package commands

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

func PauseCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	if Ctrl != nil && !Ctrl.Paused {
		Ctrl.Paused = true
		if _, err := s.ChannelMessageSend(m.ChannelID, "Playback paused."); err != nil {
			fmt.Println("Error sending message:", err)
		}
	} else if Ctrl.Paused {
		if _, err := s.ChannelMessageSend(m.ChannelID, "Playback is already paused."); err != nil {
			fmt.Println("Error sending message:", err)
		}
	} else {
		if _, err := s.ChannelMessageSend(m.ChannelID, "Nothing is playing."); err != nil {
			fmt.Println("Error sending message:", err)
		}
	}
}
