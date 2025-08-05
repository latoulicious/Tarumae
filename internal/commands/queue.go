package commands

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/latoulicious/Tarumae/pkg/common"
)

var (
	// Global queue manager to track queues per guild
	queues     = make(map[string]*common.MusicQueue)
	queueMutex sync.RWMutex
)

// QueueCommand handles the queue command
func QueueCommand(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	if len(args) < 1 {
		// Show current queue
		showQueue(s, m)
		return
	}

	// Handle subcommands
	subcommand := strings.ToLower(args[0])
	switch subcommand {
	case "add":
		if len(args) < 2 {
			s.ChannelMessageSend(m.ChannelID, "Usage: `!queue add <youtube_url>`")
			return
		}
		addToQueue(s, m, args[1:])
	case "remove":
		if len(args) < 2 {
			s.ChannelMessageSend(m.ChannelID, "Usage: `!queue remove <index>`")
			return
		}
		removeFromQueue(s, m, args[1:])
	case "clear":
		clearQueue(s, m)
	case "list":
		showQueue(s, m)
	default:
		s.ChannelMessageSend(m.ChannelID, "Usage: `!queue [add|remove|clear|list] [args...]`")
	}
}

// addToQueue adds a song to the queue
func addToQueue(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	guildID := m.GuildID
	url := args[0]

	// Get or create queue for this guild
	queue := getOrCreateQueue(guildID)

	// Validate and get stream URL
	streamURL, title, err := getYouTubeAudioStreamWithMetadata(url)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, "❌ Failed to get audio stream. Please check the URL.")
		return
	}

	// Add to queue
	queue.Add(streamURL, title, m.Author.Username)

	// Send confirmation
	queueSize := queue.Size()
	response := fmt.Sprintf("✅ Added **%s** to queue (Position: %d)", title, queueSize)
	s.ChannelMessageSend(m.ChannelID, response)

	// If nothing is currently playing, start playing
	if !queue.IsPlaying() {
		startNextInQueue(s, m, queue)
	}
}

// removeFromQueue removes a song from the queue
func removeFromQueue(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	guildID := m.GuildID
	queue := getQueue(guildID)

	if queue == nil {
		s.ChannelMessageSend(m.ChannelID, "❌ No queue found for this server.")
		return
	}

	// Parse index
	var index int
	_, err := fmt.Sscanf(args[0], "%d", &index)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, "❌ Invalid index. Use `!queue list` to see queue positions.")
		return
	}

	// Adjust for 1-based indexing
	index--

	err = queue.Remove(index)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("❌ %s", err.Error()))
		return
	}

	s.ChannelMessageSend(m.ChannelID, "✅ Removed song from queue.")
}

// clearQueue clears the entire queue
func clearQueue(s *discordgo.Session, m *discordgo.MessageCreate) {
	guildID := m.GuildID
	queue := getQueue(guildID)

	if queue == nil {
		s.ChannelMessageSend(m.ChannelID, "❌ No queue found for this server.")
		return
	}

	queue.Clear()
	s.ChannelMessageSend(m.ChannelID, "✅ Queue cleared.")
}

// showQueue shows the current queue
func showQueue(s *discordgo.Session, m *discordgo.MessageCreate) {
	guildID := m.GuildID
	queue := getQueue(guildID)

	if queue == nil || (queue.Size() == 0 && queue.Current() == nil) {
		s.ChannelMessageSend(m.ChannelID, "📭 Queue is empty.")
		return
	}

	var response strings.Builder
	response.WriteString("🎵 **Music Queue**\n\n")

	// Show currently playing
	if current := queue.Current(); current != nil {
		response.WriteString(fmt.Sprintf("🎶 **Now Playing:** %s (Requested by: %s)\n\n",
			current.Title, current.RequestedBy))
	}

	// Show queue items
	items := queue.List()
	if len(items) > 0 {
		response.WriteString("📋 **Up Next:**\n")
		for i, item := range items {
			response.WriteString(fmt.Sprintf("%d. **%s** (Requested by: %s)\n",
				i+1, item.Title, item.RequestedBy))
		}
	} else {
		response.WriteString("📋 No songs in queue.\n")
	}

	s.ChannelMessageSend(m.ChannelID, response.String())
}

// startNextInQueue starts playing the next song in the queue
func startNextInQueue(s *discordgo.Session, m *discordgo.MessageCreate, queue *common.MusicQueue) {
	item := queue.Next()
	if item == nil {
		queue.SetPlaying(false)
		return
	}

	queue.SetPlaying(true)

	// Find user's voice channel and connect
	vc, err := common.FindAndJoinUserVoiceChannel(s, m.Author.ID, m.GuildID)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, "❌ "+err.Error())
		queue.SetPlaying(false)
		return
	}

	queue.SetVoiceConnection(vc)

	// Create and start the audio pipeline
	pipeline := common.NewAudioPipeline(vc)
	queue.SetPipeline(pipeline)

	// Send now playing message
	nowPlayingMsg := fmt.Sprintf("🎶 Now playing: **%s** (Requested by: %s)",
		item.Title, item.RequestedBy)
	s.ChannelMessageSend(m.ChannelID, nowPlayingMsg)

	// Start streaming
	err = pipeline.PlayStream(item.URL)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, "❌ Failed to start audio playback.")
		vc.Disconnect()
		queue.SetPlaying(false)
		return
	}

	// Monitor the pipeline and handle completion
	go func() {
		// Wait for pipeline to finish
		for pipeline.IsPlaying() {
			time.Sleep(1 * time.Second)
		}

		// Play next song in queue
		startNextInQueue(s, m, queue)
	}()
}

// getOrCreateQueue gets or creates a queue for a guild
func getOrCreateQueue(guildID string) *common.MusicQueue {
	queueMutex.Lock()
	defer queueMutex.Unlock()

	if queue, exists := queues[guildID]; exists {
		return queue
	}

	queue := common.NewMusicQueue(guildID)
	queues[guildID] = queue
	return queue
}

// getQueue gets a queue for a guild
func getQueue(guildID string) *common.MusicQueue {
	queueMutex.RLock()
	defer queueMutex.RUnlock()
	return queues[guildID]
}
