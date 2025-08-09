package database

import "errors"

// Database configuration errors
var (
	ErrInvalidDatabasePath            = errors.New("invalid database path")
	ErrInvalidMaxConnections          = errors.New("invalid max connections")
	ErrInvalidConnectionTimeout       = errors.New("invalid connection timeout")
	ErrInvalidMetricsBatchSize        = errors.New("invalid metrics batch size")
	ErrInvalidMetricsFlushInterval    = errors.New("invalid metrics flush interval")
	ErrInvalidMetricsRetention        = errors.New("invalid metrics retention")
	ErrInvalidUMACacheRetention       = errors.New("invalid UMA cache retention")
	ErrInvalidUMACacheCleanupInterval = errors.New("invalid UMA cache cleanup interval")
	ErrInvalidSynchronousMode         = errors.New("invalid synchronous mode")
)

// Database operation errors
var (
	ErrDatabaseNotConnected    = errors.New("database not connected")
	ErrConnectionPoolExhausted = errors.New("connection pool exhausted")
	ErrConnectionTimeout       = errors.New("connection timeout")
	ErrTransactionFailed       = errors.New("transaction failed")
	ErrMigrationFailed         = errors.New("migration failed")
	ErrBackupFailed            = errors.New("backup failed")
	ErrRestoreFailed           = errors.New("restore failed")
)

// Repository errors
var (
	ErrMetricNotFound     = errors.New("metric not found")
	ErrSessionNotFound    = errors.New("session not found")
	ErrEventNotFound      = errors.New("event not found")
	ErrInvalidMetricType  = errors.New("invalid metric type")
	ErrInvalidEventType   = errors.New("invalid event type")
	ErrInvalidSeverity    = errors.New("invalid severity")
	ErrInvalidAggregation = errors.New("invalid aggregation")
)

// Migration errors
var (
	ErrMigrationNotFound       = errors.New("migration not found")
	ErrInvalidMigrationVersion = errors.New("invalid migration version")
	ErrMigrationAlreadyApplied = errors.New("migration already applied")
	ErrCannotRollback          = errors.New("cannot rollback migration")
)
