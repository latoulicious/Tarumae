package common

import (
	"fmt"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
)

// FindAndJoinUserVoiceChannel finds the user's voice channel and joins it with retry logic
func FindAndJoinUserVoiceChannel(s *discordgo.Session, userID, guildID string) (*discordgo.VoiceConnection, error) {
	guild, err := s.State.Guild(guildID)
	if err != nil {
		return nil, fmt.Errorf("could not find guild: %v", err)
	}

	var userChannelID string
	for _, vs := range guild.VoiceStates {
		if vs.UserID == userID {
			userChannelID = vs.ChannelID
			break
		}
	}

	if userChannelID == "" {
		return nil, fmt.Errorf("you must be in a voice channel to play music")
	}

	// Get channel info for logging
	channel, err := s.State.Channel(userChannelID)
	channelName := "Unknown"
	if err == nil {
		channelName = channel.Name
	}

	log.Printf("Joining voice channel: %s (%s) in guild: %s", channelName, userChannelID, guildID)

	// Join with retry logic
	var vc *discordgo.VoiceConnection
	maxRetries := 3

	for i := 0; i < maxRetries; i++ {
		vc, err = s.ChannelVoiceJoin(guildID, userChannelID, false, true)
		if err == nil {
			break
		}

		log.Printf("Voice join attempt %d/%d failed: %v", i+1, maxRetries, err)
		if i < maxRetries-1 {
			time.Sleep(time.Duration(i+1) * time.Second)
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to join voice channel after %d attempts: %v", maxRetries, err)
	}

	// Wait for connection to be ready with timeout
	timeout := time.After(10 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			vc.Disconnect()
			return nil, fmt.Errorf("voice connection timed out")
		case <-ticker.C:
			if vc.Ready {
				log.Printf("Voice connection ready for guild: %s", guildID)
				return vc, nil
			}
		}
	}
}

// DisconnectFromVoiceChannel disconnects from the voice channel in the specified guild
func DisconnectFromVoiceChannel(s *discordgo.Session, guildID string) error {
	// Get all voice connections for the guild
	for _, vc := range s.VoiceConnections {
		if vc.GuildID == guildID {
			vc.Disconnect()
			log.Printf("Disconnected from voice channel in guild: %s", guildID)
			return nil
		}
	}

	log.Printf("No voice connection found for guild: %s", guildID)
	return nil
}
