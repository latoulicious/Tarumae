package database

import (
	"crypto/md5"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// migrationManager implements the MigrationManager interface
type migrationManager struct {
	db         *sql.DB
	migrations map[int]*migrationScript
	config     *MigrationConfig
}

// migrationScript represents a single database migration
type migrationScript struct {
	Version     int
	Name        string
	Description string
	UpSQL       string
	DownSQL     string
	Checksum    string
}

// MigrationConfig holds configuration for the migration manager
type MigrationConfig struct {
	BackupEnabled    bool   `json:"backup_enabled" yaml:"backup_enabled"`
	BackupDirectory  string `json:"backup_directory" yaml:"backup_directory"`
	BackupRetention  int    `json:"backup_retention" yaml:"backup_retention"`
	ValidateChecksum bool   `json:"validate_checksum" yaml:"validate_checksum"`
}

// DefaultMigrationConfig returns default migration configuration
func DefaultMigrationConfig() *MigrationConfig {
	return &MigrationConfig{
		BackupEnabled:    true,
		BackupDirectory:  "./backups",
		BackupRetention:  5,
		ValidateChecksum: true,
	}
}

// NewMigrationManager creates a new migration manager
func NewMigrationManager(db *sql.DB) (MigrationManager, error) {
	return NewMigrationManagerWithConfig(db, DefaultMigrationConfig())
}

// NewMigrationManagerWithConfig creates a new migration manager with custom configuration
func NewMigrationManagerWithConfig(db *sql.DB, config *MigrationConfig) (MigrationManager, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}

	if config == nil {
		config = DefaultMigrationConfig()
	}

	mm := &migrationManager{
		db:         db,
		migrations: make(map[int]*migrationScript),
		config:     config,
	}

	// Initialize migration tracking table
	if err := mm.initializeMigrationTable(); err != nil {
		return nil, fmt.Errorf("failed to initialize migration table: %w", err)
	}

	// Create backup directory if backup is enabled
	if mm.config.BackupEnabled {
		if err := mm.ensureBackupDirectory(); err != nil {
			return nil, fmt.Errorf("failed to create backup directory: %w", err)
		}
	}

	// Load migration scripts
	mm.loadMigrations()

	return mm, nil
}

// initializeMigrationTable creates the migration tracking table
func (mm *migrationManager) initializeMigrationTable() error {
	query := `
	CREATE TABLE IF NOT EXISTS schema_migrations (
		version INTEGER PRIMARY KEY,
		name TEXT NOT NULL,
		description TEXT,
		checksum TEXT NOT NULL,
		applied_at DATETIME NOT NULL
	)
	`

	if _, err := mm.db.Exec(query); err != nil {
		return fmt.Errorf("failed to create schema_migrations table: %w", err)
	}

	return nil
}

