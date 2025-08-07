package uma

import (
	"fmt"
	"strings"
	"sync"

	"github.com/bwmarrin/discordgo"
)

// NavigationManager manages image navigation for Uma character embeds
type NavigationManager struct {
	activeNavigations map[string]*NavigationState
	mutex             sync.RWMutex
}

// NavigationState tracks the current state of image navigation
type NavigationState struct {
	Character    *Character
	ImagesResult *CharacterImagesResult
	CurrentIndex int
	MessageID    string
	ChannelID    string
}

// SupportCardNavigationState tracks the current state of support card version navigation
type SupportCardNavigationState struct {
	SupportCards []*SimplifiedSupportCard
	CurrentIndex int
	MessageID    string
	ChannelID    string
	Query        string
}

var navigationManager = &NavigationManager{
	activeNavigations: make(map[string]*NavigationState),
}

// SupportCardNavigationManager manages version navigation for support card embeds
type SupportCardNavigationManager struct {
	activeNavigations map[string]*SupportCardNavigationState
	mutex             sync.RWMutex
}

var supportCardNavigationManager = &SupportCardNavigationManager{
	activeNavigations: make(map[string]*SupportCardNavigationState),
}

// HandleReaction handles reaction events for Uma character image navigation
func (nm *NavigationManager) HandleReaction(s *discordgo.Session, r *discordgo.MessageReactionAdd) {
	// Only handle reactions from the bot's own messages
	if r.UserID == s.State.User.ID {
		return
	}

	// Check if this is a navigation reaction
	reaction := r.Emoji.Name
	if reaction != "⬅️" && reaction != "➡️" && reaction != "🔄" {
		return
	}

	// Get navigation state
	nm.mutex.RLock()
	state, exists := nm.activeNavigations[r.MessageID]
	nm.mutex.RUnlock()

	if !exists {
		return
	}

	// Handle navigation
	switch reaction {
	case "⬅️":
		// Previous image
		if state.CurrentIndex > 0 {
			state.CurrentIndex--
		} else {
			// Get total number of images
			totalImages := nm.getTotalImages(state.ImagesResult)
			state.CurrentIndex = totalImages - 1
		}
	case "➡️":
		// Next image
		totalImages := nm.getTotalImages(state.ImagesResult)
		if state.CurrentIndex < totalImages-1 {
			state.CurrentIndex++
		} else {
			state.CurrentIndex = 0
		}
	case "🔄":
		// Refresh images (re-fetch from API)
		client := NewClient()
		imagesResult := client.GetCharacterImages(state.Character.ID)
		if imagesResult.Found {
			state.ImagesResult = imagesResult
			state.CurrentIndex = 0
		}
	}

	// Update the embed
	embed := nm.createCharacterEmbed(state.Character, state.ImagesResult, state.CurrentIndex)

	// Update the message
	_, err := s.ChannelMessageEditEmbed(state.ChannelID, state.MessageID, embed)
	if err != nil {
		fmt.Printf("Error updating Uma navigation embed: %v\n", err)
		return
	}

	// Remove the user's reaction
	s.MessageReactionRemove(state.ChannelID, state.MessageID, reaction, r.UserID)
}

// RegisterNavigation registers a new Uma character navigation
func (nm *NavigationManager) RegisterNavigation(messageID string, character *Character, imagesResult *CharacterImagesResult, channelID string) {
	nm.mutex.Lock()
	defer nm.mutex.Unlock()

	nm.activeNavigations[messageID] = &NavigationState{
		Character:    character,
		ImagesResult: imagesResult,
		CurrentIndex: 0,
		MessageID:    messageID,
		ChannelID:    channelID,
	}
}

// CleanupNavigation removes a navigation state when no longer needed
func (nm *NavigationManager) CleanupNavigation(messageID string) {
	nm.mutex.Lock()
	defer nm.mutex.Unlock()

	delete(nm.activeNavigations, messageID)
}

