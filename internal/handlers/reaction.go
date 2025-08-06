package handlers

import (
	"github.com/bwmarrin/discordgo"
	"github.com/latoulicious/HKTM/pkg/uma"
)

// ReactionAddHandler handles reaction add events
func ReactionAddHandler(s *discordgo.Session, r *discordgo.MessageReactionAdd) {
	// Add comprehensive nil checks
	if s == nil || r == nil {
		return
	}

	// Ignore reactions from the bot itself
	if s.State.User != nil && r.UserID == s.State.User.ID {
		return
	}

	// Handle Uma character image navigation
	navigationManager := uma.GetNavigationManager()
	navigationManager.HandleReaction(s, r)
}

// ReactionRemoveHandler handles reaction remove events (for cleanup)
func ReactionRemoveHandler(s *discordgo.Session, r *discordgo.MessageReactionRemove) {
	// Add comprehensive nil checks
	if s == nil || r == nil {
		return
	}

	// For now, we don't need to handle reaction removal
	// The navigation state is managed separately
}
