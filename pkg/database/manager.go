package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// databaseManager implements the DatabaseManager interface
type databaseManager struct {
	config            *DatabaseConfig
	db                *sql.DB
	connectionPool    ConnectionPool
	migrationManager  MigrationManager
	umaRepository     UMARepository
	metricsRepository MetricsRepository

	// State management
	connected bool
	mutex     sync.RWMutex

	// Background tasks
	cleanupTicker *time.Ticker
	backupTicker  *time.Ticker
	stopChan      chan struct{}
}

// NewDatabaseManager creates a new enhanced database manager
func NewDatabaseManager(config *DatabaseConfig) (DatabaseManager, error) {
	if config == nil {
		config = DefaultDatabaseConfig()
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid database configuration: %w", err)
	}

	dm := &databaseManager{
		config:   config,
		stopChan: make(chan struct{}),
	}

	return dm, nil
}

// Connect establishes database connection and initializes components
func (dm *databaseManager) Connect() error {
	dm.mutex.Lock()
	defer dm.mutex.Unlock()

	return dm.connectInternal()
}

// connectInternal establishes database connection without acquiring lock
func (dm *databaseManager) connectInternal() error {
	if dm.connected {
		return nil
	}

	// Open database connection
	db, err := sql.Open("sqlite3", dm.buildConnectionString())
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(dm.config.MaxConnections)
	db.SetMaxIdleConns(dm.config.MaxConnections / 2)
	db.SetConnMaxLifetime(time.Hour)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), dm.config.ConnectionTimeout)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return fmt.Errorf("failed to ping database: %w", err)
	}

	dm.db = db

	// Initialize components
	if err := dm.initializeComponents(); err != nil {
		db.Close()
		return fmt.Errorf("failed to initialize components: %w", err)
	}

	// Run migrations
	if err := dm.migrationManager.Migrate(); err != nil {
		db.Close()
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	dm.connected = true

	// Start background tasks
	dm.startBackgroundTasks()

	log.Printf("Database manager connected successfully")
	return nil
}

// Close closes the database connection and stops background tasks
func (dm *databaseManager) Close() error {
	dm.mutex.Lock()
	defer dm.mutex.Unlock()

	if !dm.connected {
		return nil
	}

	// Stop background tasks
	select {
	case <-dm.stopChan:
		// Channel already closed
	default:
		close(dm.stopChan)
	}

	if dm.cleanupTicker != nil {
		dm.cleanupTicker.Stop()
	}

	if dm.backupTicker != nil {
		dm.backupTicker.Stop()
	}

	// Close connection pool if exists
	if dm.connectionPool != nil {
		if err := dm.connectionPool.Close(); err != nil {
			log.Printf("Error closing connection pool: %v", err)
		}
	}

	// Close database
	if dm.db != nil {
		if err := dm.db.Close(); err != nil {
			return fmt.Errorf("failed to close database: %w", err)
		}
	}

	dm.connected = false
	log.Printf("Database manager closed successfully")
	return nil
}

// Ping tests the database connection
func (dm *databaseManager) Ping(ctx context.Context) error {
	dm.mutex.RLock()
	defer dm.mutex.RUnlock()

	if !dm.connected || dm.db == nil {
		return ErrDatabaseNotConnected
	}

	return dm.db.PingContext(ctx)
}

// UMARepository returns the UMA repository
func (dm *databaseManager) UMARepository() UMARepository {
	return dm.umaRepository
}

// MetricsRepository returns the metrics repository
func (dm *databaseManager) MetricsRepository() MetricsRepository {
	return dm.metricsRepository
}

// Migrate runs database migrations
func (dm *databaseManager) Migrate() error {
	if dm.migrationManager == nil {
		return fmt.Errorf("migration manager not initialized")
	}
	return dm.migrationManager.Migrate()
}

// GetSchemaVersion returns the current schema version
func (dm *databaseManager) GetSchemaVersion() (int, error) {
	if dm.migrationManager == nil {
		return 0, fmt.Errorf("migration manager not initialized")
	}
	return dm.migrationManager.GetCurrentVersion()
}

// CleanExpiredData removes expired data from all repositories
func (dm *databaseManager) CleanExpiredData() error {
	dm.mutex.RLock()
	defer dm.mutex.RUnlock()

	if !dm.connected {
		return ErrDatabaseNotConnected
	}

	// Clean UMA cache
	if err := dm.umaRepository.CleanExpiredCache(); err != nil {
		return fmt.Errorf("failed to clean UMA cache: %w", err)
	}

	// Clean metrics
	ctx := context.Background()
	if err := dm.metricsRepository.CleanExpiredMetrics(ctx, dm.config.MetricsRetention); err != nil {
		return fmt.Errorf("failed to clean expired metrics: %w", err)
	}

	return nil
}

