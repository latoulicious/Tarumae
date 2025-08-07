# Test Suite Documentation

This directory contains comprehensive tests for the HKTM project. The tests are organized into four main categories: client tests, search tests, audio pipeline tests, and support card tests.

## Test Categories

### 1. Client Tests (`client_test.go`)
Tests for the Uma Musume API client functionality.

### 2. Search Tests (`search_test.go`)
Tests for YouTube search and URL detection functionality.

### 3. Audio Tests (`audio_test.go`)
Tests for the audio pipeline components including yt-dlp and FFmpeg integration.

### 4. Support Tests (`support_test.go`)
Tests for the Uma Musume support card functionality and Gametora integration.

## Available Test Functions

### Client Tests

| Test Function | Description |
|---------------|-------------|
| `TestNewClient` | Tests client initialization |
| `TestGetCharacterImages` | Tests character image fetching |
| `TestSearchCharacterExactMatch` | Tests exact character search |
| `TestSearchCharacterPartialMatch` | Tests partial character search |
| `TestSearchCharacterNoMatch` | Tests when no character is found |
| `TestSearchSupportCard` | Tests support card search |

### Search Tests

| Test Function | Description |
|---------------|-------------|
| `TestURLDetection` | Tests URL detection functionality |
| `TestYouTubeSearch` | Tests YouTube search functionality |
| `TestAudioStreamExtraction` | Tests audio stream extraction |

### Audio Tests

| Test Function | Description |
|---------------|-------------|
| `TestYtDlpAvailability` | Tests if yt-dlp is available |
| `TestFFmpegAvailability` | Tests if FFmpeg is available |
| `TestYtDlpURLExtraction` | Tests yt-dlp URL extraction |
| `TestFFmpegPCMConversion` | Tests FFmpeg PCM conversion |
| `TestFormatAvailability` | Tests format availability |
| `TestAudioPipelineIntegration` | Tests the complete audio pipeline |

### Support Tests

| Test Function | Description |
|---------------|-------------|
| `TestSupportCardSearch` | Tests support card search functionality |
| `TestSupportCardMultipleVersions` | Tests when multiple versions of a support card are found |
| `TestSupportCardHints` | Tests support card hints functionality |
| `TestSupportCardEventSkills` | Tests support card event skills functionality |
| `TestSupportCardNotFound` | Tests the behavior when a support card is not found |
| `TestSupportCardDebugSearch` | Tests the debug search functionality |
| `TestSupportCardIntegration` | Tests the complete support card functionality |

## Running Tests

### Run All Tests
```bash
go test ./test/ -v
```

### Run Specific Test Categories

#### Client Tests
```bash
# Run all client tests
go test ./test/ -v -run "Test.*Client"

# Run specific client test
go test ./test/ -v -run TestNewClient
go test ./test/ -v -run TestSearchCharacterExactMatch
```

#### Search Tests
```bash
# Run all search tests
go test ./test/ -v -run "Test.*Search"

# Run specific search test
go test ./test/ -v -run TestURLDetection
go test ./test/ -v -run TestYouTubeSearch
```

#### Audio Tests
```bash
# Run all audio tests
go test ./test/ -v -run "Test.*Audio"

# Run specific audio test
go test ./test/ -v -run TestYtDlpAvailability
go test ./test/ -v -run TestFFmpegAvailability
```

#### Support Tests
```bash
# Run all support tests
go test ./test/ -v -run "TestSupportCard"

# Run specific support test
go test ./test/ -v -run TestSupportCardSearch
go test ./test/ -v -run TestSupportCardHints
```

### Run Tests by Pattern

```bash
# Run all tests containing "Search" in the name
go test ./test/ -v -run ".*Search.*"

# Run all tests containing "Character" in the name
go test ./test/ -v -run ".*Character.*"

# Run all tests containing "Audio" in the name
go test ./test/ -v -run ".*Audio.*"

# Run all tests containing "Support" in the name
go test ./test/ -v -run ".*Support.*"
```

### Run Individual Tests

```bash
# Client tests
go test ./test/ -v -run TestNewClient
go test ./test/ -v -run TestGetCharacterImages
go test ./test/ -v -run TestSearchCharacterExactMatch
go test ./test/ -v -run TestSearchCharacterPartialMatch
go test ./test/ -v -run TestSearchCharacterNoMatch
go test ./test/ -v -run TestSearchSupportCard

# Search tests
go test ./test/ -v -run TestURLDetection
go test ./test/ -v -run TestYouTubeSearch
go test ./test/ -v -run TestAudioStreamExtraction

# Audio tests
go test ./test/ -v -run TestYtDlpAvailability
go test ./test/ -v -run TestFFmpegAvailability
go test ./test/ -v -run TestYtDlpURLExtraction
go test ./test/ -v -run TestFFmpegPCMConversion
go test ./test/ -v -run TestFormatAvailability
go test ./test/ -v -run TestAudioPipelineIntegration

# Support tests
go test ./test/ -v -run TestSupportCardSearch
go test ./test/ -v -run TestSupportCardMultipleVersions
go test ./test/ -v -run TestSupportCardHints
go test ./test/ -v -run TestSupportCardEventSkills
go test ./test/ -v -run TestSupportCardNotFound
go test ./test/ -v -run TestSupportCardDebugSearch
go test ./test/ -v -run TestSupportCardIntegration
```

## Test Dependencies

### External Tools Required
- **yt-dlp**: For YouTube video downloading and URL extraction
- **FFmpeg**: For audio conversion and processing

### Installation

#### Arch Linux
```bash
sudo pacman -S yt-dlp ffmpeg
```

#### Ubuntu/Debian
```bash
sudo apt update
sudo apt install yt-dlp ffmpeg
```

#### macOS
```bash
brew install yt-dlp ffmpeg
```

## Test Details

### Client Tests
The client tests verify the Uma Musume API client functionality:
- Client initialization
- Character search (exact, partial, no match)
- Character image fetching
- Support card search

### Search Tests
The search tests verify YouTube integration:
- URL detection for various formats
- YouTube search functionality
- Audio stream extraction

### Audio Tests
The audio tests verify the complete audio pipeline:
- Tool availability (yt-dlp, FFmpeg)
- URL extraction from YouTube
- Audio format conversion
- Complete pipeline integration

### Support Tests
The support tests verify the Uma Musume support card functionality:
- Support card search and retrieval
- Multiple version handling
- Support hints and event skills
- Debug search functionality
- Integration with Gametora API

## Notes

- Some tests make actual API calls to external services
- Audio tests require yt-dlp and FFmpeg to be installed
- Support tests require internet connectivity for Gametora API
- Tests may take several seconds to complete due to network requests
- The warnings during compilation are from the gopus library and don't affect test functionality

## Troubleshooting

### Common Issues

1. **yt-dlp not found**: Install yt-dlp using your package manager
2. **FFmpeg not found**: Install FFmpeg using your package manager
3. **Network timeouts**: Some tests require internet connectivity
4. **API rate limits**: Some tests may fail if API rate limits are exceeded
5. **Gametora API issues**: Support tests may fail if Gametora is down

### Debug Mode
To see more detailed output, use the `-v` flag:
```bash
go test ./test/ -v -run TestSupportCardSearch
```

This will show the test logs including success messages and detailed error information. 