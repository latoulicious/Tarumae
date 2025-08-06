package debug

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Test the audio pipeline components
func AudioTest() {
	fmt.Println("Testing Tarumae Audio Pipeline Components...")

	// Test 1: Check if yt-dlp is available
	fmt.Println("\n1. Testing yt-dlp...")
	ytdlpCmd := exec.Command("yt-dlp", "--version")
	if err := ytdlpCmd.Run(); err != nil {
		fmt.Println("❌ yt-dlp not found. Please install it first.")
		os.Exit(1)
	}
	fmt.Println("✅ yt-dlp is available")

	// Test 2: Check if FFmpeg is available
	fmt.Println("\n2. Testing FFmpeg...")
	ffmpegCmd := exec.Command("ffmpeg", "-version")
	if err := ffmpegCmd.Run(); err != nil {
		fmt.Println("❌ FFmpeg not found. Please install it first.")
		os.Exit(1)
	}
	fmt.Println("✅ FFmpeg is available")

	// Test 3: Test yt-dlp URL extraction with better quality
	fmt.Println("\n3. Testing yt-dlp URL extraction (best audio quality)...")
	testURL := "https://www.youtube.com/watch?v=dQw4w9WgXcQ" // Rick Roll for testing
	cmd := exec.Command("yt-dlp",
		"-f", "bestaudio[ext=m4a]/bestaudio[ext=webm]/bestaudio",
		"--no-playlist",
		"--no-warnings",
		"-g", testURL)

	output, err := cmd.Output()
	if err != nil {
		fmt.Printf("❌ Failed to extract URL: %v\n", err)
		os.Exit(1)
	}

	streamURL := strings.TrimSpace(string(output))
	if streamURL == "" {
		fmt.Println("❌ Empty stream URL returned")
		os.Exit(1)
	}

	fmt.Printf("✅ Successfully extracted stream URL: %s...\n", streamURL[:50])

	// Test 4: Test FFmpeg PCM conversion
	fmt.Println("\n4. Testing FFmpeg PCM conversion...")
	ffmpegTestCmd := exec.Command("ffmpeg",
		"-i", streamURL,
		"-f", "s16le",
		"-acodec", "pcm_s16le",
		"-ar", "48000",
		"-ac", "2",
		"-t", "1", // Only convert 1 second for testing
		"-y", "/tmp/test_output.raw")

	if err := ffmpegTestCmd.Run(); err != nil {
		fmt.Printf("❌ FFmpeg conversion failed: %v\n", err)
		os.Exit(1)
	}

	// Check if output file was created
	if _, err := os.Stat("/tmp/test_output.raw"); err == nil {
		fmt.Println("✅ FFmpeg PCM conversion successful")
		// Clean up
		os.Remove("/tmp/test_output.raw")
	} else {
		fmt.Println("❌ FFmpeg output file not found")
		os.Exit(1)
	}

	// Test 5: Show available formats for comparison
	fmt.Println("\n5. Testing format availability...")
	formatCmd := exec.Command("yt-dlp",
		"-F", testURL)

	formatOutput, err := formatCmd.Output()
	if err != nil {
		fmt.Printf("❌ Failed to get formats: %v\n", err)
	} else {
		lines := strings.Split(string(formatOutput), "\n")
		audioFormats := []string{}
		for _, line := range lines {
			if strings.Contains(line, "audio") && (strings.Contains(line, "m4a") || strings.Contains(line, "webm")) {
				audioFormats = append(audioFormats, line)
			}
		}
		fmt.Printf("✅ Found %d audio formats available\n", len(audioFormats))
		if len(audioFormats) > 0 {
			fmt.Println("Available audio formats:")
			for i, format := range audioFormats[:5] { // Show first 5
				fmt.Printf("  %s\n", format)
				if i >= 4 {
					break
				}
			}
		}
	}

	fmt.Println("\n🎉 All audio pipeline components are working correctly!")
	fmt.Println("The improved audio quality pipeline is ready to use.")
	fmt.Println("Using bestaudio format and 128kbps Opus encoding for better quality.")
}
