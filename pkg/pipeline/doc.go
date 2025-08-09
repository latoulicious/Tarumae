// Package pipeline provides a robust, fault-tolerant audio streaming pipeline
// for Discord music bots. This package implements a comprehensive audio pipeline
// with enhanced error handling, monitoring, recovery mechanisms, and structured logging.
//
// # Core Components
//
// The pipeline consists of several key components:
//
//   - AudioPipelineManager: Central coordinator that manages the entire pipeline
//   - Structured Logging: JSON/text logging with configurable levels and fields
//   - Metrics Collection: Comprehensive metrics tracking for performance monitoring
//   - Configuration Management: Flexible configuration with validation and environment variable support
//   - Error Classification: Systematic error categorization and handling
//   - State Management: Well-defined pipeline states with proper transitions
//
// # Architecture
//
// The pipeline follows a layered architecture with clear separation of concerns:
//
//   1. Management Layer: AudioPipelineManager coordinates all operations
//   2. Processing Layer: Stream acquisition, processing, encoding, and streaming
//   3. Monitoring Layer: Health checks, metrics collection, and logging
//   4. Recovery Layer: Error handling and automatic recovery strategies
//
// # Usage Example
//
//	// Create configuration
//	config := pipeline.DefaultPipelineConfig()
//	config.LoadFromEnvironment()
//
//	// Create logger
//	logger := pipeline.NewStructuredLogger(config.Logging)
//
//	// Create pipeline manager
//	manager, err := pipeline.NewAudioPipelineManager(config, logger)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Start streaming
//	ctx := context.Background()
//	err = manager.Start(ctx, "https://example.com/stream")
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Monitor pipeline
//	for {
//		state := manager.GetState()
//		metrics := manager.GetMetrics()
//		healthy := manager.IsHealthy()
//
//		logger.Info("Pipeline status",
//			pipeline.String("state", state.String()),
//			pipeline.Bool("healthy", healthy),
//			pipeline.Any("metrics", metrics),
//		)
//
//		time.Sleep(5 * time.Second)
//	}
//
// # Configuration
//
// The pipeline supports comprehensive configuration through the PipelineConfig struct.
// Configuration can be loaded from environment variables using LoadFromEnvironment().
// All configuration is validated before use.
//
// # Logging
//
// The structured logging system supports multiple output formats (JSON, text, console)
// and log levels (debug, info, warn, error, fatal). Loggers can be created with
// persistent fields and support contextual logging.
//
// # Metrics
//
// The metrics system supports counters, gauges, histograms, and timing measurements.
// Metrics are tagged and can be aggregated for monitoring and alerting.
//
// # Error Handling
//
// Errors are classified by category (network, stream, process, voice, system) and
// severity (low, medium, high, critical). This classification drives recovery
// strategies and user notifications.
//
// # State Management
//
// The pipeline operates through well-defined states:
//   - Idle: No active streaming
//   - Initializing: Setting up components
//   - Streaming: Active audio streaming
//   - Recovering: Attempting error recovery
//   - Paused: Temporarily suspended
//   - Stopping: Gracefully shutting down
//   - Failed: Unrecoverable error state
//
// State transitions are logged and can trigger specific behaviors.
//
// # Thread Safety
//
// All components are designed to be thread-safe and can be used concurrently.
// The pipeline manager coordinates all operations through channels and proper
// synchronization primitives.
//
// # Extensibility
//
// The pipeline is designed with interfaces that allow for easy extension and
// testing. Components can be mocked or replaced with alternative implementations.
package pipeline