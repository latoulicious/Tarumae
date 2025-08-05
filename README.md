# Tarumae - Discord Music Bot

A Discord music bot built in Go that streams audio directly from YouTube using a robust FFmpeg + gopus pipeline.

## Features

- **Direct Audio Pipeline**: Uses yt-dlp → FFmpeg → gopus for maximum reliability
- **No DCA Dependency**: Eliminates the problematic DCA wrapper that fails silently
- **Full Control**: Complete visibility into the audio processing pipeline
- **Battle-Tested Tools**: Leverages proven FFmpeg and yt-dlp for audio extraction

## Architecture

The audio pipeline works as follows:

1. **yt-dlp**: Extracts direct audio stream URLs from YouTube
2. **FFmpeg**: Converts the stream to raw PCM audio (48kHz, stereo, 16-bit)
3. **gopus**: Directly encodes the PCM data to Opus format
4. **Discord**: Streams the Opus data directly to voice channels

This approach eliminates the DCA black box and gives us complete control over the audio processing pipeline.

## Requirements

- Go 1.23+
- FFmpeg
- yt-dlp
- Discord Bot Token

## Installation

1. Clone the repository
2. Install dependencies: `go mod tidy`
3. Set up your Discord bot token
4. Run: `go run cmd/main.go`

## Commands

- `!play <url>` - Play audio from a YouTube URL
- `!stop` - Stop playback
- `!pause` - Pause playback
- `!resume` - Resume playback
- `!skip` - Skip current track

## Why This Approach Works

- **Direct Control**: You manage the entire audio pipeline yourself
- **No Black Box**: When something breaks, you know exactly where (process died, pipe broken, etc.)
- **Battle-Tested Tools**: yt-dlp and FFmpeg are rock-solid when used correctly
- **Proper Error Handling**: You can detect and recover from specific failure points
- **gopus Integration**: Direct Opus encoding without external dependencies