// getTotalImages calculates the total number of images across all categories
func (nm *NavigationManager) getTotalImages(imagesResult *CharacterImagesResult) int {
	totalImages := 0
	if imagesResult.Found {
		for _, category := range imagesResult.Images {
			totalImages += len(category.Images)
		}
	}
	return totalImages
}

// CreateCharacterEmbed creates an embed for character display with image navigation
func (nm *NavigationManager) CreateCharacterEmbed(character *Character, imagesResult *CharacterImagesResult, imageIndex int) *discordgo.MessageEmbed {
	return nm.createCharacterEmbed(character, imagesResult, imageIndex)
}

// createCharacterEmbed creates an embed for character display with image navigation
func (nm *NavigationManager) createCharacterEmbed(character *Character, imagesResult *CharacterImagesResult, imageIndex int) *discordgo.MessageEmbed {
	// Build footer text
	footerText := "Data from umapyoi.net"
	if imagesResult.Found && len(imagesResult.Images) > 0 {
		totalImages := 0
		for _, category := range imagesResult.Images {
			totalImages += len(category.Images)
		}
		if totalImages > 1 {
			footerText = fmt.Sprintf("Data from umapyoi.net | Image %d of %d", imageIndex+1, totalImages)
		}
	}

	embed := &discordgo.MessageEmbed{
		Title:       "🏇 Character Found",
		Description: fmt.Sprintf("**%s** (%s)", character.NameEn, character.NameJp),
		Color:       0x00ff00, // Green color
		Footer: &discordgo.MessageEmbedFooter{
			Text: footerText,
		},
		Fields: []*discordgo.MessageEmbedField{},
	}

	// Add character metadata fields
	if character.CategoryLabelEn != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Category",
			Value:  character.CategoryLabelEn,
			Inline: true,
		})
	}

	embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
		Name:   "Character ID",
		Value:  fmt.Sprintf("%d", character.ID),
		Inline: true,
	})

	// Add character image if available
	if imagesResult.Found && len(imagesResult.Images) > 0 {
		// Flatten all images from all categories for navigation
		var allImages []CharacterImage
		var allCategories []string
		for _, category := range imagesResult.Images {
			for _, image := range category.Images {
				allImages = append(allImages, image)
				allCategories = append(allCategories, category.LabelEn)
			}
		}

		if len(allImages) > 0 {
			if imageIndex >= len(allImages) {
				imageIndex = 0
			}
			image := allImages[imageIndex]
			category := allCategories[imageIndex]

			// Add image details
			if len(allImages) > 1 {
				embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
					Value:  fmt.Sprintf("**Type:** %s", category),
					Inline: false,
				})
			}

			// Add the image
			embed.Image = &discordgo.MessageEmbedImage{
				URL: image.Image,
			}
		}
	} else if character.ThumbImg != "" {
		// Fallback to thumbnail if no images available
		embed.Image = &discordgo.MessageEmbedImage{
			URL: character.ThumbImg,
		}
	}

	return embed
}

// GetNavigationManager returns the global navigation manager instance
func GetNavigationManager() *NavigationManager {
	return navigationManager
}

