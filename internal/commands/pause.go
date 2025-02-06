package commands

import (
	"github.com/bwmarrin/discordgo"
)

func PauseCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	if ctrl != nil {
		ctrl.Paused = true
		s.ChannelMessageSend(m.ChannelID, "Playback paused.")
	} else {
		s.ChannelMessageSend(m.ChannelID, "Nothing is playing.")
	}
}
