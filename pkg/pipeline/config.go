package pipeline

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// PipelineConfig contains comprehensive configuration for the audio pipeline
type PipelineConfig struct {
	StreamAcquisition StreamAcquisitionConfig `json:"stream_acquisition"`
	FFmpeg           FFmpegConfig            `json:"ffmpeg"`
	Opus             OpusConfig              `json:"opus"`
	Health           HealthConfig            `json:"health"`
	Recovery         RecoveryConfig          `json:"recovery"`
	Resources        ResourceConfig          `json:"resources"`
	Logging          LoggingConfig           `json:"logging"`
	Discord          DiscordConfig           `json:"discord"`
}

// StreamAcquisitionConfig contains configuration for stream acquisition
type StreamAcquisitionConfig struct {
	MaxRetries       int           `json:"max_retries"`
	RetryDelay       time.Duration `json:"retry_delay"`
	CacheTimeout     time.Duration `json:"cache_timeout"`
	ValidationTimeout time.Duration `json:"validation_timeout"`
	UserAgent        string        `json:"user_agent"`
	Strategies       []string      `json:"strategies"`
}

// FFmpegConfig contains configuration for FFmpeg processing
type FFmpegConfig struct {
	BinaryPath       string            `json:"binary_path"`
	Args             []string          `json:"args"`
	BufferSize       string            `json:"buffer_size"`
	ReconnectOptions map[string]string `json:"reconnect_options"`
	Timeout          time.Duration     `json:"timeout"`
	MaxRestarts      int               `json:"max_restarts"`
}

// OpusConfig contains configuration for Opus encoding
type OpusConfig struct {
	SampleRate       int  `json:"sample_rate"`
	Channels         int  `json:"channels"`
	Bitrate          int  `json:"bitrate"`
	Complexity       int  `json:"complexity"`
	FrameSize        int  `json:"frame_size"`
	AdaptiveMode     bool `json:"adaptive_mode"`
	MaxBitrate       int  `json:"max_bitrate"`
	MinBitrate       int  `json:"min_bitrate"`
}

// HealthConfig contains configuration for health monitoring
type HealthConfig struct {
	Enabled          bool          `json:"enabled"`
	CheckInterval    time.Duration `json:"check_interval"`
	FailureThreshold int           `json:"failure_threshold"`
	Checks           []string      `json:"checks"`
	AlertThresholds  map[string]float64 `json:"alert_thresholds"`
}

// RecoveryConfig contains configuration for recovery strategies
type RecoveryConfig struct {
	Enabled          bool          `json:"enabled"`
	MaxAttempts      int           `json:"max_attempts"`
	BackoffStrategy  string        `json:"backoff_strategy"`
	InitialDelay     time.Duration `json:"initial_delay"`
	MaxDelay         time.Duration `json:"max_delay"`
	Strategies       []string      `json:"strategies"`
}

// ResourceConfig contains configuration for resource management
type ResourceConfig struct {
	MaxCPUUsage      float64 `json:"max_cpu_usage"`
	MaxMemoryUsage   int64   `json:"max_memory_usage"`
	MaxBandwidth     int64   `json:"max_bandwidth"`
	MonitorInterval  time.Duration `json:"monitor_interval"`
	CleanupInterval  time.Duration `json:"cleanup_interval"`
}

// LoggingConfig contains configuration for logging
type LoggingConfig struct {
	Level            string `json:"level"`
	Format           string `json:"format"`
	Output           string `json:"output"`
	EnableMetrics    bool   `json:"enable_metrics"`
	EnableTracing    bool   `json:"enable_tracing"`
	RotateSize       int64  `json:"rotate_size"`
	RotateCount      int    `json:"rotate_count"`
}

// DiscordConfig contains configuration for Discord integration
type DiscordConfig struct {
	ReconnectAttempts int           `json:"reconnect_attempts"`
	ReconnectDelay    time.Duration `json:"reconnect_delay"`
	SpeakingTimeout   time.Duration `json:"speaking_timeout"`
	BufferSize        int           `json:"buffer_size"`
	SendTimeout       time.Duration `json:"send_timeout"`
}

