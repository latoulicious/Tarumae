package database

import (
	"database/sql"
	"time"
)

// DatabaseConfig holds configuration for the database manager
type DatabaseConfig struct {
	// Connection settings
	DatabasePath      string        `json:"database_path" yaml:"database_path"`
	MaxConnections    int           `json:"max_connections" yaml:"max_connections"`
	ConnectionTimeout time.Duration `json:"connection_timeout" yaml:"connection_timeout"`

	// Metrics persistence settings
	MetricsBatchSize     int           `json:"metrics_batch_size" yaml:"metrics_batch_size"`
	MetricsFlushInterval time.Duration `json:"metrics_flush_interval" yaml:"metrics_flush_interval"`
	MetricsRetention     time.Duration `json:"metrics_retention" yaml:"metrics_retention"`

	// UMA cache settings
	UMACacheRetention       time.Duration `json:"uma_cache_retention" yaml:"uma_cache_retention"`
	UMACacheCleanupInterval time.Duration `json:"uma_cache_cleanup_interval" yaml:"uma_cache_cleanup_interval"`

	// Performance settings
	WALMode         bool   `json:"wal_mode" yaml:"wal_mode"`
	SynchronousMode string `json:"synchronous_mode" yaml:"synchronous_mode"`
	CacheSize       int    `json:"cache_size" yaml:"cache_size"`

	// Backup settings
	BackupEnabled   bool          `json:"backup_enabled" yaml:"backup_enabled"`
	BackupInterval  time.Duration `json:"backup_interval" yaml:"backup_interval"`
	BackupRetention int           `json:"backup_retention" yaml:"backup_retention"`
}

// DefaultDatabaseConfig returns a configuration with sensible defaults
func DefaultDatabaseConfig() *DatabaseConfig {
	return &DatabaseConfig{
		DatabasePath:      "uma_cache.db",
		MaxConnections:    10,
		ConnectionTimeout: 30 * time.Second,

		MetricsBatchSize:     100,
		MetricsFlushInterval: 30 * time.Second,
		MetricsRetention:     7 * 24 * time.Hour, // 7 days

		UMACacheRetention:       24 * time.Hour, // 1 day
		UMACacheCleanupInterval: 1 * time.Hour,  // 1 hour

		WALMode:         true,
		SynchronousMode: "NORMAL",
		CacheSize:       -64000, // 64MB

		BackupEnabled:   false,
		BackupInterval:  24 * time.Hour, // Daily
		BackupRetention: 7,              // Keep 7 backups
	}
}

// Validate validates the database configuration
func (c *DatabaseConfig) Validate() error {
	if c.DatabasePath == "" {
		return ErrInvalidDatabasePath
	}
	if c.MaxConnections <= 0 {
		return ErrInvalidMaxConnections
	}
	if c.ConnectionTimeout <= 0 {
		return ErrInvalidConnectionTimeout
	}
	if c.MetricsBatchSize <= 0 {
		return ErrInvalidMetricsBatchSize
	}
	if c.MetricsFlushInterval <= 0 {
		return ErrInvalidMetricsFlushInterval
	}
	if c.MetricsRetention <= 0 {
		return ErrInvalidMetricsRetention
	}
	if c.UMACacheRetention <= 0 {
		return ErrInvalidUMACacheRetention
	}
	if c.UMACacheCleanupInterval <= 0 {
		return ErrInvalidUMACacheCleanupInterval
	}
	if c.SynchronousMode != "OFF" && c.SynchronousMode != "NORMAL" && c.SynchronousMode != "FULL" {
		return ErrInvalidSynchronousMode
	}
	return nil
}

// DatabaseStats holds statistics about the database
type DatabaseStats struct {
	UMAStats     *UMAStats     `json:"uma_stats"`
	MetricsStats *MetricsStats `json:"metrics_stats"`
	PoolStats    *PoolStats    `json:"pool_stats"`
	FileSize     int64         `json:"file_size"`
	LastBackup   *time.Time    `json:"last_backup,omitempty"`
}

// UMAStats holds statistics about UMA cache
type UMAStats struct {
	CharacterSearchCount   int `json:"character_search_count"`
	CharacterImagesCount   int `json:"character_images_count"`
	SupportCardSearchCount int `json:"support_card_search_count"`
	SupportCardListCount   int `json:"support_card_list_count"`
	GametoraSkillsCount    int `json:"gametora_skills_count"`
	TotalCacheCount        int `json:"total_cache_count"`
	ExpiredEntriesCount    int `json:"expired_entries_count"`
}

// MetricsStats holds statistics about pipeline metrics
type MetricsStats struct {
	TotalMetrics     int64            `json:"total_metrics"`
	TotalSessions    int64            `json:"total_sessions"`
	TotalEvents      int64            `json:"total_events"`
	ActiveSessions   int              `json:"active_sessions"`
	OldestMetric     *time.Time       `json:"oldest_metric,omitempty"`
	NewestMetric     *time.Time       `json:"newest_metric,omitempty"`
	MetricsByType    map[string]int64 `json:"metrics_by_type"`
	EventsBySeverity map[string]int64 `json:"events_by_severity"`
}

