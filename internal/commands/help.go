package commands

import "github.com/bwmarrin/discordgo"

// showHelpCommand displays all available commands with their descriptions
func ShowHelpCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	helpMessage := "🎵 **Hokko Tarumae - Music Bot Commands**\n\n" +
		"**🎶 Music Commands:**\n" +
		"• `!play <youtube_url>` - Play audio from a YouTube URL\n" +
		"• `!pause` - Pause the current playback\n" +
		"• `!resume` - Resume paused playback\n" +
		"• `!skip` - Skip the current track\n" +
		"• `!stop` - Stop playback and disconnect from voice channel\n\n" +
		"**📊 Information Commands:**\n" +
		"• `!servers` - Show all servers the bot is joined to\n" +
		"• `!help` - Show this help message\n\n" +
		"**🔧 Admin Commands (Bot Owner Only):**\n" +
		"• `!leave <server_id>` - Leave a server by ID (requires confirmation)\n" +
		"• `!leavebyname <server_name>` - Leave a server by name (requires confirmation)\n" +
		"• `!confirm` - Confirm leaving a server by ID\n" +
		"• `!confirmbyname` - Confirm leaving a server by name\n\n" +
		"**💡 Tips:**\n" +
		"• Make sure you're in a voice channel before using music commands\n" +
		"• Use `!servers` to get server IDs for the leave commands\n" +
		"• Only the bot owner can use admin commands\n\n" +
		"*I'm Hokko Tarumae, Tomakomai's Tourism Ambassador!★*"

	s.ChannelMessageSend(m.ChannelID, helpMessage)
}
