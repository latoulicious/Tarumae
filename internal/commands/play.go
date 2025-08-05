package commands

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"layeh.com/gopus"
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

	// Give the voice connection a moment to establish
	log.Println("Voice connection established, waiting for readiness...")
	time.Sleep(1 * time.Second)

	log.Printf("Voice connection state - Ready: %v, Speaking: %v, Connected: %v", vc.Ready, vc.Speaking(false), vc.Ready)

	// Stream audio using direct gopus encoding
	err = streamAudioWithGopus(vc, streamURL)
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
	// Try different format options to avoid 403 errors
	cmd := exec.Command("yt-dlp",
		"-f", "bestaudio[ext=m4a]/bestaudio[ext=webm]/bestaudio",
		"--no-playlist",
		"--no-warnings",
		"-g", url)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		// If the first attempt fails, try with different options
		log.Println("First attempt failed, trying alternative format...")
		cmd = exec.Command("yt-dlp",
			"-f", "bestaudio",
			"--no-playlist",
			"--no-warnings",
			"--extractor-args", "youtube:player_client=android",
			"-g", url)
		cmd.Stdout = &out
		cmd.Stderr = os.Stderr
		err = cmd.Run()
		if err != nil {
			return "", fmt.Errorf("yt-dlp error: %v", err)
		}
	}

	streamURL := strings.TrimSpace(out.String())
	if streamURL == "" {
		return "", errors.New("empty stream URL")
	}

	// Split by newlines and take the first URL (in case multiple formats are returned)
	urls := strings.Split(streamURL, "\n")
	if len(urls) > 0 && urls[0] != "" {
		streamURL = urls[0]
	}

	log.Println("Extracted stream URL:", streamURL)
	return streamURL, nil
}

// streamAudioWithGopus streams audio using gopus for direct Opus encoding
func streamAudioWithGopus(vc *discordgo.VoiceConnection, streamURL string) error {
	log.Println("Starting gopus-based audio stream...")

	// Wait for voice connection to be ready
	log.Println("Waiting for voice connection to be ready...")
	for !vc.Ready {
		time.Sleep(100 * time.Millisecond)
	}
	log.Println("Voice connection is ready!")

	// Create FFmpeg command to extract raw PCM audio
	// FFmpeg will convert the stream to raw PCM at Discord's required format
	cmd := exec.Command("ffmpeg",
		"-i", streamURL,
		"-f", "s16le", // Output format: signed 16-bit little-endian
		"-acodec", "pcm_s16le", // Audio codec: PCM 16-bit
		"-ar", "48000", // Sample rate: 48kHz (Discord standard)
		"-ac", "2", // Audio channels: 2 (stereo)
		"-") // Output to stdout

	// Get stdout pipe for reading the raw PCM audio
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("error creating stdout pipe: %v", err)
	}

	// Start the FFmpeg process
	log.Println("Starting FFmpeg process for PCM extraction...")
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("error starting FFmpeg: %v", err)
	}
	defer cmd.Wait()

	// Create Opus encoder
	opusEncoder, err := gopus.NewEncoder(48000, 2, gopus.Voip)
	if err != nil {
		return fmt.Errorf("error creating Opus encoder: %v", err)
	}

	// Set Opus encoding parameters
	opusEncoder.SetBitrate(64000) // 64kbps

	// Add a small delay to ensure voice connection is fully ready
	time.Sleep(500 * time.Millisecond)

	// Start speaking
	vc.Speaking(true)
	defer vc.Speaking(false)

	log.Println("Starting audio stream to Discord...")

	// Read and encode the PCM data
	pcmBuffer := make([]byte, 960*4) // 20ms of stereo 16-bit audio at 48kHz
	for {
		n, err := stdout.Read(pcmBuffer)
		if err != nil {
			if err == io.EOF {
				log.Println("FFmpeg stream ended")
				break
			}
			return fmt.Errorf("error reading FFmpeg output: %v", err)
		}

		if n > 0 {
			// Convert bytes to int16 samples
			samples := make([]int16, n/2)
			for i := 0; i < n/2; i++ {
				samples[i] = int16(pcmBuffer[i*2]) | int16(pcmBuffer[i*2+1])<<8
			}

			// Encode to Opus
			opusData, err := opusEncoder.Encode(samples, 960, 960*4)
			if err != nil {
				log.Printf("Error encoding to Opus: %v", err)
				continue
			}

			// Send the Opus data to Discord
			vc.OpusSend <- opusData
		}
	}

	log.Println("Audio stream completed successfully")
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
			log.Printf("Found user in voice channel: %s", vs.ChannelID)

			// Check bot permissions in the voice channel
			channel, err := s.State.Channel(vs.ChannelID)
			if err != nil {
				log.Printf("Warning: Could not get channel info: %v", err)
			} else {
				log.Printf("Joining voice channel: %s (%s)", channel.Name, vs.ChannelID)
			}

			vc, err := s.ChannelVoiceJoin(guildID, vs.ChannelID, false, true)
			if err != nil {
				return nil, fmt.Errorf("error joining voice channel: %v", err)
			}

			log.Printf("Successfully joined voice channel")
			return vc, nil
		}
	}

	return nil, errors.New("user not in a voice channel")
}
