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

func setupTestRetentionManager(t *testing.T) (*MetricsRetentionManager, *sql.DB, func()) {
	// Create temporary database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	db, err := sql.Open("sqlite3", dbPath)
	require.NoError(t, err)

	// Create test tables
	queries := []string{
		`CREATE TABLE pipeline_metrics (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			pipeline_id TEXT NOT NULL,
			metric_name TEXT NOT NULL,
			metric_type TEXT NOT NULL,
			metric_value REAL NOT NULL,
			tags TEXT,
			metadata TEXT,
			timestamp DATETIME NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE pipeline_events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			pipeline_id TEXT NOT NULL,
			event_type TEXT NOT NULL,
			event_data TEXT NOT NULL,
			severity TEXT,
			timestamp DATETIME NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE pipeline_sessions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			pipeline_id TEXT UNIQUE NOT NULL,
			guild_id TEXT,
			channel_id TEXT,
			user_id TEXT,
			stream_url TEXT,
			started_at DATETIME NOT NULL,
			ended_at DATETIME,
			final_state TEXT,
			total_errors INTEGER DEFAULT 0,
			total_recoveries INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
	}

	for _, query := range queries {
		_, err = db.Exec(query)
		require.NoError(t, err)
	}

	config := &DatabaseConfig{
		MetricsRetention:        24 * time.Hour,
		UMACacheCleanupInterval: 100 * time.Millisecond, // Fast for testing
	}

	manager := NewMetricsRetentionManager(db, config)

	cleanup := func() {
		manager.Stop()
		db.Close()
	}

	return manager, db, cleanup
}

func TestNewMetricsRetentionManager(t *testing.T) {
	manager, _, cleanup := setupTestRetentionManager(t)
	defer cleanup()

	assert.NotNil(t, manager)

	policies := manager.GetPolicies()
	assert.NotEmpty(t, policies)
	assert.Len(t, policies, 4) // Default policies

	// Check default policies
	policyNames := make(map[string]bool)
	for _, policy := range policies {
		policyNames[policy.Name] = true
		assert.True(t, policy.Enabled)
		assert.Greater(t, policy.RetentionPeriod, time.Duration(0))
	}

	assert.True(t, policyNames["metrics_retention"])
	assert.True(t, policyNames["events_retention"])
	assert.True(t, policyNames["completed_sessions_retention"])
	assert.True(t, policyNames["low_priority_metrics_retention"])
}

func TestMetricsRetentionManager_AddRemovePolicy(t *testing.T) {
	manager, _, cleanup := setupTestRetentionManager(t)
	defer cleanup()

	initialCount := len(manager.GetPolicies())

	// Add custom policy
	customPolicy := RetentionPolicy{
		Name:            "custom_policy",
		Description:     "Custom test policy",
		RetentionPeriod: 1 * time.Hour,
		TableName:       "pipeline_metrics",
		TimestampColumn: "timestamp",
		Priority:        0, // High priority
		Enabled:         true,
	}

	manager.AddPolicy(customPolicy)

	policies := manager.GetPolicies()
	assert.Len(t, policies, initialCount+1)

	// Should be first due to priority
	assert.Equal(t, "custom_policy", policies[0].Name)
	assert.Equal(t, 0, policies[0].Priority)

	// Remove policy
	removed := manager.RemovePolicy("custom_policy")
	assert.True(t, removed)

	policies = manager.GetPolicies()
	assert.Len(t, policies, initialCount)

	// Try to remove non-existent policy
	removed = manager.RemovePolicy("non_existent")
	assert.False(t, removed)
}

func TestMetricsRetentionManager_StartStop(t *testing.T) {
	manager, _, cleanup := setupTestRetentionManager(t)
	defer cleanup()

	logger := &testLogger{}
	manager.SetLogger(logger)

	// Test start
	err := manager.Start()
	assert.NoError(t, err)

	// Test double start
	err = manager.Start()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already running")

	// Give it time to start
	time.Sleep(50 * time.Millisecond)

	// Test stop
	err = manager.Stop()
	assert.NoError(t, err)

	// Check logs
	messages := logger.GetMessages()
	assert.Contains(t, messages[0], "MetricsRetentionManager started")
	assert.Contains(t, messages[len(messages)-1], "MetricsRetentionManager stopped gracefully")
}

func TestMetricsRetentionManager_RunCleanup(t *testing.T) {
	manager, db, cleanup := setupTestRetentionManager(t)
	defer cleanup()

	logger := &testLogger{}
	manager.SetLogger(logger)

	// Insert old test data
	oldTime := time.Now().Add(-48 * time.Hour)
	recentTime := time.Now().Add(-1 * time.Hour)

	// Insert old metrics (should be cleaned)
	_, err := db.Exec(`
		INSERT INTO pipeline_metrics (pipeline_id, metric_name, metric_type, metric_value, tags, metadata, timestamp)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, "test-pipeline", "old_metric", "counter", 1.0, "{}", "{}", oldTime)
	require.NoError(t, err)

	// Insert recent metrics (should be kept)
	_, err = db.Exec(`
		INSERT INTO pipeline_metrics (pipeline_id, metric_name, metric_type, metric_value, tags, metadata, timestamp)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, "test-pipeline", "recent_metric", "counter", 2.0, "{}", "{}", recentTime)
	require.NoError(t, err)

	// Insert old events (should be cleaned)
	_, err = db.Exec(`
		INSERT INTO pipeline_events (pipeline_id, event_type, event_data, severity, timestamp)
		VALUES (?, ?, ?, ?, ?)
	`, "test-pipeline", "error", "{}", "high", oldTime)
	require.NoError(t, err)

	// Run cleanup
	ctx := context.Background()
	stats, err := manager.RunCleanup(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, stats)

	// Verify old data was cleaned
	var metricCount int
	err = db.QueryRow("SELECT COUNT(*) FROM pipeline_metrics").Scan(&metricCount)
	assert.NoError(t, err)
	assert.Equal(t, 1, metricCount) // Only recent metric should remain

	var eventCount int
	err = db.QueryRow("SELECT COUNT(*) FROM pipeline_events").Scan(&eventCount)
	assert.NoError(t, err)
	assert.Equal(t, 0, eventCount) // Old event should be cleaned

	// Check stats
	assert.Greater(t, stats.TotalCleaned, int64(0))
	assert.NotZero(t, stats.LastCleanupTime)
	assert.NotEmpty(t, stats.LastPolicyResults)

	// Check logs
	messages := logger.GetMessages()
	found := false
	for _, msg := range messages {
		if contains(msg, "Starting metrics retention cleanup") {
			found = true
			break
		}
	}
	assert.True(t, found, "Should log cleanup start")
}

func TestMetricsRetentionManager_PolicyExecution(t *testing.T) {
	manager, db, cleanup := setupTestRetentionManager(t)
	defer cleanup()

	// Create a custom policy for testing
	testPolicy := RetentionPolicy{
		Name:            "test_policy",
		Description:     "Test policy",
		RetentionPeriod: 1 * time.Hour,
		TableName:       "pipeline_metrics",
		TimestampColumn: "timestamp",
		Conditions:      []string{"metric_name = 'test_metric'"},
		Priority:        1,
		Enabled:         true,
	}

	// Clear default policies and add test policy
	manager.policies = []RetentionPolicy{testPolicy}

	// Insert test data
	oldTime := time.Now().Add(-2 * time.Hour)
	recentTime := time.Now().Add(-30 * time.Minute)

	// Insert old test_metric (should be cleaned)
	_, err := db.Exec(`
		INSERT INTO pipeline_metrics (pipeline_id, metric_name, metric_type, metric_value, tags, metadata, timestamp)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, "test-pipeline", "test_metric", "counter", 1.0, "{}", "{}", oldTime)
	require.NoError(t, err)

	// Insert old other_metric (should NOT be cleaned due to condition)
	_, err = db.Exec(`
		INSERT INTO pipeline_metrics (pipeline_id, metric_name, metric_type, metric_value, tags, metadata, timestamp)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, "test-pipeline", "other_metric", "counter", 2.0, "{}", "{}", oldTime)
	require.NoError(t, err)

	// Insert recent test_metric (should NOT be cleaned due to time)
	_, err = db.Exec(`
		INSERT INTO pipeline_metrics (pipeline_id, metric_name, metric_type, metric_value, tags, metadata, timestamp)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, "test-pipeline", "test_metric", "counter", 3.0, "{}", "{}", recentTime)
	require.NoError(t, err)

	// Run cleanup
	ctx := context.Background()
	stats, err := manager.RunCleanup(ctx)
	assert.NoError(t, err)

	// Verify only the old test_metric was cleaned
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM pipeline_metrics").Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 2, count) // other_metric and recent test_metric should remain

	// Verify the correct record was cleaned
	var metricName string
	err = db.QueryRow("SELECT metric_name FROM pipeline_metrics WHERE timestamp = ?", oldTime).Scan(&metricName)
	assert.NoError(t, err)
	assert.Equal(t, "other_metric", metricName)

	// Check policy result
	assert.Contains(t, stats.LastPolicyResults, "test_policy")
	result := stats.LastPolicyResults["test_policy"]
	assert.Equal(t, int64(1), result.RecordsFound)
	assert.Equal(t, int64(1), result.RecordsCleaned)
	assert.Empty(t, result.Error)
}

