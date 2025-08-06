package debug

import (
	"fmt"
	"log"

	"github.com/latoulicious/HKTM/pkg/common"
)

// TestSearchFunctionality tests the YouTube search functionality
func SearchTest() {
	fmt.Println("Testing YouTube Search Functionality...")

	// Test 1: Test URL detection
	fmt.Println("\n1. Testing URL detection...")
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
		if isURL == testCase.expected {
			fmt.Printf("✅ Test case %d passed: %s -> %v\n", i+1, testCase.input, isURL)
		} else {
			fmt.Printf("❌ Test case %d failed: %s -> %v (expected %v)\n", i+1, testCase.input, isURL, testCase.expected)
		}
	}

	// Test 2: Test YouTube search
	fmt.Println("\n2. Testing YouTube search...")
	testQuery := "siqlo one way street"
	fmt.Printf("Searching for: %s\n", testQuery)

	url, title, duration, err := common.SearchYouTubeAndGetURL(testQuery)
	if err != nil {
		log.Printf("❌ Search failed: %v", err)
		return
	}

	fmt.Printf("✅ Search successful!\n")
	fmt.Printf("   URL: %s\n", url)
	fmt.Printf("   Title: %s\n", title)
	fmt.Printf("   Duration: %v\n", duration)

	// Test 3: Test audio stream extraction from search result
	fmt.Println("\n3. Testing audio stream extraction...")
	streamURL, streamTitle, streamDuration, streamErr := common.GetYouTubeAudioStreamWithMetadata(url)
	if streamErr != nil {
		log.Printf("❌ Stream extraction failed: %v", streamErr)
		return
	}

	fmt.Printf("✅ Stream extraction successful!\n")
	fmt.Printf("   Stream URL: %s...\n", streamURL[:50])
	fmt.Printf("   Title: %s\n", streamTitle)
	fmt.Printf("   Duration: %v\n", streamDuration)

	fmt.Println("\n🎉 All tests completed successfully!")
}
