package database

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// SessionManager provides high-level session management functionality
type SessionManager struct {
	metricsRepo MetricsRepository
	config      *DatabaseConfig

	// Session tracking
	activeSessions map[string]*PipelineSession
	sessionMutex   sync.RWMutex

	// Cleanup management
	cleanupTicker *time.Ticker
	stopChan      chan struct{}

	// Analytics cache
	analyticsCache *SessionAnalyticsCache
}

// SessionAnalyticsCache caches frequently requested analytics data
type SessionAnalyticsCache struct {
	lastUpdate    time.Time
	cacheDuration time.Duration

	// Cached analytics
	totalSessions          int64
	activeSessions         int64
	averageSessionDuration time.Duration
	errorRate              float64
	recoveryRate           float64

	mutex sync.RWMutex
}

// SessionAnalytics represents session analytics data
type SessionAnalytics struct {
	TotalSessions          int64            `json:"total_sessions"`
	ActiveSessions         int64            `json:"active_sessions"`
	CompletedSessions      int64            `json:"completed_sessions"`
	AverageSessionDuration time.Duration    `json:"average_session_duration"`
	ErrorRate              float64          `json:"error_rate"`
	RecoveryRate           float64          `json:"recovery_rate"`
	SessionsByState        map[string]int64 `json:"sessions_by_state"`
	SessionsByHour         map[int]int64    `json:"sessions_by_hour"`
	TopErrorTypes          []ErrorTypeCount `json:"top_error_types"`
}

// ErrorTypeCount represents error type statistics
type ErrorTypeCount struct {
	ErrorType string `json:"error_type"`
	Count     int64  `json:"count"`
}

// SessionCleanupStats represents cleanup operation statistics
type SessionCleanupStats struct {
	OrphanedSessionsCleaned int           `json:"orphaned_sessions_cleaned"`
	StaleSessionsUpdated    int           `json:"stale_sessions_updated"`
	CleanupDuration         time.Duration `json:"cleanup_duration"`
}

// NewSessionManager creates a new session manager
func NewSessionManager(metricsRepo MetricsRepository, config *DatabaseConfig) *SessionManager {
	return &SessionManager{
		metricsRepo:    metricsRepo,
		config:         config,
		activeSessions: make(map[string]*PipelineSession),
		stopChan:       make(chan struct{}),
		analyticsCache: &SessionAnalyticsCache{
			cacheDuration: 5 * time.Minute, // Cache analytics for 5 minutes
		},
	}
}

// Start starts the session manager and its background tasks
func (sm *SessionManager) Start() error {
	// Load active sessions from database
	if err := sm.loadActiveSessions(); err != nil {
		return fmt.Errorf("failed to load active sessions: %w", err)
	}

	// Start cleanup task
	sm.cleanupTicker = time.NewTicker(sm.config.UMACacheCleanupInterval)
	go sm.runCleanupTask()

	log.Printf("Session manager started with %d active sessions", len(sm.activeSessions))
	return nil
}

// Stop stops the session manager and its background tasks
func (sm *SessionManager) Stop() error {
	// Stop cleanup task
	select {
	case <-sm.stopChan:
		// Channel already closed
	default:
		close(sm.stopChan)
	}

	if sm.cleanupTicker != nil {
		sm.cleanupTicker.Stop()
	}

	log.Printf("Session manager stopped")
	return nil
}

// CreateSession creates a new pipeline session
func (sm *SessionManager) CreateSession(ctx context.Context, session *PipelineSession) error {
	// Store in database
	if err := sm.metricsRepo.CreateSession(ctx, session); err != nil {
		return fmt.Errorf("failed to create session in database: %w", err)
	}

	// Add to active sessions cache
	sm.sessionMutex.Lock()
	sm.activeSessions[session.PipelineID] = session
	sm.sessionMutex.Unlock()

	// Invalidate analytics cache
	sm.invalidateAnalyticsCache()

	log.Printf("Created session %s for guild %s", session.PipelineID, session.GuildID)
	return nil
}

