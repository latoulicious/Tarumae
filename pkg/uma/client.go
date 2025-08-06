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

// SearchSupportCard searches for a support card by name
func (c *Client) SearchSupportCard(query string) *SupportCardSearchResult {
	// Check cache first
	cacheKey := fmt.Sprintf("support_search_%s", strings.ToLower(query))
	if cached := c.getFromCache(cacheKey); cached != nil {
		if result, ok := cached.(*SupportCardSearchResult); ok {
			return result
		}
	}

	// First, get the list of support cards
	listResult := c.GetSupportCardList()
	if !listResult.Found {
		result := &SupportCardSearchResult{
			Found: false,
			Error: listResult.Error,
			Query: query,
		}
		c.setCache(cacheKey, result)
		return result
	}

	// Find the best match
	bestMatch := c.findBestSupportCardMatch(query, listResult.SupportCards)
	if bestMatch == nil {
		result := &SupportCardSearchResult{
			Found: false,
			Query: query,
		}
		c.setCache(cacheKey, result)
		return result
	}

	// Get detailed information for the matched card
	detailedResult := c.GetSupportCard(bestMatch.ID)
	if !detailedResult.Found {
		result := &SupportCardSearchResult{
			Found: false,
			Error: detailedResult.Error,
			Query: query,
		}
		c.setCache(cacheKey, result)
		return result
	}

	result := &SupportCardSearchResult{
		Found:       true,
		SupportCard: detailedResult.SupportCard,
		Query:       query,
	}

	c.setCache(cacheKey, result)
	return result
}

// GetSupportCardList fetches the list of all support cards
func (c *Client) GetSupportCardList() *SupportCardListResult {
	// Check cache first
	cacheKey := "support_list"
	if cached := c.getFromCache(cacheKey); cached != nil {
		if result, ok := cached.(*SupportCardListResult); ok {
			return result
		}
	}

	// Make API request
	url := fmt.Sprintf("%s/v1/support", c.baseURL)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		result := &SupportCardListResult{
			Found: false,
			Error: fmt.Errorf("failed to fetch support card list: %v", err),
		}
		c.setCache(cacheKey, result)
		return result
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		result := &SupportCardListResult{
			Found: false,
			Error: fmt.Errorf("API returned status code: %d", resp.StatusCode),
		}
		c.setCache(cacheKey, result)
		return result
	}

	var supportCards []SupportCard
	if err := json.NewDecoder(resp.Body).Decode(&supportCards); err != nil {
		result := &SupportCardListResult{
			Found: false,
			Error: fmt.Errorf("failed to decode API response: %v", err),
		}
		c.setCache(cacheKey, result)
		return result
	}

	result := &SupportCardListResult{
		Found:        true,
		SupportCards: supportCards,
	}

	c.setCache(cacheKey, result)
	return result
}

// GetSupportCard fetches detailed information for a specific support card
func (c *Client) GetSupportCard(supportID int) *SupportCardSearchResult {
	// Check cache first
	cacheKey := fmt.Sprintf("support_detail_%d", supportID)
	if cached := c.getFromCache(cacheKey); cached != nil {
		if result, ok := cached.(*SupportCardSearchResult); ok {
			return result
		}
	}

	// Make API request
	url := fmt.Sprintf("%s/v1/support/%d", c.baseURL, supportID)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		result := &SupportCardSearchResult{
			Found: false,
			Error: fmt.Errorf("failed to fetch support card details: %v", err),
		}
		c.setCache(cacheKey, result)
		return result
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		result := &SupportCardSearchResult{
			Found: false,
			Error: fmt.Errorf("API returned status code: %d", resp.StatusCode),
		}
		c.setCache(cacheKey, result)
		return result
	}

	var supportCard SupportCard
	if err := json.NewDecoder(resp.Body).Decode(&supportCard); err != nil {
		result := &SupportCardSearchResult{
			Found: false,
			Error: fmt.Errorf("failed to decode API response: %v", err),
		}
		c.setCache(cacheKey, result)
		return result
	}

	result := &SupportCardSearchResult{
		Found:       true,
		SupportCard: &supportCard,
	}

	c.setCache(cacheKey, result)
	return result
}

// findBestSupportCardMatch finds the best support card match for the given query
func (c *Client) findBestSupportCardMatch(query string, supportCards []SupportCard) *SupportCard {
	query = strings.ToLower(query)

	// First, try exact match with English title
	for _, card := range supportCards {
		if strings.ToLower(card.TitleEn) == query {
			return &card
		}
	}

	// Then, try contains match with English title
	for _, card := range supportCards {
		if strings.Contains(strings.ToLower(card.TitleEn), query) {
			return &card
		}
	}

	// Try Japanese title
	for _, card := range supportCards {
		if strings.Contains(strings.ToLower(card.Title), query) {
			return &card
		}
	}

	// Try gametora identifier
	for _, card := range supportCards {
		if strings.Contains(strings.ToLower(card.Gametora), query) {
			return &card
		}
	}

	// Finally, try partial word match
	queryWords := strings.Fields(query)
	for _, card := range supportCards {
		titleEn := strings.ToLower(card.TitleEn)
		titleJp := strings.ToLower(card.Title)
		gametora := strings.ToLower(card.Gametora)

		for _, word := range queryWords {
			if strings.Contains(titleEn, word) || strings.Contains(titleJp, word) || strings.Contains(gametora, word) {
				return &card
			}
		}
	}

	return nil
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
