package commands

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/faiface/beep/mp3"
	"github.com/kkdai/youtube/v2"
)

func PlayCommand(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	if len(args) == 0 {
		if _, err := s.ChannelMessageSend(m.ChannelID, "Please provide a song name or URL."); err != nil {
			fmt.Println("Error sending message:", err)
		}
		return
	}

	query := strings.Join(args, " ")
	yt := youtube.Client{}

	video, err := yt.GetVideo(query)
	if err != nil {
		if _, err := s.ChannelMessageSend(m.ChannelID, "Could not find the video."); err != nil {
			fmt.Println("Error sending message:", err)
		}
		return
	}

	stream, _, err := yt.GetStream(video, &youtube.Format{ItagNo: 140})
	if err != nil {
		if _, err := s.ChannelMessageSend(m.ChannelID, "Could not get the audio stream."); err != nil {
			fmt.Println("Error sending message:", err)
		}
		return
	}
	defer stream.Close()

	tempFile, err := os.Create("temp.mp3")
	if err != nil {
		if _, err := s.ChannelMessageSend(m.ChannelID, "Could not create temporary file."); err != nil {
			fmt.Println("Error sending message:", err)
		}
		return
	}
	defer os.Remove(tempFile.Name())

	_, err = io.Copy(tempFile, stream)
	if err != nil {
		if _, err := s.ChannelMessageSend(m.ChannelID, "Could not save the audio."); err != nil {
			fmt.Println("Error sending message:", err)
		}
		return
	}

	if _, err := tempFile.Seek(0, 0); err != nil {
		if _, err := s.ChannelMessageSend(m.ChannelID, "Could not rewind the audio file."); err != nil {
			fmt.Println("Error sending message:", err)
		}
		return
	}

	streamer, _, err := mp3.Decode(tempFile)
	if err != nil {
		if _, err := s.ChannelMessageSend(m.ChannelID, "Could not decode the audio."); err != nil {
			fmt.Println("Error sending message:", err)
		}
		return
	}
	defer streamer.Close()

	if _, err := s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Now playing: %s", video.Title)); err != nil {
		fmt.Println("Error sending message:", err)
	}
}