// HandleSupportCardReaction handles reaction events for support card version navigation
func (scnm *SupportCardNavigationManager) HandleSupportCardReaction(s *discordgo.Session, r *discordgo.MessageReactionAdd) {
	// Only handle reactions from the bot's own messages
	if r.UserID == s.State.User.ID {
		return
	}

	// Check if this is a navigation reaction
	reaction := r.Emoji.Name

	// Define all possible number reactions (1️⃣ through 9️⃣)
	validReactions := []string{"1️⃣", "2️⃣", "3️⃣", "4️⃣", "5️⃣", "6️⃣", "7️⃣", "8️⃣", "9️⃣", "🔄"}

	isValidReaction := false
	for _, validReaction := range validReactions {
		if reaction == validReaction {
			isValidReaction = true
			break
		}
	}

	if !isValidReaction {
		return
	}

	// Get navigation state
	scnm.mutex.RLock()
	state, exists := scnm.activeNavigations[r.MessageID]
	scnm.mutex.RUnlock()

	if !exists {
		return
	}

	// Handle navigation
	switch reaction {
	case "1️⃣":
		// First version
		state.CurrentIndex = 0
	case "2️⃣":
		// Second version
		if len(state.SupportCards) > 1 {
			state.CurrentIndex = 1
		}
	case "3️⃣":
		// Third version
		if len(state.SupportCards) > 2 {
			state.CurrentIndex = 2
		}
	case "4️⃣":
		// Fourth version
		if len(state.SupportCards) > 3 {
			state.CurrentIndex = 3
		}
	case "5️⃣":
		// Fifth version
		if len(state.SupportCards) > 4 {
			state.CurrentIndex = 4
		}
	case "6️⃣":
		// Sixth version
		if len(state.SupportCards) > 5 {
			state.CurrentIndex = 5
		}
	case "7️⃣":
		// Seventh version
		if len(state.SupportCards) > 6 {
			state.CurrentIndex = 6
		}
	case "8️⃣":
		// Eighth version
		if len(state.SupportCards) > 7 {
			state.CurrentIndex = 7
		}
	case "9️⃣":
		// Ninth version
		if len(state.SupportCards) > 8 {
			state.CurrentIndex = 8
		}
	case "🔄":
		// Refresh (re-search)
		client := NewGametoraClient()
		result := client.SearchSimplifiedSupportCard(state.Query)
		if result.Found && len(result.SupportCards) > 0 {
			state.SupportCards = result.SupportCards
			state.CurrentIndex = 0
		}
	}

	// Update the embed
	embed := scnm.createSupportCardEmbed(state.SupportCards[state.CurrentIndex], state.SupportCards, state.CurrentIndex)

	// Update the message
	_, err := s.ChannelMessageEditEmbed(state.ChannelID, state.MessageID, embed)
	if err != nil {
		fmt.Printf("Error updating support card navigation embed: %v\n", err)
		return
	}

	// Remove the user's reaction
	s.MessageReactionRemove(state.ChannelID, state.MessageID, reaction, r.UserID)
}

// RegisterSupportCardNavigation registers a new support card version navigation
func (scnm *SupportCardNavigationManager) RegisterSupportCardNavigation(messageID string, supportCards []*SimplifiedSupportCard, channelID string, query string) {
	scnm.mutex.Lock()
	defer scnm.mutex.Unlock()

	scnm.activeNavigations[messageID] = &SupportCardNavigationState{
		SupportCards: supportCards,
		CurrentIndex: 0,
		MessageID:    messageID,
		ChannelID:    channelID,
		Query:        query,
	}
}

// CleanupSupportCardNavigation removes a support card navigation state
func (scnm *SupportCardNavigationManager) CleanupSupportCardNavigation(messageID string) {
	scnm.mutex.Lock()
	defer scnm.mutex.Unlock()

	delete(scnm.activeNavigations, messageID)
}

// CreateSupportCardEmbed creates an embed for support card display with version navigation
func (scnm *SupportCardNavigationManager) CreateSupportCardEmbed(supportCard *SimplifiedSupportCard, allCards []*SimplifiedSupportCard, currentIndex int) *discordgo.MessageEmbed {
	return scnm.createSupportCardEmbed(supportCard, allCards, currentIndex)
}