// loadMigrations loads all migration scripts
func (mm *migrationManager) loadMigrations() {
	// Migration 1: Initial UMA cache schema (existing tables)
	mm.migrations[1] = &migrationScript{
		Version:     1,
		Name:        "initial_uma_cache",
		Description: "Create initial UMA cache tables",
		UpSQL: `
			-- UMA cache tables already exist, this migration marks them as version 1
			CREATE TABLE IF NOT EXISTS uma_cache (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				cache_key TEXT UNIQUE NOT NULL,
				data TEXT NOT NULL,
				type TEXT NOT NULL,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				expires_at DATETIME NOT NULL
			);
			
			CREATE TABLE IF NOT EXISTS character_search_cache (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				query TEXT UNIQUE NOT NULL,
				character_id INTEGER NOT NULL,
				character_data TEXT NOT NULL,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				expires_at DATETIME NOT NULL
			);
			
			CREATE TABLE IF NOT EXISTS character_images_cache (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				character_id INTEGER UNIQUE NOT NULL,
				images_data TEXT NOT NULL,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				expires_at DATETIME NOT NULL
			);
			
			CREATE TABLE IF NOT EXISTS support_card_search_cache (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				query TEXT UNIQUE NOT NULL,
				support_cards_data TEXT NOT NULL,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				expires_at DATETIME NOT NULL
			);
			
			CREATE TABLE IF NOT EXISTS support_card_list_cache (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				list_data TEXT NOT NULL,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				expires_at DATETIME NOT NULL
			);
			
			CREATE TABLE IF NOT EXISTS gametora_skills_cache (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				query TEXT UNIQUE NOT NULL,
				skills_data TEXT NOT NULL,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				expires_at DATETIME NOT NULL
			);
			
			-- Create indexes
			CREATE INDEX IF NOT EXISTS idx_uma_cache_key ON uma_cache(cache_key);
			CREATE INDEX IF NOT EXISTS idx_uma_cache_expires ON uma_cache(expires_at);
			CREATE INDEX IF NOT EXISTS idx_character_search_query ON character_search_cache(query);
			CREATE INDEX IF NOT EXISTS idx_character_search_expires ON character_search_cache(expires_at);
			CREATE INDEX IF NOT EXISTS idx_character_images_id ON character_images_cache(character_id);
			CREATE INDEX IF NOT EXISTS idx_character_images_expires ON character_images_cache(expires_at);
			CREATE INDEX IF NOT EXISTS idx_support_card_search_query ON support_card_search_cache(query);
			CREATE INDEX IF NOT EXISTS idx_support_card_search_expires ON support_card_search_cache(expires_at);
			CREATE INDEX IF NOT EXISTS idx_support_card_list_expires ON support_card_list_cache(expires_at);
			CREATE INDEX IF NOT EXISTS idx_gametora_skills_query ON gametora_skills_cache(query);
			CREATE INDEX IF NOT EXISTS idx_gametora_skills_expires ON gametora_skills_cache(expires_at);
		`,
		DownSQL: `
			DROP INDEX IF EXISTS idx_gametora_skills_expires;
			DROP INDEX IF EXISTS idx_gametora_skills_query;
			DROP INDEX IF EXISTS idx_support_card_list_expires;
			DROP INDEX IF EXISTS idx_support_card_search_expires;
			DROP INDEX IF EXISTS idx_support_card_search_query;
			DROP INDEX IF EXISTS idx_character_images_expires;
			DROP INDEX IF EXISTS idx_character_images_id;
			DROP INDEX IF EXISTS idx_character_search_expires;
			DROP INDEX IF EXISTS idx_character_search_query;
			DROP INDEX IF EXISTS idx_uma_cache_expires;
			DROP INDEX IF EXISTS idx_uma_cache_key;
			
			DROP TABLE IF EXISTS gametora_skills_cache;
			DROP TABLE IF EXISTS support_card_list_cache;
			DROP TABLE IF EXISTS support_card_search_cache;
			DROP TABLE IF EXISTS character_images_cache;
			DROP TABLE IF EXISTS character_search_cache;
			DROP TABLE IF EXISTS uma_cache;
		`,
	}

	// Migration 2: Add pipeline metrics tables
	mm.migrations[2] = &migrationScript{
		Version:     2,
		Name:        "add_pipeline_metrics",
		Description: "Add pipeline metrics, sessions, and events tables",
		UpSQL: `
			CREATE TABLE IF NOT EXISTS pipeline_metrics (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				pipeline_id TEXT NOT NULL,
				metric_name TEXT NOT NULL,
				metric_type TEXT NOT NULL,
				metric_value REAL NOT NULL,
				tags TEXT,
				metadata TEXT,
				timestamp DATETIME NOT NULL,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP
			);
			
			CREATE TABLE IF NOT EXISTS pipeline_sessions (
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
			);
			
			CREATE TABLE IF NOT EXISTS pipeline_events (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				pipeline_id TEXT NOT NULL,
				event_type TEXT NOT NULL,
				event_data TEXT NOT NULL,
				severity TEXT,
				timestamp DATETIME NOT NULL,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP
			);
			
			-- Create indexes for performance
			CREATE INDEX IF NOT EXISTS idx_pipeline_metrics_pipeline_id ON pipeline_metrics(pipeline_id);
			CREATE INDEX IF NOT EXISTS idx_pipeline_metrics_timestamp ON pipeline_metrics(timestamp);
			CREATE INDEX IF NOT EXISTS idx_pipeline_metrics_name ON pipeline_metrics(metric_name);
			CREATE INDEX IF NOT EXISTS idx_pipeline_metrics_type ON pipeline_metrics(metric_type);
			CREATE INDEX IF NOT EXISTS idx_pipeline_sessions_pipeline_id ON pipeline_sessions(pipeline_id);
			CREATE INDEX IF NOT EXISTS idx_pipeline_sessions_started_at ON pipeline_sessions(started_at);
			CREATE INDEX IF NOT EXISTS idx_pipeline_events_pipeline_id ON pipeline_events(pipeline_id);
			CREATE INDEX IF NOT EXISTS idx_pipeline_events_timestamp ON pipeline_events(timestamp);
			CREATE INDEX IF NOT EXISTS idx_pipeline_events_type ON pipeline_events(event_type);
			CREATE INDEX IF NOT EXISTS idx_pipeline_events_severity ON pipeline_events(severity);
		`,
		DownSQL: `
			DROP INDEX IF EXISTS idx_pipeline_events_severity;
			DROP INDEX IF EXISTS idx_pipeline_events_type;
			DROP INDEX IF EXISTS idx_pipeline_events_timestamp;
			DROP INDEX IF EXISTS idx_pipeline_events_pipeline_id;
			DROP INDEX IF EXISTS idx_pipeline_sessions_started_at;
			DROP INDEX IF EXISTS idx_pipeline_sessions_pipeline_id;
			DROP INDEX IF EXISTS idx_pipeline_metrics_type;
			DROP INDEX IF EXISTS idx_pipeline_metrics_name;
			DROP INDEX IF EXISTS idx_pipeline_metrics_timestamp;
			DROP INDEX IF EXISTS idx_pipeline_metrics_pipeline_id;
			
			DROP TABLE IF EXISTS pipeline_events;
			DROP TABLE IF EXISTS pipeline_sessions;
			DROP TABLE IF EXISTS pipeline_metrics;
		`,
	}

	// Migration 3: Add enhanced indexes and constraints for better performance
	mm.migrations[3] = &migrationScript{
		Version:     3,
		Name:        "enhance_performance_indexes",
		Description: "Add composite indexes and constraints for better query performance",
		UpSQL: `
			-- Add composite indexes for common query patterns
			CREATE INDEX IF NOT EXISTS idx_pipeline_metrics_name_timestamp ON pipeline_metrics(metric_name, timestamp);
			CREATE INDEX IF NOT EXISTS idx_pipeline_metrics_pipeline_name ON pipeline_metrics(pipeline_id, metric_name);
			CREATE INDEX IF NOT EXISTS idx_pipeline_events_pipeline_type ON pipeline_events(pipeline_id, event_type);
			CREATE INDEX IF NOT EXISTS idx_pipeline_events_severity_timestamp ON pipeline_events(severity, timestamp);
			CREATE INDEX IF NOT EXISTS idx_pipeline_sessions_guild_started ON pipeline_sessions(guild_id, started_at);
			CREATE INDEX IF NOT EXISTS idx_pipeline_sessions_ended_state ON pipeline_sessions(ended_at, final_state);
			
			-- Add partial indexes for active sessions
			CREATE INDEX IF NOT EXISTS idx_pipeline_sessions_active ON pipeline_sessions(pipeline_id) WHERE ended_at IS NULL;
			
			-- Add indexes for cleanup operations
			CREATE INDEX IF NOT EXISTS idx_uma_cache_expires_type ON uma_cache(expires_at, type);
			CREATE INDEX IF NOT EXISTS idx_character_search_expires_created ON character_search_cache(expires_at, created_at);
			CREATE INDEX IF NOT EXISTS idx_character_images_expires_created ON character_images_cache(expires_at, created_at);
			CREATE INDEX IF NOT EXISTS idx_support_card_search_expires_created ON support_card_search_cache(expires_at, created_at);
			CREATE INDEX IF NOT EXISTS idx_support_card_list_expires_created ON support_card_list_cache(expires_at, created_at);
			CREATE INDEX IF NOT EXISTS idx_gametora_skills_expires_created ON gametora_skills_cache(expires_at, created_at);
		`,
		DownSQL: `
			DROP INDEX IF EXISTS idx_gametora_skills_expires_created;
			DROP INDEX IF EXISTS idx_support_card_list_expires_created;
			DROP INDEX IF EXISTS idx_support_card_search_expires_created;
			DROP INDEX IF EXISTS idx_character_images_expires_created;
			DROP INDEX IF EXISTS idx_character_search_expires_created;
			DROP INDEX IF EXISTS idx_uma_cache_expires_type;
			DROP INDEX IF EXISTS idx_pipeline_sessions_active;
			DROP INDEX IF EXISTS idx_pipeline_sessions_ended_state;
			DROP INDEX IF EXISTS idx_pipeline_sessions_guild_started;
			DROP INDEX IF EXISTS idx_pipeline_events_severity_timestamp;
			DROP INDEX IF EXISTS idx_pipeline_events_pipeline_type;
			DROP INDEX IF EXISTS idx_pipeline_metrics_pipeline_name;
			DROP INDEX IF EXISTS idx_pipeline_metrics_name_timestamp;
		`,
	}

	// Migration 4: Add data integrity constraints and triggers
	mm.migrations[4] = &migrationScript{
		Version:     4,
		Name:        "add_data_integrity",
		Description: "Add constraints and triggers for data integrity",
		UpSQL: `
			-- Add check constraints for data validation
			-- Note: SQLite doesn't support adding constraints to existing tables,
			-- so we'll create triggers instead for validation
			
			-- Trigger to validate metric types
			CREATE TRIGGER IF NOT EXISTS validate_metric_type
			BEFORE INSERT ON pipeline_metrics
			FOR EACH ROW
			WHEN NEW.metric_type NOT IN ('counter', 'gauge', 'histogram', 'timing')
			BEGIN
				SELECT RAISE(ABORT, 'Invalid metric_type. Must be one of: counter, gauge, histogram, timing');
			END;
			
			-- Trigger to validate event severity
			CREATE TRIGGER IF NOT EXISTS validate_event_severity
			BEFORE INSERT ON pipeline_events
			FOR EACH ROW
			WHEN NEW.severity IS NOT NULL AND NEW.severity NOT IN ('low', 'medium', 'high', 'critical')
			BEGIN
				SELECT RAISE(ABORT, 'Invalid severity. Must be one of: low, medium, high, critical');
			END;
			
			-- Trigger to validate session state
			CREATE TRIGGER IF NOT EXISTS validate_session_state
			BEFORE UPDATE ON pipeline_sessions
			FOR EACH ROW
			WHEN NEW.final_state IS NOT NULL AND NEW.final_state NOT IN ('completed', 'failed', 'cancelled', 'timeout')
			BEGIN
				SELECT RAISE(ABORT, 'Invalid final_state. Must be one of: completed, failed, cancelled, timeout');
			END;
			
			-- Trigger to automatically set ended_at when final_state is set
			CREATE TRIGGER IF NOT EXISTS auto_set_ended_at
			BEFORE UPDATE ON pipeline_sessions
			FOR EACH ROW
			WHEN NEW.final_state IS NOT NULL AND OLD.final_state IS NULL AND NEW.ended_at IS NULL
			BEGIN
				UPDATE pipeline_sessions SET ended_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
			END;
		`,
		DownSQL: `
			DROP TRIGGER IF EXISTS auto_set_ended_at;
			DROP TRIGGER IF EXISTS validate_session_state;
			DROP TRIGGER IF EXISTS validate_event_severity;
			DROP TRIGGER IF EXISTS validate_metric_type;
		`,
	}

	// Calculate checksums for all migrations
	for _, migration := range mm.migrations {
		migration.Checksum = mm.calculateChecksum(migration.UpSQL)
	}
}

