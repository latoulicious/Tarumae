package common

import (
	"log"
	"math/rand"
	"os"
)

var panicMessages = []string{
	// Base warnings
	"Whoa there cowboy — production mode is disabled.",
	"Nope. This bot isn't meant for deployment. Read the README.",
	"You thought ENABLE_PRODUCTION_MODE=true would work? Cute.",
	"This is a personal project, not a microservice.",
	"Kaboom. You tripped the anti-prod nuke. Bye.",

	// Developer confessionals
	"If you deploy this, you legally become the new maintainer.",
	"SLA: 0%. Uptime: also 0%. Confidence: negative.",
	"Some parts were copy-pasted from a StackOverflow answer... from 2013.",

	// Discord bot dissuasion
	"This isn't a drop-in music bot. It's a spiritual experience.",
	"Congrats! You just deployed a fully operational stress test.",

	// Mysterious/absurd confusion
	"Only me and God know what lies beyond this point. Good luck.",
	"Only me and God understand what's going on here. And even we're guessing.",
	"You weren't supposed to see this message. Forget it now.",
	"Check the README. Then the source. Then your life choices.",

	// Fake support / passive-aggressive
	"Thank you for calling support. Please hang up and rethink your actions.",
	"Please submit a support ticket via smoke signal.",
	"Usage exceeded expected stupidity threshold. Shutting down.",
}

func CheckPersonalUse() {
	if os.Getenv("ENABLE_PRODUCTION_MODE") == "true" {
		panic(panicMessages[rand.Intn(len(panicMessages))])
	}
}

func EnforceGuildAndDev(id string) {
	if os.Getenv("ENABLE_PRODUCTION_MODE") == "true" {
		return
	}

	ownerID := os.Getenv("BOT_OWNER_ID")

	if id != ownerID {
		log.Fatalf("You're not my owner. User ID: %s not allowed.", id)
	} else {
		log.Printf("Welcome back, %s!", ownerID)
	}
}
