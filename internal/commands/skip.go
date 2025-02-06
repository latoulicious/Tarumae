package commands

import (
	"github.com/bwmarrin/discordgo"
)

func SkipCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	if audioStream != nil {
		audioStream.Seek(audioStream.Len())
		s.ChannelMessageSend(m.ChannelID, "Skipped to the next song.")
	} else {
		s.ChannelMessageSend(m.ChannelID, "Nothing is playing.")
	}
}
