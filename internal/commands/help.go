package commands

import (
	"strings"
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
				Name: "Music Commands",
				Value: strings.Join([]string{
					"• `!play <url>` / `!p <url>` - Play a YouTube video by URL",
					"• `!p search <keywords>` - Search and play a YouTube video",
					"• `!nowplaying` / `!np` - Show the currently playing track",
					"• `!queue add <url>` - Add a YouTube video to the queue",
					"• `!queue list` - List the current queue",
					"• `!queue remove <index>` - Remove a track from the queue",
					"• `!clear` - Clear the entire queue (confirmation for non-admins)",
					"• `!shuffle` - Shuffle the queue (announces new top song for large queues)",
					"• `!pause` - Pause the current playback",
					"• `!resume` - Resume paused playback",
					"• `!skip` - Skip the currently playing track",
					"• `!stop` - Stop playback and disconnect from voice channel",
				}, "\n"),
				Inline: false,
			},
			{
				Name: "ℹInformation Commands",
				Value: strings.Join([]string{
					"• `!about` - Show bot info, uptime, and stats",
					"• `!servers` - List servers the bot is connected to (bot owner only)",
					"• `!help` / `!h` - Show this help message",
				}, "\n"),
				Inline: false,
			},
			{
				Name:   "Fun Commands",
				Value:  "• `!gremlin` - Post a random gremlin image\n• `!lyrics <song>` - Search for anime song lyrics\n• `!uma char <name>` - Search for Uma Musume characters (with image navigation)",
				Inline: false,
			},
			{
				Name: "Admin Commands (Bot Owner Only)",
				Value: strings.Join([]string{
					"• `!leave <server_id>` - Force bot to leave a server by ID",
				}, "\n"),
				Inline: false,
			},
			{
				Name: "💡 Tips",
				Value: strings.Join([]string{
					"• Join a voice channel **before** using music commands",
					"• Only **YouTube links and searches** are currently supported",
					"• For lyrics, try using **Japanese titles** for better results",
				}, "\n"),
				Inline: false,
			},
		},
	}

	s.ChannelMessageSendEmbed(m.ChannelID, embed)
}

// Unused commands
// • `!shuffle announce` - Shuffle and always announce the new top song