// UpdateSession updates an existing pipeline session
func (sm *SessionManager) UpdateSession(ctx context.Context, sessionID string, updates *SessionUpdate) error {
	// Update in database
	if err := sm.metricsRepo.UpdateSession(ctx, sessionID, updates); err != nil {
		return fmt.Errorf("failed to update session in database: %w", err)
	}

	// Update active sessions cache
	sm.sessionMutex.Lock()
	if session, exists := sm.activeSessions[sessionID]; exists {
		if updates.EndedAt != nil {
			session.EndedAt = updates.EndedAt
			// Remove from active sessions if ended
			delete(sm.activeSessions, sessionID)
		}
		if updates.FinalState != nil {
			session.FinalState = *updates.FinalState
		}
		if updates.TotalErrors != nil {
			session.TotalErrors = *updates.TotalErrors
		}
		if updates.TotalRecoveries != nil {
			session.TotalRecoveries = *updates.TotalRecoveries
		}
	}
	sm.sessionMutex.Unlock()

	// Invalidate analytics cache
	sm.invalidateAnalyticsCache()

	log.Printf("Updated session %s", sessionID)
	return nil
}

// EndSession ends a pipeline session
func (sm *SessionManager) EndSession(ctx context.Context, sessionID string, finalState string) error {
	now := time.Now()
	updates := &SessionUpdate{
		EndedAt:    &now,
		FinalState: &finalState,
	}

	return sm.UpdateSession(ctx, sessionID, updates)
}

// GetSession retrieves a pipeline session by ID
func (sm *SessionManager) GetSession(ctx context.Context, sessionID string) (*PipelineSession, error) {
	// Try active sessions cache first
	sm.sessionMutex.RLock()
	if session, exists := sm.activeSessions[sessionID]; exists {
		sm.sessionMutex.RUnlock()
		return session, nil
	}
	sm.sessionMutex.RUnlock()

	// Fallback to database
	return sm.metricsRepo.GetSession(ctx, sessionID)
}

// GetActiveSessions returns all active sessions
func (sm *SessionManager) GetActiveSessions(ctx context.Context) ([]*PipelineSession, error) {
	sm.sessionMutex.RLock()
	sessions := make([]*PipelineSession, 0, len(sm.activeSessions))
	for _, session := range sm.activeSessions {
		sessions = append(sessions, session)
	}
	sm.sessionMutex.RUnlock()

	return sessions, nil
}

// GetSessionAnalytics returns comprehensive session analytics
func (sm *SessionManager) GetSessionAnalytics(ctx context.Context) (*SessionAnalytics, error) {
	// Check cache first
	sm.analyticsCache.mutex.RLock()
	if time.Since(sm.analyticsCache.lastUpdate) < sm.analyticsCache.cacheDuration {
		analytics := &SessionAnalytics{
			TotalSessions:          sm.analyticsCache.totalSessions,
			ActiveSessions:         sm.analyticsCache.activeSessions,
			AverageSessionDuration: sm.analyticsCache.averageSessionDuration,
			ErrorRate:              sm.analyticsCache.errorRate,
			RecoveryRate:           sm.analyticsCache.recoveryRate,
		}
		sm.analyticsCache.mutex.RUnlock()

		// Still need to fetch non-cached data
		if err := sm.populateDetailedAnalytics(ctx, analytics); err != nil {
			return nil, err
		}

		return analytics, nil
	}
	sm.analyticsCache.mutex.RUnlock()

	// Generate fresh analytics
	return sm.generateSessionAnalytics(ctx)
}

// GetSessionsByTimeRange returns sessions within a time range
func (sm *SessionManager) GetSessionsByTimeRange(ctx context.Context, startTime, endTime time.Time) ([]*PipelineSession, error) {
	// Use the metrics repository's session query extensions
	return sm.metricsRepo.GetSessionsByTimeRange(ctx, startTime, endTime)
}

// CleanupOrphanedSessions cleans up sessions that may have been left in an inconsistent state
func (sm *SessionManager) CleanupOrphanedSessions(ctx context.Context) (*SessionCleanupStats, error) {
	startTime := time.Now()
	stats := &SessionCleanupStats{}

	// Find sessions that have been active for too long (likely orphaned)
	cutoffTime := time.Now().Add(-2 * time.Hour) // Sessions active for more than 2 hours

	// Use the metrics repository's query extensions to get orphaned sessions
	orphanedSessions, err := sm.metricsRepo.GetOrphanedSessions(ctx, cutoffTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get orphaned sessions: %w", err)
	}

	for _, session := range orphanedSessions {
		// Mark as orphaned/timeout
		now := time.Now()
		finalState := "timeout"
		updates := &SessionUpdate{
			EndedAt:    &now,
			FinalState: &finalState,
		}

		if err := sm.UpdateSession(ctx, session.PipelineID, updates); err != nil {
			log.Printf("Failed to cleanup orphaned session %s: %v", session.PipelineID, err)
		} else {
			stats.OrphanedSessionsCleaned++
		}
	}

	stats.CleanupDuration = time.Since(startTime)
	log.Printf("Cleaned up %d orphaned sessions in %v", stats.OrphanedSessionsCleaned, stats.CleanupDuration)

	return stats, nil
}

