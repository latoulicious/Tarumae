package uma

import (
	"fmt"
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

var navigationManager = &NavigationManager{
	activeNavigations: make(map[string]*NavigationState),
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
	embed := &discordgo.MessageEmbed{
		Title:       "🏇 Character Found",
		Description: fmt.Sprintf("**%s** (%s)", character.NameEn, character.NameJp),
		Color:       0x00ff00, // Green color
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Data from umapyoi.net",
		},
		Fields: []*discordgo.MessageEmbedField{},
	}

	// Flatten all images from all categories for navigation
	var allImages []CharacterImage
	var allCategories []string
	if imagesResult.Found {
		for _, category := range imagesResult.Images {
			for _, image := range category.Images {
				allImages = append(allImages, image)
				allCategories = append(allCategories, category.LabelEn)
			}
		}
	}

	// Add character image if available
	if len(allImages) > 0 {
		if imageIndex >= len(allImages) {
			imageIndex = 0
		}
		image := allImages[imageIndex]
		category := allCategories[imageIndex]
		embed.Image = &discordgo.MessageEmbedImage{
			URL: image.Image,
		}

		// Add image navigation info
		if len(allImages) > 1 {
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:   "🖼️ Image Navigation",
				Value:  fmt.Sprintf("Image %d of %d\nCategory: %s\nUploaded: %s", imageIndex+1, len(allImages), category, image.Uploaded),
				Inline: false,
			})
		}
	} else if character.ThumbImg != "" {
		// Fallback to thumbnail if no images available
		embed.Image = &discordgo.MessageEmbedImage{
			URL: character.ThumbImg,
		}
	}

	// Add category field
	if character.CategoryLabelEn != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "🏷️ Category",
			Value:  character.CategoryLabelEn,
			Inline: true,
		})
	}

	// Add character ID field
	embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
		Name:   "🆔 Character ID",
		Value:  fmt.Sprintf("%d", character.ID),
		Inline: true,
	})

	// Add row number field
	embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
		Name:   "📊 Row Number",
		Value:  fmt.Sprintf("%d", character.RowNumber),
		Inline: true,
	})

	return embed
}

// GetNavigationManager returns the global navigation manager instance
func GetNavigationManager() *NavigationManager {
	return navigationManager
}
