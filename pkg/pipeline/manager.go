package pipeline

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// AudioPipelineManager is the enhanced central coordinator for the audio pipeline
type AudioPipelineManager struct {
	// Configuration
	config *PipelineConfig
	
	// Core components (interfaces to be implemented in later tasks)
	streamAcquisition StreamAcquisition
	streamProcessor   StreamProcessor
	audioEncoder      AudioEncoder
	discordStreamer   DiscordStreamer
	
	// Management components (interfaces to be implemented in later tasks)
	healthChecker   []HealthCheck
	recoveryManager RecoveryStrategy
	errorClassifier ErrorClassifier
	userNotifier    UserNotifier
	resourceManager ResourceManager
	
	// State management
	state      PipelineState
	stateMutex sync.RWMutex
	
	// Monitoring and logging
	metrics *PipelineMetricsCollector
	logger  Logger
	
	// Control channels
	controlChan chan ControlMessage
	errorChan   chan *PipelineError
	stateChan   chan StateChange
	
	// Context management
	ctx    context.Context
	cancel context.CancelFunc
	
	// Pipeline metadata
	pipelineID string
	startTime  time.Time
}

// NewAudioPipelineManager creates a new enhanced audio pipeline manager
func NewAudioPipelineManager(config *PipelineConfig, logger Logger) (*AudioPipelineManager, error) {
	if config == nil {
		config = DefaultPipelineConfig()
	}
	
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}
	
	if logger == nil {
		logger = DefaultLogger()
	}
	
	pipelineID := fmt.Sprintf("pipeline-%d", time.Now().UnixNano())
	
	ctx, cancel := context.WithCancel(context.Background())
	
	manager := &AudioPipelineManager{
		config:      config,
		state:       StateIdle,
		metrics:     NewPipelineMetricsCollector(pipelineID, logger),
		logger:      logger.With(String("component", "pipeline_manager"), String("pipeline_id", pipelineID)),
		controlChan: make(chan ControlMessage, 100),
		errorChan:   make(chan *PipelineError, 100),
		stateChan:   make(chan StateChange, 100),
		ctx:         ctx,
		cancel:      cancel,
		pipelineID:  pipelineID,
	}
	
	manager.logger.Info("Created new audio pipeline manager",
		String("pipeline_id", pipelineID),
		Any("config", config),
	)
	
	return manager, nil
}

// Start starts the audio pipeline with the given stream URL
func (apm *AudioPipelineManager) Start(ctx context.Context, streamURL string) error {
	apm.stateMutex.Lock()
	defer apm.stateMutex.Unlock()
	
	if apm.state != StateIdle {
		return fmt.Errorf("pipeline is not in idle state, current state: %s", apm.state)
	}
	
	apm.logger.Info("Starting audio pipeline", String("stream_url", streamURL))
	
	// Record start time
	apm.startTime = time.Now()
	
	// Transition to initializing state
	apm.changeState(StateInitializing, "pipeline start requested")
	
	// Start control loop
	go apm.controlLoop()
	
	// TODO: In later tasks, this will initialize and start all components
	// For now, we just simulate the initialization
	apm.logger.Info("Pipeline initialization complete")
	apm.changeState(StateStreaming, "initialization complete")
	
	return nil
}

// Stop stops the audio pipeline
func (apm *AudioPipelineManager) Stop() error {
	apm.stateMutex.Lock()
	defer apm.stateMutex.Unlock()
	
	if apm.state == StateIdle || apm.state == StateStopping {
		return nil
	}
	
	apm.logger.Info("Stopping audio pipeline")
	
	apm.changeState(StateStopping, "stop requested")
	
	// Cancel context to stop all operations
	apm.cancel()
	
	// TODO: In later tasks, this will properly stop all components
	
	apm.changeState(StateIdle, "pipeline stopped")
	
	return nil
}

// Pause pauses the audio pipeline (placeholder for future implementation)
func (apm *AudioPipelineManager) Pause() error {
	apm.stateMutex.Lock()
	defer apm.stateMutex.Unlock()
	
	if apm.state != StateStreaming {
		return fmt.Errorf("cannot pause pipeline in state: %s", apm.state)
	}
	
	apm.logger.Info("Pausing audio pipeline")
	apm.changeState(StatePaused, "pause requested")
	
	// TODO: Implement pause functionality in later tasks
	
	return nil
}

// Resume resumes the audio pipeline (placeholder for future implementation)
func (apm *AudioPipelineManager) Resume() error {
	apm.stateMutex.Lock()
	defer apm.stateMutex.Unlock()
	
	if apm.state != StatePaused {
		return fmt.Errorf("cannot resume pipeline in state: %s", apm.state)
	}
	
	apm.logger.Info("Resuming audio pipeline")
	apm.changeState(StateStreaming, "resume requested")
	
	// TODO: Implement resume functionality in later tasks
	
	return nil
}

