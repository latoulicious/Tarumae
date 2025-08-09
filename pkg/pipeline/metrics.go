package pipeline

import (
	"fmt"
	"sync"
	"time"
)

// MetricType represents the type of metric
type MetricType int

const (
	CounterType MetricType = iota
	GaugeType
	HistogramType
	TimingType
)

func (mt MetricType) String() string {
	switch mt {
	case CounterType:
		return "counter"
	case GaugeType:
		return "gauge"
	case HistogramType:
		return "histogram"
	case TimingType:
		return "timing"
	default:
		return "unknown"
	}
}

// Metric represents a single metric measurement
type Metric struct {
	Name      string                 `json:"name"`
	Type      MetricType             `json:"type"`
	Value     float64                `json:"value"`
	Tags      map[string]string      `json:"tags,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// MetricSnapshot represents a snapshot of metrics at a point in time
type MetricSnapshot struct {
	Timestamp time.Time         `json:"timestamp"`
	Metrics   map[string]Metric `json:"metrics"`
}

// BasicMetricsCollector implements the MetricsCollector interface
type BasicMetricsCollector struct {
	metrics map[string]Metric
	mu      sync.RWMutex
	logger  Logger
}

// NewBasicMetricsCollector creates a new basic metrics collector
func NewBasicMetricsCollector(logger Logger) *BasicMetricsCollector {
	return &BasicMetricsCollector{
		metrics: make(map[string]Metric),
		logger:  logger,
	}
}

// RecordCounter records a counter metric
func (c *BasicMetricsCollector) RecordCounter(name string, value int64, tags map[string]string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	key := c.buildMetricKey(name, tags)
	existing, exists := c.metrics[key]
	
	var newValue float64
	if exists && existing.Type == CounterType {
		newValue = existing.Value + float64(value)
	} else {
		newValue = float64(value)
	}
	
	c.metrics[key] = Metric{
		Name:      name,
		Type:      CounterType,
		Value:     newValue,
		Tags:      c.copyTags(tags),
		Timestamp: time.Now(),
	}
	
	c.logger.Debug("Recorded counter metric",
		String("name", name),
		Int64("value", value),
		Float64("total", newValue),
		Any("tags", tags),
	)
}

// RecordGauge records a gauge metric
func (c *BasicMetricsCollector) RecordGauge(name string, value float64, tags map[string]string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	key := c.buildMetricKey(name, tags)
	c.metrics[key] = Metric{
		Name:      name,
		Type:      GaugeType,
		Value:     value,
		Tags:      c.copyTags(tags),
		Timestamp: time.Now(),
	}
	
	c.logger.Debug("Recorded gauge metric",
		String("name", name),
		Float64("value", value),
		Any("tags", tags),
	)
}

// RecordHistogram records a histogram metric
func (c *BasicMetricsCollector) RecordHistogram(name string, value float64, tags map[string]string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	key := c.buildMetricKey(name, tags)
	existing, exists := c.metrics[key]
	
	var metadata map[string]interface{}
	if exists && existing.Type == HistogramType {
		metadata = existing.Metadata
		if metadata == nil {
			metadata = make(map[string]interface{})
		}
		
		// Update histogram statistics
		count, _ := metadata["count"].(float64)
		sum, _ := metadata["sum"].(float64)
		min, hasMin := metadata["min"].(float64)
		max, hasMax := metadata["max"].(float64)
		
		count++
		sum += value
		
		if !hasMin || value < min {
			min = value
		}
		if !hasMax || value > max {
			max = value
		}
		
		metadata["count"] = count
		metadata["sum"] = sum
		metadata["min"] = min
		metadata["max"] = max
		metadata["avg"] = sum / count
	} else {
		metadata = map[string]interface{}{
			"count": 1.0,
			"sum":   value,
			"min":   value,
			"max":   value,
			"avg":   value,
		}
	}
	
	c.metrics[key] = Metric{
		Name:      name,
		Type:      HistogramType,
		Value:     value,
		Tags:      c.copyTags(tags),
		Timestamp: time.Now(),
		Metadata:  metadata,
	}
	
	c.logger.Debug("Recorded histogram metric",
		String("name", name),
		Float64("value", value),
		Any("tags", tags),
		Any("stats", metadata),
	)
}

// RecordTiming records a timing metric
func (c *BasicMetricsCollector) RecordTiming(name string, duration time.Duration, tags map[string]string) {
	c.RecordHistogram(name, float64(duration.Nanoseconds())/1e6, tags) // Convert to milliseconds
	
	c.logger.Debug("Recorded timing metric",
		String("name", name),
		Duration("duration", duration),
		Any("tags", tags),
	)
}

// GetMetric retrieves a specific metric
func (c *BasicMetricsCollector) GetMetric(name string, tags map[string]string) (Metric, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	key := c.buildMetricKey(name, tags)
	metric, exists := c.metrics[key]
	return metric, exists
}

// GetAllMetrics returns a snapshot of all current metrics
func (c *BasicMetricsCollector) GetAllMetrics() MetricSnapshot {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	snapshot := MetricSnapshot{
		Timestamp: time.Now(),
		Metrics:   make(map[string]Metric),
	}
	
	for key, metric := range c.metrics {
		snapshot.Metrics[key] = metric
	}
	
	return snapshot
}

// GetMetricsByName returns all metrics with the given name
func (c *BasicMetricsCollector) GetMetricsByName(name string) []Metric {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	var metrics []Metric
	for _, metric := range c.metrics {
		if metric.Name == name {
			metrics = append(metrics, metric)
		}
	}
	
	return metrics
}

// Reset clears all metrics
func (c *BasicMetricsCollector) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.metrics = make(map[string]Metric)
	c.logger.Debug("Reset all metrics")
}

// buildMetricKey creates a unique key for a metric based on name and tags
func (c *BasicMetricsCollector) buildMetricKey(name string, tags map[string]string) string {
	if len(tags) == 0 {
		return name
	}
	
	key := name
	for k, v := range tags {
		key += fmt.Sprintf(",%s=%s", k, v)
	}
	return key
}

// copyTags creates a copy of the tags map
func (c *BasicMetricsCollector) copyTags(tags map[string]string) map[string]string {
	if tags == nil {
		return nil
	}
	
	copy := make(map[string]string)
	for k, v := range tags {
		copy[k] = v
	}
	return copy
}

// PipelineMetricsCollector is a specialized metrics collector for pipeline metrics
type PipelineMetricsCollector struct {
	*BasicMetricsCollector
	pipelineID string
}

// NewPipelineMetricsCollector creates a new pipeline-specific metrics collector
func NewPipelineMetricsCollector(pipelineID string, logger Logger) *PipelineMetricsCollector {
	return &PipelineMetricsCollector{
		BasicMetricsCollector: NewBasicMetricsCollector(logger),
		pipelineID:           pipelineID,
	}
}

// RecordPipelineCounter records a counter with pipeline tags
func (c *PipelineMetricsCollector) RecordPipelineCounter(name string, value int64, tags map[string]string) {
	pipelineTags := c.addPipelineTags(tags)
	c.RecordCounter(name, value, pipelineTags)
}

// RecordPipelineGauge records a gauge with pipeline tags
func (c *PipelineMetricsCollector) RecordPipelineGauge(name string, value float64, tags map[string]string) {
	pipelineTags := c.addPipelineTags(tags)
	c.RecordGauge(name, value, pipelineTags)
}

// RecordPipelineHistogram records a histogram with pipeline tags
func (c *PipelineMetricsCollector) RecordPipelineHistogram(name string, value float64, tags map[string]string) {
	pipelineTags := c.addPipelineTags(tags)
	c.RecordHistogram(name, value, pipelineTags)
}

// RecordPipelineTiming records a timing with pipeline tags
func (c *PipelineMetricsCollector) RecordPipelineTiming(name string, duration time.Duration, tags map[string]string) {
	pipelineTags := c.addPipelineTags(tags)
	c.RecordTiming(name, duration, pipelineTags)
}

// addPipelineTags adds pipeline-specific tags to the provided tags
func (c *PipelineMetricsCollector) addPipelineTags(tags map[string]string) map[string]string {
	pipelineTags := make(map[string]string)
	
	// Add pipeline ID
	pipelineTags["pipeline_id"] = c.pipelineID
	
	// Copy existing tags
	for k, v := range tags {
		pipelineTags[k] = v
	}
	
	return pipelineTags
}

// Common pipeline metrics helpers

// RecordStreamLatency records stream latency metric
func (c *PipelineMetricsCollector) RecordStreamLatency(latency time.Duration) {
	c.RecordPipelineTiming("pipeline.stream.latency", latency, nil)
}

// RecordProcessingDelay records processing delay metric
func (c *PipelineMetricsCollector) RecordProcessingDelay(delay time.Duration) {
	c.RecordPipelineTiming("pipeline.processing.delay", delay, nil)
}

// RecordEncodingTime records encoding time metric
func (c *PipelineMetricsCollector) RecordEncodingTime(duration time.Duration) {
	c.RecordPipelineTiming("pipeline.encoding.time", duration, nil)
}

// RecordError records an error metric
func (c *PipelineMetricsCollector) RecordError(errorType string, category ErrorCategory) {
	tags := map[string]string{
		"error_type": errorType,
		"category":   category.String(),
	}
	c.RecordPipelineCounter("pipeline.errors.total", 1, tags)
}

// RecordRecoveryAttempt records a recovery attempt metric
func (c *PipelineMetricsCollector) RecordRecoveryAttempt(strategy string, success bool) {
	tags := map[string]string{
		"strategy": strategy,
		"success":  fmt.Sprintf("%t", success),
	}
	c.RecordPipelineCounter("pipeline.recovery.attempts", 1, tags)
}

// RecordResourceUsage records resource usage metrics
func (c *PipelineMetricsCollector) RecordResourceUsage(cpuUsage float64, memoryUsage int64, networkBandwidth int64) {
	c.RecordPipelineGauge("pipeline.resources.cpu_usage", cpuUsage, nil)
	c.RecordPipelineGauge("pipeline.resources.memory_usage", float64(memoryUsage), nil)
	c.RecordPipelineGauge("pipeline.resources.network_bandwidth", float64(networkBandwidth), nil)
}

// RecordStateChange records a pipeline state change
func (c *PipelineMetricsCollector) RecordStateChange(from, to PipelineState) {
	tags := map[string]string{
		"from_state": from.String(),
		"to_state":   to.String(),
	}
	c.RecordPipelineCounter("pipeline.state.changes", 1, tags)
}