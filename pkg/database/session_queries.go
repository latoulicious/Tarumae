package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// SessionQueryExtensions provides additional query methods for session analytics
type SessionQueryExtensions struct {
	db *sql.DB
}

// NewSessionQueryExtensions creates a new session query extensions instance
func NewSessionQueryExtensions(db *sql.DB) *SessionQueryExtensions {
	return &SessionQueryExtensions{db: db}
}

// scanSession scans a database row into a PipelineSession struct
func scanSession(rows *sql.Rows) (*PipelineSession, error) {
	session := &PipelineSession{}
	var finalState sql.NullString
	var totalErrors, totalRecoveries sql.NullInt64

	err := rows.Scan(
		&session.PipelineID,
		&session.GuildID,
		&session.ChannelID,
		&session.UserID,
		&session.StreamURL,
		&session.StartedAt,
		&session.EndedAt,
		&finalState,
		&totalErrors,
		&totalRecoveries,
		&session.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	if finalState.Valid {
		session.FinalState = finalState.String
	}
	if totalErrors.Valid {
		session.TotalErrors = int(totalErrors.Int64)
	}
	if totalRecoveries.Valid {
		session.TotalRecoveries = int(totalRecoveries.Int64)
	}

	return session, nil
}

// GetSessionsByTimeRange retrieves sessions within a specific time range
func (sq *SessionQueryExtensions) GetSessionsByTimeRange(ctx context.Context, startTime, endTime time.Time) ([]*PipelineSession, error) {
	query := `
		SELECT pipeline_id, guild_id, channel_id, user_id, stream_url, started_at, ended_at, 
		       final_state, total_errors, total_recoveries, created_at
		FROM pipeline_sessions 
		WHERE started_at >= ? AND started_at <= ?
		ORDER BY started_at DESC
	`

	rows, err := sq.db.QueryContext(ctx, query, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to query sessions by time range: %w", err)
	}
	defer rows.Close()

	var sessions []*PipelineSession
	for rows.Next() {
		session, err := scanSession(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan session: %w", err)
		}
		sessions = append(sessions, session)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating sessions: %w", err)
	}

	return sessions, nil
}

// GetSessionsByGuild retrieves sessions for a specific guild
func (sq *SessionQueryExtensions) GetSessionsByGuild(ctx context.Context, guildID string, limit int) ([]*PipelineSession, error) {
	query := `
		SELECT pipeline_id, guild_id, channel_id, user_id, stream_url, started_at, ended_at, 
		       final_state, total_errors, total_recoveries, created_at
		FROM pipeline_sessions 
		WHERE guild_id = ?
		ORDER BY started_at DESC
		LIMIT ?
	`

	rows, err := sq.db.QueryContext(ctx, query, guildID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query sessions by guild: %w", err)
	}
	defer rows.Close()

	var sessions []*PipelineSession
	for rows.Next() {
		session, err := scanSession(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan session: %w", err)
		}
		sessions = append(sessions, session)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating sessions: %w", err)
	}

	return sessions, nil
}

// GetSessionDurationStats calculates session duration statistics
func (sq *SessionQueryExtensions) GetSessionDurationStats(ctx context.Context) (*SessionDurationStats, error) {
	query := `
		SELECT 
			COUNT(*) as total_sessions,
			AVG(CASE WHEN ended_at IS NOT NULL THEN 
				(julianday(ended_at) - julianday(started_at)) * 24 * 60 * 60 
				ELSE NULL END) as avg_duration_seconds,
			MIN(CASE WHEN ended_at IS NOT NULL THEN 
				(julianday(ended_at) - julianday(started_at)) * 24 * 60 * 60 
				ELSE NULL END) as min_duration_seconds,
			MAX(CASE WHEN ended_at IS NOT NULL THEN 
				(julianday(ended_at) - julianday(started_at)) * 24 * 60 * 60 
				ELSE NULL END) as max_duration_seconds
		FROM pipeline_sessions 
		WHERE ended_at IS NOT NULL
	`

	stats := &SessionDurationStats{}
	var avgDuration, minDuration, maxDuration sql.NullFloat64

	err := sq.db.QueryRowContext(ctx, query).Scan(
		&stats.TotalCompletedSessions,
		&avgDuration,
		&minDuration,
		&maxDuration,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get session duration stats: %w", err)
	}

	if avgDuration.Valid {
		stats.AverageDuration = time.Duration(avgDuration.Float64) * time.Second
	}
	if minDuration.Valid {
		stats.MinDuration = time.Duration(minDuration.Float64) * time.Second
	}
	if maxDuration.Valid {
		stats.MaxDuration = time.Duration(maxDuration.Float64) * time.Second
	}

	return stats, nil
}

// GetSessionsByState returns session counts grouped by final state
func (sq *SessionQueryExtensions) GetSessionsByState(ctx context.Context) (map[string]int64, error) {
	query := `
		SELECT 
			COALESCE(final_state, 'active') as state,
			COUNT(*) as count
		FROM pipeline_sessions 
		GROUP BY COALESCE(final_state, 'active')
	`

	rows, err := sq.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query sessions by state: %w", err)
	}
	defer rows.Close()

	stateCount := make(map[string]int64)
	for rows.Next() {
		var state string
		var count int64
		if err := rows.Scan(&state, &count); err != nil {
			return nil, fmt.Errorf("failed to scan state count: %w", err)
		}
		stateCount[state] = count
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating state counts: %w", err)
	}

	return stateCount, nil
}

