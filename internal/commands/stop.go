package commands

import (
	"github.com/bwmarrin/discordgo"
	"github.com/latoulicious/Tarumae/pkg/common"
)

// StopCommand stops the current audio playback and clears queue
func StopCommand(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	guildID := m.GuildID

	// Get queue for this guild
	queue := getQueue(guildID)
	if queue == nil || !queue.IsPlaying() {
		sendEmbedMessage(s, m.ChannelID, "❌ Error", "No audio is currently playing.", 0xff0000)
		return
	}

	// Stop current pipeline
	if pipeline := queue.GetPipeline(); pipeline != nil {
		pipeline.Stop()
	}

	// Clear queue and stop playing
	queue.Clear()
	queue.SetPlaying(false)

	// Clear presence
	if presenceManager != nil {
		presenceManager.ClearMusicPresence()
	}

	// Send stop embed
	sendBotStoppedEmbed(s, m.ChannelID, m.Author.Username)

	// Find and disconnect from voice channel
	common.DisconnectFromVoiceChannel(s, guildID)
}
