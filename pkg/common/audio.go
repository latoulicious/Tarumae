package common

import (
	"context"
	"fmt"
	"io"
	"log"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"layeh.com/gopus"
)

// AudioPipeline manages the entire audio streaming pipeline
type AudioPipeline struct {
	ctx         context.Context
	cancel      context.CancelFunc
	voiceConn   *discordgo.VoiceConnection
	ffmpegCmd   *exec.Cmd
	opusEncoder *gopus.Encoder
	isPlaying   bool
	mu          sync.RWMutex

	// Health monitoring
	lastFrameTime time.Time
	healthTicker  *time.Ticker

	// Error handling
	errorChan    chan error
	restartChan  chan struct{}
	maxRestarts  int
	restartCount int
}

// NewAudioPipeline creates a new audio pipeline
func NewAudioPipeline(vc *discordgo.VoiceConnection) *AudioPipeline {
	ctx, cancel := context.WithCancel(context.Background())

	return &AudioPipeline{
		ctx:           ctx,
		cancel:        cancel,
		voiceConn:     vc,
		maxRestarts:   3,
		errorChan:     make(chan error, 10),
		restartChan:   make(chan struct{}, 1),
		lastFrameTime: time.Now(),
	}
}

// PlayStream starts streaming audio from the given URL
func (ap *AudioPipeline) PlayStream(streamURL string) error {
	ap.mu.Lock()
	defer ap.mu.Unlock()

	if ap.isPlaying {
		return fmt.Errorf("pipeline is already playing")
	}

	// Initialize Opus encoder
	encoder, err := gopus.NewEncoder(48000, 2, gopus.Audio)
	if err != nil {
		return fmt.Errorf("failed to create opus encoder: %v", err)
	}
	encoder.SetBitrate(128000) // Higher bitrate for better quality
	ap.opusEncoder = encoder

	ap.isPlaying = true

	// Start health monitoring
	ap.startHealthMonitoring()

	// Start the main streaming goroutine
	go ap.streamLoop(streamURL)

	// Start error handler
	go ap.errorHandler(streamURL)

	return nil
}

// streamLoop is the main audio streaming loop with restart capability
func (ap *AudioPipeline) streamLoop(streamURL string) {
	defer func() {
		ap.mu.Lock()
		ap.isPlaying = false
		ap.mu.Unlock()
	}()

	for {
		select {
		case <-ap.ctx.Done():
			log.Println("Audio pipeline context cancelled")
			return
		case <-ap.restartChan:
			if ap.restartCount >= ap.maxRestarts {
				log.Printf("Max restart attempts (%d) reached, stopping", ap.maxRestarts)
				ap.errorChan <- fmt.Errorf("max restarts exceeded")
				return
			}
			ap.restartCount++
			log.Printf("Restarting audio pipeline (attempt %d/%d)", ap.restartCount, ap.maxRestarts)
			time.Sleep(2 * time.Second) // Brief delay before restart
		}

		err := ap.streamAudio(streamURL)
		if err != nil {
			log.Printf("Stream error: %v", err)
			ap.errorChan <- err

			// Check if we should restart
			if ap.shouldRestart(err) {
				select {
				case ap.restartChan <- struct{}{}:
				default:
				}
				continue
			}
			return
		}

		// Normal completion
		log.Println("Audio stream completed normally")
		return
	}
}

// streamAudio handles the actual audio streaming
func (ap *AudioPipeline) streamAudio(streamURL string) error {
	// Create FFmpeg command with better error handling and buffering
	cmd := exec.CommandContext(ap.ctx, "ffmpeg",
		"-reconnect", "1",
		"-reconnect_streamed", "1",
		"-reconnect_delay_max", "5",
		"-i", streamURL,
		"-f", "s16le",
		"-acodec", "pcm_s16le",
		"-ar", "48000",
		"-ac", "2",
		"-bufsize", "64k",
		"-")

	ap.ffmpegCmd = cmd

	// Capture stderr for debugging
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %v", err)
	}

	// Start stderr consumer to prevent blocking
	go ap.consumeStderr(stderrPipe)

	// Get stdout pipe
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %v", err)
	}

	// Start FFmpeg
	log.Println("Starting FFmpeg process...")
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start ffmpeg: %v", err)
	}

	// Ensure process cleanup
	defer func() {
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		cmd.Wait()
	}()

	// Wait for voice connection readiness
	if err := ap.waitForVoiceReady(); err != nil {
		return err
	}

	// Start speaking
	ap.voiceConn.Speaking(true)
	defer ap.voiceConn.Speaking(false)

	log.Println("Starting audio stream to Discord...")

	// Stream audio with proper buffering and error handling
	return ap.streamPCMToDiscord(stdout)
}

