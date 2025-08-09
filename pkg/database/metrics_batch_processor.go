package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"
)

// MetricsBatchProcessor handles efficient bulk insertion of pipeline metrics
type MetricsBatchProcessor struct {
	db     *sql.DB
	config *DatabaseConfig
	logger Logger

	// Processing state
	metricBuffer    chan *PipelineMetric
	batchBuffer     []*PipelineMetric
	bufferMutex     sync.RWMutex
	processingQueue chan []*PipelineMetric
	errorRetryQueue chan []*PipelineMetric

	// Configuration
	batchSize     int
	flushInterval time.Duration
	maxRetries    int
	retryDelay    time.Duration

	// Control channels
	stopChan chan struct{}
	doneChan chan struct{}

	// State
	running  bool
	runMutex sync.RWMutex

	// Metrics
	processedCount int64
	errorCount     int64
	retryCount     int64
	statsMutex     sync.RWMutex

	// Prepared statements
	insertStmt *sql.Stmt
}

// Logger interface for the batch processor
type Logger interface {
	Printf(format string, v ...interface{})
	Errorf(format string, v ...interface{})
}

// defaultLogger implements Logger using the standard log package
type defaultLogger struct{}

func (l *defaultLogger) Printf(format string, v ...interface{}) {
	log.Printf(format, v...)
}

func (l *defaultLogger) Errorf(format string, v ...interface{}) {
	log.Printf("ERROR: "+format, v...)
}

// NewMetricsBatchProcessor creates a new metrics batch processor
func NewMetricsBatchProcessor(db *sql.DB, config *DatabaseConfig) (*MetricsBatchProcessor, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}

	if config == nil {
		config = DefaultDatabaseConfig()
	}

	processor := &MetricsBatchProcessor{
		db:              db,
		config:          config,
		logger:          &defaultLogger{},
		batchSize:       config.MetricsBatchSize,
		flushInterval:   config.MetricsFlushInterval,
		maxRetries:      3,
		retryDelay:      5 * time.Second,
		metricBuffer:    make(chan *PipelineMetric, config.MetricsBatchSize*2),
		batchBuffer:     make([]*PipelineMetric, 0, config.MetricsBatchSize),
		processingQueue: make(chan []*PipelineMetric, 10),
		errorRetryQueue: make(chan []*PipelineMetric, 5),
		stopChan:        make(chan struct{}),
		doneChan:        make(chan struct{}),
	}

	// Prepare insert statement
	if err := processor.prepareStatements(); err != nil {
		return nil, fmt.Errorf("failed to prepare statements: %w", err)
	}

	return processor, nil
}

// SetLogger sets a custom logger for the batch processor
func (p *MetricsBatchProcessor) SetLogger(logger Logger) {
	p.logger = logger
}

// Start begins the batch processing goroutines
func (p *MetricsBatchProcessor) Start() error {
	p.runMutex.Lock()
	defer p.runMutex.Unlock()

	if p.running {
		return fmt.Errorf("batch processor is already running")
	}

	// Start batch collector
	go p.runBatchCollector()

	// Start batch processor
	go p.runBatchProcessor()

	// Start retry processor
	go p.runRetryProcessor()

	p.running = true

	p.logger.Printf("MetricsBatchProcessor started with batch size %d and flush interval %v",
		p.batchSize, p.flushInterval)

	return nil
}

// Stop gracefully shuts down the batch processor
func (p *MetricsBatchProcessor) Stop() error {
	p.runMutex.Lock()
	defer p.runMutex.Unlock()

	if !p.running {
		return nil // Already stopped or never started
	}

	// Close stop channel only if not already closed
	select {
	case <-p.stopChan:
		// Already closed
	default:
		close(p.stopChan)
	}

	// Wait for processing to complete with timeout
	select {
	case <-p.doneChan:
		p.logger.Printf("MetricsBatchProcessor stopped gracefully")
	case <-time.After(5 * time.Second): // Reduced timeout for tests
		p.logger.Errorf("MetricsBatchProcessor stop timeout")
	}

	p.running = false

	// Close prepared statements
	if p.insertStmt != nil {
		p.insertStmt.Close()
	}

	return nil
}

