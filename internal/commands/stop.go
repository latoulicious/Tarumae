package commands

import (
	"github.com/bwmarrin/discordgo"
)

func StopCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	if audioStream != nil {
		audioStream.Close()
		s.ChannelMessageSend(m.ChannelID, "Playback stopped.")
	} else {
		s.ChannelMessageSend(m.ChannelID, "Nothing is playing.")
	}
}