// GetState returns the current pipeline state
func (apm *AudioPipelineManager) GetState() PipelineState {
	apm.stateMutex.RLock()
	defer apm.stateMutex.RUnlock()
	return apm.state
}

// GetMetrics returns the current pipeline metrics
func (apm *AudioPipelineManager) GetMetrics() *PipelineMetrics {
	snapshot := apm.metrics.GetAllMetrics()
	
	// Convert to PipelineMetrics format
	metrics := NewPipelineMetrics()
	
	// TODO: In later tasks, populate metrics from actual components
	// For now, return basic metrics
	metrics.LastUpdated = snapshot.Timestamp
	
	return metrics
}

// IsHealthy returns whether the pipeline is healthy
func (apm *AudioPipelineManager) IsHealthy() bool {
	// TODO: In later tasks, this will check all health checks
	// For now, return true if pipeline is in a good state
	state := apm.GetState()
	return state == StateStreaming || state == StateIdle
}

// GetPipelineID returns the unique pipeline identifier
func (apm *AudioPipelineManager) GetPipelineID() string {
	return apm.pipelineID
}

// GetUptime returns how long the pipeline has been running
func (apm *AudioPipelineManager) GetUptime() time.Duration {
	if apm.startTime.IsZero() {
		return 0
	}
	return time.Since(apm.startTime)
}

// changeState changes the pipeline state and notifies listeners
func (apm *AudioPipelineManager) changeState(newState PipelineState, reason string) {
	oldState := apm.state
	apm.state = newState
	
	change := StateChange{
		From:      oldState,
		To:        newState,
		Timestamp: time.Now(),
		Reason:    reason,
	}
	
	apm.logger.Info("Pipeline state changed",
		String("from", oldState.String()),
		String("to", newState.String()),
		String("reason", reason),
	)
	
	// Record state change metric
	apm.metrics.RecordStateChange(oldState, newState)
	
	// Send state change notification (non-blocking)
	select {
	case apm.stateChan <- change:
	default:
		apm.logger.Warn("State change channel full, dropping notification")
	}
}

// controlLoop handles control messages and coordinates pipeline operations
func (apm *AudioPipelineManager) controlLoop() {
	apm.logger.Debug("Starting pipeline control loop")
	
	defer func() {
		apm.logger.Debug("Pipeline control loop stopped")
	}()
	
	for {
		select {
		case <-apm.ctx.Done():
			return
			
		case msg := <-apm.controlChan:
			apm.handleControlMessage(msg)
			
		case err := <-apm.errorChan:
			apm.handleError(err)
			
		case change := <-apm.stateChan:
			apm.handleStateChange(change)
		}
	}
}

// handleControlMessage processes control messages
func (apm *AudioPipelineManager) handleControlMessage(msg ControlMessage) {
	apm.logger.Debug("Received control message",
		String("type", msg.Type),
		Any("data", msg.Data),
	)
	
	// TODO: In later tasks, implement specific control message handling
}

// handleError processes pipeline errors
func (apm *AudioPipelineManager) handleError(err *PipelineError) {
	apm.logger.Error("Pipeline error occurred",
		Error(err.Err),
		String("category", err.Category.String()),
		String("severity", err.Severity.String()),
		Bool("retryable", err.Retryable),
		Any("context", err.Context),
	)
	
	// Record error metric
	apm.metrics.RecordError(err.Err.Error(), err.Category)
	
	// TODO: In later tasks, implement error recovery logic
	if err.Severity == SeverityCritical {
		apm.logger.Error("Critical error, stopping pipeline")
		apm.changeState(StateFailed, fmt.Sprintf("critical error: %s", err.Err.Error()))
	}
}

// handleStateChange processes state changes
func (apm *AudioPipelineManager) handleStateChange(change StateChange) {
	apm.logger.Debug("Processing state change",
		String("from", change.From.String()),
		String("to", change.To.String()),
		String("reason", change.Reason),
	)
	
	// TODO: In later tasks, implement state-specific logic
}

// SendControlMessage sends a control message to the pipeline
func (apm *AudioPipelineManager) SendControlMessage(msgType string, data interface{}) error {
	msg := ControlMessage{
		Type:      msgType,
		Data:      data,
		Timestamp: time.Now(),
	}
	
	select {
	case apm.controlChan <- msg:
		return nil
	case <-time.After(time.Second):
		return fmt.Errorf("timeout sending control message")
	}
}

// ReportError reports an error to the pipeline
func (apm *AudioPipelineManager) ReportError(err error, category ErrorCategory, severity ErrorSeverity) {
	pipelineErr := NewPipelineError(err, category, severity)
	
	select {
	case apm.errorChan <- pipelineErr:
	case <-time.After(time.Second):
		apm.logger.Error("Failed to report error, channel full", Error(err))
	}
}