// GetSessionsByHour returns session counts grouped by hour of day
func (sq *SessionQueryExtensions) GetSessionsByHour(ctx context.Context) (map[int]int64, error) {
	query := `
		SELECT 
			CAST(strftime('%H', started_at) AS INTEGER) as hour,
			COUNT(*) as count
		FROM pipeline_sessions 
		GROUP BY CAST(strftime('%H', started_at) AS INTEGER)
		ORDER BY hour
	`

	rows, err := sq.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query sessions by hour: %w", err)
	}
	defer rows.Close()

	hourCount := make(map[int]int64)
	for rows.Next() {
		var hour int
		var count int64
		if err := rows.Scan(&hour, &count); err != nil {
			return nil, fmt.Errorf("failed to scan hour count: %w", err)
		}
		hourCount[hour] = count
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating hour counts: %w", err)
	}

	return hourCount, nil
}

// GetTopErrorTypes returns the most common error types from events
func (sq *SessionQueryExtensions) GetTopErrorTypes(ctx context.Context, limit int) ([]ErrorTypeCount, error) {
	query := `
		SELECT 
			json_extract(event_data, '$.error_type') as error_type,
			COUNT(*) as count
		FROM pipeline_events 
		WHERE event_type = 'error' 
		AND json_extract(event_data, '$.error_type') IS NOT NULL
		GROUP BY json_extract(event_data, '$.error_type')
		ORDER BY count DESC
		LIMIT ?
	`

	rows, err := sq.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query top error types: %w", err)
	}
	defer rows.Close()

	var errorTypes []ErrorTypeCount
	for rows.Next() {
		var errorType ErrorTypeCount
		if err := rows.Scan(&errorType.ErrorType, &errorType.Count); err != nil {
			return nil, fmt.Errorf("failed to scan error type: %w", err)
		}
		errorTypes = append(errorTypes, errorType)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating error types: %w", err)
	}

	return errorTypes, nil
}

// GetOrphanedSessions returns sessions that have been active for too long
func (sq *SessionQueryExtensions) GetOrphanedSessions(ctx context.Context, cutoffTime time.Time) ([]*PipelineSession, error) {
	query := `
		SELECT pipeline_id, guild_id, channel_id, user_id, stream_url, started_at, ended_at, 
		       final_state, total_errors, total_recoveries, created_at
		FROM pipeline_sessions 
		WHERE started_at < ? AND ended_at IS NULL
		ORDER BY started_at ASC
	`

	rows, err := sq.db.QueryContext(ctx, query, cutoffTime)
	if err != nil {
		return nil, fmt.Errorf("failed to query orphaned sessions: %w", err)
	}
	defer rows.Close()

	var sessions []*PipelineSession
	for rows.Next() {
		session, err := scanSession(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan session: %w", err)
		}
		sessions = append(sessions, session)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating sessions: %w", err)
	}

	return sessions, nil
}

// GetSessionErrorRates calculates error rates for sessions
func (sq *SessionQueryExtensions) GetSessionErrorRates(ctx context.Context) (*SessionErrorRates, error) {
	query := `
		SELECT 
			COUNT(*) as total_sessions,
			COUNT(CASE WHEN total_errors > 0 THEN 1 END) as sessions_with_errors,
			AVG(CAST(total_errors AS REAL)) as avg_errors_per_session,
			AVG(CAST(total_recoveries AS REAL)) as avg_recoveries_per_session
		FROM pipeline_sessions
	`

	rates := &SessionErrorRates{}
	var avgErrors, avgRecoveries sql.NullFloat64

	err := sq.db.QueryRowContext(ctx, query).Scan(
		&rates.TotalSessions,
		&rates.SessionsWithErrors,
		&avgErrors,
		&avgRecoveries,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get session error rates: %w", err)
	}

	if avgErrors.Valid {
		rates.AverageErrorsPerSession = avgErrors.Float64
	}
	if avgRecoveries.Valid {
		rates.AverageRecoveriesPerSession = avgRecoveries.Float64
	}

	if rates.TotalSessions > 0 {
		rates.ErrorRate = float64(rates.SessionsWithErrors) / float64(rates.TotalSessions)
	}

	if rates.SessionsWithErrors > 0 {
		rates.RecoveryRate = rates.AverageRecoveriesPerSession / rates.AverageErrorsPerSession
	}

	return rates, nil
}

// SessionDurationStats represents session duration statistics
type SessionDurationStats struct {
	TotalCompletedSessions int           `json:"total_completed_sessions"`
	AverageDuration        time.Duration `json:"average_duration"`
	MinDuration            time.Duration `json:"min_duration"`
	MaxDuration            time.Duration `json:"max_duration"`
}

// SessionErrorRates represents session error rate statistics
type SessionErrorRates struct {
	TotalSessions               int64   `json:"total_sessions"`
	SessionsWithErrors          int64   `json:"sessions_with_errors"`
	ErrorRate                   float64 `json:"error_rate"`
	RecoveryRate                float64 `json:"recovery_rate"`
	AverageErrorsPerSession     float64 `json:"average_errors_per_session"`
	AverageRecoveriesPerSession float64 `json:"average_recoveries_per_session"`
}
