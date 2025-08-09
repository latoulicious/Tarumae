package pipeline

import (
	"context"
	"time"
)

// HealthCheck defines the interface for pipeline health checks
type HealthCheck interface {
	Name() string
	Check(ctx context.Context) *HealthResult
	Interval() time.Duration
	Critical() bool
}

// RecoveryStrategy defines the interface for pipeline recovery strategies
type RecoveryStrategy interface {
	CanRecover(error *PipelineError) bool
	Recover(ctx context.Context, pipeline PipelineManager) error
	Priority() int
	MaxAttempts() int
}

// AcquisitionStrategy defines the interface for stream acquisition strategies
type AcquisitionStrategy interface {
	GetStreamURL(source string) (*StreamInfo, error)
	CanHandle(source string) bool
	Priority() int
}

// Logger defines the interface for structured logging
type Logger interface {
	Debug(msg string, fields ...Field)
	Info(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	Error(msg string, fields ...Field)
	Fatal(msg string, fields ...Field)
	With(fields ...Field) Logger
}

// MetricsCollector defines the interface for metrics collection
type MetricsCollector interface {
	RecordCounter(name string, value int64, tags map[string]string)
	RecordGauge(name string, value float64, tags map[string]string)
	RecordHistogram(name string, value float64, tags map[string]string)
	RecordTiming(name string, duration time.Duration, tags map[string]string)
}

// PipelineManager defines the interface for the main pipeline manager
type PipelineManager interface {
	Start(ctx context.Context, streamURL string) error
	Stop() error
	Pause() error
	Resume() error
	GetState() PipelineState
	GetMetrics() *PipelineMetrics
	IsHealthy() bool
}

// StreamAcquisition defines the interface for stream URL acquisition
type StreamAcquisition interface {
	GetStreamURL(source string) (*StreamInfo, error)
	RefreshStreamURL(info *StreamInfo) (*StreamInfo, error)
	ValidateStreamURL(url string) error
}

// StreamProcessor defines the interface for audio stream processing
type StreamProcessor interface {
	Start(ctx context.Context, streamInfo *StreamInfo) error
	Stop() error
	IsRunning() bool
	GetProcessMetrics() map[string]interface{}
}

// AudioEncoder defines the interface for audio encoding
type AudioEncoder interface {
	Encode(pcmData []int16, frameSize int) ([]byte, error)
	SetBitrate(bitrate int) error
	SetComplexity(complexity int) error
	GetEncodingMetrics() map[string]interface{}
}

// DiscordStreamer defines the interface for Discord voice streaming
type DiscordStreamer interface {
	Start(ctx context.Context) error
	Stop() error
	SendOpusFrame(data []byte) error
	IsConnected() bool
	GetConnectionMetrics() *ConnectionMetrics
}

// ErrorClassifier defines the interface for error classification
type ErrorClassifier interface {
	Classify(err error) *PipelineError
	IsRetryable(err error) bool
	GetSeverity(err error) ErrorSeverity
	GetCategory(err error) ErrorCategory
}

// UserNotifier defines the interface for user notifications
type UserNotifier interface {
	NotifyError(err *PipelineError) error
	NotifyRecovery(message string) error
	NotifyStateChange(change *StateChange) error
	NotifyQualityChange(quality *QualityMetrics) error
}

// ResourceManager defines the interface for resource management
type ResourceManager interface {
	GetCPUUsage() float64
	GetMemoryUsage() int64
	GetNetworkBandwidth() int64
	CheckResourceLimits() error
	CleanupResources() error
}

// ConfigValidator defines the interface for configuration validation
type ConfigValidator interface {
	Validate(config interface{}) error
	GetValidationErrors() []error
}