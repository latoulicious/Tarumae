package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/kkdai/youtube/v2"
)

func PlayCommand(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	if len(args) == 0 {
		s.ChannelMessageSend(m.ChannelID, "Please provide a song name or URL.")
		return
	}

	query := strings.Join(args, " ")
	video, err := youtube.New().GetVideo(query)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, "Could not find the video.")
		return
	}

	// Download the audio
	stream, _, err := youtube.New().GetStream(video, &youtube.StreamOptions{Quality: "140"})
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, "Could not download the audio.")
		return
	}

	// Save the audio to a file
	fileName := fmt.Sprintf("%s.mp4", video.Title)
	out, err := os.Create(fileName)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, "Could not create audio file.")
		return
	}
	defer out.Close()

	_, err = stream.WriteTo(out)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, "Could not write audio to file.")
		return
	}

	s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Playing: %s", video.Title))

	// TODO: Implement audio playback logic here
}
