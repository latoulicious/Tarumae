package commands

import (
	"github.com/bwmarrin/discordgo"
)

func PauseCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	guildID := m.GuildID
	
	// Update activity for idle monitoring
	updateActivity(guildID)
	
	if Ctrl != nil && !Ctrl.Paused {
		Ctrl.Paused = true
		sendEmbedMessage(s, m.ChannelID, "⏸️ Playback Paused", "Music playback has been paused.", 0xffa500)
	} else if Ctrl.Paused {
		sendEmbedMessage(s, m.ChannelID, "❌ Error", "Playback is already paused.", 0xff0000)
	} else {
		sendEmbedMessage(s, m.ChannelID, "❌ Error", "Nothing is playing.", 0xff0000)
	}
}
