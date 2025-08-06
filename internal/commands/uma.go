package commands

import (
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/latoulicious/HKTM/pkg/uma"
)

var umaClient = uma.NewClient()
var navigationManager = uma.GetNavigationManager()

// UmaCommand handles Uma Musume related commands
func UmaCommand(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	if len(args) == 0 {
		s.ChannelMessageSend(m.ChannelID, "❌ Please specify a subcommand.\n\n**Usage:** `!uma char <character name>`\n**Example:** `!uma char Oguri Cap`")
		return
	}

	subcommand := strings.ToLower(args[0])

	switch subcommand {
	case "char", "character":
		CharacterCommand(s, m, args[1:])
	default:
		s.ChannelMessageSend(m.ChannelID, "❌ Unknown subcommand.\n\n**Available subcommands:**\n• `char <name>` - Search for a character\n\n**Example:** `!uma char Oguri Cap`")
	}
}

// CharacterCommand searches for and displays character information
func CharacterCommand(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	// Check if user provided a character name
	if len(args) == 0 {
		s.ChannelMessageSend(m.ChannelID, "❌ Please provide a character name to search for.\n\n**Usage:** `!uma char <character name>`\n**Example:** `!uma char Oguri Cap`")
		return
	}

	// Join the arguments to form the search query
	query := strings.Join(args, " ")

	// Send a loading message
	loadingMsg, _ := s.ChannelMessageSend(m.ChannelID, "🔍 Searching for character...")

	// Search for character
	result := umaClient.SearchCharacter(query)

	// Delete the loading message
	s.ChannelMessageDelete(m.ChannelID, loadingMsg.ID)

	if !result.Found {
		// Create error embed
		embed := &discordgo.MessageEmbed{
			Title:       "❌ Character Not Found",
			Description: fmt.Sprintf("Could not find character: **%s**", query),
			Color:       0xff0000, // Red color
			Timestamp:   time.Now().Format(time.RFC3339),
			Footer: &discordgo.MessageEmbedFooter{
				Text: "Hokko Tarumae | Uma Musume Search",
			},
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:   "💡 Tips",
					Value:  "• Try using the Japanese name\n• Check spelling and try alternative names\n• Try partial names (e.g., 'oguri' for 'Oguri Cap')",
					Inline: false,
				},
			},
		}

		if result.Error != nil {
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:   "🔧 Error",
				Value:  result.Error.Error(),
				Inline: false,
			})
		}

		s.ChannelMessageSendEmbed(m.ChannelID, embed)
		return
	}

	// Fetch character images
	imagesResult := umaClient.GetCharacterImages(result.Character.ID)

	// Create success embed with image navigation
	embed := navigationManager.CreateCharacterEmbed(result.Character, imagesResult, 0)

	// Send the initial embed
	msg, err := s.ChannelMessageSendEmbed(m.ChannelID, embed)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, "❌ Failed to send character information.")
		return
	}

	// Register navigation if there are multiple images
	totalImages := 0
	if imagesResult.Found {
		for _, category := range imagesResult.Images {
			totalImages += len(category.Images)
		}
	}

	if totalImages > 1 {
		navigationManager.RegisterNavigation(msg.ID, result.Character, imagesResult, m.ChannelID)

		// Add navigation emotes
		reactions := []string{"⬅️", "➡️", "🔄"}
		for _, reaction := range reactions {
			s.MessageReactionAdd(m.ChannelID, msg.ID, reaction)
		}
	}
}
