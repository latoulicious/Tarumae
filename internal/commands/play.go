package commands

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

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
		sendEmbedMessage(s, m.ChannelID, "❌ Usage Error", "Please provide a YouTube URL or search query.", 0xff0000)
		return
	}

	guildID := m.GuildID

	// Update activity for idle monitoring
	updateActivity(guildID)

	input := args[0]
	var url, title string
	var duration time.Duration
	var videoURL string // Store the video URL for search results

	// Check if input is a URL or search query
	if common.IsURL(input) {
		// Input is a URL, use existing logic
		streamURL, streamTitle, streamDuration, err := common.GetYouTubeAudioStreamWithMetadata(input)
		if err != nil {
			log.Printf("Error fetching stream URL: %v", err)
			sendEmbedMessage(s, m.ChannelID, "❌ Error", "Failed to get audio stream. Please check the URL.", 0xff0000)
			return
		}
		url = streamURL
		title = streamTitle
		duration = streamDuration
		videoURL = input // For direct URLs, use the input as video URL
	} else {
		// Input is a search query, search YouTube and get the first result
		searchQuery := strings.Join(args, " ") // Join all args as search query
		log.Printf("Treating input as search query: %s", searchQuery)

		// Search for the video and get its URL
		foundVideoURL, _, _, searchErr := common.SearchYouTubeAndGetURL(searchQuery)
		if searchErr != nil {
			log.Printf("Error searching YouTube: %v", searchErr)
			sendEmbedMessage(s, m.ChannelID, "❌ Search Error", "Failed to find any videos for your search query.", 0xff0000)
			return
		}

		// Now get the audio stream from the found video URL
		streamURL, streamTitle, streamDuration, streamErr := common.GetYouTubeAudioStreamWithMetadata(foundVideoURL)
		if streamErr != nil {
			log.Printf("Error fetching stream URL from search result: %v", streamErr)
			sendEmbedMessage(s, m.ChannelID, "❌ Error", "Failed to get audio stream from search result.", 0xff0000)
			return
		}

		url = streamURL
		title = streamTitle
		duration = streamDuration
		videoURL = foundVideoURL // Store the found video URL
	}

	// Get or create queue for this guild
	queue := getOrCreateQueue(guildID)

	// Check if it's a YouTube URL and extract video ID
	var videoID string
	var originalURL string

	// Check if the video URL is a YouTube URL
	if common.IsYouTubeURL(videoURL) {
		videoID = common.ExtractYouTubeVideoID(videoURL)
		originalURL = videoURL
		// Use the new method for YouTube videos
		queue.AddWithYouTubeData(url, originalURL, videoID, title, m.Author.Username, duration)
	} else {
		// Use the original method for non-YouTube URLs
		queue.Add(url, title, m.Author.Username)
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
