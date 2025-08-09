package database

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestMetricsRepository(t *testing.T) (MetricsRepository, func()) {
	// Create temporary database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	db, err := sql.Open("sqlite3", dbPath)
	require.NoError(t, err)

	config := DefaultDatabaseConfig()
	repo, err := NewMetricsRepository(db, config)
	require.NoError(t, err)

	cleanup := func() {
		db.Close()
	}

	return repo, cleanup
}

func TestMetricsRepository_StoreAndGetMetric(t *testing.T) {
	repo, cleanup := setupTestMetricsRepository(t)
	defer cleanup()

	ctx := context.Background()

	// Test data
	metric := &PipelineMetric{
		PipelineID:  "test-pipeline-1",
		MetricName:  "test_metric",
		MetricType:  "counter",
		MetricValue: 42.5,
		Tags: map[string]string{
			"environment": "test",
			"version":     "1.0",
		},
		Metadata: map[string]interface{}{
			"source": "unit_test",
		},
		Timestamp: time.Now(),
	}

	// Test storing metric
	err := repo.StoreMetric(ctx, metric)
	assert.NoError(t, err)

	// Test retrieving metrics
	query := &MetricsQuery{
		PipelineID: "test-pipeline-1",
		Limit:      10,
	}

	metrics, err := repo.GetMetrics(ctx, query)
	assert.NoError(t, err)
	assert.Len(t, metrics, 1)

	retrieved := metrics[0]
	assert.Equal(t, metric.PipelineID, retrieved.PipelineID)
	assert.Equal(t, metric.MetricName, retrieved.MetricName)
	assert.Equal(t, metric.MetricType, retrieved.MetricType)
	assert.Equal(t, metric.MetricValue, retrieved.MetricValue)
	assert.Equal(t, metric.Tags["environment"], retrieved.Tags["environment"])
	assert.Equal(t, metric.Metadata["source"], retrieved.Metadata["source"])
}

func TestMetricsRepository_StoreBatchMetrics(t *testing.T) {
	repo, cleanup := setupTestMetricsRepository(t)
	defer cleanup()

	ctx := context.Background()

	// Test data
	metrics := []*PipelineMetric{
		{
			PipelineID:  "test-pipeline-1",
			MetricName:  "metric_1",
			MetricType:  "counter",
			MetricValue: 10,
			Tags:        map[string]string{"type": "test"},
			Metadata:    map[string]interface{}{},
			Timestamp:   time.Now(),
		},
		{
			PipelineID:  "test-pipeline-1",
			MetricName:  "metric_2",
			MetricType:  "gauge",
			MetricValue: 20,
			Tags:        map[string]string{"type": "test"},
			Metadata:    map[string]interface{}{},
			Timestamp:   time.Now(),
		},
	}

	// Test batch storage
	err := repo.StoreBatchMetrics(ctx, metrics)
	assert.NoError(t, err)

	// Test retrieval
	query := &MetricsQuery{
		PipelineID: "test-pipeline-1",
		Limit:      10,
	}

	retrieved, err := repo.GetMetrics(ctx, query)
	assert.NoError(t, err)
	assert.Len(t, retrieved, 2)
}

func TestMetricsRepository_GetAggregatedMetrics(t *testing.T) {
	repo, cleanup := setupTestMetricsRepository(t)
	defer cleanup()

	ctx := context.Background()

	// Store test metrics
	baseTime := time.Now().Truncate(time.Hour)
	metrics := []*PipelineMetric{
		{
			PipelineID:  "test-pipeline-1",
			MetricName:  "cpu_usage",
			MetricType:  "gauge",
			MetricValue: 50.0,
			Tags:        map[string]string{},
			Metadata:    map[string]interface{}{},
			Timestamp:   baseTime,
		},
		{
			PipelineID:  "test-pipeline-1",
			MetricName:  "cpu_usage",
			MetricType:  "gauge",
			MetricValue: 60.0,
			Tags:        map[string]string{},
			Metadata:    map[string]interface{}{},
			Timestamp:   baseTime.Add(1 * time.Minute),
		},
	}

	err := repo.StoreBatchMetrics(ctx, metrics)
	require.NoError(t, err)

	// Test aggregation
	query := &AggregationQuery{
		PipelineID:  "test-pipeline-1",
		MetricName:  "cpu_usage",
		Aggregation: "avg",
		StartTime:   &baseTime,
		EndTime:     func() *time.Time { t := baseTime.Add(2 * time.Minute); return &t }(),
	}

	aggregated, err := repo.GetAggregatedMetrics(ctx, query)
	assert.NoError(t, err)
	assert.NotNil(t, aggregated)
	assert.Equal(t, "cpu_usage", aggregated.MetricName)
	assert.Equal(t, "avg", aggregated.Aggregation)
	assert.NotEmpty(t, aggregated.Results)
}

