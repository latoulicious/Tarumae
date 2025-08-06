package commands

import (
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/latoulicious/HKTM/pkg/common"
)

// ClearCommand handles the !clear command to empty the queue
func ClearCommand(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	guildID := m.GuildID

	// Update activity
	updateActivity(guildID)

	queue := getQueue(guildID)

	if queue == nil {
		sendEmbedMessage(s, m.ChannelID, "❌ Error", "No queue found for this server.", 0xff0000)
		return
	}

	// Check if queue is empty
	if queue.Size() == 0 && queue.Current() == nil {
		sendEmbedMessage(s, m.ChannelID, "📭 Queue Already Empty", "The queue is already empty.", 0x808080)
		return
	}

	// Handle confirmation arguments
	if len(args) > 0 {
		arg := strings.ToLower(args[0])
		switch arg {
		case "confirm":
			// User confirmed the clear
			clearQueueInternal(s, m, queue)
			return
		case "cancel":
			// User cancelled the clear
			sendEmbedMessage(s, m.ChannelID, "❌ Cancelled", "Queue clear operation cancelled.", 0x808080)
			return
		}
	}

	// Check if user has admin permissions
	hasAdmin := hasAdminPermissions(s, m.GuildID, m.Author.ID)

	// If user is not admin and there are multiple songs, ask for confirmation
	if !hasAdmin && queue.Size() > 3 {
		embed := &discordgo.MessageEmbed{
			Title:     "⚠️ Confirm Queue Clear",
			Color:     0xffa500, // Orange
			Timestamp: time.Now().Format(time.RFC3339),
			Footer: &discordgo.MessageEmbedFooter{
				Text: "Hokko Tarumae",
			},
			Description: "You're about to clear the entire queue with multiple songs. Are you sure?\n\n" +
				"Reply with `!clear confirm` to proceed or `!clear cancel` to cancel.",
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:   "Queue Size",
					Value:  fmt.Sprintf("%d songs", queue.Size()),
					Inline: true,
				},
				{
					Name:   "Requested By",
					Value:  m.Author.Username,
					Inline: true,
				},
			},
		}
		s.ChannelMessageSendEmbed(m.ChannelID, embed)
		return
	}

	// Clear the queue
	clearQueueInternal(s, m, queue)
}

// clearQueueInternal performs the actual queue clearing
func clearQueueInternal(s *discordgo.Session, m *discordgo.MessageCreate, queue *common.MusicQueue) {
	// Get queue size before clearing for the message
	queueSize := queue.Size()

	// Clear the queue
	queue.Clear()

	// Send confirmation embed
	embed := &discordgo.MessageEmbed{
		Title:     "🗑️ Queue Cleared",
		Color:     0x00ff00, // Green
		Timestamp: time.Now().Format(time.RFC3339),
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Hokko Tarumae",
		},
		Description: "The queue has been successfully cleared.",
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Songs Removed",
				Value:  fmt.Sprintf("%d songs", queueSize),
				Inline: true,
			},
			{
				Name:   "Cleared By",
				Value:  m.Author.Username,
				Inline: true,
			},
		},
	}
	s.ChannelMessageSendEmbed(m.ChannelID, embed)
}

// hasAdminPermissions checks if a user has admin permissions
func hasAdminPermissions(s *discordgo.Session, guildID, userID string) bool {
	// Get guild member
	member, err := s.GuildMember(guildID, userID)
	if err != nil {
		return false
	}

	// Check if user has admin permission
	for _, roleID := range member.Roles {
		role, err := s.State.Role(guildID, roleID)
		if err != nil {
			continue
		}
		if role.Permissions&discordgo.PermissionAdministrator != 0 {
			return true
		}
	}

	return false
}
