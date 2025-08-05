package commands

import (
	"github.com/bwmarrin/discordgo"
)

func SkipCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	guildID := m.GuildID

	// Get queue for this guild
	queue := getQueue(guildID)
	if queue == nil || !queue.IsPlaying() {
		s.ChannelMessageSend(m.ChannelID, "Nothing is currently playing.")
		return
	}

	// Stop current pipeline
	if pipeline := queue.GetPipeline(); pipeline != nil {
		pipeline.Stop()
	}

	s.ChannelMessageSend(m.ChannelID, "⏭️ Skipped current song.")

	// Start next song in queue
	startNextInQueue(s, m, queue)
}
