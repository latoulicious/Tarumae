package commands

import (
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/latoulicious/HKTM/pkg/scrapper"
)

var lyricsScraper = scrapper.NewLyricsScraper()

// LyricsCommand searches for and displays lyrics
func LyricsCommand(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	// Check if user provided a search query
	if len(args) == 0 {
		s.ChannelMessageSend(m.ChannelID, "❌ Please provide a song title to search for lyrics.\n\n**Usage:** `!lyrics <song title>`\n**Example:** `!lyrics Cruel Angel's Thesis`")
		return
	}

	// Join the arguments to form the search query
	query := strings.Join(args, " ")

	// Send a loading message
	loadingMsg, _ := s.ChannelMessageSend(m.ChannelID, "🔍 Searching for lyrics...")

	// Search for lyrics
	result := lyricsScraper.SearchLyrics(query)

	// Delete the loading message
	s.ChannelMessageDelete(m.ChannelID, loadingMsg.ID)

	if !result.Found {
		// Create error embed
		embed := &discordgo.MessageEmbed{
			Title:       "❌ Lyrics Not Found",
			Description: fmt.Sprintf("Could not find lyrics for: **%s**", query),
			Color:       0xff0000, // Red color
			Timestamp:   time.Now().Format(time.RFC3339),
			Footer: &discordgo.MessageEmbedFooter{
				Text: "Hokko Tarumae | Lyrics Search",
			},
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:   "💡 Tips",
					Value:  "• Try using the original Japanese title\n• Check spelling and try alternative titles\n• Try searching for the artist name as well",
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

	// Create success embed
	embed := &discordgo.MessageEmbed{
		Title:       "🎵 Lyrics Found",
		Description: fmt.Sprintf("**%s**", result.Title),
		Color:       0x00ff00, // Green color
		Timestamp:   time.Now().Format(time.RFC3339),
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Source: %s", result.Source),
		},
		Fields: []*discordgo.MessageEmbedField{},
	}

	// Add artist field if available
	if result.Artist != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "🎤 Artist",
			Value:  result.Artist,
			Inline: true,
		})
	}

	// Add lyrics field
	embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
		Name:   "📝 Lyrics",
		Value:  result.Lyrics,
		Inline: false,
	})

	// Add source URL if available
	if result.URL != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "🔗 Source",
			Value:  fmt.Sprintf("[View on %s](%s)", result.Source, result.URL),
			Inline: false,
		})
	}

	s.ChannelMessageSendEmbed(m.ChannelID, embed)
}