func TestMetricsRepository_SessionManagement(t *testing.T) {
	repo, cleanup := setupTestMetricsRepository(t)
	defer cleanup()

	ctx := context.Background()

	// Test data
	session := &PipelineSession{
		PipelineID: "test-pipeline-1",
		GuildID:    "guild-123",
		ChannelID:  "channel-456",
		UserID:     "user-789",
		StreamURL:  "https://example.com/stream",
		StartedAt:  time.Now(),
	}

	// Test creating session
	err := repo.CreateSession(ctx, session)
	assert.NoError(t, err)

	// Test retrieving session
	retrieved, err := repo.GetSession(ctx, session.PipelineID)
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, session.PipelineID, retrieved.PipelineID)
	assert.Equal(t, session.GuildID, retrieved.GuildID)
	assert.Equal(t, session.ChannelID, retrieved.ChannelID)

	// Test updating session
	endTime := time.Now()
	updates := &SessionUpdate{
		EndedAt:         &endTime,
		FinalState:      stringPtr("completed"),
		TotalErrors:     intPtr(2),
		TotalRecoveries: intPtr(1),
	}

	err = repo.UpdateSession(ctx, session.PipelineID, updates)
	assert.NoError(t, err)

	// Verify update
	updated, err := repo.GetSession(ctx, session.PipelineID)
	assert.NoError(t, err)
	assert.NotNil(t, updated.EndedAt)
	assert.Equal(t, "completed", updated.FinalState)
	assert.Equal(t, 2, updated.TotalErrors)
	assert.Equal(t, 1, updated.TotalRecoveries)

	// Test getting active sessions (should be empty after update)
	activeSessions, err := repo.GetActiveSessions(ctx)
	assert.NoError(t, err)
	assert.Empty(t, activeSessions)
}

func TestMetricsRepository_EventManagement(t *testing.T) {
	repo, cleanup := setupTestMetricsRepository(t)
	defer cleanup()

	ctx := context.Background()

	// Test data
	event := &PipelineEvent{
		PipelineID: "test-pipeline-1",
		EventType:  "error",
		EventData: map[string]interface{}{
			"error_message": "test error",
			"component":     "stream_processor",
		},
		Severity:  "high",
		Timestamp: time.Now(),
	}

	// Test storing event
	err := repo.StoreEvent(ctx, event)
	assert.NoError(t, err)

	// Test retrieving events
	query := &EventQuery{
		PipelineID: "test-pipeline-1",
		Limit:      10,
	}

	events, err := repo.GetEvents(ctx, query)
	assert.NoError(t, err)
	assert.Len(t, events, 1)

	retrieved := events[0]
	assert.Equal(t, event.PipelineID, retrieved.PipelineID)
	assert.Equal(t, event.EventType, retrieved.EventType)
	assert.Equal(t, event.Severity, retrieved.Severity)
	assert.Equal(t, event.EventData["error_message"], retrieved.EventData["error_message"])
}

func TestMetricsRepository_CleanExpiredMetrics(t *testing.T) {
	repo, cleanup := setupTestMetricsRepository(t)
	defer cleanup()

	ctx := context.Background()

	// Store old metric
	oldTime := time.Now().Add(-48 * time.Hour)
	metric := &PipelineMetric{
		PipelineID:  "test-pipeline-1",
		MetricName:  "old_metric",
		MetricType:  "counter",
		MetricValue: 1,
		Tags:        map[string]string{},
		Metadata:    map[string]interface{}{},
		Timestamp:   oldTime,
	}

	err := repo.StoreMetric(ctx, metric)
	require.NoError(t, err)

	// Clean with 24 hour retention
	retentionPeriod := 24 * time.Hour
	err = repo.CleanExpiredMetrics(ctx, retentionPeriod)
	assert.NoError(t, err)

	// Verify old metric is gone
	query := &MetricsQuery{
		PipelineID: "test-pipeline-1",
	}

	metrics, err := repo.GetMetrics(ctx, query)
	assert.NoError(t, err)
	assert.Empty(t, metrics)
}

func TestMetricsRepository_GetMetricsStats(t *testing.T) {
	repo, cleanup := setupTestMetricsRepository(t)
	defer cleanup()

	ctx := context.Background()

	// Store some test data
	metric := &PipelineMetric{
		PipelineID:  "test-pipeline-1",
		MetricName:  "test_metric",
		MetricType:  "counter",
		MetricValue: 1,
		Tags:        map[string]string{},
		Metadata:    map[string]interface{}{},
		Timestamp:   time.Now(),
	}

	err := repo.StoreMetric(ctx, metric)
	require.NoError(t, err)

	// Get stats
	stats, err := repo.GetMetricsStats(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, stats)
	assert.Equal(t, int64(1), stats.TotalMetrics)
	assert.Contains(t, stats.MetricsByType, "counter")
	assert.Equal(t, int64(1), stats.MetricsByType["counter"])
}

func TestNewMetricsRepository_NilDB(t *testing.T) {
	config := DefaultDatabaseConfig()
	repo, err := NewMetricsRepository(nil, config)
	assert.Error(t, err)
	assert.Nil(t, repo)
	assert.Contains(t, err.Error(), "database connection is nil")
}

// Helper functions
func stringPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}
