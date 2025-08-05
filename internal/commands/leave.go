package commands

import (
	"fmt"
	"os"

	"github.com/bwmarrin/discordgo"
)

// LeaveCommand allows the bot owner to make the bot leave a specific server
func LeaveCommand(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
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

	// If no arguments provided, show server list
	if len(args) < 1 {
		ServersCommand(s, m)
		return
	}

	// Get the server ID from arguments
	serverID := args[0]

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

	// Leave the server directly
	err = s.GuildLeave(serverID)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("❌ Failed to leave server: %v", err))
		return
	}

	// Send confirmation message
	leaveMsg := fmt.Sprintf("✅ Successfully left **%s** (ID: %s)", guild.Name, serverID)
	s.ChannelMessageSend(m.ChannelID, leaveMsg)
}
