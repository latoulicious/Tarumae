package common

import (
	"bytes"
	"fmt"
	"log"
	"net/url"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// IsYouTubeURL checks if a URL appears to be from YouTube
func IsYouTubeURL(urlStr string) bool {
	return strings.Contains(urlStr, "youtube.com") || strings.Contains(urlStr, "youtu.be")
}

// ExtractYouTubeVideoID extracts the video ID from a YouTube URL
func ExtractYouTubeVideoID(youtubeURL string) string {
	// Handle youtube.com URLs
	if strings.Contains(youtubeURL, "youtube.com") {
		parsedURL, err := url.Parse(youtubeURL)
		if err != nil {
			return ""
		}

		// Check for v parameter
		if videoID := parsedURL.Query().Get("v"); videoID != "" {
			return videoID
		}

		// Check for embed URLs like /embed/VIDEO_ID
		if strings.Contains(parsedURL.Path, "/embed/") {
			parts := strings.Split(parsedURL.Path, "/embed/")
			if len(parts) > 1 {
				return strings.Split(parts[1], "?")[0] // Remove any query params
			}
		}
	}

	// Handle youtu.be URLs
	if strings.Contains(youtubeURL, "youtu.be") {
		parsedURL, err := url.Parse(youtubeURL)
		if err != nil {
			return ""
		}

		// Extract video ID from path
		videoID := strings.TrimPrefix(parsedURL.Path, "/")
		return strings.Split(videoID, "?")[0] // Remove any query params
	}

	// Fallback: use regex to find 11-character alphanumeric video ID
	re := regexp.MustCompile(`[a-zA-Z0-9_-]{11}`)
	matches := re.FindAllString(youtubeURL, -1)
	if len(matches) > 0 {
		return matches[0]
	}

	return ""
}

// GetYouTubeThumbnailURL generates a thumbnail URL from a video ID
func GetYouTubeThumbnailURL(videoID string) string {
	if videoID == "" {
		return ""
	}
	// Use maxresdefault for best quality, fallback to hqdefault if needed
	return fmt.Sprintf("https://img.youtube.com/vi/%s/maxresdefault.jpg", videoID)
}

// GetYouTubeMetadata extracts both title and duration from a YouTube URL
func GetYouTubeMetadata(urlStr string) (title string, duration time.Duration, err error) {
	log.Printf("Extracting metadata from: %s", urlStr)

	// Use yt-dlp to get both title and duration
	cmd := exec.Command("yt-dlp",
		"--no-playlist",
		"--no-warnings",
		"--print", "title",
		"--print", "duration",
		urlStr)

	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		log.Printf("Failed to get metadata: %v", err)
		return "Unknown Title", 0, fmt.Errorf("failed to extract metadata: %v", err)
	}

	output := strings.TrimSpace(out.String())
	lines := strings.Split(output, "\n")

	if len(lines) >= 1 {
		title = strings.TrimSpace(lines[0])
	}
	if len(lines) >= 2 {
		durationStr := strings.TrimSpace(lines[1])
		if durationStr != "" && durationStr != "None" {
			// yt-dlp returns duration in seconds
			if seconds, parseErr := strconv.ParseFloat(durationStr, 64); parseErr == nil {
				duration = time.Duration(seconds * float64(time.Second))
			}
		}
	}

	if title == "" {
		title = "Unknown Title"
	}

	log.Printf("Extracted metadata - Title: %s, Duration: %v", title, duration)
	return title, duration, nil
}

// GetYouTubeAudioStreamWithMetadata extracts stream URL, title, and duration
func GetYouTubeAudioStreamWithMetadata(urlStr string) (streamURL, title string, duration time.Duration, err error) {
	log.Printf("Extracting audio stream and metadata from: %s", urlStr)

	// First, get metadata (title and duration)
	title, duration, metaErr := GetYouTubeMetadata(urlStr)
	if metaErr != nil {
		log.Printf("Warning: Failed to get metadata: %v", metaErr)
		title = "Unknown Title"
		duration = 0
	}

	// Then get stream URL with multiple fallback strategies
	strategies := [][]string{
		// Strategy 1: Best audio with format preference
		{"-f", "bestaudio[ext=m4a]/bestaudio[ext=webm]/bestaudio[ext=mp4]/bestaudio"},

		// Strategy 2: Android client (often bypasses restrictions)
		{"-f", "bestaudio", "--extractor-args", "youtube:player_client=android"},

		// Strategy 3: Web client with cookies
		{"-f", "bestaudio", "--extractor-args", "youtube:player_client=web"},

		// Strategy 4: Last resort - any audio
		{"-f", "worst[ext=m4a]/worst"},
	}

	for i, strategy := range strategies {
		log.Printf("Trying extraction strategy %d/%d", i+1, len(strategies))

		args := append([]string{"--no-playlist", "--no-warnings", "-g"}, strategy...)
		args = append(args, urlStr)

		cmd := exec.Command("yt-dlp", args...)
		var out bytes.Buffer
		cmd.Stdout = &out

		if err := cmd.Run(); err != nil {
			log.Printf("Strategy %d failed: %v", i+1, err)
			continue
		}

		streamURL = strings.TrimSpace(out.String())
		if streamURL != "" {
			// Take first URL if multiple are returned
			urls := strings.Split(streamURL, "\n")
			if len(urls) > 0 && urls[0] != "" {
				streamURL = urls[0]
				log.Printf("Successfully extracted stream URL using strategy %d", i+1)
				return streamURL, title, duration, nil
			}
		}
	}

	return "", title, duration, fmt.Errorf("failed to extract audio stream URL after trying all strategies")
}