// GetStats returns database statistics
func (dm *databaseManager) GetStats() (*DatabaseStats, error) {
	dm.mutex.RLock()
	defer dm.mutex.RUnlock()

	if !dm.connected {
		return nil, ErrDatabaseNotConnected
	}

	stats := &DatabaseStats{}

	// Get UMA stats
	if umaStats, err := dm.umaRepository.GetCacheStats(); err == nil {
		stats.UMAStats = &UMAStats{
			CharacterSearchCount:   umaStats["character_search"],
			CharacterImagesCount:   umaStats["character_images"],
			SupportCardSearchCount: umaStats["support_card_search"],
			SupportCardListCount:   umaStats["support_card_list"],
			GametoraSkillsCount:    umaStats["gametora_skills"],
			TotalCacheCount:        umaStats["total_cache"],
		}
	}

	// Get metrics stats
	ctx := context.Background()
	if metricsStats, err := dm.metricsRepository.GetMetricsStats(ctx); err == nil {
		stats.MetricsStats = metricsStats
	}

	// Get connection pool stats
	if dm.connectionPool != nil {
		stats.PoolStats = dm.connectionPool.Stats()
	}

	// Get file size
	if fileInfo, err := os.Stat(dm.config.DatabasePath); err == nil {
		stats.FileSize = fileInfo.Size()
	}

	return stats, nil
}

// Backup creates a backup of the database
func (dm *databaseManager) Backup(path string) error {
	dm.mutex.RLock()
	defer dm.mutex.RUnlock()

	if !dm.connected {
		return ErrDatabaseNotConnected
	}

	// Use SQLite backup API
	backupQuery := fmt.Sprintf("VACUUM INTO '%s'", path)
	if _, err := dm.db.Exec(backupQuery); err != nil {
		return fmt.Errorf("backup failed: %w", err)
	}

	log.Printf("Database backup created: %s", path)
	return nil
}

// Restore restores the database from a backup
func (dm *databaseManager) Restore(path string) error {
	dm.mutex.Lock()
	defer dm.mutex.Unlock()

	// Check if backup file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("backup file does not exist: %s", path)
	}

	// Close current connection
	if dm.connected && dm.db != nil {
		// Stop background tasks first
		if dm.cleanupTicker != nil {
			dm.cleanupTicker.Stop()
		}
		if dm.backupTicker != nil {
			dm.backupTicker.Stop()
		}

		dm.db.Close()
		dm.connected = false

		// Create new stop channel for the restored connection
		dm.stopChan = make(chan struct{})
	}

	// Copy backup file to database path
	if err := copyFile(path, dm.config.DatabasePath); err != nil {
		return fmt.Errorf("failed to restore backup: %w", err)
	}

	// Reconnect
	if err := dm.connectInternal(); err != nil {
		return fmt.Errorf("failed to reconnect after restore: %w", err)
	}

	log.Printf("Database restored from backup: %s", path)
	return nil
}

// buildConnectionString builds the SQLite connection string with options
func (dm *databaseManager) buildConnectionString() string {
	connStr := dm.config.DatabasePath + "?"

	if dm.config.WALMode {
		connStr += "journal_mode=WAL&"
	}

	connStr += fmt.Sprintf("synchronous=%s&", dm.config.SynchronousMode)
	connStr += fmt.Sprintf("cache_size=%d&", dm.config.CacheSize)
	connStr += "foreign_keys=ON"

	return connStr
}

// initializeComponents initializes all database components
func (dm *databaseManager) initializeComponents() error {
	// Initialize migration manager
	migrationManager, err := NewMigrationManager(dm.db)
	if err != nil {
		return fmt.Errorf("failed to create migration manager: %w", err)
	}
	dm.migrationManager = migrationManager

	// Initialize UMA repository
	umaRepository, err := NewUMARepository(dm.db)
	if err != nil {
		return fmt.Errorf("failed to create UMA repository: %w", err)
	}
	dm.umaRepository = umaRepository

	// Initialize metrics repository
	metricsRepository, err := NewMetricsRepository(dm.db, dm.config)
	if err != nil {
		return fmt.Errorf("failed to create metrics repository: %w", err)
	}
	dm.metricsRepository = metricsRepository

	return nil
}

// startBackgroundTasks starts background maintenance tasks
func (dm *databaseManager) startBackgroundTasks() {
	// Start cleanup task
	dm.cleanupTicker = time.NewTicker(dm.config.UMACacheCleanupInterval)
	go dm.runCleanupTask()

	// Start backup task if enabled
	if dm.config.BackupEnabled {
		dm.backupTicker = time.NewTicker(dm.config.BackupInterval)
		go dm.runBackupTask()
	}
}

// runCleanupTask runs the periodic cleanup task
func (dm *databaseManager) runCleanupTask() {
	for {
		select {
		case <-dm.cleanupTicker.C:
			if err := dm.CleanExpiredData(); err != nil {
				log.Printf("Error during cleanup: %v", err)
			}
		case <-dm.stopChan:
			return
		}
	}
}

// runBackupTask runs the periodic backup task
func (dm *databaseManager) runBackupTask() {
	for {
		select {
		case <-dm.backupTicker.C:
			backupPath := fmt.Sprintf("%s.backup.%d", dm.config.DatabasePath, time.Now().Unix())
			if err := dm.Backup(backupPath); err != nil {
				log.Printf("Error during backup: %v", err)
			}
		case <-dm.stopChan:
			return
		}
	}
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = destFile.ReadFrom(sourceFile)
	return err
}