// GetCurrentVersion returns the current schema version
func (mm *migrationManager) GetCurrentVersion() (int, error) {
	query := "SELECT COALESCE(MAX(version), 0) FROM schema_migrations"

	var version int
	err := mm.db.QueryRow(query).Scan(&version)
	if err != nil {
		return 0, fmt.Errorf("failed to get current version: %w", err)
	}

	return version, nil
}

// GetLatestVersion returns the latest available migration version
func (mm *migrationManager) GetLatestVersion() int {
	maxVersion := 0
	for version := range mm.migrations {
		if version > maxVersion {
			maxVersion = version
		}
	}
	return maxVersion
}

// Migrate runs all pending migrations
func (mm *migrationManager) Migrate() error {
	currentVersion, err := mm.GetCurrentVersion()
	if err != nil {
		return fmt.Errorf("failed to get current version: %w", err)
	}

	latestVersion := mm.GetLatestVersion()

	if currentVersion >= latestVersion {
		log.Printf("Database is up to date (version %d)", currentVersion)
		return nil
	}

	log.Printf("Migrating database from version %d to %d", currentVersion, latestVersion)

	// Create backup before migration if enabled
	var backupPath string
	if mm.config.BackupEnabled {
		backupPath, err = mm.createPreMigrationBackup(currentVersion, latestVersion)
		if err != nil {
			return fmt.Errorf("failed to create pre-migration backup: %w", err)
		}
		log.Printf("Created pre-migration backup: %s", backupPath)
	}

	// Get sorted list of versions to migrate
	var versionsToMigrate []int
	for version := range mm.migrations {
		if version > currentVersion {
			versionsToMigrate = append(versionsToMigrate, version)
		}
	}
	sort.Ints(versionsToMigrate)

	// Run migrations in order
	for _, version := range versionsToMigrate {
		if err := mm.runMigrationWithBackup(version, true); err != nil {
			log.Printf("Migration %d failed, attempting rollback", version)
			if backupPath != "" {
				log.Printf("Restoring from backup: %s", backupPath)
				if restoreErr := mm.restoreFromBackup(backupPath); restoreErr != nil {
					return fmt.Errorf("migration failed and backup restore failed: migration error: %w, restore error: %v", err, restoreErr)
				}
				return fmt.Errorf("migration failed, database restored from backup: %w", err)
			}
			return fmt.Errorf("failed to run migration %d: %w", version, err)
		}
		log.Printf("Applied migration %d: %s", version, mm.migrations[version].Name)
	}

	// Clean up old backups
	if mm.config.BackupEnabled {
		if err := mm.cleanupOldBackups(); err != nil {
			log.Printf("Warning: failed to cleanup old backups: %v", err)
		}
	}

	log.Printf("Database migration completed successfully")
	return nil
}

