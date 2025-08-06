# Lyrics Scraper Package

A lightweight web scraper for fetching anime song lyrics from AnimeLyrics.com.

## Features

- **Lightweight**: Only ~150 LOC for scraping logic
- **Lazy-loaded**: Only fetches on demand when `!lyrics` command is used
- **Caching**: Built-in cache to skip repeated fetches (30-minute TTL)
- **Error handling**: Comprehensive error handling and user-friendly messages
- **Discord integration**: Optimized for Discord embed display

## Usage

### Basic Usage

```go
import "github.com/latoulicious/HKTM/pkg/scrapper"

// Create a new scraper instance
scraper := scrapper.NewLyricsScraper()

// Search for lyrics
result := scraper.SearchLyrics("Cruel Angel's Thesis")

if result.Found {
    fmt.Printf("Title: %s\n", result.Title)
    fmt.Printf("Artist: %s\n", result.Artist)
    fmt.Printf("Lyrics: %s\n", result.Lyrics)
    fmt.Printf("Source: %s\n", result.Source)
    fmt.Printf("URL: %s\n", result.URL)
} else {
    fmt.Printf("Error: %s\n", result.Error)
}
```

### Discord Bot Integration

The scraper is already integrated into the HKTM Discord bot with the `!lyrics` command:

```
!lyrics <song title>
```

Examples:
- `!lyrics Cruel Angel's Thesis`
- `!lyrics 残酷な天使のテーゼ`
- `!lyrics 君の名は`

## Architecture

### Components

1. **LyricsScraper**: Main scraper struct with HTTP client and cache
2. **LyricsResult**: Result struct containing song information and lyrics
3. **Cache**: In-memory cache to avoid repeated requests

### Scraping Process

1. **Search**: Query AnimeLyrics.com search page
2. **Parse**: Extract first result URL
3. **Fetch**: Download lyrics page
4. **Extract**: Parse title, artist, and lyrics content
5. **Clean**: Format and limit text for Discord
6. **Cache**: Store result for future requests

## Error Handling

The scraper handles various error scenarios:

- Network timeouts (10-second timeout)
- HTTP errors (non-200 status codes)
- Parse errors (malformed HTML)
- Empty results (no lyrics found)
- Invalid queries (empty search terms)

## Performance

- **Timeout**: 10 seconds per request
- **Cache TTL**: 30 minutes
- **Text limit**: 2000 characters for Discord compatibility
- **Concurrent requests**: Single-threaded (no concurrency)

## Dependencies

- `github.com/PuerkitoBio/goquery` - HTML parsing
- `net/http` - HTTP client
- `regexp` - Text cleaning
- `time` - Cache TTL

## Future Enhancements

- Support for additional lyrics sites
- Better Japanese text handling
- Improved caching with file storage
- Rate limiting to respect site policies
- Concurrent scraping for faster results 