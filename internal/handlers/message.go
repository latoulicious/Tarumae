package handlers

import (
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/latoulicious/Tarumae/internal/commands"
)

func MessageHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore all messages created by the bot itself
	if m.Author.ID == s.State.User.ID {
		return
	}

	// Check if the message is a command
	if strings.HasPrefix(m.Content, "!") {
		args := strings.Split(m.Content, " ")
		command := strings.TrimPrefix(args[0], "!")

		switch command {
		case "play":
			commands.PlayCommand(s, m, args[1:])
		case "pause":
			commands.PauseCommand(s, m)
		case "resume":
			commands.ResumeCommand(s, m)
		case "skip":
			commands.SkipCommand(s, m)
		case "stop":
			commands.StopCommand(s, m)
		default:
			s.ChannelMessageSend(m.ChannelID, "Unknown command. Try !play, !pause, !resume, !skip, or !stop.")
		}
	}
}
