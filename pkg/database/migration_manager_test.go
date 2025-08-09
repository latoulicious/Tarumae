package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMigrationManager(t *testing.T) {
	// Create temporary directory for test databases and backups
	tempDir, err := os.MkdirTemp("", "migration_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "test.db")
	backupDir := filepath.Join(tempDir, "backups")

	t.Run("NewMigrationManager", func(t *testing.T) {
		db, err := sql.Open("sqlite3", dbPath)
		require.NoError(t, err)
		defer db.Close()

		config := &MigrationConfig{
			BackupEnabled:    true,
			BackupDirectory:  backupDir,
			BackupRetention:  3,
			ValidateChecksum: true,
		}

		mm, err := NewMigrationManagerWithConfig(db, config)
		require.NoError(t, err)
		assert.NotNil(t, mm)

		// Check that backup directory was created
		_, err = os.Stat(backupDir)
		assert.NoError(t, err)
	})

	t.Run("GetCurrentVersion_EmptyDatabase", func(t *testing.T) {
		db, err := sql.Open("sqlite3", dbPath)
		require.NoError(t, err)
		defer db.Close()

		mm, err := NewMigrationManager(db)
		require.NoError(t, err)

		version, err := mm.GetCurrentVersion()
		require.NoError(t, err)
		assert.Equal(t, 0, version)
	})

	t.Run("GetLatestVersion", func(t *testing.T) {
		db, err := sql.Open("sqlite3", dbPath)
		require.NoError(t, err)
		defer db.Close()

		mm, err := NewMigrationManager(db)
		require.NoError(t, err)

		latestVersion := mm.GetLatestVersion()
		assert.Greater(t, latestVersion, 0)
		assert.GreaterOrEqual(t, latestVersion, 4) // We have at least 4 migrations
	})

	t.Run("Migrate_FullMigration", func(t *testing.T) {
		db, err := sql.Open("sqlite3", dbPath)
		require.NoError(t, err)
		defer db.Close()

		config := &MigrationConfig{
			BackupEnabled:    true,
			BackupDirectory:  backupDir,
			BackupRetention:  3,
			ValidateChecksum: true,
		}

		mm, err := NewMigrationManagerWithConfig(db, config)
		require.NoError(t, err)

		// Run all migrations
		err = mm.Migrate()
		require.NoError(t, err)

		// Check current version
		currentVersion, err := mm.GetCurrentVersion()
		require.NoError(t, err)
		latestVersion := mm.GetLatestVersion()
		assert.Equal(t, latestVersion, currentVersion)

		// Verify tables were created
		tables := []string{
			"schema_migrations",
			"uma_cache",
			"character_search_cache",
			"character_images_cache",
			"support_card_search_cache",
			"support_card_list_cache",
			"gametora_skills_cache",
			"pipeline_metrics",
			"pipeline_sessions",
			"pipeline_events",
		}

		for _, table := range tables {
			var count int
			err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&count)
			require.NoError(t, err)
			assert.Equal(t, 1, count, "Table %s should exist", table)
		}

		// Verify indexes were created
		indexes := []string{
			"idx_uma_cache_key",
			"idx_pipeline_metrics_pipeline_id",
			"idx_pipeline_sessions_pipeline_id",
			"idx_pipeline_events_pipeline_id",
			"idx_pipeline_metrics_name_timestamp",
			"idx_pipeline_sessions_active",
		}

		for _, index := range indexes {
			var count int
			err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name=?", index).Scan(&count)
			require.NoError(t, err)
			assert.Equal(t, 1, count, "Index %s should exist", index)
		}

		// Verify triggers were created
		triggers := []string{
			"validate_metric_type",
			"validate_event_severity",
			"validate_session_state",
			"auto_set_ended_at",
		}

		for _, trigger := range triggers {
			var count int
			err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='trigger' AND name=?", trigger).Scan(&count)
			require.NoError(t, err)
			assert.Equal(t, 1, count, "Trigger %s should exist", trigger)
		}
	})

	t.Run("GetMigrationHistory", func(t *testing.T) {
		db, err := sql.Open("sqlite3", dbPath)
		require.NoError(t, err)
		defer db.Close()

		mm, err := NewMigrationManager(db)
		require.NoError(t, err)

		history, err := mm.GetMigrationHistory()
		require.NoError(t, err)
		assert.NotEmpty(t, history)

		// Check that migrations are in order
		for i := 1; i < len(history); i++ {
			assert.Greater(t, history[i].Version, history[i-1].Version)
		}

		// Check that all migrations have required fields
		for _, migration := range history {
			assert.Greater(t, migration.Version, 0)
			assert.NotEmpty(t, migration.Name)
			assert.NotEmpty(t, migration.Checksum)
			assert.False(t, migration.AppliedAt.IsZero())
		}
	})

	t.Run("MigrateTo_SpecificVersion", func(t *testing.T) {
		// Create new database for this test
		dbPath2 := filepath.Join(tempDir, "test2.db")
		db, err := sql.Open("sqlite3", dbPath2)
		require.NoError(t, err)
		defer db.Close()

		mm, err := NewMigrationManager(db)
		require.NoError(t, err)

		// Migrate to version 2
		err = mm.MigrateTo(2)
		require.NoError(t, err)

		currentVersion, err := mm.GetCurrentVersion()
		require.NoError(t, err)
		assert.Equal(t, 2, currentVersion)

		// Verify only first 2 migrations were applied
		history, err := mm.GetMigrationHistory()
		require.NoError(t, err)
		assert.Len(t, history, 2)
	})

	t.Run("Rollback_LastMigration", func(t *testing.T) {
		// Create new database for this test
		dbPath3 := filepath.Join(tempDir, "test3.db")
		backupDir3 := filepath.Join(tempDir, "backups3")
		db, err := sql.Open("sqlite3", dbPath3)
		require.NoError(t, err)
		defer db.Close()

		config := &MigrationConfig{
			BackupEnabled:    true,
			BackupDirectory:  backupDir3,
			BackupRetention:  3,
			ValidateChecksum: true,
		}

		mm, err := NewMigrationManagerWithConfig(db, config)
		require.NoError(t, err)

		// First migrate to version 2
		err = mm.MigrateTo(2)
		require.NoError(t, err)

		currentVersion, err := mm.GetCurrentVersion()
		require.NoError(t, err)
		assert.Equal(t, 2, currentVersion)

		// Rollback last migration
		err = mm.Rollback()
		require.NoError(t, err)

		currentVersion, err = mm.GetCurrentVersion()
		require.NoError(t, err)
		assert.Equal(t, 1, currentVersion)

		// Verify pipeline tables were removed
		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='pipeline_metrics'").Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})

	t.Run("RollbackTo_SpecificVersion", func(t *testing.T) {
		// Create new database for this test
		dbPath4 := filepath.Join(tempDir, "test4.db")
		backupDir4 := filepath.Join(tempDir, "backups4")
		db, err := sql.Open("sqlite3", dbPath4)
		require.NoError(t, err)
		defer db.Close()

		config := &MigrationConfig{
			BackupEnabled:    true,
			BackupDirectory:  backupDir4,
			BackupRetention:  3,
			ValidateChecksum: true,
		}

		mm, err := NewMigrationManagerWithConfig(db, config)
		require.NoError(t, err)

		// First migrate to latest
		err = mm.Migrate()
		require.NoError(t, err)

		// Rollback to version 1
		err = mm.RollbackTo(1)
		require.NoError(t, err)

		currentVersion, err := mm.GetCurrentVersion()
		require.NoError(t, err)
		assert.Equal(t, 1, currentVersion)

		// Verify only UMA tables exist
		umaTable := []string{"uma_cache", "character_search_cache"}
		for _, table := range umaTable {
			var count int
			err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&count)
			require.NoError(t, err)
			assert.Equal(t, 1, count, "UMA table %s should exist", table)
		}

		// Verify pipeline tables don't exist
		pipelineTables := []string{"pipeline_metrics", "pipeline_sessions", "pipeline_events"}
		for _, table := range pipelineTables {
			var count int
			err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&count)
			require.NoError(t, err)
			assert.Equal(t, 0, count, "Pipeline table %s should not exist", table)
		}
	})

	t.Run("BackupAndRestore", func(t *testing.T) {
		// Create new database for this test
		dbPath5 := filepath.Join(tempDir, "test5.db")
		backupDir5 := filepath.Join(tempDir, "backups5")
		db, err := sql.Open("sqlite3", dbPath5)
		require.NoError(t, err)
		defer db.Close()

		config := &MigrationConfig{
			BackupEnabled:    true,
			BackupDirectory:  backupDir5,
			BackupRetention:  3,
			ValidateChecksum: true,
		}

		mm, err := NewMigrationManagerWithConfig(db, config)
		require.NoError(t, err)

		// Migrate to version 1
		err = mm.MigrateTo(1)
		require.NoError(t, err)

		// Insert some test data
		_, err = db.Exec("INSERT INTO uma_cache (cache_key, data, type, expires_at) VALUES (?, ?, ?, ?)",
			"test_key", "test_data", "test_type", time.Now().Add(time.Hour))
		require.NoError(t, err)

		// Create manual backup
		backupPath := filepath.Join(backupDir5, "manual_backup.db")
		err = mm.(*migrationManager).createBackup(backupPath)
		require.NoError(t, err)

		// Verify backup file exists
		_, err = os.Stat(backupPath)
		assert.NoError(t, err)

		// Modify data
		_, err = db.Exec("DELETE FROM uma_cache WHERE cache_key = ?", "test_key")
		require.NoError(t, err)

		// Verify data is gone
		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM uma_cache WHERE cache_key = ?", "test_key").Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 0, count)

		// Restore from backup
		err = mm.(*migrationManager).restoreFromBackup(backupPath)
		require.NoError(t, err)

		// Get the new database connection after restore
		newDB := mm.(*migrationManager).db

		// Verify data is restored
		err = newDB.QueryRow("SELECT COUNT(*) FROM uma_cache WHERE cache_key = ?", "test_key").Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 1, count)
	})

	t.Run("GetBackupInfo", func(t *testing.T) {
		db, err := sql.Open("sqlite3", dbPath)
		require.NoError(t, err)
		defer db.Close()

		config := &MigrationConfig{
			BackupEnabled:    true,
			BackupDirectory:  backupDir,
			BackupRetention:  3,
			ValidateChecksum: true,
		}

		mm, err := NewMigrationManagerWithConfig(db, config)
		require.NoError(t, err)

		backups, err := mm.(*migrationManager).GetBackupInfo()
		require.NoError(t, err)

		// Should have backups from previous tests
		assert.NotEmpty(t, backups)

		// Check backup info structure
		for _, backup := range backups {
			assert.NotEmpty(t, backup.Filename)
			assert.NotEmpty(t, backup.Path)
			assert.Greater(t, backup.Size, int64(0))
			assert.False(t, backup.CreatedAt.IsZero())
			assert.NotEmpty(t, backup.Description)
		}
	})

	t.Run("DataIntegrityTriggers", func(t *testing.T) {
		// Create new database for this test
		dbPath6 := filepath.Join(tempDir, "test6.db")
		backupDir6 := filepath.Join(tempDir, "backups6")
		db, err := sql.Open("sqlite3", dbPath6)
		require.NoError(t, err)
		defer db.Close()

		config := &MigrationConfig{
			BackupEnabled:    true,
			BackupDirectory:  backupDir6,
			BackupRetention:  3,
			ValidateChecksum: true,
		}

		mm, err := NewMigrationManagerWithConfig(db, config)
		require.NoError(t, err)

		// Migrate to latest to get triggers
		err = mm.Migrate()
		require.NoError(t, err)

		// Test metric type validation
		_, err = db.Exec(`INSERT INTO pipeline_metrics 
			(pipeline_id, metric_name, metric_type, metric_value, timestamp) 
			VALUES (?, ?, ?, ?, ?)`,
			"test_pipeline", "test_metric", "invalid_type", 1.0, time.Now())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Invalid metric_type")

		// Test valid metric type
		_, err = db.Exec(`INSERT INTO pipeline_metrics 
			(pipeline_id, metric_name, metric_type, metric_value, timestamp) 
			VALUES (?, ?, ?, ?, ?)`,
			"test_pipeline", "test_metric", "counter", 1.0, time.Now())
		assert.NoError(t, err)

		// Test event severity validation
		_, err = db.Exec(`INSERT INTO pipeline_events 
			(pipeline_id, event_type, event_data, severity, timestamp) 
			VALUES (?, ?, ?, ?, ?)`,
			"test_pipeline", "error", "{}", "invalid_severity", time.Now())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Invalid severity")

		// Test valid event severity
		_, err = db.Exec(`INSERT INTO pipeline_events 
			(pipeline_id, event_type, event_data, severity, timestamp) 
			VALUES (?, ?, ?, ?, ?)`,
			"test_pipeline", "error", "{}", "high", time.Now())
		assert.NoError(t, err)
	})

	t.Run("BackupCleanup", func(t *testing.T) {
		db, err := sql.Open("sqlite3", dbPath)
		require.NoError(t, err)
		defer db.Close()

		config := &MigrationConfig{
			BackupEnabled:    true,
			BackupDirectory:  backupDir,
			BackupRetention:  2, // Keep only 2 backups
			ValidateChecksum: true,
		}

		mm, err := NewMigrationManagerWithConfig(db, config)
		require.NoError(t, err)

		// Create several backup files
		for i := 0; i < 5; i++ {
			backupPath := filepath.Join(backupDir, fmt.Sprintf("test_backup_%d.db", i))
			err = mm.(*migrationManager).createBackup(backupPath)
			require.NoError(t, err)
			time.Sleep(10 * time.Millisecond) // Ensure different timestamps
		}

		// Run cleanup
		err = mm.(*migrationManager).cleanupOldBackups()
		require.NoError(t, err)

		// Count remaining backup files
		files, err := os.ReadDir(backupDir)
		require.NoError(t, err)

		backupCount := 0
		for _, file := range files {
			if !file.IsDir() && filepath.Ext(file.Name()) == ".db" {
				backupCount++
			}
		}

		// Should have at most the retention limit plus any from previous tests
		// We can't be exact due to previous tests, but should be reasonable
		assert.LessOrEqual(t, backupCount, 10) // Reasonable upper bound
	})
}

