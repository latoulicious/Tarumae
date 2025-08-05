package commands

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
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

// PlayCommand handles the play command with robust error handling and recovery
func PlayCommand(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	if len(args) < 1 {
		s.ChannelMessageSend(m.ChannelID, "Please provide a YouTube URL.")
		return
	}

	guildID := m.GuildID
	url := args[0]

	// Check if there's already an active pipeline for this guild
	pipelineMutex.RLock()
	if pipeline, exists := activePipelines[guildID]; exists && pipeline.IsPlaying() {
		pipelineMutex.RUnlock()
		s.ChannelMessageSend(m.ChannelID, "Already playing audio! Use `!stop` first.")
		return
	}
	pipelineMutex.RUnlock()

	log.Printf("Processing play command for URL: %s in guild: %s", url, guildID)
	s.ChannelMessageSend(m.ChannelID, "🎵 Fetching audio stream, please wait...")

	// Validate and get stream URL
	streamURL, title, err := getYouTubeAudioStreamWithMetadata(url)
	if err != nil {
		log.Printf("Error fetching stream URL: %v", err)
		s.ChannelMessageSend(m.ChannelID, "❌ Failed to get audio stream. Please check the URL.")
		return
	}

	// Find user's voice channel and connect
	vc, err := common.FindAndJoinUserVoiceChannel(s, m.Author.ID, guildID)
	if err != nil {
		log.Printf("Error joining voice channel: %v", err)
		s.ChannelMessageSend(m.ChannelID, "❌ "+err.Error())
		return
	}

	// Create and start the robust audio pipeline
	pipeline := common.NewAudioPipeline(vc)

	// Store the pipeline for management
	pipelineMutex.Lock()
	activePipelines[guildID] = pipeline
	pipelineMutex.Unlock()

	// Clean up pipeline when done
	defer func() {
		pipelineMutex.Lock()
		delete(activePipelines, guildID)
		pipelineMutex.Unlock()
	}()

	// Send now playing message
	nowPlayingMsg := fmt.Sprintf("🎶 Now playing: **%s**", title)
	s.ChannelMessageSend(m.ChannelID, nowPlayingMsg)

	// Start streaming with error handling
	err = pipeline.PlayStream(streamURL)
	if err != nil {
		log.Printf("Error starting audio pipeline: %v", err)
		s.ChannelMessageSend(m.ChannelID, "❌ Failed to start audio playback.")
		vc.Disconnect()
		return
	}

	log.Printf("Audio pipeline started successfully for guild: %s", guildID)

	// Monitor the pipeline and handle completion
	go func() {
		// Wait for pipeline to finish (either naturally or due to error)
		for pipeline.IsPlaying() {
			time.Sleep(1 * time.Second)
		}

		log.Printf("Audio playback completed for guild: %s", guildID)
		s.ChannelMessageSend(m.ChannelID, "✅ Playback finished.")

		// Disconnect after a brief delay
		time.Sleep(2 * time.Second)
		vc.Disconnect()
	}()
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
		s.ChannelMessageSend(m.ChannelID, "🔇 No audio is currently playing.")
		return
	}

	s.ChannelMessageSend(m.ChannelID, "🎵 Audio is currently playing.")
}