func TestMetricsRetentionManager_DisabledPolicy(t *testing.T) {
	manager, db, cleanup := setupTestRetentionManager(t)
	defer cleanup()

	// Create a disabled policy
	disabledPolicy := RetentionPolicy{
		Name:            "disabled_policy",
		Description:     "Disabled test policy",
		RetentionPeriod: 1 * time.Hour,
		TableName:       "pipeline_metrics",
		TimestampColumn: "timestamp",
		Priority:        1,
		Enabled:         false, // Disabled
	}

	// Clear default policies and add disabled policy
	manager.policies = []RetentionPolicy{disabledPolicy}

	// Insert old data
	oldTime := time.Now().Add(-2 * time.Hour)
	_, err := db.Exec(`
		INSERT INTO pipeline_metrics (pipeline_id, metric_name, metric_type, metric_value, tags, metadata, timestamp)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, "test-pipeline", "test_metric", "counter", 1.0, "{}", "{}", oldTime)
	require.NoError(t, err)

	// Run cleanup
	ctx := context.Background()
	stats, err := manager.RunCleanup(ctx)
	assert.NoError(t, err)

	// Verify no data was cleaned (policy was disabled)
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM pipeline_metrics").Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 1, count) // Data should remain

	// Check that no policy results were recorded
	assert.Empty(t, stats.LastPolicyResults)
}

func TestMetricsRetentionManager_GetStats(t *testing.T) {
	manager, _, cleanup := setupTestRetentionManager(t)
	defer cleanup()

	// Initial stats
	stats := manager.GetStats()
	assert.NotNil(t, stats)
	assert.Zero(t, stats.TotalCleaned)
	assert.True(t, stats.LastCleanupTime.IsZero())
	assert.NotZero(t, stats.NextScheduledRun)

	// Run cleanup to update stats
	ctx := context.Background()
	_, err := manager.RunCleanup(ctx)
	assert.NoError(t, err)

	// Updated stats
	stats = manager.GetStats()
	assert.NotZero(t, stats.LastCleanupTime)
	assert.NotZero(t, stats.NextScheduledRun)
}

func TestMetricsRetentionManager_AutomaticCleanup(t *testing.T) {
	// This test verifies that automatic cleanup runs
	manager, db, cleanup := setupTestRetentionManager(t)
	defer cleanup()

	logger := &testLogger{}
	manager.SetLogger(logger)

	// Insert old data
	oldTime := time.Now().Add(-48 * time.Hour)
	_, err := db.Exec(`
		INSERT INTO pipeline_metrics (pipeline_id, metric_name, metric_type, metric_value, tags, metadata, timestamp)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, "test-pipeline", "old_metric", "counter", 1.0, "{}", "{}", oldTime)
	require.NoError(t, err)

	// Start manager
	err = manager.Start()
	require.NoError(t, err)

	// Wait for automatic cleanup (initial delay + some buffer)
	time.Sleep(200 * time.Millisecond)

	// Verify cleanup ran
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM pipeline_metrics").Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 0, count) // Old data should be cleaned

	// Check logs for cleanup activity
	messages := logger.GetMessages()
	found := false
	for _, msg := range messages {
		if contains(msg, "Starting metrics retention cleanup") {
			found = true
			break
		}
	}
	assert.True(t, found, "Should log automatic cleanup")
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) &&
			(s[:len(substr)] == substr ||
				s[len(s)-len(substr):] == substr ||
				containsSubstring(s, substr))))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
