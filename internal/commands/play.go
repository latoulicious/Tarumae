package commands

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/cyfdecyf/youtube-dl-go"
	"github.com/faiface/beep"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
)

var (
	audioStream *beep.StreamSeekCloser
	ctrl        *beep.Ctrl
)

func PlayCommand(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	if len(args) == 0 {
		s.ChannelMessageSend(m.ChannelID, "Please provide a song name or URL.")
		return
	}

	query := strings.Join(args, " ")
	s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Playing: %s", query))

	// Download the audio from YouTube
	audioPath, err := downloadAudio(query)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Failed to download audio: %v", err))
		return
	}

	// Open the audio file
	f, err := os.Open(audioPath)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Failed to open audio file: %v", err))
		return
	}
	defer f.Close()

	// Decode the audio file
	streamer, format, err := mp3.Decode(f)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Failed to decode audio file: %v", err))
		return
	}
	defer streamer.Close()

	// Initialize the audio stream
	audioStream = beep.StreamSeekCloser(streamer)
	ctrl = &beep.Ctrl{Streamer: audioStream}

	// Initialize the speaker
	speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))

	// Play the audio
	speaker.Play(ctrl)
}

func downloadAudio(query string) (string, error) {
	// Use youtube-dl to download the audio
	ydl := youtube.NewClient()
	video, err := ydl.GetVideoInfo(query)
	if err != nil {
		return "", err
	}

	// Get the audio format
	audioFormat := video.Formats.Extremes(youtube.FormatAudioBitrateKey, true)[0]

	// Download the audio
	err = ydl.Download(video, audioFormat, "audio.mp3")
	if err != nil {
		return "", err
	}

	return "audio.mp3", nil
}
