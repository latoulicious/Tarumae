package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// metricsRepository implements the MetricsRepository interface
type metricsRepository struct {
	db     *sql.DB
	config *DatabaseConfig

	// Enhanced components
	batchProcessor   *MetricsBatchProcessor
	retentionManager *MetricsRetentionManager

	// Prepared statements for performance
	insertMetricStmt  *sql.Stmt
	insertSessionStmt *sql.Stmt
	insertEventStmt   *sql.Stmt
	updateSessionStmt *sql.Stmt
}

// NewMetricsRepository creates a new metrics repository
func NewMetricsRepository(db *sql.DB, config *DatabaseConfig) (MetricsRepository, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}

	repo := &metricsRepository{
		db:     db,
		config: config,
	}

	// Initialize metrics tables
	if err := repo.initializeTables(); err != nil {
		return nil, fmt.Errorf("failed to initialize metrics tables: %w", err)
	}

	// Prepare statements
	if err := repo.prepareStatements(); err != nil {
		return nil, fmt.Errorf("failed to prepare statements: %w", err)
	}

	// Initialize batch processor
	batchProcessor, err := NewMetricsBatchProcessor(db, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create batch processor: %w", err)
	}
	repo.batchProcessor = batchProcessor

	// Initialize retention manager
	retentionManager := NewMetricsRetentionManager(db, config)
	repo.retentionManager = retentionManager

	// Start background components
	if err := repo.batchProcessor.Start(); err != nil {
		return nil, fmt.Errorf("failed to start batch processor: %w", err)
	}

	if err := repo.retentionManager.Start(); err != nil {
		repo.batchProcessor.Stop() // Cleanup on failure
		return nil, fmt.Errorf("failed to start retention manager: %w", err)
	}

	return repo, nil
}

// initializeTables creates the metrics tables if they don't exist
func (r *metricsRepository) initializeTables() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS pipeline_metrics (
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

		`CREATE TABLE IF NOT EXISTS pipeline_sessions (
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

		`CREATE TABLE IF NOT EXISTS pipeline_events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			pipeline_id TEXT NOT NULL,
			event_type TEXT NOT NULL,
			event_data TEXT NOT NULL,
			severity TEXT,
			timestamp DATETIME NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,

		// Create indexes for performance
		`CREATE INDEX IF NOT EXISTS idx_pipeline_metrics_pipeline_id ON pipeline_metrics(pipeline_id)`,
		`CREATE INDEX IF NOT EXISTS idx_pipeline_metrics_timestamp ON pipeline_metrics(timestamp)`,
		`CREATE INDEX IF NOT EXISTS idx_pipeline_metrics_name ON pipeline_metrics(metric_name)`,
		`CREATE INDEX IF NOT EXISTS idx_pipeline_metrics_type ON pipeline_metrics(metric_type)`,
		`CREATE INDEX IF NOT EXISTS idx_pipeline_sessions_pipeline_id ON pipeline_sessions(pipeline_id)`,
		`CREATE INDEX IF NOT EXISTS idx_pipeline_sessions_started_at ON pipeline_sessions(started_at)`,
		`CREATE INDEX IF NOT EXISTS idx_pipeline_events_pipeline_id ON pipeline_events(pipeline_id)`,
		`CREATE INDEX IF NOT EXISTS idx_pipeline_events_timestamp ON pipeline_events(timestamp)`,
		`CREATE INDEX IF NOT EXISTS idx_pipeline_events_type ON pipeline_events(event_type)`,
		`CREATE INDEX IF NOT EXISTS idx_pipeline_events_severity ON pipeline_events(severity)`,
	}

	for _, query := range queries {
		if _, err := r.db.Exec(query); err != nil {
			return fmt.Errorf("failed to execute query: %v", err)
		}
	}

	return nil
}