// createSupportCardEmbed creates an embed for support card display with version navigation
func (scnm *SupportCardNavigationManager) createSupportCardEmbed(supportCard *SimplifiedSupportCard, allCards []*SimplifiedSupportCard, currentIndex int) *discordgo.MessageEmbed {
	// Determine embed color based on rarity
	var color int
	switch supportCard.Rarity {
	case 3: // SSR
		color = 0xFFD700 // Gold
	case 2: // SR
		color = 0xC0C0C0 // Silver
	case 1: // R
		color = 0xCD7F32 // Bronze
	default:
		color = 0x7289DA // Default Discord blue
	}

	// Build footer text
	footerText := "Data from Gametora API | Hokko Tarumae"
	if len(allCards) > 1 {
		rarityText := "R"
		switch supportCard.Rarity {
		case 2:
			rarityText = "SR"
		case 3:
			rarityText = "SSR"
		}
		footerText = fmt.Sprintf("Data from Gametora API | Hokko Tarumae | %s Version (%d of %d)", rarityText, currentIndex+1, len(allCards))
	}

	// Create embed
	embed := &discordgo.MessageEmbed{
		Title:       supportCard.NameJp,
		Description: fmt.Sprintf("**Character:** %s", supportCard.CharName),
		Color:       color,
		Footer: &discordgo.MessageEmbedFooter{
			Text: footerText,
		},
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "🎴 Rarity",
				Value:  fmt.Sprintf("%d", supportCard.Rarity),
				Inline: true,
			},
			{
				Name:   "🎯 Type",
				Value:  supportCard.Type,
				Inline: true,
			},
			{
				Name:   "🆔 Support ID",
				Value:  fmt.Sprintf("%d", supportCard.SupportID),
				Inline: true,
			},
		},
	}

	// Add obtained info if available
	if supportCard.Obtained != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "📦 Obtained",
			Value:  supportCard.Obtained,
			Inline: true,
		})
	}

	// Add card image if available
	if supportCard.URLName != "" {
		client := NewGametoraClient()
		imageURL := client.GetSupportCardImageURL(supportCard.URLName)
		if imageURL != "" {
			embed.Image = &discordgo.MessageEmbedImage{
				URL: imageURL,
			}
		}
	}

	// Add support hints if available
	if len(supportCard.Hints.HintSkills) > 0 {
		var hintsText strings.Builder
		for i, hint := range supportCard.Hints.HintSkills {
			hintsText.WriteString(fmt.Sprintf("• %s", hint.NameEn))
			if i < len(supportCard.Hints.HintSkills)-1 {
				hintsText.WriteString("\n")
			}
		}

		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   fmt.Sprintf("💡 Support Hints (%d)", len(supportCard.Hints.HintSkills)),
			Value:  hintsText.String(),
			Inline: false,
		})
	}

	// Add event skills if available
	if len(supportCard.EventSkills) > 0 {
		var eventsText strings.Builder
		for i, event := range supportCard.EventSkills {
			eventsText.WriteString(fmt.Sprintf("• %s", event.NameEn))
			if i < len(supportCard.EventSkills)-1 {
				eventsText.WriteString("\n")
			}
		}

		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   fmt.Sprintf("🎉 Event Skills (%d)", len(supportCard.EventSkills)),
			Value:  eventsText.String(),
			Inline: false,
		})
	}

	// Add version info if multiple versions exist
	if len(allCards) > 1 {
		var versionsText strings.Builder
		for i, card := range allCards {
			rarityText := "R"
			switch card.Rarity {
			case 2:
				rarityText = "SR"
			case 3:
				rarityText = "SSR"
			}

			indicator := "○"
			if i == currentIndex {
				indicator = "●"
			}

			versionsText.WriteString(fmt.Sprintf("%s %s (%s) - ID: %d", indicator, card.NameJp, rarityText, card.SupportID))
			if i < len(allCards)-1 {
				versionsText.WriteString("\n")
			}
		}

		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   fmt.Sprintf("📋 Available Versions (%d)", len(allCards)),
			Value:  versionsText.String(),
			Inline: false,
		})
	}

	return embed
}

// GetSupportCardNavigationManager returns the global support card navigation manager instance
func GetSupportCardNavigationManager() *SupportCardNavigationManager {
	return supportCardNavigationManager
}