// MigrateTo migrates to a specific version
func (mm *migrationManager) MigrateTo(targetVersion int) error {
	currentVersion, err := mm.GetCurrentVersion()
	if err != nil {
		return fmt.Errorf("failed to get current version: %w", err)
	}

	if currentVersion == targetVersion {
		return nil
	}

	if targetVersion > currentVersion {
		// Migrate up - find all migrations between current and target
		var versionsToMigrate []int
		for version := range mm.migrations {
			if version > currentVersion && version <= targetVersion {
				versionsToMigrate = append(versionsToMigrate, version)
			}
		}
		sort.Ints(versionsToMigrate)

		// If no migrations to run, return early
		if len(versionsToMigrate) == 0 {
			return nil
		}

		// If target version doesn't exist but we have migrations to run,
		// only run up to the highest existing version
		if _, exists := mm.migrations[targetVersion]; !exists && len(versionsToMigrate) > 0 {
			// Find the highest version we can actually migrate to
			highestAvailable := versionsToMigrate[len(versionsToMigrate)-1]
			if highestAvailable < targetVersion {
				targetVersion = highestAvailable
			}
		}

		// Create backup before migration if enabled
		var backupPath string
		if mm.config.BackupEnabled {
			backupPath, err = mm.createPreMigrationBackup(currentVersion, targetVersion)
			if err != nil {
				return fmt.Errorf("failed to create pre-migration backup: %w", err)
			}
			log.Printf("Created pre-migration backup: %s", backupPath)
		}

		for _, version := range versionsToMigrate {
			if err := mm.runMigrationWithBackup(version, true); err != nil {
				if backupPath != "" {
					log.Printf("Migration failed, backup available at: %s", backupPath)
				}
				return fmt.Errorf("failed to run migration %d: %w", version, err)
			}
		}

		// Clean up old backups after successful migration
		if mm.config.BackupEnabled {
			if err := mm.cleanupOldBackups(); err != nil {
				log.Printf("Warning: failed to cleanup old backups: %v", err)
			}
		}
	} else {
		// Migrate down
		var versionsToRollback []int
		for version := range mm.migrations {
			if version > targetVersion && version <= currentVersion {
				versionsToRollback = append(versionsToRollback, version)
			}
		}
		sort.Sort(sort.Reverse(sort.IntSlice(versionsToRollback)))

		// If no rollbacks to run, return early
		if len(versionsToRollback) == 0 {
			return nil
		}

		// Create backup before rollback if enabled
		var backupPath string
		if mm.config.BackupEnabled {
			backupPath, err = mm.createPreMigrationBackup(currentVersion, targetVersion)
			if err != nil {
				return fmt.Errorf("failed to create pre-migration backup: %w", err)
			}
			log.Printf("Created pre-migration backup: %s", backupPath)
		}

		for _, version := range versionsToRollback {
			if err := mm.runMigrationWithBackup(version, false); err != nil {
				if backupPath != "" {
					log.Printf("Rollback failed, backup available at: %s", backupPath)
				}
				return fmt.Errorf("failed to rollback migration %d: %w", version, err)
			}
		}

		// Clean up old backups after successful rollback
		if mm.config.BackupEnabled {
			if err := mm.cleanupOldBackups(); err != nil {
				log.Printf("Warning: failed to cleanup old backups: %v", err)
			}
		}
	}

	return nil
}

