package database

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"
)

// MetricsRetentionManager handles cleanup and retention policies for pipeline metrics
type MetricsRetentionManager struct {
	db     *sql.DB
	config *DatabaseConfig
	logger Logger

	// Retention policies
	policies []RetentionPolicy

	// Control channels
	stopChan chan struct{}
	doneChan chan struct{}

	// State
	running bool
	mutex   sync.RWMutex

	// Statistics
	lastCleanupTime time.Time
	totalCleaned    int64
	statsMutex      sync.RWMutex
}

// RetentionPolicy defines a cleanup policy for metrics
type RetentionPolicy struct {
	Name            string        `json:"name"`
	Description     string        `json:"description"`
	RetentionPeriod time.Duration `json:"retention_period"`
	TableName       string        `json:"table_name"`
	TimestampColumn string        `json:"timestamp_column"`
	Conditions      []string      `json:"conditions,omitempty"` // Additional WHERE conditions
	Priority        int           `json:"priority"`             // Lower number = higher priority
	Enabled         bool          `json:"enabled"`
}

// RetentionStats holds statistics about retention operations
type RetentionStats struct {
	LastCleanupTime    time.Time                `json:"last_cleanup_time"`
	TotalCleaned       int64                    `json:"total_cleaned"`
	CleanupsByPolicy   map[string]int64         `json:"cleanups_by_policy"`
	LastPolicyResults  map[string]*PolicyResult `json:"last_policy_results"`
	NextScheduledRun   time.Time                `json:"next_scheduled_run"`
	AverageCleanupTime time.Duration            `json:"average_cleanup_time"`
}

// PolicyResult holds the result of applying a retention policy
type PolicyResult struct {
	PolicyName     string        `json:"policy_name"`
	RecordsFound   int64         `json:"records_found"`
	RecordsCleaned int64         `json:"records_cleaned"`
	ExecutionTime  time.Duration `json:"execution_time"`
	Error          string        `json:"error,omitempty"`
	Timestamp      time.Time     `json:"timestamp"`
}

// NewMetricsRetentionManager creates a new metrics retention manager
func NewMetricsRetentionManager(db *sql.DB, config *DatabaseConfig) *MetricsRetentionManager {
	manager := &MetricsRetentionManager{
		db:       db,
		config:   config,
		logger:   &defaultLogger{},
		policies: getDefaultRetentionPolicies(config),
		stopChan: make(chan struct{}),
		doneChan: make(chan struct{}),
	}

	return manager
}

// SetLogger sets a custom logger for the retention manager
func (m *MetricsRetentionManager) SetLogger(logger Logger) {
	m.logger = logger
}

// AddPolicy adds a custom retention policy
func (m *MetricsRetentionManager) AddPolicy(policy RetentionPolicy) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Insert policy in priority order
	inserted := false
	for i, existing := range m.policies {
		if policy.Priority < existing.Priority {
			// Insert at position i
			m.policies = append(m.policies[:i], append([]RetentionPolicy{policy}, m.policies[i:]...)...)
			inserted = true
			break
		}
	}

	if !inserted {
		m.policies = append(m.policies, policy)
	}

	m.logger.Printf("Added retention policy: %s (priority: %d)", policy.Name, policy.Priority)
}

// RemovePolicy removes a retention policy by name
func (m *MetricsRetentionManager) RemovePolicy(name string) bool {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	for i, policy := range m.policies {
		if policy.Name == name {
			m.policies = append(m.policies[:i], m.policies[i+1:]...)
			m.logger.Printf("Removed retention policy: %s", name)
			return true
		}
	}

	return false
}

// GetPolicies returns a copy of all retention policies
func (m *MetricsRetentionManager) GetPolicies() []RetentionPolicy {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	policies := make([]RetentionPolicy, len(m.policies))
	copy(policies, m.policies)
	return policies
}

// Start begins the retention manager background process
func (m *MetricsRetentionManager) Start() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.running {
		return fmt.Errorf("retention manager is already running")
	}

	m.running = true
	go m.runRetentionLoop()

	m.logger.Printf("MetricsRetentionManager started with %d policies", len(m.policies))
	return nil
}