// AddMetric adds a metric to the processing queue
func (p *MetricsBatchProcessor) AddMetric(metric *PipelineMetric) error {
	select {
	case p.metricBuffer <- metric:
		return nil
	case <-p.stopChan:
		return fmt.Errorf("batch processor is stopping")
	default:
		return fmt.Errorf("metric buffer is full")
	}
}

// AddMetrics adds multiple metrics to the processing queue
func (p *MetricsBatchProcessor) AddMetrics(metrics []*PipelineMetric) error {
	for _, metric := range metrics {
		if err := p.AddMetric(metric); err != nil {
			return fmt.Errorf("failed to add metric: %w", err)
		}
	}
	return nil
}

// Flush forces processing of any buffered metrics
func (p *MetricsBatchProcessor) Flush() error {
	p.bufferMutex.Lock()
	defer p.bufferMutex.Unlock()

	if len(p.batchBuffer) > 0 {
		batch := make([]*PipelineMetric, len(p.batchBuffer))
		copy(batch, p.batchBuffer)
		p.batchBuffer = p.batchBuffer[:0]

		select {
		case p.processingQueue <- batch:
			return nil
		case <-time.After(5 * time.Second):
			return fmt.Errorf("flush timeout")
		}
	}

	return nil
}

// GetStats returns processing statistics
func (p *MetricsBatchProcessor) GetStats() *BatchProcessorStats {
	p.statsMutex.RLock()
	defer p.statsMutex.RUnlock()

	p.bufferMutex.RLock()
	bufferSize := len(p.batchBuffer)
	p.bufferMutex.RUnlock()

	return &BatchProcessorStats{
		ProcessedCount:   p.processedCount,
		ErrorCount:       p.errorCount,
		RetryCount:       p.retryCount,
		BufferSize:       bufferSize,
		QueueSize:        len(p.processingQueue),
		RetryQueueSize:   len(p.errorRetryQueue),
		MetricBufferSize: len(p.metricBuffer),
	}
}