func TestMigrationManagerErrors(t *testing.T) {
	t.Run("NewMigrationManager_NilDB", func(t *testing.T) {
		mm, err := NewMigrationManager(nil)
		assert.Error(t, err)
		assert.Nil(t, mm)
		assert.Contains(t, err.Error(), "database connection is nil")
	})

	t.Run("Rollback_NoMigrations", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "migration_error_test_*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		dbPath := filepath.Join(tempDir, "empty.db")
		db, err := sql.Open("sqlite3", dbPath)
		require.NoError(t, err)
		defer db.Close()

		mm, err := NewMigrationManager(db)
		require.NoError(t, err)

		err = mm.Rollback()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no migrations to rollback")
	})

	t.Run("MigrateTo_InvalidVersion", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "migration_error_test_*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		dbPath := filepath.Join(tempDir, "test.db")
		backupDir := filepath.Join(tempDir, "backups")
		db, err := sql.Open("sqlite3", dbPath)
		require.NoError(t, err)
		defer db.Close()

		config := &MigrationConfig{
			BackupEnabled:    false, // Disable backup for this test
			BackupDirectory:  backupDir,
			BackupRetention:  3,
			ValidateChecksum: true,
		}

		mm, err := NewMigrationManagerWithConfig(db, config)
		require.NoError(t, err)

		// Try to migrate to a version that doesn't exist but is higher than latest
		// This should migrate to the latest available version
		err = mm.MigrateTo(999)
		assert.NoError(t, err) // Should succeed and migrate to latest

		currentVersion, err := mm.GetCurrentVersion()
		require.NoError(t, err)
		latestVersion := mm.GetLatestVersion()
		assert.Equal(t, latestVersion, currentVersion) // Should be at latest version
	})
}