// DefaultPipelineConfig returns a configuration with sensible defaults
func DefaultPipelineConfig() *PipelineConfig {
	return &PipelineConfig{
		StreamAcquisition: StreamAcquisitionConfig{
			MaxRetries:        3,
			RetryDelay:        2 * time.Second,
			CacheTimeout:      30 * time.Minute,
			ValidationTimeout: 5 * time.Second,
			UserAgent:         "HKTM-Bot/1.0",
			Strategies:        []string{"yt-dlp-default", "yt-dlp-fallback"},
		},
		FFmpeg: FFmpegConfig{
			BinaryPath:  "ffmpeg",
			BufferSize:  "64k",
			Timeout:     30 * time.Second,
			MaxRestarts: 3,
			Args: []string{
				"-reconnect", "1",
				"-reconnect_streamed", "1",
				"-reconnect_delay_max", "5",
			},
			ReconnectOptions: map[string]string{
				"reconnect":           "1",
				"reconnect_streamed":  "1",
				"reconnect_delay_max": "5",
			},
		},
		Opus: OpusConfig{
			SampleRate:   48000,
			Channels:     2,
			Bitrate:      128000,
			Complexity:   10,
			FrameSize:    960,
			AdaptiveMode: true,
			MaxBitrate:   256000,
			MinBitrate:   64000,
		},
		Health: HealthConfig{
			Enabled:          true,
			CheckInterval:    5 * time.Second,
			FailureThreshold: 3,
			Checks:           []string{"stream", "process", "voice", "resources"},
			AlertThresholds: map[string]float64{
				"cpu_usage":    80.0,
				"memory_usage": 85.0,
				"latency_ms":   500.0,
			},
		},
		Recovery: RecoveryConfig{
			Enabled:         true,
			MaxAttempts:     3,
			BackoffStrategy: "exponential",
			InitialDelay:    1 * time.Second,
			MaxDelay:        30 * time.Second,
			Strategies:      []string{"quick-retry", "stream-refresh", "process-restart"},
		},
		Resources: ResourceConfig{
			MaxCPUUsage:     80.0,
			MaxMemoryUsage:  100 * 1024 * 1024, // 100MB
			MaxBandwidth:    10 * 1024 * 1024,  // 10MB/s
			MonitorInterval: 10 * time.Second,
			CleanupInterval: 5 * time.Minute,
		},
		Logging: LoggingConfig{
			Level:         "info",
			Format:        "json",
			Output:        "stdout",
			EnableMetrics: true,
			EnableTracing: false,
			RotateSize:    10 * 1024 * 1024, // 10MB
			RotateCount:   5,
		},
		Discord: DiscordConfig{
			ReconnectAttempts: 3,
			ReconnectDelay:    2 * time.Second,
			SpeakingTimeout:   10 * time.Second,
			BufferSize:        100,
			SendTimeout:       100 * time.Millisecond,
		},
	}
}

// LoadFromEnvironment loads configuration values from environment variables
func (c *PipelineConfig) LoadFromEnvironment() {
	// Stream acquisition
	if val := os.Getenv("PIPELINE_STREAM_MAX_RETRIES"); val != "" {
		if retries, err := strconv.Atoi(val); err == nil {
			c.StreamAcquisition.MaxRetries = retries
		}
	}
	
	if val := os.Getenv("PIPELINE_STREAM_RETRY_DELAY"); val != "" {
		if delay, err := time.ParseDuration(val); err == nil {
			c.StreamAcquisition.RetryDelay = delay
		}
	}
	
	// FFmpeg
	if val := os.Getenv("PIPELINE_FFMPEG_PATH"); val != "" {
		c.FFmpeg.BinaryPath = val
	}
	
	if val := os.Getenv("PIPELINE_FFMPEG_MAX_RESTARTS"); val != "" {
		if restarts, err := strconv.Atoi(val); err == nil {
			c.FFmpeg.MaxRestarts = restarts
		}
	}
	
	// Opus
	if val := os.Getenv("PIPELINE_OPUS_BITRATE"); val != "" {
		if bitrate, err := strconv.Atoi(val); err == nil {
			c.Opus.Bitrate = bitrate
		}
	}
	
	if val := os.Getenv("PIPELINE_OPUS_COMPLEXITY"); val != "" {
		if complexity, err := strconv.Atoi(val); err == nil {
			c.Opus.Complexity = complexity
		}
	}
	
	// Health
	if val := os.Getenv("PIPELINE_HEALTH_ENABLED"); val != "" {
		c.Health.Enabled = val == "true" || val == "1"
	}
	
	if val := os.Getenv("PIPELINE_HEALTH_CHECK_INTERVAL"); val != "" {
		if interval, err := time.ParseDuration(val); err == nil {
			c.Health.CheckInterval = interval
		}
	}
	
	// Recovery
	if val := os.Getenv("PIPELINE_RECOVERY_ENABLED"); val != "" {
		c.Recovery.Enabled = val == "true" || val == "1"
	}
	
	if val := os.Getenv("PIPELINE_RECOVERY_MAX_ATTEMPTS"); val != "" {
		if attempts, err := strconv.Atoi(val); err == nil {
			c.Recovery.MaxAttempts = attempts
		}
	}
	
	// Resources
	if val := os.Getenv("PIPELINE_MAX_CPU_USAGE"); val != "" {
		if cpu, err := strconv.ParseFloat(val, 64); err == nil {
			c.Resources.MaxCPUUsage = cpu
		}
	}
	
	if val := os.Getenv("PIPELINE_MAX_MEMORY_USAGE"); val != "" {
		if memory, err := strconv.ParseInt(val, 10, 64); err == nil {
			c.Resources.MaxMemoryUsage = memory
		}
	}
	
	// Logging
	if val := os.Getenv("PIPELINE_LOG_LEVEL"); val != "" {
		c.Logging.Level = val
	}
	
	if val := os.Getenv("PIPELINE_LOG_FORMAT"); val != "" {
		c.Logging.Format = val
	}
}

