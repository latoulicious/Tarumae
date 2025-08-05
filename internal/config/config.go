package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DiscordToken string
	OwnerID      string
}

var (
	ErrDiscordTokenNotSet = os.ErrInvalid
	ErrOwnerIDNotSet      = os.ErrInvalid
)

func LoadConfig() (*Config, error) {
	// Load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		return nil, err
	}

	discordToken := os.Getenv("DISCORD_TOKEN")
	if discordToken == "" {
		return nil, ErrDiscordTokenNotSet
	}

	ownerID := os.Getenv("BOT_OWNER_ID")
	if ownerID == "" {
		return nil, ErrOwnerIDNotSet
	}

	return &Config{
		DiscordToken: discordToken,
		OwnerID:      ownerID,
	}, nil
}