// Rollback rolls back the last migration
func (mm *migrationManager) Rollback() error {
	currentVersion, err := mm.GetCurrentVersion()
	if err != nil {
		return fmt.Errorf("failed to get current version: %w", err)
	}

	if currentVersion == 0 {
		return fmt.Errorf("no migrations to rollback")
	}

	// Create backup before rollback if enabled
	var backupPath string
	if mm.config.BackupEnabled {
		timestamp := time.Now().Format("20060102_150405")
		nanos := time.Now().UnixNano() % 1000000 // Add nanoseconds for uniqueness
		filename := fmt.Sprintf("before_rollback_v%d_%s_%d.db", currentVersion, timestamp, nanos)
		backupPath = filepath.Join(mm.config.BackupDirectory, filename)

		if err := mm.createBackup(backupPath); err != nil {
			return fmt.Errorf("failed to create pre-rollback backup: %w", err)
		}
		log.Printf("Created pre-rollback backup: %s", backupPath)
	}

	if err := mm.runMigration(currentVersion, false); err != nil {
		if backupPath != "" {
			log.Printf("Rollback failed, backup available at: %s", backupPath)
		}
		return fmt.Errorf("failed to rollback migration %d: %w", currentVersion, err)
	}

	log.Printf("Rolled back migration %d: %s", currentVersion, mm.migrations[currentVersion].Name)
	return nil
}

