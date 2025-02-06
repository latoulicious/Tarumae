package commands

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

func ResumeCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	if Ctrl != nil && !Ctrl.Paused {
		Ctrl.Paused = true
		if _, err := s.ChannelMessageSend(m.ChannelID, "Playback Resumed."); err != nil {
			fmt.Println("Error sending message:", err)
		}
	} else if Ctrl.Paused {
		if _, err := s.ChannelMessageSend(m.ChannelID, "Playback is already resumed."); err != nil {
			fmt.Println("Error sending message:", err)
		}
	} else {
		if _, err := s.ChannelMessageSend(m.ChannelID, "Nothing is playing."); err != nil {
			fmt.Println("Error sending message:", err)
		}
	}
}
