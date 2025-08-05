package commands

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

// ServersCommand displays information about which servers the bot is joined to
func ServersCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	guilds := s.State.Guilds

	if len(guilds) == 0 {
		s.ChannelMessageSend(m.ChannelID, "I'm not joined to any servers.")
		return
	}

	// Create a response message with server information including IDs
	var response string
	if len(guilds) == 1 {
		response = fmt.Sprintf("I'm joined to **1 server**:\n• **%s** (ID: `%s`)", guilds[0].Name, guilds[0].ID)
	} else {
		response = fmt.Sprintf("I'm joined to **%d servers**:\n", len(guilds))
		for i, guild := range guilds {
			response += fmt.Sprintf("• **%s** (ID: `%s`)", guild.Name, guild.ID)
			if i < len(guilds)-1 {
				response += "\n"
			}
		}
	}

	response += "\n\n💡 **Tip**: Use `!leave <server_id>` to leave a server."
	s.ChannelMessageSend(m.ChannelID, response)
}
