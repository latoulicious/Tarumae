package commands

import (
	"fmt"
	"log"
	"sync"

	"github.com/bwmarrin/discordgo"
	"github.com/latoulicious/Tarumae/pkg/common"
)

var (
	// Global pipeline manager to track active streams
	activePipelines = make(map[string]*common.AudioPipeline)
	pipelineMutex   sync.RWMutex
)

// PlayCommand handles the play command with queue integration
func PlayCommand(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	if len(args) < 1 {
		sendEmbedMessage(s, m.ChannelID, "❌ Usage Error", "Please provide a YouTube URL.", 0xff0000)
		return
	}

	guildID := m.GuildID

	// Update activity for idle monitoring
	updateActivity(guildID)

	url := args[0]

	// Get or create queue for this guild
	queue := getOrCreateQueue(guildID)

	// Validate and get stream URL with metadata
	streamURL, title, duration, err := common.GetYouTubeAudioStreamWithMetadata(url)
	if err != nil {
		log.Printf("Error fetching stream URL: %v", err)
		sendEmbedMessage(s, m.ChannelID, "❌ Error", "Failed to get audio stream. Please check the URL.", 0xff0000)
		return
	}

	// Check if it's a YouTube URL and extract video ID
	var videoID string
	var originalURL string
	if common.IsYouTubeURL(url) {
		videoID = common.ExtractYouTubeVideoID(url)
		originalURL = url
		// Use the new method for YouTube videos
		queue.AddWithYouTubeData(streamURL, originalURL, videoID, title, m.Author.Username, duration)
	} else {
		// Use the original method for non-YouTube URLs
		queue.Add(streamURL, title, m.Author.Username)
	}

	// Send confirmation with embed
	queueSize := queue.Size()
	description := fmt.Sprintf("✅ Added **%s** to queue (Position: %d)", title, queueSize)
	sendEmbedMessage(s, m.ChannelID, "🎵 Song Added", description, 0x00ff00)

	// If nothing is currently playing, start playing
	if !queue.IsPlaying() {
		startNextInQueue(s, m, queue)
	}
}

// StatusCommand shows the current playback status
func StatusCommand(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	guildID := m.GuildID

	pipelineMutex.RLock()
	pipeline, exists := activePipelines[guildID]
	pipelineMutex.RUnlock()

	if !exists || !pipeline.IsPlaying() {
		sendEmbedMessage(s, m.ChannelID, "🔇 No Audio", "No audio is currently playing.", 0x808080)
		return
	}

	sendEmbedMessage(s, m.ChannelID, "🎵 Audio Playing", "Audio is currently playing.", 0x00ff00)
}