// Stop gracefully shuts down the retention manager
func (m *MetricsRetentionManager) Stop() error {
	m.mutex.Lock()
	if !m.running {
		m.mutex.Unlock()
		return nil
	}
	m.mutex.Unlock()

	close(m.stopChan)

	// Wait for shutdown with timeout
	select {
	case <-m.doneChan:
		m.logger.Printf("MetricsRetentionManager stopped gracefully")
	case <-time.After(30 * time.Second):
		m.logger.Errorf("MetricsRetentionManager stop timeout")
	}

	m.mutex.Lock()
	m.running = false
	m.mutex.Unlock()

	return nil
}

// RunCleanup manually triggers a cleanup operation
func (m *MetricsRetentionManager) RunCleanup(ctx context.Context) (*RetentionStats, error) {
	return m.executeCleanup(ctx)
}

// GetStats returns current retention statistics
func (m *MetricsRetentionManager) GetStats() *RetentionStats {
	m.statsMutex.RLock()
	defer m.statsMutex.RUnlock()

	return &RetentionStats{
		LastCleanupTime:   m.lastCleanupTime,
		TotalCleaned:      m.totalCleaned,
		CleanupsByPolicy:  make(map[string]int64),         // TODO: Implement detailed tracking
		LastPolicyResults: make(map[string]*PolicyResult), // TODO: Implement result tracking
		NextScheduledRun:  m.getNextScheduledRun(),
	}
}

// runRetentionLoop runs the main retention cleanup loop
func (m *MetricsRetentionManager) runRetentionLoop() {
	defer close(m.doneChan)

	// Calculate cleanup interval (default to 1 hour)
	cleanupInterval := 1 * time.Hour
	if m.config.UMACacheCleanupInterval > 0 {
		cleanupInterval = m.config.UMACacheCleanupInterval
	}

	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()

	// Run initial cleanup after a short delay
	initialDelay := time.NewTimer(30 * time.Second)
	defer initialDelay.Stop()

	for {
		select {
		case <-initialDelay.C:
			// Run initial cleanup
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
			if _, err := m.executeCleanup(ctx); err != nil {
				m.logger.Errorf("Initial cleanup failed: %v", err)
			}
			cancel()

		case <-ticker.C:
			// Run scheduled cleanup
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
			if _, err := m.executeCleanup(ctx); err != nil {
				m.logger.Errorf("Scheduled cleanup failed: %v", err)
			}
			cancel()

		case <-m.stopChan:
			return
		}
	}
}

// executeCleanup executes all enabled retention policies
func (m *MetricsRetentionManager) executeCleanup(ctx context.Context) (*RetentionStats, error) {
	startTime := time.Now()
	m.logger.Printf("Starting metrics retention cleanup")

	m.mutex.RLock()
	policies := make([]RetentionPolicy, len(m.policies))
	copy(policies, m.policies)
	m.mutex.RUnlock()

	var totalCleaned int64
	policyResults := make(map[string]*PolicyResult)

	for _, policy := range policies {
		if !policy.Enabled {
			continue
		}

		result := m.executePolicy(ctx, policy)
		policyResults[policy.Name] = result
		totalCleaned += result.RecordsCleaned

		if result.Error != "" {
			m.logger.Errorf("Policy %s failed: %s", policy.Name, result.Error)
		} else {
			m.logger.Printf("Policy %s cleaned %d records in %v",
				policy.Name, result.RecordsCleaned, result.ExecutionTime)
		}
	}

	// Update statistics
	m.statsMutex.Lock()
	m.lastCleanupTime = startTime
	m.totalCleaned += totalCleaned
	m.statsMutex.Unlock()

	executionTime := time.Since(startTime)
	m.logger.Printf("Metrics retention cleanup completed: %d records cleaned in %v",
		totalCleaned, executionTime)

	return &RetentionStats{
		LastCleanupTime:    startTime,
		TotalCleaned:       m.totalCleaned,
		LastPolicyResults:  policyResults,
		NextScheduledRun:   m.getNextScheduledRun(),
		AverageCleanupTime: executionTime,
	}, nil
}