// prepareStatements prepares SQL statements for batch operations
func (p *MetricsBatchProcessor) prepareStatements() error {
	var err error

	p.insertStmt, err = p.db.Prepare(`
		INSERT INTO pipeline_metrics (pipeline_id, metric_name, metric_type, metric_value, tags, metadata, timestamp)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare insert statement: %w", err)
	}

	return nil
}

// runBatchCollector collects metrics into batches
func (p *MetricsBatchProcessor) runBatchCollector() {
	ticker := time.NewTicker(p.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case metric := <-p.metricBuffer:
			p.bufferMutex.Lock()
			p.batchBuffer = append(p.batchBuffer, metric)

			// Check if batch is full
			if len(p.batchBuffer) >= p.batchSize {
				batch := make([]*PipelineMetric, len(p.batchBuffer))
				copy(batch, p.batchBuffer)
				p.batchBuffer = p.batchBuffer[:0]
				p.bufferMutex.Unlock()

				// Send batch for processing
				select {
				case p.processingQueue <- batch:
				case <-p.stopChan:
					return
				}
			} else {
				p.bufferMutex.Unlock()
			}

		case <-ticker.C:
			// Flush any pending metrics
			if err := p.Flush(); err != nil {
				p.logger.Errorf("Failed to flush metrics: %v", err)
			}

		case <-p.stopChan:
			// Final flush before stopping
			p.Flush()
			return
		}
	}
}

// runBatchProcessor processes batches of metrics
func (p *MetricsBatchProcessor) runBatchProcessor() {
	defer close(p.doneChan)

	for {
		select {
		case batch := <-p.processingQueue:
			if err := p.processBatch(batch); err != nil {
				p.logger.Errorf("Failed to process batch: %v", err)

				// Add to retry queue
				select {
				case p.errorRetryQueue <- batch:
				default:
					p.logger.Errorf("Retry queue full, dropping batch of %d metrics", len(batch))
					p.incrementErrorCount(int64(len(batch)))
				}
			} else {
				p.incrementProcessedCount(int64(len(batch)))
			}

		case <-p.stopChan:
			// Process remaining batches
			for {
				select {
				case batch := <-p.processingQueue:
					if err := p.processBatch(batch); err != nil {
						p.logger.Errorf("Failed to process final batch: %v", err)
					}
				default:
					return
				}
			}
		}
	}
}

// runRetryProcessor handles failed batch retries
func (p *MetricsBatchProcessor) runRetryProcessor() {
	for {
		select {
		case batch := <-p.errorRetryQueue:
			p.retryBatch(batch)

		case <-p.stopChan:
			return
		}
	}
}

// processBatch processes a batch of metrics
func (p *MetricsBatchProcessor) processBatch(batch []*PipelineMetric) error {
	if len(batch) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt := tx.StmtContext(ctx, p.insertStmt)

	for _, metric := range batch {
		tagsJSON, err := json.Marshal(metric.Tags)
		if err != nil {
			return fmt.Errorf("failed to marshal tags: %w", err)
		}

		metadataJSON, err := json.Marshal(metric.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}

		_, err = stmt.ExecContext(ctx,
			metric.PipelineID,
			metric.MetricName,
			metric.MetricType,
			metric.MetricValue,
			string(tagsJSON),
			string(metadataJSON),
			metric.Timestamp,
		)

		if err != nil {
			return fmt.Errorf("failed to insert metric: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit batch: %w", err)
	}

	return nil
}

// retryBatch retries a failed batch with exponential backoff
func (p *MetricsBatchProcessor) retryBatch(batch []*PipelineMetric) {
	for attempt := 1; attempt <= p.maxRetries; attempt++ {
		// Wait with exponential backoff
		if attempt > 1 {
			delay := time.Duration(attempt-1) * p.retryDelay
			time.Sleep(delay)
		}

		if err := p.processBatch(batch); err != nil {
			p.logger.Errorf("Retry attempt %d failed for batch of %d metrics: %v",
				attempt, len(batch), err)
			p.incrementRetryCount(1)

			if attempt == p.maxRetries {
				p.logger.Errorf("Max retries exceeded, dropping batch of %d metrics", len(batch))
				p.incrementErrorCount(int64(len(batch)))
				return
			}
		} else {
			p.incrementProcessedCount(int64(len(batch)))
			return
		}
	}
}

// incrementProcessedCount safely increments the processed count
func (p *MetricsBatchProcessor) incrementProcessedCount(count int64) {
	p.statsMutex.Lock()
	p.processedCount += count
	p.statsMutex.Unlock()
}

// incrementErrorCount safely increments the error count
func (p *MetricsBatchProcessor) incrementErrorCount(count int64) {
	p.statsMutex.Lock()
	p.errorCount += count
	p.statsMutex.Unlock()
}

// incrementRetryCount safely increments the retry count
func (p *MetricsBatchProcessor) incrementRetryCount(count int64) {
	p.statsMutex.Lock()
	p.retryCount += count
	p.statsMutex.Unlock()
}

// BatchProcessorStats holds statistics about batch processing
type BatchProcessorStats struct {
	ProcessedCount   int64 `json:"processed_count"`
	ErrorCount       int64 `json:"error_count"`
	RetryCount       int64 `json:"retry_count"`
	BufferSize       int   `json:"buffer_size"`
	QueueSize        int   `json:"queue_size"`
	RetryQueueSize   int   `json:"retry_queue_size"`
	MetricBufferSize int   `json:"metric_buffer_size"`
}