// loadActiveSessions loads active sessions from the database into memory
func (sm *SessionManager) loadActiveSessions() error {
	ctx := context.Background()
	sessions, err := sm.metricsRepo.GetActiveSessions(ctx)
	if err != nil {
		return fmt.Errorf("failed to load active sessions: %w", err)
	}

	sm.sessionMutex.Lock()
	for _, session := range sessions {
		sm.activeSessions[session.PipelineID] = session
	}
	sm.sessionMutex.Unlock()

	return nil
}

// runCleanupTask runs the periodic cleanup task
func (sm *SessionManager) runCleanupTask() {
	for {
		select {
		case <-sm.cleanupTicker.C:
			ctx := context.Background()
			if _, err := sm.CleanupOrphanedSessions(ctx); err != nil {
				log.Printf("Error during session cleanup: %v", err)
			}
		case <-sm.stopChan:
			return
		}
	}
}

// generateSessionAnalytics generates comprehensive session analytics
func (sm *SessionManager) generateSessionAnalytics(ctx context.Context) (*SessionAnalytics, error) {
	analytics := &SessionAnalytics{
		SessionsByState: make(map[string]int64),
		SessionsByHour:  make(map[int]int64),
	}

	// Get basic stats from metrics repository
	metricsStats, err := sm.metricsRepo.GetMetricsStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get metrics stats: %w", err)
	}

	analytics.TotalSessions = metricsStats.TotalSessions
	analytics.ActiveSessions = int64(metricsStats.ActiveSessions)
	analytics.CompletedSessions = analytics.TotalSessions - analytics.ActiveSessions

	// Calculate additional analytics
	if err := sm.populateDetailedAnalytics(ctx, analytics); err != nil {
		return nil, err
	}

	// Update cache
	sm.updateAnalyticsCache(analytics)

	return analytics, nil
}

// populateDetailedAnalytics populates detailed analytics that require custom queries
func (sm *SessionManager) populateDetailedAnalytics(ctx context.Context, analytics *SessionAnalytics) error {
	// This would require custom queries in the metrics repository
	// For now, we'll provide basic implementations

	// Calculate error rate and recovery rate from events
	eventQuery := &EventQuery{
		EventTypes: []string{"error", "recovery"},
		Limit:      1000, // Last 1000 events
	}

	events, err := sm.metricsRepo.GetEvents(ctx, eventQuery)
	if err != nil {
		return fmt.Errorf("failed to get events for analytics: %w", err)
	}

	var errorCount, recoveryCount int64
	errorTypes := make(map[string]int64)

	for _, event := range events {
		switch event.EventType {
		case "error":
			errorCount++
			if errorType, ok := event.EventData["error_type"].(string); ok {
				errorTypes[errorType]++
			}
		case "recovery":
			recoveryCount++
		}
	}

	if errorCount > 0 {
		analytics.ErrorRate = float64(errorCount) / float64(len(events))
		analytics.RecoveryRate = float64(recoveryCount) / float64(errorCount)
	}

	// Convert error types to sorted list
	for errorType, count := range errorTypes {
		analytics.TopErrorTypes = append(analytics.TopErrorTypes, ErrorTypeCount{
			ErrorType: errorType,
			Count:     count,
		})
	}

	return nil
}

// updateAnalyticsCache updates the analytics cache
func (sm *SessionManager) updateAnalyticsCache(analytics *SessionAnalytics) {
	sm.analyticsCache.mutex.Lock()
	sm.analyticsCache.totalSessions = analytics.TotalSessions
	sm.analyticsCache.activeSessions = analytics.ActiveSessions
	sm.analyticsCache.averageSessionDuration = analytics.AverageSessionDuration
	sm.analyticsCache.errorRate = analytics.ErrorRate
	sm.analyticsCache.recoveryRate = analytics.RecoveryRate
	sm.analyticsCache.lastUpdate = time.Now()
	sm.analyticsCache.mutex.Unlock()
}

// invalidateAnalyticsCache invalidates the analytics cache
func (sm *SessionManager) invalidateAnalyticsCache() {
	sm.analyticsCache.mutex.Lock()
	sm.analyticsCache.lastUpdate = time.Time{} // Reset to zero time to force refresh
	sm.analyticsCache.mutex.Unlock()
}
