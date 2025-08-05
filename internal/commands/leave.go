package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// LeaveCommand allows the bot owner to make the bot leave a specific server
func LeaveCommand(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	// Check if user provided arguments
	if len(args) < 1 {
		s.ChannelMessageSend(m.ChannelID, "Usage: `!leave <server_id>`")
		return
	}

	// Get the server ID from arguments
	serverID := args[0]

	// Check if the user is the bot owner
	ownerID := os.Getenv("BOT_OWNER_ID")
	if ownerID == "" {
		s.ChannelMessageSend(m.ChannelID, "❌ Bot owner ID not configured.")
		return
	}

	if m.Author.ID != ownerID {
		s.ChannelMessageSend(m.ChannelID, "❌ You don't have permission to use this command.")
		return
	}

	// Validate server ID format (Discord IDs are 17-19 digits)
	if len(serverID) < 17 || len(serverID) > 19 {
		s.ChannelMessageSend(m.ChannelID, "❌ Invalid server ID format.")
		return
	}

	// Check if the bot is actually in the specified server
	guild, err := s.Guild(serverID)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, "❌ Server not found or bot is not in that server.")
		return
	}

	// Confirm the action
	confirmationMsg := fmt.Sprintf("Are you sure you want me to leave **%s**? Reply with `!confirm` to proceed.", guild.Name)
	s.ChannelMessageSend(m.ChannelID, confirmationMsg)

	// Store pending leave request (in a real implementation, you might want to use a more robust storage)
	// For now, we'll use a simple approach - you can enhance this with proper state management
}

// LeaveByNameCommand allows the bot owner to leave a server by name
func LeaveByNameCommand(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	// Check if user provided arguments
	if len(args) < 1 {
		s.ChannelMessageSend(m.ChannelID, "Usage: `!leavebyname <server_name>`")
		return
	}

	// Get the server name from arguments (join all args in case name has spaces)
	serverName := strings.Join(args, " ")

	// Check if the user is the bot owner
	ownerID := os.Getenv("BOT_OWNER_ID")
	if ownerID == "" {
		s.ChannelMessageSend(m.ChannelID, "❌ Bot owner ID not configured.")
		return
	}

	if m.Author.ID != ownerID {
		s.ChannelMessageSend(m.ChannelID, "❌ You don't have permission to use this command.")
		return
	}

	// Find the server by name
	guilds := s.State.Guilds
	var targetGuild *discordgo.Guild
	var exactMatch *discordgo.Guild
	var partialMatches []*discordgo.Guild

	for _, guild := range guilds {
		if strings.EqualFold(guild.Name, serverName) {
			exactMatch = guild
			break
		}
		if strings.Contains(strings.ToLower(guild.Name), strings.ToLower(serverName)) {
			partialMatches = append(partialMatches, guild)
		}
	}

	// Use exact match if found, otherwise show partial matches
	if exactMatch != nil {
		targetGuild = exactMatch
	} else if len(partialMatches) == 1 {
		targetGuild = partialMatches[0]
	} else if len(partialMatches) > 1 {
		// Show multiple matches
		response := "❌ Multiple servers found with similar names:\n"
		for _, guild := range partialMatches {
			response += fmt.Sprintf("• **%s** (ID: `%s`)\n", guild.Name, guild.ID)
		}
		response += "\nPlease use the exact server name or use `!leave <server_id>` with the specific ID."
		s.ChannelMessageSend(m.ChannelID, response)
		return
	} else {
		s.ChannelMessageSend(m.ChannelID, "❌ Server not found. Use `!servers` to see available servers.")
		return
	}

	// Confirm the action
	confirmationMsg := fmt.Sprintf("Are you sure you want me to leave **%s**? Reply with `!confirmbyname %s` to proceed.", targetGuild.Name, targetGuild.Name)
	s.ChannelMessageSend(m.ChannelID, confirmationMsg)
}

// ConfirmLeaveCommand handles the confirmation for leaving a server
func ConfirmLeaveCommand(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	// Check if the user is the bot owner
	ownerID := os.Getenv("BOT_OWNER_ID")
	if ownerID == "" {
		s.ChannelMessageSend(m.ChannelID, "❌ Bot owner ID not configured.")
		return
	}

	if m.Author.ID != ownerID {
		s.ChannelMessageSend(m.ChannelID, "❌ You don't have permission to use this command.")
		return
	}

	// Check if there are arguments (server ID should be provided)
	if len(args) < 1 {
		s.ChannelMessageSend(m.ChannelID, "Usage: `!confirm <server_id>`")
		return
	}

	serverID := args[0]

	// Validate server ID format
	if len(serverID) < 17 || len(serverID) > 19 {
		s.ChannelMessageSend(m.ChannelID, "❌ Invalid server ID format.")
		return
	}

	// Get guild information before leaving
	guild, err := s.Guild(serverID)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, "❌ Server not found or bot is not in that server.")
		return
	}

	// Leave the server
	err = s.GuildLeave(serverID)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("❌ Failed to leave server: %v", err))
		return
	}

	// Send confirmation message
	leaveMsg := fmt.Sprintf("✅ Successfully left **%s** (ID: %s)", guild.Name, serverID)
	s.ChannelMessageSend(m.ChannelID, leaveMsg)
}

// ConfirmLeaveByNameCommand handles the confirmation for leaving a server by name
func ConfirmLeaveByNameCommand(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	// Check if the user is the bot owner
	ownerID := os.Getenv("BOT_OWNER_ID")
	if ownerID == "" {
		s.ChannelMessageSend(m.ChannelID, "❌ Bot owner ID not configured.")
		return
	}

	if m.Author.ID != ownerID {
		s.ChannelMessageSend(m.ChannelID, "❌ You don't have permission to use this command.")
		return
	}

	// Check if there are arguments (server name should be provided)
	if len(args) < 1 {
		s.ChannelMessageSend(m.ChannelID, "Usage: `!confirmbyname <server_name>`")
		return
	}

	serverName := strings.Join(args, " ")

	// Find the server by name
	guilds := s.State.Guilds
	var targetGuild *discordgo.Guild

	for _, guild := range guilds {
		if strings.EqualFold(guild.Name, serverName) {
			targetGuild = guild
			break
		}
	}

	if targetGuild == nil {
		s.ChannelMessageSend(m.ChannelID, "❌ Server not found.")
		return
	}

	// Leave the server
	err := s.GuildLeave(targetGuild.ID)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("❌ Failed to leave server: %v", err))
		return
	}

	// Send confirmation message
	leaveMsg := fmt.Sprintf("✅ Successfully left **%s** (ID: %s)", targetGuild.Name, targetGuild.ID)
	s.ChannelMessageSend(m.ChannelID, leaveMsg)
}
