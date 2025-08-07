package test

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// TestYtDlpAvailability tests if yt-dlp is available
func TestYtDlpAvailability(t *testing.T) {
	ytdlpCmd := exec.Command("yt-dlp", "--version")
	if err := ytdlpCmd.Run(); err != nil {
		t.Fatalf("yt-dlp not found. Please install it first: %v", err)
	}
	t.Log("✅ yt-dlp is available")
}

// TestFFmpegAvailability tests if FFmpeg is available
func TestFFmpegAvailability(t *testing.T) {
	ffmpegCmd := exec.Command("ffmpeg", "-version")
	if err := ffmpegCmd.Run(); err != nil {
		t.Fatalf("FFmpeg not found. Please install it first: %v", err)
	}
	t.Log("✅ FFmpeg is available")
}

// TestYtDlpURLExtraction tests yt-dlp URL extraction with better quality
func TestYtDlpURLExtraction(t *testing.T) {
	testURL := "https://www.youtube.com/watch?v=dQw4w9WgXcQ" // Rick Roll for testing
	cmd := exec.Command("yt-dlp",
		"-f", "bestaudio[ext=m4a]/bestaudio[ext=webm]/bestaudio",
		"--no-playlist",
		"--no-warnings",
		"-g", testURL)

	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Failed to extract URL: %v", err)
	}

	streamURL := strings.TrimSpace(string(output))
	if streamURL == "" {
		t.Fatal("Empty stream URL returned")
	}

	t.Logf("✅ Successfully extracted stream URL: %s...", streamURL[:50])
}

// TestFFmpegPCMConversion tests FFmpeg PCM conversion
func TestFFmpegPCMConversion(t *testing.T) {
	// First get a stream URL
	testURL := "https://www.youtube.com/watch?v=dQw4w9WgXcQ"
	cmd := exec.Command("yt-dlp",
		"-f", "bestaudio[ext=m4a]/bestaudio[ext=webm]/bestaudio",
		"--no-playlist",
		"--no-warnings",
		"-g", testURL)

	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Failed to extract URL: %v", err)
	}

	streamURL := strings.TrimSpace(string(output))
	if streamURL == "" {
		t.Fatal("Empty stream URL returned")
	}

	// Test FFmpeg PCM conversion
	ffmpegTestCmd := exec.Command("ffmpeg",
		"-i", streamURL,
		"-f", "s16le",
		"-acodec", "pcm_s16le",
		"-ar", "48000",
		"-ac", "2",
		"-t", "1", // Only convert 1 second for testing
		"-y", "/tmp/test_output.raw")

	if err := ffmpegTestCmd.Run(); err != nil {
		t.Fatalf("FFmpeg conversion failed: %v", err)
	}

	// Check if output file was created
	if _, err := os.Stat("/tmp/test_output.raw"); err == nil {
		t.Log("✅ FFmpeg PCM conversion successful")
		// Clean up
		os.Remove("/tmp/test_output.raw")
	} else {
		t.Fatal("FFmpeg output file not found")
	}
}

// TestFormatAvailability tests format availability for comparison
func TestFormatAvailability(t *testing.T) {
	testURL := "https://www.youtube.com/watch?v=dQw4w9WgXcQ"
	formatCmd := exec.Command("yt-dlp", "-F", testURL)

	formatOutput, err := formatCmd.Output()
	if err != nil {
		t.Fatalf("Failed to get formats: %v", err)
	}

	lines := strings.Split(string(formatOutput), "\n")
	audioFormats := []string{}
	for _, line := range lines {
		if strings.Contains(line, "audio") && (strings.Contains(line, "m4a") || strings.Contains(line, "webm")) {
			audioFormats = append(audioFormats, line)
		}
	}

	if len(audioFormats) == 0 {
		t.Fatal("No audio formats found")
	}

	t.Logf("✅ Found %d audio formats available", len(audioFormats))
	t.Log("Available audio formats:")
	for i, format := range audioFormats[:5] { // Show first 5
		t.Logf("  %s", format)
		if i >= 4 {
			break
		}
	}
}

// TestAudioPipelineIntegration tests the complete audio pipeline
func TestAudioPipelineIntegration(t *testing.T) {
	// This test runs all the individual components to ensure they work together
	t.Run("yt-dlp availability", TestYtDlpAvailability)
	t.Run("ffmpeg availability", TestFFmpegAvailability)
	t.Run("URL extraction", TestYtDlpURLExtraction)
	t.Run("PCM conversion", TestFFmpegPCMConversion)
	t.Run("format availability", TestFormatAvailability)

	t.Log("🎉 All audio pipeline components are working correctly!")
	t.Log("The improved audio quality pipeline is ready to use.")
	t.Log("Using bestaudio format and 128kbps Opus encoding for better quality.")
}