// PoolStats holds statistics about the connection pool
type PoolStats struct {
	MaxConnections     int `json:"max_connections"`
	ActiveConnections  int `json:"active_connections"`
	IdleConnections    int `json:"idle_connections"`
	WaitingConnections int `json:"waiting_connections"`
}

// Connection represents a database connection wrapper
type Connection struct {
	DB        *sql.DB
	InUse     bool
	CreatedAt time.Time
	LastUsed  time.Time
}

// PipelineMetric represents a single pipeline metric
type PipelineMetric struct {
	ID          int64                  `json:"id"`
	PipelineID  string                 `json:"pipeline_id"`
	MetricName  string                 `json:"metric_name"`
	MetricType  string                 `json:"metric_type"` // counter, gauge, histogram, timing
	MetricValue float64                `json:"metric_value"`
	Tags        map[string]string      `json:"tags,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
	CreatedAt   time.Time              `json:"created_at"`
}

// PipelineSession represents a pipeline session
type PipelineSession struct {
	ID              string     `json:"id"`
	PipelineID      string     `json:"pipeline_id"`
	GuildID         string     `json:"guild_id,omitempty"`
	ChannelID       string     `json:"channel_id,omitempty"`
	UserID          string     `json:"user_id,omitempty"`
	StreamURL       string     `json:"stream_url,omitempty"`
	StartedAt       time.Time  `json:"started_at"`
	EndedAt         *time.Time `json:"ended_at,omitempty"`
	FinalState      string     `json:"final_state,omitempty"`
	TotalErrors     int        `json:"total_errors"`
	TotalRecoveries int        `json:"total_recoveries"`
	CreatedAt       time.Time  `json:"created_at"`
}

// SessionUpdate represents updates to a pipeline session
type SessionUpdate struct {
	EndedAt         *time.Time `json:"ended_at,omitempty"`
	FinalState      *string    `json:"final_state,omitempty"`
	TotalErrors     *int       `json:"total_errors,omitempty"`
	TotalRecoveries *int       `json:"total_recoveries,omitempty"`
}

// PipelineEvent represents a pipeline event
type PipelineEvent struct {
	ID         int64                  `json:"id"`
	PipelineID string                 `json:"pipeline_id"`
	EventType  string                 `json:"event_type"` // state_change, error, recovery
	EventData  map[string]interface{} `json:"event_data"`
	Severity   string                 `json:"severity"` // low, medium, high, critical
	Timestamp  time.Time              `json:"timestamp"`
	CreatedAt  time.Time              `json:"created_at"`
}

// MetricsQuery represents a query for metrics
type MetricsQuery struct {
	PipelineID  string            `json:"pipeline_id,omitempty"`
	MetricNames []string          `json:"metric_names,omitempty"`
	MetricTypes []string          `json:"metric_types,omitempty"`
	Tags        map[string]string `json:"tags,omitempty"`
	StartTime   *time.Time        `json:"start_time,omitempty"`
	EndTime     *time.Time        `json:"end_time,omitempty"`
	Limit       int               `json:"limit,omitempty"`
	Offset      int               `json:"offset,omitempty"`
}

// AggregationQuery represents a query for aggregated metrics
type AggregationQuery struct {
	PipelineID   string            `json:"pipeline_id,omitempty"`
	MetricName   string            `json:"metric_name"`
	MetricType   string            `json:"metric_type,omitempty"`
	Tags         map[string]string `json:"tags,omitempty"`
	StartTime    *time.Time        `json:"start_time,omitempty"`
	EndTime      *time.Time        `json:"end_time,omitempty"`
	Aggregation  string            `json:"aggregation"` // sum, avg, min, max, count
	GroupBy      []string          `json:"group_by,omitempty"`
	TimeInterval string            `json:"time_interval,omitempty"` // 1m, 5m, 1h, 1d
}

// AggregatedMetrics represents aggregated metrics results
type AggregatedMetrics struct {
	MetricName   string                   `json:"metric_name"`
	Aggregation  string                   `json:"aggregation"`
	TimeInterval string                   `json:"time_interval,omitempty"`
	Results      []*AggregatedMetricPoint `json:"results"`
}

// AggregatedMetricPoint represents a single aggregated metric point
type AggregatedMetricPoint struct {
	Value     float64           `json:"value"`
	Tags      map[string]string `json:"tags,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
}

// EventQuery represents a query for events
type EventQuery struct {
	PipelineID string     `json:"pipeline_id,omitempty"`
	EventTypes []string   `json:"event_types,omitempty"`
	Severities []string   `json:"severities,omitempty"`
	StartTime  *time.Time `json:"start_time,omitempty"`
	EndTime    *time.Time `json:"end_time,omitempty"`
	Limit      int        `json:"limit,omitempty"`
	Offset     int        `json:"offset,omitempty"`
}

// Migration represents a database migration
type Migration struct {
	Version     int       `json:"version"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	AppliedAt   time.Time `json:"applied_at"`
	Checksum    string    `json:"checksum"`
}