// prepareStatements prepares commonly used SQL statements
func (r *metricsRepository) prepareStatements() error {
	var err error

	// Prepare insert metric statement
	r.insertMetricStmt, err = r.db.Prepare(`
		INSERT INTO pipeline_metrics (pipeline_id, metric_name, metric_type, metric_value, tags, metadata, timestamp)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare insert metric statement: %w", err)
	}

	// Prepare insert session statement
	r.insertSessionStmt, err = r.db.Prepare(`
		INSERT INTO pipeline_sessions (pipeline_id, guild_id, channel_id, user_id, stream_url, started_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare insert session statement: %w", err)
	}

	// Prepare insert event statement
	r.insertEventStmt, err = r.db.Prepare(`
		INSERT INTO pipeline_events (pipeline_id, event_type, event_data, severity, timestamp)
		VALUES (?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare insert event statement: %w", err)
	}

	// Prepare update session statement
	r.updateSessionStmt, err = r.db.Prepare(`
		UPDATE pipeline_sessions 
		SET ended_at = ?, final_state = ?, total_errors = ?, total_recoveries = ?
		WHERE pipeline_id = ?
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare update session statement: %w", err)
	}

	return nil
}

// StoreMetric stores a single pipeline metric using batch processing for better performance
func (r *metricsRepository) StoreMetric(ctx context.Context, metric *PipelineMetric) error {
	// Use batch processor if available for better performance
	if r.batchProcessor != nil {
		return r.batchProcessor.AddMetric(metric)
	}

	// Fallback to direct storage if batch processor is not available
	return r.storeMetricDirect(ctx, metric)
}

// storeMetricDirect stores a single pipeline metric directly to the database
func (r *metricsRepository) storeMetricDirect(ctx context.Context, metric *PipelineMetric) error {
	tagsJSON, err := json.Marshal(metric.Tags)
	if err != nil {
		return fmt.Errorf("failed to marshal tags: %w", err)
	}

	metadataJSON, err := json.Marshal(metric.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	_, err = r.insertMetricStmt.ExecContext(ctx,
		metric.PipelineID,
		metric.MetricName,
		metric.MetricType,
		metric.MetricValue,
		string(tagsJSON),
		string(metadataJSON),
		metric.Timestamp,
	)

	if err != nil {
		return fmt.Errorf("failed to store metric: %w", err)
	}

	return nil
}

// StoreBatchMetrics stores multiple pipeline metrics efficiently using batch processing
func (r *metricsRepository) StoreBatchMetrics(ctx context.Context, metrics []*PipelineMetric) error {
	if len(metrics) == 0 {
		return nil
	}

	// Use batch processor if available for better performance
	if r.batchProcessor != nil {
		return r.batchProcessor.AddMetrics(metrics)
	}

	// Fallback to direct batch storage if batch processor is not available
	return r.storeBatchMetricsDirect(ctx, metrics)
}

// storeBatchMetricsDirect stores multiple pipeline metrics directly to the database
func (r *metricsRepository) storeBatchMetricsDirect(ctx context.Context, metrics []*PipelineMetric) error {
	if len(metrics) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt := tx.StmtContext(ctx, r.insertMetricStmt)

	for _, metric := range metrics {
		tagsJSON, err := json.Marshal(metric.Tags)
		if err != nil {
			return fmt.Errorf("failed to marshal tags: %w", err)
		}

		metadataJSON, err := json.Marshal(metric.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}

		_, err = stmt.ExecContext(ctx,
			metric.PipelineID,
			metric.MetricName,
			metric.MetricType,
			metric.MetricValue,
			string(tagsJSON),
			string(metadataJSON),
			metric.Timestamp,
		)

		if err != nil {
			return fmt.Errorf("failed to store metric in batch: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit batch metrics: %w", err)
	}

	return nil
}

// GetMetrics retrieves metrics based on query parameters
func (r *metricsRepository) GetMetrics(ctx context.Context, query *MetricsQuery) ([]*PipelineMetric, error) {
	sqlQuery, args := r.buildMetricsQuery(query)

	rows, err := r.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query metrics: %w", err)
	}
	defer rows.Close()

	var metrics []*PipelineMetric
	for rows.Next() {
		metric := &PipelineMetric{}
		var tagsJSON, metadataJSON string

		err := rows.Scan(
			&metric.ID,
			&metric.PipelineID,
			&metric.MetricName,
			&metric.MetricType,
			&metric.MetricValue,
			&tagsJSON,
			&metadataJSON,
			&metric.Timestamp,
			&metric.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan metric: %w", err)
		}

		if err := json.Unmarshal([]byte(tagsJSON), &metric.Tags); err != nil {
			return nil, fmt.Errorf("failed to unmarshal tags: %w", err)
		}

		if err := json.Unmarshal([]byte(metadataJSON), &metric.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}

		metrics = append(metrics, metric)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating metrics: %w", err)
	}

	return metrics, nil
}

// GetAggregatedMetrics retrieves aggregated metrics
func (r *metricsRepository) GetAggregatedMetrics(ctx context.Context, query *AggregationQuery) (*AggregatedMetrics, error) {
	sqlQuery, args := r.buildAggregationQuery(query)

	rows, err := r.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query aggregated metrics: %w", err)
	}
	defer rows.Close()

	result := &AggregatedMetrics{
		MetricName:   query.MetricName,
		Aggregation:  query.Aggregation,
		TimeInterval: query.TimeInterval,
		Results:      []*AggregatedMetricPoint{},
	}

	for rows.Next() {
		point := &AggregatedMetricPoint{}
		var tagsJSON string

		err := rows.Scan(&point.Value, &tagsJSON, &point.Timestamp)
		if err != nil {
			return nil, fmt.Errorf("failed to scan aggregated metric: %w", err)
		}

		if err := json.Unmarshal([]byte(tagsJSON), &point.Tags); err != nil {
			return nil, fmt.Errorf("failed to unmarshal tags: %w", err)
		}

		result.Results = append(result.Results, point)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating aggregated metrics: %w", err)
	}

	return result, nil
}

// CreateSession creates a new pipeline session
func (r *metricsRepository) CreateSession(ctx context.Context, session *PipelineSession) error {
	_, err := r.insertSessionStmt.ExecContext(ctx,
		session.PipelineID,
		session.GuildID,
		session.ChannelID,
		session.UserID,
		session.StreamURL,
		session.StartedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	return nil
}

// UpdateSession updates an existing pipeline session
func (r *metricsRepository) UpdateSession(ctx context.Context, sessionID string, updates *SessionUpdate) error {
	_, err := r.updateSessionStmt.ExecContext(ctx,
		updates.EndedAt,
		updates.FinalState,
		updates.TotalErrors,
		updates.TotalRecoveries,
		sessionID,
	)

	if err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}

	return nil
}

// GetSession retrieves a pipeline session by ID
func (r *metricsRepository) GetSession(ctx context.Context, sessionID string) (*PipelineSession, error) {
	query := `
		SELECT pipeline_id, guild_id, channel_id, user_id, stream_url, started_at, ended_at, 
		       final_state, total_errors, total_recoveries, created_at
		FROM pipeline_sessions 
		WHERE pipeline_id = ?
	`

	session := &PipelineSession{}
	var finalState sql.NullString
	err := r.db.QueryRowContext(ctx, query, sessionID).Scan(
		&session.PipelineID,
		&session.GuildID,
		&session.ChannelID,
		&session.UserID,
		&session.StreamURL,
		&session.StartedAt,
		&session.EndedAt,
		&finalState,
		&session.TotalErrors,
		&session.TotalRecoveries,
		&session.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrSessionNotFound
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	if finalState.Valid {
		session.FinalState = finalState.String
	}

	return session, nil
}

// GetActiveSessions retrieves all active pipeline sessions
func (r *metricsRepository) GetActiveSessions(ctx context.Context) ([]*PipelineSession, error) {
	query := `
		SELECT pipeline_id, guild_id, channel_id, user_id, stream_url, started_at, ended_at, 
		       final_state, total_errors, total_recoveries, created_at
		FROM pipeline_sessions 
		WHERE ended_at IS NULL
		ORDER BY started_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query active sessions: %w", err)
	}
	defer rows.Close()

	var sessions []*PipelineSession
	for rows.Next() {
		session := &PipelineSession{}
		var finalState sql.NullString
		err := rows.Scan(
			&session.PipelineID,
			&session.GuildID,
			&session.ChannelID,
			&session.UserID,
			&session.StreamURL,
			&session.StartedAt,
			&session.EndedAt,
			&finalState,
			&session.TotalErrors,
			&session.TotalRecoveries,
			&session.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan session: %w", err)
		}

		if finalState.Valid {
			session.FinalState = finalState.String
		}

		sessions = append(sessions, session)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating sessions: %w", err)
	}

	return sessions, nil
}

// StoreEvent stores a pipeline event
func (r *metricsRepository) StoreEvent(ctx context.Context, event *PipelineEvent) error {
	eventDataJSON, err := json.Marshal(event.EventData)
	if err != nil {
		return fmt.Errorf("failed to marshal event data: %w", err)
	}

	_, err = r.insertEventStmt.ExecContext(ctx,
		event.PipelineID,
		event.EventType,
		string(eventDataJSON),
		event.Severity,
		event.Timestamp,
	)

	if err != nil {
		return fmt.Errorf("failed to store event: %w", err)
	}

	return nil
}

// GetEvents retrieves events based on query parameters
func (r *metricsRepository) GetEvents(ctx context.Context, query *EventQuery) ([]*PipelineEvent, error) {
	sqlQuery, args := r.buildEventQuery(query)

	rows, err := r.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query events: %w", err)
	}
	defer rows.Close()

	var events []*PipelineEvent
	for rows.Next() {
		event := &PipelineEvent{}
		var eventDataJSON string

		err := rows.Scan(
			&event.ID,
			&event.PipelineID,
			&event.EventType,
			&eventDataJSON,
			&event.Severity,
			&event.Timestamp,
			&event.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}

		if err := json.Unmarshal([]byte(eventDataJSON), &event.EventData); err != nil {
			return nil, fmt.Errorf("failed to unmarshal event data: %w", err)
		}

		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating events: %w", err)
	}

	return events, nil
}

// CleanExpiredMetrics removes metrics older than the retention period
func (r *metricsRepository) CleanExpiredMetrics(ctx context.Context, retentionPeriod time.Duration) error {
	cutoffTime := time.Now().Add(-retentionPeriod)

	queries := []string{
		"DELETE FROM pipeline_metrics WHERE timestamp < ?",
		"DELETE FROM pipeline_events WHERE timestamp < ?",
		"DELETE FROM pipeline_sessions WHERE started_at < ? AND ended_at IS NOT NULL",
	}

	for _, query := range queries {
		if _, err := r.db.ExecContext(ctx, query, cutoffTime); err != nil {
			return fmt.Errorf("failed to clean expired metrics: %w", err)
		}
	}

	return nil
}

// GetMetricsStats returns statistics about stored metrics
func (r *metricsRepository) GetMetricsStats(ctx context.Context) (*MetricsStats, error) {
	stats := &MetricsStats{
		MetricsByType:    make(map[string]int64),
		EventsBySeverity: make(map[string]int64),
	}

	// Get total metrics count
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM pipeline_metrics").Scan(&stats.TotalMetrics)
	if err != nil {
		return nil, fmt.Errorf("failed to get total metrics count: %w", err)
	}

	// Get total sessions count
	err = r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM pipeline_sessions").Scan(&stats.TotalSessions)
	if err != nil {
		return nil, fmt.Errorf("failed to get total sessions count: %w", err)
	}

	// Get total events count
	err = r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM pipeline_events").Scan(&stats.TotalEvents)
	if err != nil {
		return nil, fmt.Errorf("failed to get total events count: %w", err)
	}

	// Get active sessions count
	err = r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM pipeline_sessions WHERE ended_at IS NULL").Scan(&stats.ActiveSessions)
	if err != nil {
		return nil, fmt.Errorf("failed to get active sessions count: %w", err)
	}

	// Get oldest and newest metric timestamps
	var oldestTimeStr, newestTimeStr sql.NullString
	err = r.db.QueryRowContext(ctx, "SELECT MIN(timestamp), MAX(timestamp) FROM pipeline_metrics").Scan(&oldestTimeStr, &newestTimeStr)
	if err != nil {
		return nil, fmt.Errorf("failed to get metric timestamp range: %w", err)
	}

	if oldestTimeStr.Valid {
		if t, err := time.Parse("2006-01-02 15:04:05", oldestTimeStr.String); err == nil {
			stats.OldestMetric = &t
		}
	}
	if newestTimeStr.Valid {
		if t, err := time.Parse("2006-01-02 15:04:05", newestTimeStr.String); err == nil {
			stats.NewestMetric = &t
		}
	}

	// Get metrics by type
	rows, err := r.db.QueryContext(ctx, "SELECT metric_type, COUNT(*) FROM pipeline_metrics GROUP BY metric_type")
	if err != nil {
		return nil, fmt.Errorf("failed to get metrics by type: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var metricType string
		var count int64
		if err := rows.Scan(&metricType, &count); err != nil {
			return nil, fmt.Errorf("failed to scan metric type stats: %w", err)
		}
		stats.MetricsByType[metricType] = count
	}

	// Get events by severity
	rows, err = r.db.QueryContext(ctx, "SELECT severity, COUNT(*) FROM pipeline_events GROUP BY severity")
	if err != nil {
		return nil, fmt.Errorf("failed to get events by severity: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var severity string
		var count int64
		if err := rows.Scan(&severity, &count); err != nil {
			return nil, fmt.Errorf("failed to scan event severity stats: %w", err)
		}
		stats.EventsBySeverity[severity] = count
	}

	return stats, nil
}

// buildMetricsQuery builds a SQL query for metrics retrieval
func (r *metricsRepository) buildMetricsQuery(query *MetricsQuery) (string, []interface{}) {
	sqlQuery := `
		SELECT id, pipeline_id, metric_name, metric_type, metric_value, tags, metadata, timestamp, created_at
		FROM pipeline_metrics
		WHERE 1=1
	`
	var args []interface{}

	if query.PipelineID != "" {
		sqlQuery += " AND pipeline_id = ?"
		args = append(args, query.PipelineID)
	}

	if len(query.MetricNames) > 0 {
		placeholders := strings.Repeat("?,", len(query.MetricNames)-1) + "?"
		sqlQuery += " AND metric_name IN (" + placeholders + ")"
		for _, name := range query.MetricNames {
			args = append(args, name)
		}
	}

	if len(query.MetricTypes) > 0 {
		placeholders := strings.Repeat("?,", len(query.MetricTypes)-1) + "?"
		sqlQuery += " AND metric_type IN (" + placeholders + ")"
		for _, metricType := range query.MetricTypes {
			args = append(args, metricType)
		}
	}

	if query.StartTime != nil {
		sqlQuery += " AND timestamp >= ?"
		args = append(args, *query.StartTime)
	}

	if query.EndTime != nil {
		sqlQuery += " AND timestamp <= ?"
		args = append(args, *query.EndTime)
	}

	sqlQuery += " ORDER BY timestamp DESC"

	if query.Limit > 0 {
		sqlQuery += " LIMIT ?"
		args = append(args, query.Limit)
	}

	if query.Offset > 0 {
		sqlQuery += " OFFSET ?"
		args = append(args, query.Offset)
	}

	return sqlQuery, args
}

// buildAggregationQuery builds a SQL query for aggregated metrics
func (r *metricsRepository) buildAggregationQuery(query *AggregationQuery) (string, []interface{}) {
	var aggregationFunc string
	switch query.Aggregation {
	case "sum":
		aggregationFunc = "SUM(metric_value)"
	case "avg":
		aggregationFunc = "AVG(metric_value)"
	case "min":
		aggregationFunc = "MIN(metric_value)"
	case "max":
		aggregationFunc = "MAX(metric_value)"
	case "count":
		aggregationFunc = "COUNT(*)"
	default:
		aggregationFunc = "AVG(metric_value)"
	}

	sqlQuery := fmt.Sprintf(`
		SELECT %s as value, tags, timestamp
		FROM pipeline_metrics
		WHERE metric_name = ?
	`, aggregationFunc)

	args := []interface{}{query.MetricName}

	if query.PipelineID != "" {
		sqlQuery += " AND pipeline_id = ?"
		args = append(args, query.PipelineID)
	}

	if query.MetricType != "" {
		sqlQuery += " AND metric_type = ?"
		args = append(args, query.MetricType)
	}

	if query.StartTime != nil {
		sqlQuery += " AND timestamp >= ?"
		args = append(args, *query.StartTime)
	}

	if query.EndTime != nil {
		sqlQuery += " AND timestamp <= ?"
		args = append(args, *query.EndTime)
	}

	if query.TimeInterval != "" {
		// Group by time interval (simplified implementation)
		sqlQuery += " GROUP BY datetime(timestamp, 'start of hour')"
	}

	sqlQuery += " ORDER BY timestamp"

	return sqlQuery, args
}

// buildEventQuery builds a SQL query for events retrieval
func (r *metricsRepository) buildEventQuery(query *EventQuery) (string, []interface{}) {
	sqlQuery := `
		SELECT id, pipeline_id, event_type, event_data, severity, timestamp, created_at
		FROM pipeline_events
		WHERE 1=1
	`
	var args []interface{}

	if query.PipelineID != "" {
		sqlQuery += " AND pipeline_id = ?"
		args = append(args, query.PipelineID)
	}

	if len(query.EventTypes) > 0 {
		placeholders := strings.Repeat("?,", len(query.EventTypes)-1) + "?"
		sqlQuery += " AND event_type IN (" + placeholders + ")"
		for _, eventType := range query.EventTypes {
			args = append(args, eventType)
		}
	}

	if len(query.Severities) > 0 {
		placeholders := strings.Repeat("?,", len(query.Severities)-1) + "?"
		sqlQuery += " AND severity IN (" + placeholders + ")"
		for _, severity := range query.Severities {
			args = append(args, severity)
		}
	}

	if query.StartTime != nil {
		sqlQuery += " AND timestamp >= ?"
		args = append(args, *query.StartTime)
	}

	if query.EndTime != nil {
		sqlQuery += " AND timestamp <= ?"
		args = append(args, *query.EndTime)
	}

	sqlQuery += " ORDER BY timestamp DESC"

	if query.Limit > 0 {
		sqlQuery += " LIMIT ?"
		args = append(args, query.Limit)
	}

	if query.Offset > 0 {
		sqlQuery += " OFFSET ?"
		args = append(args, query.Offset)
	}

	return sqlQuery, args
}

// Enhanced methods for batch processing and retention management

// GetBatchProcessorStats returns statistics from the batch processor
func (r *metricsRepository) GetBatchProcessorStats() (*BatchProcessorStats, error) {
	if r.batchProcessor == nil {
		return nil, fmt.Errorf("batch processor not initialized")
	}
	return r.batchProcessor.GetStats(), nil
}

// FlushPendingMetrics forces processing of any buffered metrics
func (r *metricsRepository) FlushPendingMetrics() error {
	if r.batchProcessor == nil {
		return fmt.Errorf("batch processor not initialized")
	}
	return r.batchProcessor.Flush()
}

// GetRetentionStats returns statistics from the retention manager
func (r *metricsRepository) GetRetentionStats() (*RetentionStats, error) {
	if r.retentionManager == nil {
		return nil, fmt.Errorf("retention manager not initialized")
	}
	return r.retentionManager.GetStats(), nil
}

// RunRetentionCleanup manually triggers a retention cleanup operation
func (r *metricsRepository) RunRetentionCleanup(ctx context.Context) (*RetentionStats, error) {
	if r.retentionManager == nil {
		return nil, fmt.Errorf("retention manager not initialized")
	}
	return r.retentionManager.RunCleanup(ctx)
}

// Close gracefully shuts down the metrics repository and its components
func (r *metricsRepository) Close() error {
	var errors []string

	// Stop batch processor
	if r.batchProcessor != nil {
		if err := r.batchProcessor.Stop(); err != nil {
			errors = append(errors, fmt.Sprintf("batch processor stop error: %v", err))
		}
	}

	// Stop retention manager
	if r.retentionManager != nil {
		if err := r.retentionManager.Stop(); err != nil {
			errors = append(errors, fmt.Sprintf("retention manager stop error: %v", err))
		}
	}

	// Close prepared statements
	if r.insertMetricStmt != nil {
		if err := r.insertMetricStmt.Close(); err != nil {
			errors = append(errors, fmt.Sprintf("insert metric statement close error: %v", err))
		}
	}

	if r.insertSessionStmt != nil {
		if err := r.insertSessionStmt.Close(); err != nil {
			errors = append(errors, fmt.Sprintf("insert session statement close error: %v", err))
		}
	}

	if r.insertEventStmt != nil {
		if err := r.insertEventStmt.Close(); err != nil {
			errors = append(errors, fmt.Sprintf("insert event statement close error: %v", err))
		}
	}

	if r.updateSessionStmt != nil {
		if err := r.updateSessionStmt.Close(); err != nil {
			errors = append(errors, fmt.Sprintf("update session statement close error: %v", err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("errors during close: %s", strings.Join(errors, "; "))
	}

	return nil
}

// Enhanced StoreMetric that uses batch processing for better performance
func (r *metricsRepository) StoreMetricBatch(ctx context.Context, metric *PipelineMetric) error {
	if r.batchProcessor == nil {
		// Fallback to direct storage if batch processor is not available
		return r.StoreMetric(ctx, metric)
	}

	return r.batchProcessor.AddMetric(metric)
}

// Enhanced StoreBatchMetrics that uses the batch processor
func (r *metricsRepository) StoreBatchMetricsEnhanced(ctx context.Context, metrics []*PipelineMetric) error {
	if r.batchProcessor == nil {
		// Fallback to direct storage if batch processor is not available
		return r.StoreBatchMetrics(ctx, metrics)
	}

	return r.batchProcessor.AddMetrics(metrics)
}
