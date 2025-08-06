package uma

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Client represents the Uma Musume API client
type Client struct {
	baseURL    string
	httpClient *http.Client
	cache      map[string]*CacheEntry
	cacheMutex sync.RWMutex
	cacheTTL   time.Duration
}

// NewClient creates a new Uma Musume API client
func NewClient() *Client {
	return &Client{
		baseURL: "https://umapyoi.net/api",
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		cache:    make(map[string]*CacheEntry),
		cacheTTL: 5 * time.Minute, // Cache for 5 minutes
	}
}

// SearchCharacter searches for a character by name
func (c *Client) SearchCharacter(query string) *CharacterSearchResult {
	// Check cache first
	cacheKey := fmt.Sprintf("char_search_%s", strings.ToLower(query))
	if cached := c.getFromCache(cacheKey); cached != nil {
		if result, ok := cached.(*CharacterSearchResult); ok {
			return result
		}
	}

	// Make API request
	url := fmt.Sprintf("%s/v1/character/list", c.baseURL)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		result := &CharacterSearchResult{
			Found: false,
			Error: fmt.Errorf("failed to fetch character data: %v", err),
			Query: query,
		}
		c.setCache(cacheKey, result)
		return result
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		result := &CharacterSearchResult{
			Found: false,
			Error: fmt.Errorf("API returned status code: %d", resp.StatusCode),
			Query: query,
		}
		c.setCache(cacheKey, result)
		return result
	}

	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		result := &CharacterSearchResult{
			Found: false,
			Error: fmt.Errorf("failed to decode API response: %v", err),
			Query: query,
		}
		c.setCache(cacheKey, result)
		return result
	}

	// Find the best match
	bestMatch := c.findBestMatch(query, apiResp)

	result := &CharacterSearchResult{
		Found:     bestMatch != nil,
		Character: bestMatch,
		Query:     query,
	}

	c.setCache(cacheKey, result)
	return result
}

// findBestMatch finds the best character match for the given query
func (c *Client) findBestMatch(query string, characters []Character) *Character {
	query = strings.ToLower(query)

	// First, try exact match with English name
	for _, char := range characters {
		if strings.ToLower(char.NameEn) == query {
			return &char
		}
	}

	// Then, try contains match with English name
	for _, char := range characters {
		if strings.Contains(strings.ToLower(char.NameEn), query) {
			return &char
		}
	}

	// Try Japanese name
	for _, char := range characters {
		if strings.Contains(strings.ToLower(char.NameJp), query) {
			return &char
		}
	}

	// Finally, try partial word match
	queryWords := strings.Fields(query)
	for _, char := range characters {
		charNameEn := strings.ToLower(char.NameEn)
		charNameJp := strings.ToLower(char.NameJp)
		for _, word := range queryWords {
			if strings.Contains(charNameEn, word) || strings.Contains(charNameJp, word) {
				return &char
			}
		}
	}

	return nil
}

// GetCharacterImages fetches all images for a character by ID
func (c *Client) GetCharacterImages(charaID int) *CharacterImagesResult {
	// Check cache first
	cacheKey := fmt.Sprintf("char_images_%d", charaID)
	if cached := c.getFromCache(cacheKey); cached != nil {
		if result, ok := cached.(*CharacterImagesResult); ok {
			return result
		}
	}

	// Make API request
	url := fmt.Sprintf("%s/v1/character/images/%d", c.baseURL, charaID)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		result := &CharacterImagesResult{
			Found:   false,
			Error:   fmt.Errorf("failed to fetch character images: %v", err),
			CharaID: charaID,
		}
		c.setCache(cacheKey, result)
		return result
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		result := &CharacterImagesResult{
			Found:   false,
			Error:   fmt.Errorf("API returned status code: %d", resp.StatusCode),
			CharaID: charaID,
		}
		c.setCache(cacheKey, result)
		return result
	}

	var images []CharacterImageCategory
	if err := json.NewDecoder(resp.Body).Decode(&images); err != nil {
		result := &CharacterImagesResult{
			Found:   false,
			Error:   fmt.Errorf("failed to decode API response: %v", err),
			CharaID: charaID,
		}
		c.setCache(cacheKey, result)
		return result
	}

	result := &CharacterImagesResult{
		Found:   true,
		Images:  images,
		CharaID: charaID,
	}

	c.setCache(cacheKey, result)
	return result
}

// getFromCache retrieves an item from cache
func (c *Client) getFromCache(key string) interface{} {
	c.cacheMutex.RLock()
	defer c.cacheMutex.RUnlock()

	if entry, exists := c.cache[key]; exists && !entry.IsExpired() {
		return entry.Data
	}

	return nil
}

// setCache stores an item in cache
func (c *Client) setCache(key string, data interface{}) {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()

	c.cache[key] = &CacheEntry{
		Data:      data,
		Timestamp: time.Now(),
		TTL:       c.cacheTTL,
	}
}
