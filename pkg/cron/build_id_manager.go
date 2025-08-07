package cron

import (
	"log"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

// BuildIDManager manages automatic refresh of build IDs
type BuildIDManager struct {
	cron        *cron.Cron
	cronEntry   cron.EntryID
	refreshFunc func() error
	mutex       sync.RWMutex
	isRunning   bool
	schedule    string
}

// NewBuildIDManager creates a new build ID manager
func NewBuildIDManager(refreshFunc func() error) *BuildIDManager {
	return NewBuildIDManagerWithSchedule(refreshFunc, "0 0 */6 * * *") // Default: every 6 hours
}

// NewBuildIDManagerWithSchedule creates a new build ID manager with custom schedule
func NewBuildIDManagerWithSchedule(refreshFunc func() error, schedule string) *BuildIDManager {
	manager := &BuildIDManager{
		cron:        cron.New(cron.WithSeconds()),
		refreshFunc: refreshFunc,
		schedule:    schedule,
	}

	// Start the cron scheduler
	manager.cron.Start()

	// Schedule build ID refresh
	entryID, err := manager.cron.AddFunc(schedule, manager.refreshBuildID)
	if err != nil {
		log.Printf("Failed to schedule build ID refresh: %v", err)
	} else {
		manager.cronEntry = entryID
		log.Printf("Scheduled build ID refresh with schedule: %s", schedule)
	}

	// Initial build ID fetch
	go manager.refreshBuildID()

	return manager
}

// refreshBuildID performs the actual build ID refresh
func (bm *BuildIDManager) refreshBuildID() {
	bm.mutex.Lock()
	if bm.isRunning {
		bm.mutex.Unlock()
		log.Println("Build ID refresh already in progress, skipping...")
		return
	}
	bm.isRunning = true
	bm.mutex.Unlock()

	defer func() {
		bm.mutex.Lock()
		bm.isRunning = false
		bm.mutex.Unlock()
	}()

	log.Println("Starting build ID refresh...")

	if bm.refreshFunc != nil {
		if err := bm.refreshFunc(); err != nil {
			log.Printf("Failed to refresh build ID: %v", err)
		} else {
			log.Println("Build ID refresh completed successfully")
		}
	}
}

// Stop stops the cron scheduler
func (bm *BuildIDManager) Stop() {
	if bm.cron != nil {
		bm.cron.Stop()
		log.Println("Build ID manager stopped")
	}
}

// GetNextRun returns the next scheduled run time
func (bm *BuildIDManager) GetNextRun() time.Time {
	if bm.cron != nil {
		entries := bm.cron.Entries()
		if len(entries) > 0 {
			return entries[0].Next
		}
	}
	return time.Time{}
}

// IsRunning returns whether a refresh is currently in progress
func (bm *BuildIDManager) IsRunning() bool {
	bm.mutex.RLock()
	defer bm.mutex.RUnlock()
	return bm.isRunning
}

// GetSchedule returns the current cron schedule
func (bm *BuildIDManager) GetSchedule() string {
	return bm.schedule
}