// RollbackTo rolls back to a specific version
func (mm *migrationManager) RollbackTo(targetVersion int) error {
	return mm.MigrateTo(targetVersion)
}

// GetMigrationHistory returns the migration history
func (mm *migrationManager) GetMigrationHistory() ([]*Migration, error) {
	query := `
		SELECT version, name, description, checksum, applied_at
		FROM schema_migrations
		ORDER BY version
	`

	rows, err := mm.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query migration history: %w", err)
	}
	defer rows.Close()

	var migrations []*Migration
	for rows.Next() {
		migration := &Migration{}
		err := rows.Scan(
			&migration.Version,
			&migration.Name,
			&migration.Description,
			&migration.Checksum,
			&migration.AppliedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan migration: %w", err)
		}

		migrations = append(migrations, migration)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating migrations: %w", err)
	}

	return migrations, nil
}

// runMigration runs a single migration up or down
func (mm *migrationManager) runMigration(version int, up bool) error {
	migration, exists := mm.migrations[version]
	if !exists {
		return fmt.Errorf("migration %d not found", version)
	}

	tx, err := mm.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	var sql string
	if up {
		sql = migration.UpSQL
	} else {
		sql = migration.DownSQL
	}

	// Execute migration SQL
	if _, err := tx.Exec(sql); err != nil {
		return fmt.Errorf("failed to execute migration SQL: %w", err)
	}

	// Update migration tracking
	if up {
		// Record migration as applied
		_, err = tx.Exec(`
			INSERT OR REPLACE INTO schema_migrations (version, name, description, checksum, applied_at)
			VALUES (?, ?, ?, ?, ?)
		`, version, migration.Name, migration.Description, migration.Checksum, time.Now())
	} else {
		// Remove migration record
		_, err = tx.Exec("DELETE FROM schema_migrations WHERE version = ?", version)
	}

	if err != nil {
		return fmt.Errorf("failed to update migration tracking: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit migration: %w", err)
	}

	return nil
}

