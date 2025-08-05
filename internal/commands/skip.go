package commands

import (
	"github.com/bwmarrin/discordgo"
)

func SkipCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	guildID := m.GuildID

	// Get queue for this guild
	queue := getQueue(guildID)
	if queue == nil || !queue.IsPlaying() {
		sendEmbedMessage(s, m.ChannelID, "❌ Error", "Nothing is currently playing.", 0xff0000)
		return
	}

	// Get current song info before stopping
	currentSong := queue.Current()
	var songTitle, requestedBy string
	if currentSong != nil {
		songTitle = currentSong.Title
		requestedBy = currentSong.RequestedBy
	}

	// Stop current pipeline
	if pipeline := queue.GetPipeline(); pipeline != nil {
		pipeline.Stop()
	}

	// Send skip embed
	if currentSong != nil {
		sendSongSkippedEmbed(s, m.ChannelID, songTitle, requestedBy, m.Author.Username)
	} else {
		sendEmbedMessage(s, m.ChannelID, "⏭️ Song Skipped", "Current song has been skipped.", 0xffa500)
	}

	// Start next song in queue
	startNextInQueue(s, m, queue)
}
