package commands

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
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

	// Validate and get stream URL
	streamURL, title, err := getYouTubeAudioStreamWithMetadata(url)
	if err != nil {
		log.Printf("Error fetching stream URL: %v", err)
		sendEmbedMessage(s, m.ChannelID, "❌ Error", "Failed to get audio stream. Please check the URL.", 0xff0000)
		return
	}

	// Add to queue
	queue.Add(streamURL, title, m.Author.Username)

	// Send confirmation with embed
	queueSize := queue.Size()
	description := fmt.Sprintf("✅ Added **%s** to queue (Position: %d)", title, queueSize)
	sendEmbedMessage(s, m.ChannelID, "🎵 Song Added", description, 0x00ff00)

	// If nothing is currently playing, start playing
	if !queue.IsPlaying() {
		startNextInQueue(s, m, queue)
	}
}

// getYouTubeAudioStreamWithMetadata extracts both stream URL and metadata
func getYouTubeAudioStreamWithMetadata(url string) (streamURL, title string, err error) {
	log.Printf("Extracting audio stream from: %s", url)

	// First, get metadata
	metadataCmd := exec.Command("yt-dlp",
		"--no-playlist",
		"--no-warnings",
		"--print", "title",
		url)

	var titleOut bytes.Buffer
	metadataCmd.Stdout = &titleOut
	metadataCmd.Stderr = os.Stderr

	if err := metadataCmd.Run(); err != nil {
		log.Printf("Failed to get metadata: %v", err)
		title = "Unknown Title"
	} else {
		title = strings.TrimSpace(titleOut.String())
	}

	// Then get stream URL with multiple fallback strategies
	strategies := [][]string{
		// Strategy 1: Best audio with format preference
		{"-f", "bestaudio[ext=m4a]/bestaudio[ext=webm]/bestaudio[ext=mp4]/bestaudio"},

		// Strategy 2: Android client (often bypasses restrictions)
		{"-f", "bestaudio", "--extractor-args", "youtube:player_client=android"},

		// Strategy 3: Web client with cookies
		{"-f", "bestaudio", "--extractor-args", "youtube:player_client=web"},

		// Strategy 4: Last resort - any audio
		{"-f", "worst[ext=m4a]/worst"},
	}

	for i, strategy := range strategies {
		log.Printf("Trying extraction strategy %d/%d", i+1, len(strategies))

		args := append([]string{"--no-playlist", "--no-warnings", "-g"}, strategy...)
		args = append(args, url)

		cmd := exec.Command("yt-dlp", args...)
		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			log.Printf("Strategy %d failed: %v", i+1, err)
			continue
		}

		streamURL = strings.TrimSpace(out.String())
		if streamURL != "" {
			// Take first URL if multiple are returned
			urls := strings.Split(streamURL, "\n")
			if len(urls) > 0 && urls[0] != "" {
				streamURL = urls[0]
				log.Printf("Successfully extracted stream URL using strategy %d", i+1)
				return streamURL, title, nil
			}
		}
	}

	return "", title, fmt.Errorf("failed to extract audio stream URL after trying all strategies")
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
