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
		fmt.Println("Example: go run uma_debug.go '10001-special-week'")
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

	result := client.SearchSimplifiedSupportCard(query)
	if result.Found {
		fmt.Printf("✅ Found support card: %s\n", result.SupportCard.NameJp)
		fmt.Printf("   Character: %s\n", result.SupportCard.CharName)
		fmt.Printf("   Rarity: %d\n", result.SupportCard.Rarity)
		fmt.Printf("   Type: %s\n", result.SupportCard.Type)
		fmt.Printf("   Support ID: %d\n", result.SupportCard.SupportID)
		fmt.Printf("   URL Name: %s\n", result.SupportCard.URLName)

		// Test image URL generation
		imageURL := client.GetSupportCardImageURL(result.SupportCard.URLName)
		fmt.Printf("   Image URL: %s\n", imageURL)

		// Show all versions if multiple found
		if len(result.SupportCards) > 1 {
			fmt.Printf("\n📋 All versions found (%d):\n", len(result.SupportCards))
			for i, card := range result.SupportCards {
				rarityText := "R"
				switch card.Rarity {
				case 2:
					rarityText = "SR"
				case 3:
					rarityText = "SSR"
				}
				fmt.Printf("  %d. %s (%s) - ID: %d\n", i+1, card.NameJp, rarityText, card.SupportID)
			}

			// Test navigation embed creation
			fmt.Printf("\n🧭 Testing navigation embed for version 1 (SSR):\n")
			navManager := uma.GetSupportCardNavigationManager()
			navEmbed := navManager.CreateSupportCardEmbed(result.SupportCards[0], result.SupportCards, 0)
			fmt.Printf("  Title: %s\n", navEmbed.Title)
			fmt.Printf("  Footer: %s\n", navEmbed.Footer.Text)
			fmt.Printf("  Fields: %d\n", len(navEmbed.Fields))
		}

		// Show support hints
		if len(result.SupportCard.Hints.HintSkills) > 0 {
			fmt.Printf("\n💡 Support Hints (%d):\n", len(result.SupportCard.Hints.HintSkills))
			for i, hint := range result.SupportCard.Hints.HintSkills {
				fmt.Printf("  %d. %s\n", i+1, hint.NameEn)
			}
		}

		// Show event skills
		if len(result.SupportCard.EventSkills) > 0 {
			fmt.Printf("\n🎉 Event Skills (%d):\n", len(result.SupportCard.EventSkills))
			for i, event := range result.SupportCard.EventSkills {
				fmt.Printf("  %d. %s\n", i+1, event.NameEn)
			}
		}
	} else {
		fmt.Printf("❌ Support card not found: %s\n", query)
		if result.Error != nil {
			fmt.Printf("   Error: %v\n", result.Error)
		}
	}
}
