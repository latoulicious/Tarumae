package commands

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

func StopCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	if Ctrl.AudioStream != nil {
		Ctrl.AudioStream = nil // Clear the current stream
		if _, err := s.ChannelMessageSend(m.ChannelID, "Playback stopped."); err != nil {
			fmt.Println("Error sending message:", err)
		}
	} else {
		if _, err := s.ChannelMessageSend(m.ChannelID, "Nothing is playing."); err != nil {
			fmt.Println("Error sending message:", err)
		}
	}
}
