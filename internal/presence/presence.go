package presence

import (
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

var (
	currentPresence string
	presenceMutex   sync.RWMutex
)

// PresenceManager manages the bot's presence
type PresenceManager struct {
	session *discordgo.Session
}

// NewPresenceManager creates a new presence manager
func NewPresenceManager(session *discordgo.Session) *PresenceManager {
	return &PresenceManager{
		session: session,
	}
}

// UpdateDefaultPresence updates the bot's presence with server statistics
func (pm *PresenceManager) UpdateDefaultPresence() {
	// Get all guilds (servers) the bot is in
	guilds := pm.session.State.Guilds
	if len(guilds) == 0 {
		return
	}

	// Count total channels across all servers
	totalChannels := 0
	for _, guild := range guilds {
		if guild != nil {
			// Count text channels
			channels, err := pm.session.GuildChannels(guild.ID)
			if err != nil {
				log.Printf("Error getting channels for guild %s: %v", guild.ID, err)
				continue
			}
			totalChannels += len(channels)
		}
	}

	// Create presence data
	presence := &discordgo.UpdateStatusData{
		Status: "online",
		Activities: []*discordgo.Activity{
			{
				Name:  strconv.Itoa(totalChannels) + " channels",
				Type:  discordgo.ActivityTypeWatching,
				State: "in " + strconv.Itoa(len(guilds)) + " servers",
			},
		},
	}

	// Update the bot's presence
	err := pm.session.UpdateStatusComplex(*presence)
	if err != nil {
		log.Printf("Failed to update bot presence: %v", err)
	}

	presenceMutex.Lock()
	currentPresence = "default"
	presenceMutex.Unlock()
}

// UpdateMusicPresence updates the bot's presence to show currently playing music
func (pm *PresenceManager) UpdateMusicPresence(songTitle string) {
	presence := &discordgo.UpdateStatusData{
		Status: "online",
		Activities: []*discordgo.Activity{
			{
				Name:  "to",
				Type:  discordgo.ActivityTypeListening,
				State: songTitle,
			},
		},
	}

	err := pm.session.UpdateStatusComplex(*presence)
	if err != nil {
		log.Printf("Failed to update music presence: %v", err)
	}

	presenceMutex.Lock()
	currentPresence = "music"
	presenceMutex.Unlock()
}

// ClearMusicPresence clears the music presence and returns to default
func (pm *PresenceManager) ClearMusicPresence() {
	pm.UpdateDefaultPresence()
}

// GetCurrentPresence returns the current presence type
func (pm *PresenceManager) GetCurrentPresence() string {
	presenceMutex.RLock()
	defer presenceMutex.RUnlock()
	return currentPresence
}

// StartPeriodicUpdates starts a goroutine that updates the default presence periodically
func (pm *PresenceManager) StartPeriodicUpdates() {
	go func() {
		ticker := time.NewTicker(5 * time.Minute) // Update every 5 minutes
		defer ticker.Stop()

		for range ticker.C {
			// Only update if we're not showing music
			if pm.GetCurrentPresence() != "music" {
				pm.UpdateDefaultPresence()
			}
		}
	}()
}