// streamPCMToDiscord handles the PCM to Opus conversion and Discord streaming
func (ap *AudioPipeline) streamPCMToDiscord(reader io.Reader) error {
	// Use buffered reader for better performance
	buffer := make([]byte, 3840) // 960 samples * 2 channels * 2 bytes (20ms at 48kHz)
	frameCount := 0

	for {
		select {
		case <-ap.ctx.Done():
			return nil
		default:
		}

		// Read PCM data with timeout
		readDone := make(chan int, 1)
		readErr := make(chan error, 1)

		go func() {
			n, err := io.ReadFull(reader, buffer)
			if err != nil {
				readErr <- err
				return
			}
			readDone <- n
		}()

		var n int
		var err error

		select {
		case n = <-readDone:
		case err = <-readErr:
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				log.Println("FFmpeg stream ended normally")
				return nil
			}
			return fmt.Errorf("error reading PCM data: %v", err)
		case <-time.After(5 * time.Second):
			return fmt.Errorf("timeout reading PCM data")
		}

		if n > 0 {
			// Convert bytes to int16 samples
			samples := bytesToInt16(buffer[:n])

			// Ensure we have exactly 960 samples per channel for 20ms frames
			if len(samples) != 1920 { // 960 samples * 2 channels
				// Pad or truncate to correct size
				if len(samples) < 1920 {
					padded := make([]int16, 1920)
					copy(padded, samples)
					samples = padded
				} else {
					samples = samples[:1920]
				}
			}

			// Encode to Opus
			opusData, err := ap.opusEncoder.Encode(samples, 960, len(buffer))
			if err != nil {
				log.Printf("Opus encoding error: %v", err)
				continue
			}

			// Send to Discord with non-blocking send
			select {
			case ap.voiceConn.OpusSend <- opusData:
				frameCount++
				ap.lastFrameTime = time.Now()

				// Log progress every 100 frames (2 seconds)
				if frameCount%100 == 0 {
					log.Printf("Streamed %d frames", frameCount)
				}
			case <-time.After(100 * time.Millisecond):
				log.Println("Warning: OpusSend channel blocked, skipping frame")
			}
		}
	}
}

// Error handling
func (ap *AudioPipeline) errorHandler(_ string) {
	for {
		select {
		case <-ap.ctx.Done():
			return
		case err := <-ap.errorChan:
			log.Printf("Pipeline error: %v", err)

			if ap.shouldRestart(err) {
				log.Println("Attempting to restart pipeline...")
				select {
				case ap.restartChan <- struct{}{}:
				default:
				}
			} else {
				log.Println("Error is not recoverable, stopping pipeline")
				ap.Stop()
				return
			}
		}
	}
}

// shouldRestart determines if an error is recoverable
func (ap *AudioPipeline) shouldRestart(err error) bool {
	if ap.restartCount >= ap.maxRestarts {
		return false
	}

	// Add logic to determine which errors are recoverable
	errStr := err.Error()
	recoverableErrors := []string{
		"stream health check failed",
		"timeout reading PCM data",
		"voice connection health check failed",
		"error reading PCM data",
	}

	for _, recoverable := range recoverableErrors {
		if contains(errStr, recoverable) {
			return true
		}
	}

	return false
}

// Utility functions
func (ap *AudioPipeline) waitForVoiceReady() error {
	timeout := time.After(10 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return fmt.Errorf("timeout waiting for voice connection")
		case <-ticker.C:
			if ap.voiceConn.Ready {
				return nil
			}
		}
	}
}

func (ap *AudioPipeline) consumeStderr(stderr io.ReadCloser) {
	defer stderr.Close()
	buffer := make([]byte, 1024)
	for {
		_, err := stderr.Read(buffer)
		if err != nil {
			return
		}
		// Optionally log FFmpeg stderr for debugging
		// log.Printf("FFmpeg: %s", string(buffer))
	}
}

// Stop gracefully stops the audio pipeline
func (ap *AudioPipeline) Stop() {
	ap.mu.Lock()
	defer ap.mu.Unlock()

	log.Println("Stopping audio pipeline...")
	ap.cancel()

	if ap.ffmpegCmd != nil && ap.ffmpegCmd.Process != nil {
		ap.ffmpegCmd.Process.Kill()
	}

	if ap.voiceConn != nil {
		ap.voiceConn.Speaking(false)
	}

	ap.isPlaying = false
}

// IsPlaying returns whether the pipeline is currently playing
func (ap *AudioPipeline) IsPlaying() bool {
	ap.mu.RLock()
	defer ap.mu.RUnlock()
	return ap.isPlaying
}

// Helper functions
func bytesToInt16(data []byte) []int16 {
	samples := make([]int16, len(data)/2)
	for i := 0; i < len(samples); i++ {
		samples[i] = int16(data[i*2]) | int16(data[i*2+1])<<8
	}
	return samples
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

// startHealthMonitoring starts health monitoring for the audio pipeline
func (ap *AudioPipeline) startHealthMonitoring() {
	ap.healthTicker = time.NewTicker(5 * time.Second)
	go func() {
		defer func() {
			if ap.healthTicker != nil {
				ap.healthTicker.Stop()
			}
		}()
		for {
			select {
			case <-ap.ctx.Done():
				return
			case <-ap.healthTicker.C:
				ap.checkHealth()
			}
		}
	}()
}

// checkHealth performs health checks on the audio pipeline
func (ap *AudioPipeline) checkHealth() {
	ap.mu.RLock()
	defer ap.mu.RUnlock()

	if !ap.isPlaying {
		return
	}

	// Check if we haven't received frames in a while
	if time.Since(ap.lastFrameTime) > 10*time.Second {
		log.Println("Health check failed: no frames received in 10 seconds")
		ap.errorChan <- fmt.Errorf("stream health check failed: no recent frames")
	}

	// Check voice connection state
	if ap.voiceConn == nil || !ap.voiceConn.Ready {
		log.Println("Health check failed: voice connection not ready")
		ap.errorChan <- fmt.Errorf("voice connection health check failed")
	}
}