// calculateChecksum calculates MD5 checksum of migration SQL
func (mm *migrationManager) calculateChecksum(sql string) string {
	hash := md5.Sum([]byte(sql))
	return fmt.Sprintf("%x", hash)
}

// ensureBackupDirectory creates the backup directory if it doesn't exist
func (mm *migrationManager) ensureBackupDirectory() error {
	if err := os.MkdirAll(mm.config.BackupDirectory, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}
	return nil
}

// createPreMigrationBackup creates a backup before running migrations
func (mm *migrationManager) createPreMigrationBackup(fromVersion, toVersion int) (string, error) {
	timestamp := time.Now().Format("20060102_150405")
	nanos := time.Now().UnixNano() % 1000000 // Add nanoseconds for uniqueness
	filename := fmt.Sprintf("pre_migration_v%d_to_v%d_%s_%d.db", fromVersion, toVersion, timestamp, nanos)
	backupPath := filepath.Join(mm.config.BackupDirectory, filename)

	if err := mm.createBackup(backupPath); err != nil {
		return "", fmt.Errorf("failed to create backup: %w", err)
	}

	return backupPath, nil
}

// createBackup creates a database backup using SQLite's VACUUM INTO
func (mm *migrationManager) createBackup(backupPath string) error {
	// Ensure backup directory exists
	if err := os.MkdirAll(filepath.Dir(backupPath), 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Use SQLite's VACUUM INTO for atomic backup
	query := fmt.Sprintf("VACUUM INTO '%s'", backupPath)
	if _, err := mm.db.Exec(query); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	log.Printf("Database backup created: %s", backupPath)
	return nil
}

// restoreFromBackup restores the database from a backup file
func (mm *migrationManager) restoreFromBackup(backupPath string) error {
	// Check if backup file exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup file does not exist: %s", backupPath)
	}

	// Get the current database path from the connection
	var seq int
	var name string
	var dbPath string
	if err := mm.db.QueryRow("PRAGMA database_list").Scan(&seq, &name, &dbPath); err != nil {
		return fmt.Errorf("failed to get database path: %w", err)
	}

	// Close current connection
	if err := mm.db.Close(); err != nil {
		return fmt.Errorf("failed to close database connection: %w", err)
	}

	// Copy backup file to database path
	if err := copyFile(backupPath, dbPath); err != nil {
		return fmt.Errorf("failed to restore backup: %w", err)
	}

	// Reopen database connection
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("failed to reopen database: %w", err)
	}

	mm.db = db
	log.Printf("Database restored from backup: %s", backupPath)
	return nil
}

// runMigrationWithBackup runs a migration with individual backup support
func (mm *migrationManager) runMigrationWithBackup(version int, up bool) error {
	_, exists := mm.migrations[version]
	if !exists {
		return fmt.Errorf("migration %d not found", version)
	}

	// Validate checksum if enabled
	if mm.config.ValidateChecksum && up {
		if err := mm.validateMigrationChecksum(version); err != nil {
			return fmt.Errorf("checksum validation failed for migration %d: %w", version, err)
		}
	}

	// Create individual migration backup if enabled
	var backupPath string
	if mm.config.BackupEnabled {
		timestamp := time.Now().Format("20060102_150405")
		nanos := time.Now().UnixNano() % 1000000 // Add nanoseconds for uniqueness
		var filename string
		if up {
			filename = fmt.Sprintf("before_migration_v%d_%s_%d.db", version, timestamp, nanos)
		} else {
			filename = fmt.Sprintf("before_rollback_v%d_%s_%d.db", version, timestamp, nanos)
		}
		backupPath = filepath.Join(mm.config.BackupDirectory, filename)

		if err := mm.createBackup(backupPath); err != nil {
			return fmt.Errorf("failed to create migration backup: %w", err)
		}
	}

	// Run the migration
	if err := mm.runMigration(version, up); err != nil {
		// If backup exists, offer to restore
		if backupPath != "" {
			log.Printf("Migration failed, backup available at: %s", backupPath)
		}
		return err
	}

	return nil
}

