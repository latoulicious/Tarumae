package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DiscordToken string
}

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

	return &Config{
		DiscordToken: discordToken,
	}, nil
}

var ErrDiscordTokenNotSet = os.ErrInvalid
