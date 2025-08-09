package database

import (
	"crypto/md5"
	"database/sql"
	"fmt"
	"log"
	"sort"
	"time"
)

// migrationManager implements the MigrationManager interface
type migrationManager struct {
	db         *sql.DB
	migrations map[int]*migrationScript
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

// NewMigrationManager creates a new migration manager
func NewMigrationManager(db *sql.DB) (MigrationManager, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}

	mm := &migrationManager{
		db:         db,
		migrations: make(map[int]*migrationScript),
	}

	// Initialize migration tracking table
	if err := mm.initializeMigrationTable(); err != nil {
		return nil, fmt.Errorf("failed to initialize migration table: %w", err)
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
		if err := mm.runMigration(version, true); err != nil {
			return fmt.Errorf("failed to run migration %d: %w", version, err)
		}
		log.Printf("Applied migration %d: %s", version, mm.migrations[version].Name)
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
		// Migrate up
		var versionsToMigrate []int
		for version := range mm.migrations {
			if version > currentVersion && version <= targetVersion {
				versionsToMigrate = append(versionsToMigrate, version)
			}
		}
		sort.Ints(versionsToMigrate)

		for _, version := range versionsToMigrate {
			if err := mm.runMigration(version, true); err != nil {
				return fmt.Errorf("failed to run migration %d: %w", version, err)
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

		for _, version := range versionsToRollback {
			if err := mm.runMigration(version, false); err != nil {
				return fmt.Errorf("failed to rollback migration %d: %w", version, err)
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

	return mm.runMigration(currentVersion, false)
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
