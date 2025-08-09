package pipeline

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// TestPipelineManagerCreation tests the creation of a pipeline manager
func TestPipelineManagerCreation(t *testing.T) {
	config := DefaultPipelineConfig()
	logger := NullLogger()
	
	manager, err := NewAudioPipelineManager(config, logger)
	if err != nil {
		t.Fatalf("Failed to create pipeline manager: %v", err)
	}
	
	if manager == nil {
		t.Fatal("Pipeline manager is nil")
	}
	
	if manager.GetState() != StateIdle {
		t.Errorf("Expected initial state to be idle, got %s", manager.GetState())
	}
	
	if manager.GetPipelineID() == "" {
		t.Error("Pipeline ID should not be empty")
	}
}

// TestPipelineConfiguration tests configuration validation
func TestPipelineConfiguration(t *testing.T) {
	// Test valid configuration
	config := DefaultPipelineConfig()
	if err := config.Validate(); err != nil {
		t.Errorf("Default configuration should be valid: %v", err)
	}
	
	// Test invalid configuration
	invalidConfig := DefaultPipelineConfig()
	invalidConfig.Opus.SampleRate = -1
	if err := invalidConfig.Validate(); err == nil {
		t.Error("Invalid configuration should fail validation")
	}
}

// TestStructuredLogging tests the structured logging system
func TestStructuredLogging(t *testing.T) {
	config := LoggingConfig{
		Level:  "debug",
		Format: "json",
		Output: "stdout",
	}
	
	logger := NewStructuredLogger(config)
	if logger == nil {
		t.Fatal("Logger should not be nil")
	}
	
	// Test logging with fields
	logger.Info("Test message",
		String("key1", "value1"),
		Int("key2", 42),
		Bool("key3", true),
	)
	
	// Test logger with persistent fields
	childLogger := logger.With(String("component", "test"))
	childLogger.Debug("Child logger test")
}

// TestMetricsCollection tests the metrics collection system
func TestMetricsCollection(t *testing.T) {
	logger := NullLogger()
	collector := NewPipelineMetricsCollector("test-pipeline", logger)
	
	// Test counter
	collector.RecordPipelineCounter("test.counter", 1, map[string]string{"tag": "value"})
	
	// Test gauge
	collector.RecordPipelineGauge("test.gauge", 42.5, nil)
	
	// Test histogram
	collector.RecordPipelineHistogram("test.histogram", 100.0, nil)
	collector.RecordPipelineHistogram("test.histogram", 200.0, nil)
	
	// Test timing
	collector.RecordPipelineTiming("test.timing", 50*time.Millisecond, nil)
	
	// Get metrics snapshot
	snapshot := collector.GetAllMetrics()
	if len(snapshot.Metrics) == 0 {
		t.Error("Expected metrics to be recorded")
	}
	
	// Test specific metric retrieval
	metrics := collector.GetMetricsByName("test.histogram")
	if len(metrics) == 0 {
		t.Error("Expected histogram metric to exist")
	}
	
	if metrics[0].Metadata == nil {
		t.Error("Histogram should have metadata")
	}
	
	count, ok := metrics[0].Metadata["count"].(float64)
	if !ok || count != 2.0 {
		t.Errorf("Expected histogram count to be 2, got %v", count)
	}
}

// TestPipelineStateManagement tests pipeline state transitions
func TestPipelineStateManagement(t *testing.T) {
	config := DefaultPipelineConfig()
	logger := NullLogger()
	
	manager, err := NewAudioPipelineManager(config, logger)
	if err != nil {
		t.Fatalf("Failed to create pipeline manager: %v", err)
	}
	
	// Test initial state
	if manager.GetState() != StateIdle {
		t.Errorf("Expected initial state to be idle, got %s", manager.GetState())
	}
	
	// Test start
	ctx := context.Background()
	err = manager.Start(ctx, "test://stream")
	if err != nil {
		t.Errorf("Failed to start pipeline: %v", err)
	}
	
	// Give some time for state transition
	time.Sleep(10 * time.Millisecond)
	
	if manager.GetState() != StateStreaming {
		t.Errorf("Expected state to be streaming after start, got %s", manager.GetState())
	}
	
	// Test stop
	err = manager.Stop()
	if err != nil {
		t.Errorf("Failed to stop pipeline: %v", err)
	}
	
	// Give some time for state transition
	time.Sleep(10 * time.Millisecond)
	
	if manager.GetState() != StateIdle {
		t.Errorf("Expected state to be idle after stop, got %s", manager.GetState())
	}
}

// TestErrorClassification tests error classification
func TestErrorClassification(t *testing.T) {
	err := NewPipelineError(
		fmt.Errorf("test error"),
		CategoryNetwork,
		SeverityMedium,
	)
	
	if err.Category != CategoryNetwork {
		t.Errorf("Expected category to be network, got %s", err.Category)
	}
	
	if err.Severity != SeverityMedium {
		t.Errorf("Expected severity to be medium, got %s", err.Severity)
	}
	
	if !err.Retryable {
		t.Error("Medium severity errors should be retryable")
	}
	
	if err.Timestamp.IsZero() {
		t.Error("Error timestamp should be set")
	}
}

// TestHealthResult tests health check results
func TestHealthResult(t *testing.T) {
	result := NewHealthResult("test-check", true, "All good")
	
	if result.Name != "test-check" {
		t.Errorf("Expected name to be 'test-check', got %s", result.Name)
	}
	
	if !result.Healthy {
		t.Error("Expected result to be healthy")
	}
	
	if result.Message != "All good" {
		t.Errorf("Expected message to be 'All good', got %s", result.Message)
	}
	
	if result.Timestamp.IsZero() {
		t.Error("Result timestamp should be set")
	}
}