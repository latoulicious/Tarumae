package pipeline

import (
	"sync"
	"time"
)

// PipelineState represents the current state of the audio pipeline
type PipelineState int

const (
	StateIdle PipelineState = iota
	StateInitializing
	StateStreaming
	StateRecovering
	StatePaused
	StateStopping
	StateFailed
)

func (s PipelineState) String() string {
	switch s {
	case StateIdle:
		return "idle"
	case StateInitializing:
		return "initializing"
	case StateStreaming:
		return "streaming"
	case StateRecovering:
		return "recovering"
	case StatePaused:
		return "paused"
	case StateStopping:
		return "stopping"
	case StateFailed:
		return "failed"
	default:
		return "unknown"
	}
}

// ErrorSeverity represents the severity level of pipeline errors
type ErrorSeverity int

const (
	SeverityLow ErrorSeverity = iota
	SeverityMedium
	SeverityHigh
	SeverityCritical
)

func (s ErrorSeverity) String() string {
	switch s {
	case SeverityLow:
		return "low"
	case SeverityMedium:
		return "medium"
	case SeverityHigh:
		return "high"
	case SeverityCritical:
		return "critical"
	default:
		return "unknown"
	}
}

// ErrorCategory represents the category of pipeline errors
type ErrorCategory int

const (
	CategoryNetwork ErrorCategory = iota
	CategoryStream
	CategoryProcess
	CategoryVoice
	CategorySystem
	CategoryUnknown
)

func (c ErrorCategory) String() string {
	switch c {
	case CategoryNetwork:
		return "network"
	case CategoryStream:
		return "stream"
	case CategoryProcess:
		return "process"
	case CategoryVoice:
		return "voice"
	case CategorySystem:
		return "system"
	case CategoryUnknown:
		return "unknown"
	default:
		return "unknown"
	}
}

// PipelineError represents an error in the audio pipeline with classification
type PipelineError struct {
	Err       error
	Category  ErrorCategory
	Severity  ErrorSeverity
	Timestamp time.Time
	Context   map[string]interface{}
	Retryable bool
}

func (pe *PipelineError) Error() string {
	return pe.Err.Error()
}

// NewPipelineError creates a new classified pipeline error
func NewPipelineError(err error, category ErrorCategory, severity ErrorSeverity) *PipelineError {
	return &PipelineError{
		Err:       err,
		Category:  category,
		Severity:  severity,
		Timestamp: time.Now(),
		Context:   make(map[string]interface{}),
		Retryable: severity <= SeverityMedium,
	}
}

// StateChange represents a pipeline state transition
type StateChange struct {
	From      PipelineState
	To        PipelineState
	Timestamp time.Time
	Reason    string
}

// ControlMessage represents control messages for pipeline management
type ControlMessage struct {
	Type      string
	Data      interface{}
	Timestamp time.Time
}

// StreamInfo contains metadata about an audio stream
type StreamInfo struct {
	URL               string
	OriginalURL       string
	VideoID           string
	Title             string
	Duration          time.Duration
	Quality           string
	Format            string
	Bitrate           int
	Metadata          map[string]interface{}
	AcquiredAt        time.Time
	AcquisitionMethod string
	ExpiresAt         time.Time
	Validated         bool
	ValidationErrors  []error
}

// QualityMetrics represents audio quality measurements
type QualityMetrics struct {
	Bitrate           int
	SampleRate        int
	Channels          int
	Complexity        int
	PacketLoss        float64
	Jitter            time.Duration
	LastUpdated       time.Time
}

// ConnectionMetrics represents voice connection quality measurements
type ConnectionMetrics struct {
	Latency           time.Duration
	PacketsSent       int64
	PacketsLost       int64
	BytesSent         int64
	ConnectionUptime  time.Duration
	ReconnectCount    int
	LastReconnect     time.Time
	LastUpdated       time.Time
}

// PipelineMetrics contains comprehensive pipeline performance metrics
type PipelineMetrics struct {
	// Performance metrics
	StreamLatency     time.Duration
	ProcessingDelay   time.Duration
	EncodingTime      time.Duration
	
	// Quality metrics
	AudioQuality      QualityMetrics
	ConnectionQuality ConnectionMetrics
	
	// Error metrics
	ErrorCounts       map[string]int
	RecoverySuccess   int
	RecoveryFailures  int
	
	// Resource metrics
	CPUUsage          float64
	MemoryUsage       int64
	NetworkBandwidth  int64
	
	// Timestamps
	LastUpdated       time.Time
	
	// Thread safety
	mu                sync.RWMutex
}

// NewPipelineMetrics creates a new metrics instance
func NewPipelineMetrics() *PipelineMetrics {
	return &PipelineMetrics{
		ErrorCounts: make(map[string]int),
		LastUpdated: time.Now(),
	}
}

// UpdateStreamLatency updates the stream latency metric
func (pm *PipelineMetrics) UpdateStreamLatency(latency time.Duration) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.StreamLatency = latency
	pm.LastUpdated = time.Now()
}

// IncrementErrorCount increments the error count for a specific error type
func (pm *PipelineMetrics) IncrementErrorCount(errorType string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.ErrorCounts[errorType]++
	pm.LastUpdated = time.Now()
}

// GetErrorCount returns the error count for a specific error type
func (pm *PipelineMetrics) GetErrorCount(errorType string) int {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.ErrorCounts[errorType]
}

// HealthResult represents the result of a health check
type HealthResult struct {
	Name      string
	Healthy   bool
	Message   string
	Timestamp time.Time
	Duration  time.Duration
	Metadata  map[string]interface{}
}

// NewHealthResult creates a new health check result
func NewHealthResult(name string, healthy bool, message string) *HealthResult {
	return &HealthResult{
		Name:      name,
		Healthy:   healthy,
		Message:   message,
		Timestamp: time.Now(),
		Metadata:  make(map[string]interface{}),
	}
}