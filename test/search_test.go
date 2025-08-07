package test

import (
	"testing"

	"github.com/latoulicious/HKTM/pkg/common"
)

// TestURLDetection tests the URL detection functionality
func TestURLDetection(t *testing.T) {
	testCases := []struct {
		input    string
		expected bool
	}{
		{"https://www.youtube.com/watch?v=dQw4w9WgXcQ", true},
		{"https://youtu.be/dQw4w9WgXcQ", true},
		{"http://www.example.com", true},
		{"www.example.com", true},
		{"rick astley", false},
		{"never gonna give you up", false},
	}

	for i, testCase := range testCases {
		isURL := common.IsURL(testCase.input)
		if isURL != testCase.expected {
			t.Errorf("Test case %d failed: %s -> %v (expected %v)", i+1, testCase.input, isURL, testCase.expected)
		}
	}
}

// TestYouTubeSearch tests the YouTube search functionality
func TestYouTubeSearch(t *testing.T) {
	testQuery := "siqlo one way street"
	t.Logf("Searching for: %s", testQuery)

	url, title, duration, err := common.SearchYouTubeAndGetURL(testQuery)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if url == "" {
		t.Error("Expected URL to be returned")
	}

	if title == "" {
		t.Error("Expected title to be returned")
	}

	t.Logf("Search successful - URL: %s, Title: %s, Duration: %v", url, title, duration)
}

// TestAudioStreamExtraction tests the audio stream extraction functionality
func TestAudioStreamExtraction(t *testing.T) {
	// First get a URL from search
	testQuery := "siqlo one way street"
	url, _, _, err := common.SearchYouTubeAndGetURL(testQuery)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	// Test audio stream extraction from search result
	streamURL, streamTitle, streamDuration, streamErr := common.GetYouTubeAudioStreamWithMetadata(url)
	if streamErr != nil {
		t.Fatalf("Stream extraction failed: %v", streamErr)
	}

	if streamURL == "" {
		t.Error("Expected stream URL to be returned")
	}

	if streamTitle == "" {
		t.Error("Expected stream title to be returned")
	}

	t.Logf("Stream extraction successful - Stream URL: %s..., Title: %s, Duration: %v",
		streamURL[:min(50, len(streamURL))], streamTitle, streamDuration)
}

// Helper function to get minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
