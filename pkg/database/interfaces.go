package database

import (
	"context"
	"time"

	"github.com/latoulicious/HKTM/pkg/uma"
)

// DatabaseManager defines the interface for the enhanced database manager
type DatabaseManager interface {
	// Connection management
	Connect() error
	Close() error
	Ping(ctx context.Context) error

	// Repository access
	UMARepository() UMARepository
	MetricsRepository() MetricsRepository

	// Migration management
	Migrate() error
	GetSchemaVersion() (int, error)

	// Health and maintenance
	CleanExpiredData() error
	GetStats() (*DatabaseStats, error)
	Backup(path string) error
	Restore(path string) error
}

// UMARepository defines the interface for UMA cache operations
type UMARepository interface {
	// Character operations
	CacheCharacterSearch(query string, result *uma.CharacterSearchResult, ttl time.Duration) error
	GetCachedCharacterSearch(query string) (*uma.CharacterSearchResult, error)
	CacheCharacterImages(characterID int, result *uma.CharacterImagesResult, ttl time.Duration) error
	GetCachedCharacterImages(characterID int) (*uma.CharacterImagesResult, error)

	// Support card operations
	CacheSupportCardSearch(query string, result *uma.SupportCardSearchResult, ttl time.Duration) error
	GetCachedSupportCardSearch(query string) (*uma.SupportCardSearchResult, error)
	CacheSupportCardList(result *uma.SupportCardListResult, ttl time.Duration) error
	GetCachedSupportCardList() (*uma.SupportCardListResult, error)

	// Gametora operations
	CacheGametoraSkills(query string, result *uma.SimplifiedGametoraSearchResult, ttl time.Duration) error
	GetCachedGametoraSkills(query string) (*uma.SimplifiedGametoraSearchResult, error)

	// Maintenance
	CleanExpiredCache() error
	GetCacheStats() (map[string]int, error)
}

// MetricsRepository defines the interface for pipeline metrics operations
type MetricsRepository interface {
	// Metrics operations
	StoreMetric(ctx context.Context, metric *PipelineMetric) error
	StoreBatchMetrics(ctx context.Context, metrics []*PipelineMetric) error
	GetMetrics(ctx context.Context, query *MetricsQuery) ([]*PipelineMetric, error)
	GetAggregatedMetrics(ctx context.Context, query *AggregationQuery) (*AggregatedMetrics, error)

	// Session operations
	CreateSession(ctx context.Context, session *PipelineSession) error
	UpdateSession(ctx context.Context, sessionID string, updates *SessionUpdate) error
	GetSession(ctx context.Context, sessionID string) (*PipelineSession, error)
	GetActiveSessions(ctx context.Context) ([]*PipelineSession, error)

	// Event operations
	StoreEvent(ctx context.Context, event *PipelineEvent) error
	GetEvents(ctx context.Context, query *EventQuery) ([]*PipelineEvent, error)

	// Maintenance
	CleanExpiredMetrics(ctx context.Context, retentionPeriod time.Duration) error
	GetMetricsStats(ctx context.Context) (*MetricsStats, error)
}

// MigrationManager defines the interface for database migrations
type MigrationManager interface {
	GetCurrentVersion() (int, error)
	GetLatestVersion() int
	Migrate() error
	MigrateTo(version int) error
	Rollback() error
	RollbackTo(version int) error
	GetMigrationHistory() ([]*Migration, error)
}

// ConnectionPool defines the interface for database connection pooling
type ConnectionPool interface {
	Get() (*Connection, error)
	Put(conn *Connection) error
	Close() error
	Stats() *PoolStats
}
