package commands

import (
	"github.com/bwmarrin/discordgo"
	"github.com/latoulicious/Tarumae/pkg/common"
)

// StopCommand stops the current audio playback
func StopCommand(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	guildID := m.GuildID

	pipelineMutex.RLock()
	pipeline, exists := activePipelines[guildID]
	pipelineMutex.RUnlock()

	if !exists || !pipeline.IsPlaying() {
		s.ChannelMessageSend(m.ChannelID, "No audio is currently playing.")
		return
	}

	pipeline.Stop()
	s.ChannelMessageSend(m.ChannelID, "Stopped playback.")

	// Find and disconnect from voice channel
	common.DisconnectFromVoiceChannel(s, guildID)
}
