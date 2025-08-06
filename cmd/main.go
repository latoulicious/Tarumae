package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
	"github.com/latoulicious/Tarumae/internal/commands"
	"github.com/latoulicious/Tarumae/internal/config"
	"github.com/latoulicious/Tarumae/internal/handlers"
	"github.com/latoulicious/Tarumae/internal/presence"
)

func main() {
	// Load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Create a new Discord session using the provided token
	dg, err := discordgo.New("Bot " + cfg.DiscordToken)
	if err != nil {
		log.Fatalf("Failed to create Discord session: %v", err)
	}

	// Create presence manager
	presenceManager := presence.NewPresenceManager(dg)

	// Set the presence manager in the commands package
	commands.SetPresenceManager(presenceManager)

	// Register the message handler
	dg.AddHandler(handlers.MessageHandler)

	// Register the slash command handler
	dg.AddHandler(handlers.SlashCommandHandler)

	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		log.Fatalf("Failed to open Discord session: %v", err)
	}

	// Set initial presence
	presenceManager.UpdateDefaultPresence()

	// Start periodic presence updates
	presenceManager.StartPeriodicUpdates()

	// Start idle monitor
	idleMonitor := commands.GetIdleMonitor()
	idleMonitor(dg)

	log.Println("Bot is running. Press CTRL-C to exit.")
	// Wait here until CTRL-C or other term signal is received.
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	// Cleanly close down the Discord session.
	dg.Close()
}