// validateMigrationChecksum validates that a migration hasn't been tampered with
func (mm *migrationManager) validateMigrationChecksum(version int) error {
	migration, exists := mm.migrations[version]
	if !exists {
		return fmt.Errorf("migration %d not found", version)
	}

	// Check if migration was already applied
	query := "SELECT checksum FROM schema_migrations WHERE version = ?"
	var storedChecksum string
	err := mm.db.QueryRow(query, version).Scan(&storedChecksum)
	if err != nil {
		if err == sql.ErrNoRows {
			// Migration not applied yet, checksum validation passes
			return nil
		}
		return fmt.Errorf("failed to get stored checksum: %w", err)
	}

	// Compare checksums
	currentChecksum := mm.calculateChecksum(migration.UpSQL)
	if storedChecksum != currentChecksum {
		return fmt.Errorf("checksum mismatch: stored=%s, current=%s", storedChecksum, currentChecksum)
	}

	return nil
}

// cleanupOldBackups removes old backup files based on retention policy
func (mm *migrationManager) cleanupOldBackups() error {
	if mm.config.BackupRetention <= 0 {
		return nil // No cleanup if retention is 0 or negative
	}

	files, err := os.ReadDir(mm.config.BackupDirectory)
	if err != nil {
		return fmt.Errorf("failed to read backup directory: %w", err)
	}

	// Filter and sort backup files by modification time
	var backupFiles []os.FileInfo
	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".db" {
			info, err := file.Info()
			if err != nil {
				continue
			}
			backupFiles = append(backupFiles, info)
		}
	}

	// Sort by modification time (newest first)
	sort.Slice(backupFiles, func(i, j int) bool {
		return backupFiles[i].ModTime().After(backupFiles[j].ModTime())
	})

	// Remove old backups beyond retention limit
	for i := mm.config.BackupRetention; i < len(backupFiles); i++ {
		backupPath := filepath.Join(mm.config.BackupDirectory, backupFiles[i].Name())
		if err := os.Remove(backupPath); err != nil {
			log.Printf("Warning: failed to remove old backup %s: %v", backupPath, err)
		} else {
			log.Printf("Removed old backup: %s", backupPath)
		}
	}

	return nil
}

// GetBackupInfo returns information about available backups
func (mm *migrationManager) GetBackupInfo() ([]*BackupInfo, error) {
	if !mm.config.BackupEnabled {
		return nil, fmt.Errorf("backup is not enabled")
	}

	files, err := os.ReadDir(mm.config.BackupDirectory)
	if err != nil {
		return nil, fmt.Errorf("failed to read backup directory: %w", err)
	}

	var backups []*BackupInfo
	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".db" {
			info, err := file.Info()
			if err != nil {
				continue
			}

			backup := &BackupInfo{
				Filename:    file.Name(),
				Path:        filepath.Join(mm.config.BackupDirectory, file.Name()),
				Size:        info.Size(),
				CreatedAt:   info.ModTime(),
				Description: mm.parseBackupDescription(file.Name()),
			}
			backups = append(backups, backup)
		}
	}

	// Sort by creation time (newest first)
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].CreatedAt.After(backups[j].CreatedAt)
	})

	return backups, nil
}

// parseBackupDescription extracts description from backup filename
func (mm *migrationManager) parseBackupDescription(filename string) string {
	if filepath.Ext(filename) == ".db" {
		name := filename[:len(filename)-3] // Remove .db extension

		// Parse different backup types
		if len(name) > 14 && name[:14] == "pre_migration_" {
			return "Pre-migration backup"
		} else if len(name) > 17 && name[:17] == "before_migration_" {
			return "Before migration backup"
		} else if len(name) > 15 && name[:15] == "before_rollback_" {
			return "Before rollback backup"
		}
	}

	return "Database backup"
}

// BackupInfo represents information about a backup file
type BackupInfo struct {
	Filename    string    `json:"filename"`
	Path        string    `json:"path"`
	Size        int64     `json:"size"`
	CreatedAt   time.Time `json:"created_at"`
	Description string    `json:"description"`
}
