package commands

import (
	"github.com/bwmarrin/discordgo"
)

func ResumeCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	if ctrl != nil {
		ctrl.Paused = false
		s.ChannelMessageSend(m.ChannelID, "Playback resumed.")
	} else {
		s.ChannelMessageSend(m.ChannelID, "Nothing is playing.")
	}
}
