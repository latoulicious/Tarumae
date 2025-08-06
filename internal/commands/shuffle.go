package commands

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/latoulicious/HKTM/pkg/common"
)

// ShuffleCommand handles the !shuffle command to shuffle the queue
func ShuffleCommand(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	guildID := m.GuildID

	// Update activity
	updateActivity(guildID)

	queue := getQueue(guildID)

	if queue == nil {
		sendEmbedMessage(s, m.ChannelID, "❌ Error", "No queue found for this server.", 0xff0000)
		return
	}

	// Check if queue has enough songs to shuffle
	queueSize := queue.Size()
	if queueSize < 2 {
		sendEmbedMessage(s, m.ChannelID, "📭 Not Enough Songs", "Need at least 2 songs to shuffle the queue.", 0x808080)
		return
	}

	// Get current queue items
	items := queue.List()
	if len(items) == 0 {
		sendEmbedMessage(s, m.ChannelID, "📭 Queue Empty", "No songs in queue to shuffle.", 0x808080)
		return
	}

	// Shuffle the queue
	shuffledItems := shuffleQueueItems(items)

	// Clear the current queue and add shuffled items back
	queue.Clear()
	for _, item := range shuffledItems {
		queue.Add(item.URL, item.Title, item.RequestedBy)
	}

	// Create embed for shuffle confirmation
	embed := &discordgo.MessageEmbed{
		Title:     "🔀 Queue Shuffled",
		Color:     0x00ff00, // Green
		Timestamp: time.Now().Format(time.RFC3339),
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Hokko Tarumae",
		},
		Description: "The queue has been shuffled successfully!",
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Songs Shuffled",
				Value:  fmt.Sprintf("%d songs", queueSize),
				Inline: true,
			},
			{
				Name:   "Shuffled By",
				Value:  m.Author.Username,
				Inline: true,
			},
		},
	}

	// Add new top song announcement if requested or if it's a large queue
	announceTop := len(args) > 0 && strings.ToLower(args[0]) == "announce"
	if announceTop || queueSize > 5 {
		if len(shuffledItems) > 0 {
			topSong := shuffledItems[0]
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:   "🎵 New Top Song",
				Value:  fmt.Sprintf("**%s**\nRequested by: %s", topSong.Title, topSong.RequestedBy),
				Inline: false,
			})
		}
	}

	s.ChannelMessageSendEmbed(m.ChannelID, embed)
}

// shuffleQueueItems shuffles a slice of queue items using Fisher-Yates algorithm
func shuffleQueueItems(items []*common.QueueItem) []*common.QueueItem {
	// Create a copy to avoid modifying the original slice
	shuffled := make([]*common.QueueItem, len(items))
	copy(shuffled, items)

	// Create a new random source with current time
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Fisher-Yates shuffle algorithm
	for i := len(shuffled) - 1; i > 0; i-- {
		j := r.Intn(i + 1)
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	}

	return shuffled
}
