# Cron Package

This package provides automated cron job functionality for the HKTM bot, specifically for managing build ID refreshes.

## BuildIDManager

The `BuildIDManager` automatically refreshes build IDs from external APIs at scheduled intervals.

### Features

- **Automatic Scheduling**: Runs build ID refresh jobs at configurable intervals
- **Concurrent Safety**: Prevents multiple refresh operations from running simultaneously
- **Graceful Shutdown**: Properly stops cron jobs when the application shuts down
- **Configurable Schedule**: Supports custom cron schedules

### Usage

```go
// Create a new build ID manager with default schedule (every 6 hours)
manager := cron.NewBuildIDManager(refreshFunction)

// Create with custom schedule (every 2 hours)
manager := cron.NewBuildIDManagerWithSchedule(refreshFunction, "0 0 */2 * * *")

// Stop the manager
manager.Stop()

// Check if refresh is running
if manager.IsRunning() {
    fmt.Println("Refresh in progress...")
}

// Get next scheduled run
nextRun := manager.GetNextRun()
fmt.Printf("Next refresh at: %s\n", nextRun)

// Get current schedule
schedule := manager.GetSchedule()
fmt.Printf("Current schedule: %s\n", schedule)
```

### Cron Schedule Format

The schedule uses the standard cron format with seconds: `{second} {minute} {hour} {day} {month} {day-of-week}`

Examples:
- `"0 0 */6 * * *"` - Every 6 hours
- `"0 0 */2 * * *"` - Every 2 hours  
- `"0 0 0 * * *"` - Daily at midnight
- `"0 30 9 * * *"` - Daily at 9:30 AM

### Integration with Gametora Client

The build ID manager is automatically integrated with the `GametoraClient`:

```go
// The client automatically creates a build ID manager
client := uma.NewGametoraClient()

// The manager runs automatically in the background
// To stop it during shutdown:
client.StopBuildIDManager()
``` 