// executePolicy executes a single retention policy
func (m *MetricsRetentionManager) executePolicy(ctx context.Context, policy RetentionPolicy) *PolicyResult {
	startTime := time.Now()
	result := &PolicyResult{
		PolicyName: policy.Name,
		Timestamp:  startTime,
	}

	// Calculate cutoff time
	cutoffTime := time.Now().Add(-policy.RetentionPeriod)

	// Build query
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s < ?",
		policy.TableName, policy.TimestampColumn)

	// Add additional conditions
	args := []interface{}{cutoffTime}
	for _, condition := range policy.Conditions {
		query += " AND " + condition
	}

	// Count records to be deleted
	err := m.db.QueryRowContext(ctx, query, args...).Scan(&result.RecordsFound)
	if err != nil {
		result.Error = fmt.Sprintf("failed to count records: %v", err)
		result.ExecutionTime = time.Since(startTime)
		return result
	}

	if result.RecordsFound == 0 {
		result.ExecutionTime = time.Since(startTime)
		return result
	}

	// Execute deletion
	deleteQuery := fmt.Sprintf("DELETE FROM %s WHERE %s < ?",
		policy.TableName, policy.TimestampColumn)

	// Add additional conditions to delete query
	for _, condition := range policy.Conditions {
		deleteQuery += " AND " + condition
	}

	deleteResult, err := m.db.ExecContext(ctx, deleteQuery, args...)
	if err != nil {
		result.Error = fmt.Sprintf("failed to delete records: %v", err)
		result.ExecutionTime = time.Since(startTime)
		return result
	}

	rowsAffected, err := deleteResult.RowsAffected()
	if err != nil {
		result.Error = fmt.Sprintf("failed to get rows affected: %v", err)
		result.ExecutionTime = time.Since(startTime)
		return result
	}

	result.RecordsCleaned = rowsAffected
	result.ExecutionTime = time.Since(startTime)
	return result
}

// getNextScheduledRun calculates the next scheduled cleanup run
func (m *MetricsRetentionManager) getNextScheduledRun() time.Time {
	cleanupInterval := 1 * time.Hour
	if m.config.UMACacheCleanupInterval > 0 {
		cleanupInterval = m.config.UMACacheCleanupInterval
	}

	m.statsMutex.RLock()
	lastRun := m.lastCleanupTime
	m.statsMutex.RUnlock()

	if lastRun.IsZero() {
		return time.Now().Add(30 * time.Second) // Initial run
	}

	return lastRun.Add(cleanupInterval)
}

// getDefaultRetentionPolicies returns the default set of retention policies
func getDefaultRetentionPolicies(config *DatabaseConfig) []RetentionPolicy {
	policies := []RetentionPolicy{
		{
			Name:            "metrics_retention",
			Description:     "Clean up old pipeline metrics",
			RetentionPeriod: config.MetricsRetention,
			TableName:       "pipeline_metrics",
			TimestampColumn: "timestamp",
			Priority:        1,
			Enabled:         true,
		},
		{
			Name:            "events_retention",
			Description:     "Clean up old pipeline events",
			RetentionPeriod: config.MetricsRetention,
			TableName:       "pipeline_events",
			TimestampColumn: "timestamp",
			Priority:        2,
			Enabled:         true,
		},
		{
			Name:            "completed_sessions_retention",
			Description:     "Clean up old completed pipeline sessions",
			RetentionPeriod: config.MetricsRetention * 2, // Keep sessions longer
			TableName:       "pipeline_sessions",
			TimestampColumn: "started_at",
			Conditions:      []string{"ended_at IS NOT NULL"}, // Only completed sessions
			Priority:        3,
			Enabled:         true,
		},
		{
			Name:            "low_priority_metrics_retention",
			Description:     "Aggressive cleanup of debug/trace level metrics",
			RetentionPeriod: config.MetricsRetention / 2, // Clean up faster
			TableName:       "pipeline_metrics",
			TimestampColumn: "timestamp",
			Conditions:      []string{"JSON_EXTRACT(tags, '$.level') IN ('debug', 'trace')"},
			Priority:        4,
			Enabled:         true,
		},
	}

	return policies
}
