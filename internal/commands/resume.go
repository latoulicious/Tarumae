package commands

import (
	"github.com/bwmarrin/discordgo"
)

func ResumeCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	guildID := m.GuildID
	
	// Update activity for idle monitoring
	updateActivity(guildID)
	
	if Ctrl != nil && !Ctrl.Paused {
		Ctrl.Paused = true
		sendEmbedMessage(s, m.ChannelID, "▶️ Playback Resumed", "Music playback has been resumed.", 0x00ff00)
	} else if Ctrl.Paused {
		sendEmbedMessage(s, m.ChannelID, "❌ Error", "Playback is already resumed.", 0xff0000)
	} else {
		sendEmbedMessage(s, m.ChannelID, "❌ Error", "Nothing is playing.", 0xff0000)
	}
}