// Validate validates the configuration and returns any errors
func (c *PipelineConfig) Validate() error {
	var errors []string
	
	// Validate stream acquisition
	if c.StreamAcquisition.MaxRetries < 0 {
		errors = append(errors, "stream acquisition max_retries must be >= 0")
	}
	
	if c.StreamAcquisition.RetryDelay < 0 {
		errors = append(errors, "stream acquisition retry_delay must be >= 0")
	}
	
	// Validate FFmpeg
	if c.FFmpeg.BinaryPath == "" {
		errors = append(errors, "ffmpeg binary_path cannot be empty")
	}
	
	if c.FFmpeg.MaxRestarts < 0 {
		errors = append(errors, "ffmpeg max_restarts must be >= 0")
	}
	
	// Validate Opus
	if c.Opus.SampleRate <= 0 {
		errors = append(errors, "opus sample_rate must be > 0")
	}
	
	if c.Opus.Channels <= 0 {
		errors = append(errors, "opus channels must be > 0")
	}
	
	if c.Opus.Bitrate <= 0 {
		errors = append(errors, "opus bitrate must be > 0")
	}
	
	if c.Opus.Complexity < 0 || c.Opus.Complexity > 10 {
		errors = append(errors, "opus complexity must be between 0 and 10")
	}
	
	// Validate health
	if c.Health.CheckInterval <= 0 {
		errors = append(errors, "health check_interval must be > 0")
	}
	
	if c.Health.FailureThreshold <= 0 {
		errors = append(errors, "health failure_threshold must be > 0")
	}
	
	// Validate recovery
	if c.Recovery.MaxAttempts < 0 {
		errors = append(errors, "recovery max_attempts must be >= 0")
	}
	
	if c.Recovery.InitialDelay < 0 {
		errors = append(errors, "recovery initial_delay must be >= 0")
	}
	
	// Validate resources
	if c.Resources.MaxCPUUsage < 0 || c.Resources.MaxCPUUsage > 100 {
		errors = append(errors, "resources max_cpu_usage must be between 0 and 100")
	}
	
	if c.Resources.MaxMemoryUsage < 0 {
		errors = append(errors, "resources max_memory_usage must be >= 0")
	}
	
	// Validate logging
	validLogLevels := map[string]bool{
		"debug": true, "info": true, "warn": true, "error": true, "fatal": true,
	}
	if !validLogLevels[c.Logging.Level] {
		errors = append(errors, "logging level must be one of: debug, info, warn, error, fatal")
	}
	
	validLogFormats := map[string]bool{
		"json": true, "text": true, "console": true,
	}
	if !validLogFormats[c.Logging.Format] {
		errors = append(errors, "logging format must be one of: json, text, console")
	}
	
	if len(errors) > 0 {
		return fmt.Errorf("configuration validation failed: %v", errors)
	}
	
	return nil
}