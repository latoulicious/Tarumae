package commands

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/jonas747/dca"
)

// PlayCommand handles the play command for the Discord bot.
func PlayCommand(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	if len(args) < 1 {
		s.ChannelMessageSend(m.ChannelID, "Please provide a YouTube URL.")
		return
	}

	url := args[0]
	log.Println("Fetching audio stream URL for:", url)
	s.ChannelMessageSend(m.ChannelID, "Fetching audio stream, please wait...")

	// Get direct stream URL
	streamURL, err := getYouTubeAudioStream(url)
	if err != nil {
		log.Println("Error fetching stream URL:", err)
		s.ChannelMessageSend(m.ChannelID, "Failed to get audio stream.")
		return
	}

	log.Println("Streaming audio from:", streamURL)
	s.ChannelMessageSend(m.ChannelID, "Now playing audio...")

	// Connect to VC
	vc, err := findUserVoiceState(s, m.Author.ID, m.GuildID)
	if err != nil {
		log.Println("Error finding voice state:", err)
		s.ChannelMessageSend(m.ChannelID, "You must be in a voice channel!")
		return
	}

	// Stream audio
	err = streamAudio(vc, streamURL)
	if err != nil {
		log.Println("Error streaming audio:", err)
		s.ChannelMessageSend(m.ChannelID, "Error playing audio.")
		return
	}

	log.Println("Playback finished.")
	s.ChannelMessageSend(m.ChannelID, "Playback finished.")

	// Wait 5s before disconnecting (debug)
	time.Sleep(5 * time.Second)

	log.Println("Disconnecting from VC...")
	vc.Disconnect()
}

// getYouTubeAudioStream extracts the direct audio URL using yt-dlp.
func getYouTubeAudioStream(url string) (string, error) {
	cmd := exec.Command("yt-dlp", "-f", "bestaudio", "-g", url)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("yt-dlp error: %v", err)
	}

	streamURL := out.String()
	if streamURL == "" {
		return "", errors.New("empty stream URL")
	}
	return streamURL, nil
}

// streamAudio streams the given audio URL to a Discord VC.
func streamAudio(vc *discordgo.VoiceConnection, streamURL string) error {
	log.Println("Starting audio stream...")

	// Fetch the audio data
	resp, err := http.Get(streamURL)
	if err != nil {
		return fmt.Errorf("error fetching audio: %v", err)
	}
	defer resp.Body.Close()

	options := dca.StdEncodeOptions
	options.RawOutput = true
	options.Bitrate = 96
	options.Application = "lowdelay"

	// Encode audio stream
	encodingSession, err := dca.EncodeMem(resp.Body, options)
	if err != nil {
		return fmt.Errorf("error encoding audio: %v", err)
	}
	defer encodingSession.Cleanup()

	done := make(chan error)
	dca.NewStream(encodingSession, vc, done)

	// Wait for playback to finish
	for err := range done {
		if err != nil && err != io.EOF {
			return fmt.Errorf("playback error: %v", err)
		}
	}

	log.Println("Audio stream ended.")
	return nil
}

// findUserVoiceState finds the user's voice channel and joins it.
func findUserVoiceState(s *discordgo.Session, userID, guildID string) (*discordgo.VoiceConnection, error) {
	guild, err := s.State.Guild(guildID)
	if err != nil {
		return nil, fmt.Errorf("could not find guild: %v", err)
	}

	for _, vs := range guild.VoiceStates {
		if vs.UserID == userID {
			vc, err := s.ChannelVoiceJoin(guildID, vs.ChannelID, false, true)
			if err != nil {
				return nil, fmt.Errorf("error joining voice channel: %v", err)
			}
			return vc, nil
		}
	}

	return nil, errors.New("user not in a voice channel")
}
