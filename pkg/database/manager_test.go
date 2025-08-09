package database

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDatabaseManager(t *testing.T) {
	tests := []struct {
		name        string
		config      *DatabaseConfig
		expectError bool
	}{
		{
			name:        "nil config uses defaults",
			config:      nil,
			expectError: false,
		},
		{
			name:        "valid config",
			config:      DefaultDatabaseConfig(),
			expectError: false,
		},
		{
			name: "invalid config - empty database path",
			config: &DatabaseConfig{
				DatabasePath: "",
			},
			expectError: true,
		},
		{
			name: "invalid config - zero max connections",
			config: &DatabaseConfig{
				DatabasePath:   "test.db",
				MaxConnections: 0,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dm, err := NewDatabaseManager(tt.config)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, dm)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, dm)
			}
		})
	}
}

func TestDatabaseManager_ConnectAndClose(t *testing.T) {
	// Create temporary database file
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	config := DefaultDatabaseConfig()
	config.DatabasePath = dbPath

	dm, err := NewDatabaseManager(config)
	require.NoError(t, err)

	// Test connection
	err = dm.Connect()
	assert.NoError(t, err)

	// Test ping
	ctx := context.Background()
	err = dm.Ping(ctx)
	assert.NoError(t, err)

	// Test repositories are available
	assert.NotNil(t, dm.UMARepository())
	assert.NotNil(t, dm.MetricsRepository())

	// Test close
	err = dm.Close()
	assert.NoError(t, err)

	// Test ping after close should fail
	err = dm.Ping(ctx)
	assert.Error(t, err)
}

func TestDatabaseManager_Migration(t *testing.T) {
	// Create temporary database file
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	config := DefaultDatabaseConfig()
	config.DatabasePath = dbPath

	dm, err := NewDatabaseManager(config)
	require.NoError(t, err)

	err = dm.Connect()
	require.NoError(t, err)
	defer dm.Close()

	// Test schema version
	version, err := dm.GetSchemaVersion()
	assert.NoError(t, err)
	assert.Greater(t, version, 0)

	// Test migration
	err = dm.Migrate()
	assert.NoError(t, err)
}

func TestDatabaseManager_CleanExpiredData(t *testing.T) {
	// Create temporary database file
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	config := DefaultDatabaseConfig()
	config.DatabasePath = dbPath

	dm, err := NewDatabaseManager(config)
	require.NoError(t, err)

	err = dm.Connect()
	require.NoError(t, err)
	defer dm.Close()

	// Test cleanup
	err = dm.CleanExpiredData()
	assert.NoError(t, err)
}

func TestDatabaseManager_GetStats(t *testing.T) {
	// Create temporary database file
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	config := DefaultDatabaseConfig()
	config.DatabasePath = dbPath

	dm, err := NewDatabaseManager(config)
	require.NoError(t, err)

	err = dm.Connect()
	require.NoError(t, err)
	defer dm.Close()

	// Test stats
	stats, err := dm.GetStats()
	assert.NoError(t, err)
	assert.NotNil(t, stats)
	assert.NotNil(t, stats.UMAStats)
	assert.NotNil(t, stats.MetricsStats)
	assert.Greater(t, stats.FileSize, int64(0))
}

func TestDatabaseManager_BackupAndRestore(t *testing.T) {
	// Create temporary database file
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	backupPath := filepath.Join(tempDir, "backup.db")

	config := DefaultDatabaseConfig()
	config.DatabasePath = dbPath

	dm, err := NewDatabaseManager(config)
	require.NoError(t, err)

	err = dm.Connect()
	require.NoError(t, err)

	// Test backup
	err = dm.Backup(backupPath)
	assert.NoError(t, err)

	// Verify backup file exists
	_, err = os.Stat(backupPath)
	assert.NoError(t, err)

	dm.Close()

	// Test restore
	err = dm.Restore(backupPath)
	assert.NoError(t, err)

	// Verify database is functional after restore
	err = dm.Ping(context.Background())
	assert.NoError(t, err)

	dm.Close()
}

func TestDatabaseConfig_Validate(t *testing.T) {
	tests := []struct {
		name        string
		config      *DatabaseConfig
		expectError bool
	}{
		{
			name:        "valid config",
			config:      DefaultDatabaseConfig(),
			expectError: false,
		},
		{
			name: "invalid database path",
			config: &DatabaseConfig{
				DatabasePath: "",
			},
			expectError: true,
		},
		{
			name: "invalid max connections",
			config: &DatabaseConfig{
				DatabasePath:   "test.db",
				MaxConnections: 0,
			},
			expectError: true,
		},
		{
			name: "invalid connection timeout",
			config: &DatabaseConfig{
				DatabasePath:      "test.db",
				MaxConnections:    10,
				ConnectionTimeout: 0,
			},
			expectError: true,
		},
		{
			name: "invalid synchronous mode",
			config: &DatabaseConfig{
				DatabasePath:            "test.db",
				MaxConnections:          10,
				ConnectionTimeout:       30 * time.Second,
				MetricsBatchSize:        100,
				MetricsFlushInterval:    30 * time.Second,
				MetricsRetention:        24 * time.Hour,
				UMACacheRetention:       24 * time.Hour,
				UMACacheCleanupInterval: 1 * time.Hour,
				SynchronousMode:         "INVALID",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDefaultDatabaseConfig(t *testing.T) {
	config := DefaultDatabaseConfig()

	assert.NotEmpty(t, config.DatabasePath)
	assert.Greater(t, config.MaxConnections, 0)
	assert.Greater(t, config.ConnectionTimeout, time.Duration(0))
	assert.Greater(t, config.MetricsBatchSize, 0)
	assert.Greater(t, config.MetricsFlushInterval, time.Duration(0))
	assert.Greater(t, config.MetricsRetention, time.Duration(0))
	assert.Greater(t, config.UMACacheRetention, time.Duration(0))
	assert.Greater(t, config.UMACacheCleanupInterval, time.Duration(0))
	assert.True(t, config.WALMode)
	assert.NotEmpty(t, config.SynchronousMode)
	assert.NotZero(t, config.CacheSize)

	// Validate the default config
	err := config.Validate()
	assert.NoError(t, err)
}
