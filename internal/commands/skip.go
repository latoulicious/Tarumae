package commands

import (
	"fmt"
	"io"

	"github.com/bwmarrin/discordgo"
)

func SkipCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	if Ctrl.AudioStream != nil {
		if _, err := Ctrl.AudioStream.Seek(0, io.SeekEnd); err != nil {
			if _, err := s.ChannelMessageSend(m.ChannelID, "Failed to skip the audio stream."); err != nil {
				fmt.Println("Error sending message:", err)
			}
			return
		}
		if _, err := s.ChannelMessageSend(m.ChannelID, "Skipped to the next song."); err != nil {
			fmt.Println("Error sending message:", err)
		}
	} else {
		if _, err := s.ChannelMessageSend(m.ChannelID, "Nothing is playing."); err != nil {
			fmt.Println("Error sending message:", err)
		}
	}
}
