package scrapper

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// LyricsResult represents the result of a lyrics search
type LyricsResult struct {
	Title    string
	Artist   string
	Lyrics   string
	URL      string
	Source   string
	Found    bool
	Error    error
}

// LyricsScraper handles lyrics scraping operations
type LyricsScraper struct {
	client  *http.Client
	cache   map[string]*LyricsResult
	cacheTTL time.Duration
}

// NewLyricsScraper creates a new lyrics scraper instance
func NewLyricsScraper() *LyricsScraper {
	return &LyricsScraper{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		cache:   make(map[string]*LyricsResult),
		cacheTTL: 30 * time.Minute, // Cache for 30 minutes
	}
}

// SearchLyrics searches for lyrics on AnimeLyrics.com
func (ls *LyricsScraper) SearchLyrics(query string) *LyricsResult {
	// Check cache first
	cacheKey := strings.ToLower(strings.TrimSpace(query))
	if cached, exists := ls.cache[cacheKey]; exists {
		return cached
	}

	// Clean and encode the query
	cleanQuery := strings.TrimSpace(query)
	if cleanQuery == "" {
		return &LyricsResult{
			Found: false,
			Error: fmt.Errorf("empty search query"),
		}
	}

	// Try AnimeLyrics.com first
	result := ls.searchAnimeLyrics(cleanQuery)
	if result.Found {
		// Cache the result
		ls.cache[cacheKey] = result
		return result
	}

	// If not found, return the error
	ls.cache[cacheKey] = result
	return result
}

// searchAnimeLyrics searches for lyrics on AnimeLyrics.com
func (ls *LyricsScraper) searchAnimeLyrics(query string) *LyricsResult {
	// Construct search URL
	searchURL := fmt.Sprintf("https://www.animelyrics.com/search.php?search=%s", url.QueryEscape(query))
	
	// Make HTTP request
	resp, err := ls.client.Get(searchURL)
	if err != nil {
		return &LyricsResult{
			Found: false,
			Error: fmt.Errorf("failed to fetch search results: %w", err),
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return &LyricsResult{
			Found: false,
			Error: fmt.Errorf("search request failed with status: %d", resp.StatusCode),
		}
	}

	// Parse the HTML
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return &LyricsResult{
			Found: false,
			Error: fmt.Errorf("failed to parse search results: %w", err),
		}
	}

	// Look for search results
	var firstResultURL string
	doc.Find("a[href*='anime/']").Each(func(i int, s *goquery.Selection) {
		if firstResultURL == "" {
			if href, exists := s.Attr("href"); exists {
				firstResultURL = href
			}
		}
	})

	if firstResultURL == "" {
		return &LyricsResult{
			Found: false,
			Error: fmt.Errorf("no lyrics found for: %s", query),
		}
	}

	// If the URL is relative, make it absolute
	if !strings.HasPrefix(firstResultURL, "http") {
		firstResultURL = "https://www.animelyrics.com" + firstResultURL
	}

	// Fetch the lyrics page
	return ls.fetchLyricsPage(firstResultURL)
}

// fetchLyricsPage fetches and parses a specific lyrics page
func (ls *LyricsScraper) fetchLyricsPage(pageURL string) *LyricsResult {
	resp, err := ls.client.Get(pageURL)
	if err != nil {
		return &LyricsResult{
			Found: false,
			Error: fmt.Errorf("failed to fetch lyrics page: %w", err),
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return &LyricsResult{
			Found: false,
			Error: fmt.Errorf("lyrics page request failed with status: %d", resp.StatusCode),
		}
	}

	// Parse the HTML
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return &LyricsResult{
			Found: false,
			Error: fmt.Errorf("failed to parse lyrics page: %w", err),
		}
	}

	// Extract title
	title := ""
	doc.Find("h1, h2, h3").Each(func(i int, s *goquery.Selection) {
		if title == "" {
			title = strings.TrimSpace(s.Text())
		}
	})

	// Extract artist (usually in the page content)
	artist := ""
	doc.Find("p, div").Each(func(i int, s *goquery.Selection) {
		text := strings.TrimSpace(s.Text())
		if artist == "" && (strings.Contains(text, "Artist:") || strings.Contains(text, "歌手:")) {
			artist = strings.TrimSpace(strings.Split(text, ":")[1])
		}
	})

	// Extract lyrics (look for the main lyrics content)
	lyrics := ""
	doc.Find("div.lyrics, div#lyrics, pre, .lyrics-content").Each(func(i int, s *goquery.Selection) {
		if lyrics == "" {
			lyrics = strings.TrimSpace(s.Text())
		}
	})

	// If no specific lyrics div found, try to find lyrics in the main content
	if lyrics == "" {
		doc.Find("body").Each(func(i int, s *goquery.Selection) {
			text := s.Text()
			// Look for patterns that indicate lyrics
			if strings.Contains(text, "[Verse]") || strings.Contains(text, "[Chorus]") || 
			   strings.Contains(text, "♪") || strings.Contains(text, "★") {
				lyrics = text
			}
		})
	}

	// Clean up the lyrics
	lyrics = ls.cleanLyrics(lyrics)

	if lyrics == "" {
		return &LyricsResult{
			Found: false,
			Error: fmt.Errorf("no lyrics content found on the page"),
		}
	}

	return &LyricsResult{
		Title:  title,
		Artist: artist,
		Lyrics: lyrics,
		URL:    pageURL,
		Source: "AnimeLyrics.com",
		Found:  true,
	}
}

// cleanLyrics cleans and formats the lyrics text
func (ls *LyricsScraper) cleanLyrics(lyrics string) string {
	if lyrics == "" {
		return ""
	}

	// Remove excessive whitespace
	re := regexp.MustCompile(`\s+`)
	lyrics = re.ReplaceAllString(lyrics, "\n")

	// Remove common HTML artifacts
	lyrics = strings.ReplaceAll(lyrics, "&nbsp;", " ")
	lyrics = strings.ReplaceAll(lyrics, "&amp;", "&")
	lyrics = strings.ReplaceAll(lyrics, "&lt;", "<")
	lyrics = strings.ReplaceAll(lyrics, "&gt;", ">")

	// Trim and limit length
	lyrics = strings.TrimSpace(lyrics)
	
	// Limit to reasonable length for Discord
	if len(lyrics) > 2000 {
		lyrics = lyrics[:1997] + "..."
	}

	return lyrics
}

// ClearCache clears the lyrics cache
func (ls *LyricsScraper) ClearCache() {
	ls.cache = make(map[string]*LyricsResult)
} 