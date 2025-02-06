package config

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
)

type Config struct {
	DiscordToken string `json:"discord_token"`
}

func LoadConfig() (*Config, error) {
	// Load the configuration file
	configFile := "config.json"
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		return nil, err
	}

	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		return nil, err
	}

	var cfg Config
	err = json.Unmarshal(data, &cfg)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}
