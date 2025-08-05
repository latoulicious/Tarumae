package commands

import (
	"time"

	"github.com/bwmarrin/discordgo"
)

// ShowHelpCommand displays all available commands with their descriptions using embeds
func ShowHelpCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Create embed
	embed := &discordgo.MessageEmbed{
		Title:       "Hokko Tarumae",
		Description: "Here are all the available commands for the bot:",
		Color:       0x00ff00, // Green color
		Timestamp:   time.Now().Format(time.RFC3339),
		Footer: &discordgo.MessageEmbedFooter{
			Text:    "Hokko Tarumae | Created by latoulicious | 2025",
			IconURL: "https://cdn.discordapp.com/emojis/1198008186138021888.webp?size=96", // Replace with custom image URL
		},
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Music Commands",
				Value:  "• `!play <youtube_url>` - Add song to queue and play\n• `!queue add <youtube_url>` - Add song to queue\n• `!queue list` - Show current queue\n• `!queue remove <index>` - Remove song from queue\n• `!queue clear` - Clear entire queue\n• `!pause` - Pause the current playback\n• `!resume` - Resume paused playback\n• `!skip` - Skip the current track\n• `!stop` - Stop playback and disconnect from voice channel",
				Inline: false,
			},
			{
				Name:   "Information Commands",
				Value:  "• `!servers` - Show all servers the bot is joined to\n• `!help` - Show this help message",
				Inline: false,
			},
			{
				Name:   "Admin Commands (Bot Owner Only)",
				Value:  "• `!leave <server_id>` - Leave a server by ID\n• `!leave` - Show list of servers (if no ID provided)",
				Inline: false,
			},
			{
				Name:   "Tips",
				Value:  "• Make sure you're in a voice channel before using music commands\n• Use `!servers` or `!leave` to get server IDs\n• Only the bot owner can use admin commands",
				Inline: false,
			},
		},
	}

	s.ChannelMessageSendEmbed(m.ChannelID, embed)
}
