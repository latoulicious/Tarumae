package uma

import (
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	client := NewClient()
	if client == nil {
		t.Fatal("NewClient() returned nil")
	}

	if client.baseURL != "https://umapyoi.net/api" {
		t.Errorf("Expected baseURL to be 'https://umapyoi.net/api', got '%s'", client.baseURL)
	}

	if client.cacheTTL != 5*time.Minute {
		t.Errorf("Expected cacheTTL to be 5 minutes, got %v", client.cacheTTL)
	}
}

func TestFindBestMatch(t *testing.T) {
	client := NewClient()

	characters := []Character{
		{ID: 4879, NameEn: "Oguri Cap", NameJp: "オグリキャップ"},
		{ID: 4737, NameEn: "Special Week", NameJp: "スペシャルウィーク"},
		{ID: 4536, NameEn: "Silence Suzuka", NameJp: "サイレンススズカ"},
	}

	// Test exact match
	result := client.findBestMatch("oguri cap", characters)
	if result == nil || result.NameEn != "Oguri Cap" {
		t.Error("Expected to find 'Oguri Cap' with exact match")
	}

	// Test partial match
	result = client.findBestMatch("oguri", characters)
	if result == nil || result.NameEn != "Oguri Cap" {
		t.Error("Expected to find 'Oguri Cap' with partial match")
	}

	// Test no match
	result = client.findBestMatch("nonexistent", characters)
	if result != nil {
		t.Error("Expected no match for 'nonexistent'")
	}
}

func TestGetCharacterImages(t *testing.T) {
	client := NewClient()

	// Test with a known character ID (Admire Vega)
	result := client.GetCharacterImages(5191)

	if !result.Found {
		t.Error("Expected to find images for character ID 5191")
	}

	if len(result.Images) == 0 {
		t.Error("Expected to find image categories")
	}

	// Check that we have the expected categories
	expectedCategories := []string{"Uniform", "Racing Outfit", "Concept Art", "Starting Future"}
	foundCategories := make(map[string]bool)

	for _, category := range result.Images {
		foundCategories[category.LabelEn] = true
	}

	for _, expected := range expectedCategories {
		if !foundCategories[expected] {
			t.Errorf("Expected category '%s' not found", expected)
		}
	}

	// Check that each category has images
	for _, category := range result.Images {
		if len(category.Images) == 0 {
			t.Errorf("Category '%s' has no images", category.LabelEn)
		}

		for _, image := range category.Images {
			if image.Image == "" {
				t.Errorf("Image URL is empty in category '%s'", category.LabelEn)
			}
		}
	}
}
