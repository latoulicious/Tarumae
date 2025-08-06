package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/latoulicious/HKTM/pkg/uma"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run uma_debug.go <search_query>")
		fmt.Println("Example: go run uma_debug.go 'Almond Eye'")
		os.Exit(1)
	}

	query := os.Args[1]
	fmt.Printf("🔍 Testing Uma search for: %s\n", query)

	// Create Gametora client
	client := uma.NewGametoraClient()

	// Debug the search
	client.DebugSearchSupportCard(query)

	// Also test the actual search
	fmt.Printf("\n" + strings.Repeat("=", 50) + "\n")
	fmt.Printf("🔍 Testing actual search...\n")

	result := client.SearchSupportCard(query)
	if result.Found {
		fmt.Printf("✅ Found support card: %s (%s)\n", result.SupportCard.TitleEn, result.SupportCard.Title)
		fmt.Printf("   Character: %s (%s)\n", result.SupportCard.CharacterEn, result.SupportCard.Character)
		fmt.Printf("   Rarity: %s\n", result.SupportCard.Rarity)
		fmt.Printf("   Type: %s\n", result.SupportCard.Type)
		fmt.Printf("   URL Name: %s\n", result.SupportCard.URLName)
	} else {
		fmt.Printf("❌ Support card not found: %s\n", query)
		if result.Error != nil {
			fmt.Printf("   Error: %v\n", result.Error)
		}
	}
} 