package uma

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// GametoraClient represents the Gametora API client for stable JSON endpoints
type GametoraClient struct {
	baseURL    string
	httpClient *http.Client
	cache      map[string]*CacheEntry
	cacheMutex sync.RWMutex
	cacheTTL   time.Duration
	buildID    string
	buildMutex sync.RWMutex
}

// GametoraSupportCard represents the support card data from Gametora JSON API
type GametoraSupportCard struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	TitleEn     string `json:"titleEn"`
	Rarity      string `json:"rarity"`
	Type        string `json:"type"`
	Character   string `json:"character"`
	CharacterEn string `json:"characterEn"`
	URLName     string `json:"url_name"`
	Levels      []struct {
		Level int `json:"level"`
		Stats struct {
			Speed   int `json:"speed,omitempty"`
			Stamina int `json:"stamina,omitempty"`
			Power   int `json:"power,omitempty"`
			Guts    int `json:"guts,omitempty"`
			Wisdom  int `json:"wisdom,omitempty"`
		} `json:"stats"`
	} `json:"levels"`
	Skills []struct {
		Name          string `json:"name"`
		NameEn        string `json:"nameEn"`
		Description   string `json:"description"`
		DescriptionEn string `json:"descriptionEn"`
		Level         int    `json:"level"`
	} `json:"skills"`
	Events []struct {
		Title       string `json:"title"`
		TitleEn     string `json:"titleEn"`
		Description string `json:"description"`
		Choices     []struct {
			Text    string   `json:"text"`
			TextEn  string   `json:"textEn"`
			Effects []string `json:"effects"`
		} `json:"choices"`
	} `json:"events"`
	TrainingBonuses []struct {
		TrainingType   string `json:"trainingType"`
		TrainingTypeEn string `json:"trainingTypeEn"`
		Bonus          string `json:"bonus"`
		Description    string `json:"description"`
	} `json:"trainingBonuses"`
	Effects []struct {
		Name          string `json:"name"`
		NameEn        string `json:"nameEn"`
		Description   string `json:"description"`
		DescriptionEn string `json:"descriptionEn"`
		Type          string `json:"type"`
	} `json:"effects"`
	Hints []struct {
		Title         string `json:"title"`
		TitleEn       string `json:"titleEn"`
		Description   string `json:"description"`
		DescriptionEn string `json:"descriptionEn"`
	} `json:"hints"`
}

// GametoraCharacter represents the character data from Gametora JSON API
type GametoraCharacter struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	NameEn        string `json:"nameEn"`
	Profile       string `json:"profile"`
	ProfileEn     string `json:"profileEn"`
	Height        string `json:"height"`
	Weight        string `json:"weight"`
	Birthday      string `json:"birthday"`
	BloodType     string `json:"bloodType"`
	ThreeSizes    string `json:"threeSizes"`
	Hobbies       string `json:"hobbies"`
	HobbiesEn     string `json:"hobbiesEn"`
	Speciality    string `json:"speciality"`
	SpecialityEn  string `json:"specialityEn"`
	Personality   string `json:"personality"`
	PersonalityEn string `json:"personalityEn"`
	Motivation    string `json:"motivation"`
	MotivationEn  string `json:"motivationEn"`
	Images        []struct {
		URL         string `json:"url"`
		Description string `json:"description"`
		Type        string `json:"type"`
	} `json:"images"`
}

// GametoraSupportsResponse represents the response from the supports.json endpoint
type GametoraSupportsResponse struct {
	PageProps struct {
		SupportData []struct {
			URLName     string  `json:"url_name"`
			SupportID   int     `json:"support_id"`
			CharID      int     `json:"char_id"`
			CharName    string  `json:"char_name"`
			NameJp      string  `json:"name_jp"`
			NameKo      string  `json:"name_ko"`
			NameTw      string  `json:"name_tw"`
			Rarity      int     `json:"rarity"`
			Type        string  `json:"type"`
			Obtained    string  `json:"obtained"`
			Release     string  `json:"release"`
			ReleaseKo   string  `json:"release_ko,omitempty"`
			ReleaseZhTw string  `json:"release_zh_tw,omitempty"`
			ReleaseEn   string  `json:"release_en,omitempty"`
			Effects     [][]int `json:"effects"`
			Hints       struct {
				HintSkills []struct {
					ID     int      `json:"id"`
					Type   []string `json:"type"`
					NameEn string   `json:"name_en"`
					IconID int      `json:"iconid"`
				} `json:"hint_skills"`
				HintOthers []struct {
					HintType  int `json:"hint_type"`
					HintValue int `json:"hint_value"`
				} `json:"hint_others"`
			} `json:"hints"`
			EventSkills []struct {
				ID     int      `json:"id"`
				Type   []string `json:"type"`
				NameEn string   `json:"name_en"`
				Rarity int      `json:"rarity"`
				IconID int      `json:"iconid"`
			} `json:"event_skills"`
			Unique *struct {
				Level   int `json:"level"`
				Effects []struct {
					Type   int `json:"type"`
					Value  int `json:"value"`
					Value1 int `json:"value_1,omitempty"`
					Value2 int `json:"value_2,omitempty"`
					Value3 int `json:"value_3,omitempty"`
					Value4 int `json:"value_4,omitempty"`
				} `json:"effects"`
			} `json:"unique,omitempty"`
		} `json:"supportData"`
	} `json:"pageProps"`
}

// GametoraCharacterResponse represents the response from the character JSON endpoint
type GametoraCharacterResponse struct {
	PageProps struct {
		Character GametoraCharacter `json:"character"`
	} `json:"pageProps"`
}

// GametoraSupportCardResponse represents the response from the support card JSON endpoint
type GametoraSupportCardResponse struct {
	PageProps struct {
		SupportCard GametoraSupportCard `json:"supportCard"`
	} `json:"pageProps"`
}

// GametoraSearchResult represents the result of a Gametora search
type GametoraSearchResult struct {
	Found       bool
	SupportCard *GametoraSupportCard
	Character   *GametoraCharacter
	Error       error
	Query       string
}

// NewGametoraClient creates a new Gametora API client
func NewGametoraClient() *GametoraClient {
	return &GametoraClient{
		baseURL: "https://gametora.com/_next/data",
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
		cache:    make(map[string]*CacheEntry),
		cacheTTL: 30 * time.Minute, // Cache for 30 minutes
	}
}

// GetBuildID fetches the current build ID from Gametora
func (c *GametoraClient) GetBuildID() (string, error) {
	c.buildMutex.RLock()
	if c.buildID != "" {
		defer c.buildMutex.RUnlock()
		return c.buildID, nil
	}
	c.buildMutex.RUnlock()

	// Fetch the main page to get the build ID
	resp, err := c.httpClient.Get("https://gametora.com/umamusume/supports")
	if err != nil {
		return "", fmt.Errorf("failed to fetch build ID: %v", err)
	}
	defer resp.Body.Close()

	// Read the response body
	body := make([]byte, 1024*1024) // 1MB buffer
	n, err := resp.Body.Read(body)
	if err != nil && n == 0 {
		return "", fmt.Errorf("failed to read response body: %v", err)
	}

	// Look for build ID patterns
	bodyStr := string(body[:n])

	// Try to find build ID using different approaches

	// Use a simple string search approach
	if strings.Contains(bodyStr, "_next/data/") {
		// Find the pattern _next/data/{build_id}/
		start := strings.Index(bodyStr, "_next/data/")
		if start != -1 {
			start += len("_next/data/")
			end := strings.Index(bodyStr[start:], "/")
			if end != -1 {
				buildID := bodyStr[start : start+end]
				if len(buildID) > 10 && len(buildID) < 50 {
					c.buildMutex.Lock()
					c.buildID = buildID
					c.buildMutex.Unlock()
					return buildID, nil
				}
			}
		}
	}

	// Try to find buildId in JSON
	if strings.Contains(bodyStr, "buildId") {
		start := strings.Index(bodyStr, "buildId")
		if start != -1 {
			// Look for the value after buildId
			valueStart := strings.Index(bodyStr[start:], "\"")
			if valueStart != -1 {
				valueStart += start + valueStart + 1
				valueEnd := strings.Index(bodyStr[valueStart:], "\"")
				if valueEnd != -1 {
					buildID := bodyStr[valueStart : valueStart+valueEnd]
					if len(buildID) > 10 && len(buildID) < 50 {
						c.buildMutex.Lock()
						c.buildID = buildID
						c.buildMutex.Unlock()
						return buildID, nil
					}
				}
			}
		}
	}

	// If no build ID found, try a hardcoded one as fallback
	// This is the build ID from your example
	fallbackBuildID := "4Lod4e9rq2HCjy-VKjMHJ"
	c.buildMutex.Lock()
	c.buildID = fallbackBuildID
	c.buildMutex.Unlock()

	return fallbackBuildID, nil
}

// SearchSupportCard searches for a support card using the Gametora JSON API
func (c *GametoraClient) SearchSupportCard(query string) *GametoraSearchResult {
	// Check cache first
	cacheKey := fmt.Sprintf("gametora_support_%s", strings.ToLower(query))
	if cached := c.getFromCache(cacheKey); cached != nil {
		if result, ok := cached.(*GametoraSearchResult); ok {
			return result
		}
	}

	// Get build ID
	buildID, err := c.GetBuildID()
	if err != nil {
		result := &GametoraSearchResult{
			Found: false,
			Error: fmt.Errorf("failed to get build ID: %v", err),
			Query: query,
		}
		c.setCache(cacheKey, result)
		return result
	}

	// First, get the list of all support cards
	supportsURL := fmt.Sprintf("%s/%s/umamusume/supports.json", c.baseURL, buildID)
	resp, err := c.httpClient.Get(supportsURL)
	if err != nil {
		result := &GametoraSearchResult{
			Found: false,
			Error: fmt.Errorf("failed to fetch supports list: %v", err),
			Query: query,
		}
		c.setCache(cacheKey, result)
		return result
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		result := &GametoraSearchResult{
			Found: false,
			Error: fmt.Errorf("supports API returned status code: %d", resp.StatusCode),
			Query: query,
		}
		c.setCache(cacheKey, result)
		return result
	}

	var supportsResp GametoraSupportsResponse
	if err := json.NewDecoder(resp.Body).Decode(&supportsResp); err != nil {
		result := &GametoraSearchResult{
			Found: false,
			Error: fmt.Errorf("failed to decode supports response: %v", err),
			Query: query,
		}
		c.setCache(cacheKey, result)
		return result
	}

	// Find the best match
	query = strings.ToLower(strings.TrimSpace(query))
	var bestMatch *struct {
		URLName     string  `json:"url_name"`
		SupportID   int     `json:"support_id"`
		CharID      int     `json:"char_id"`
		CharName    string  `json:"char_name"`
		NameJp      string  `json:"name_jp"`
		NameKo      string  `json:"name_ko"`
		NameTw      string  `json:"name_tw"`
		Rarity      int     `json:"rarity"`
		Type        string  `json:"type"`
		Obtained    string  `json:"obtained"`
		Release     string  `json:"release"`
		ReleaseKo   string  `json:"release_ko,omitempty"`
		ReleaseZhTw string  `json:"release_zh_tw,omitempty"`
		ReleaseEn   string  `json:"release_en,omitempty"`
		Effects     [][]int `json:"effects"`
		Hints       struct {
			HintSkills []struct {
				ID     int      `json:"id"`
				Type   []string `json:"type"`
				NameEn string   `json:"name_en"`
				IconID int      `json:"iconid"`
			} `json:"hint_skills"`
			HintOthers []struct {
				HintType  int `json:"hint_type"`
				HintValue int `json:"hint_value"`
			} `json:"hint_others"`
		} `json:"hints"`
		EventSkills []struct {
			ID     int      `json:"id"`
			Type   []string `json:"type"`
			NameEn string   `json:"name_en"`
			Rarity int      `json:"rarity"`
			IconID int      `json:"iconid"`
		} `json:"event_skills"`
		Unique *struct {
			Level   int `json:"level"`
			Effects []struct {
				Type   int `json:"type"`
				Value  int `json:"value"`
				Value1 int `json:"value_1,omitempty"`
				Value2 int `json:"value_2,omitempty"`
				Value3 int `json:"value_3,omitempty"`
				Value4 int `json:"value_4,omitempty"`
			} `json:"effects"`
		} `json:"unique,omitempty"`
	}
	var bestScore int = -1

	for _, support := range supportsResp.PageProps.SupportData {
		urlName := strings.ToLower(support.URLName)
		charName := strings.ToLower(support.CharName)

		// Calculate match score (higher is better)
		score := 0

		// Exact matches get highest priority
		if urlName == query || charName == query {
			score = 100
		} else if strings.HasPrefix(urlName, query) || strings.HasPrefix(charName, query) {
			score = 80
		} else if strings.Contains(urlName, query) || strings.Contains(charName, query) {
			score = 60
		} else {
			// Try word-by-word matching
			queryWords := strings.Fields(query)
			for _, word := range queryWords {
				if len(word) > 2 { // Only consider words longer than 2 characters
					if strings.Contains(urlName, word) || strings.Contains(charName, word) {
						score += 10
					}
				}
			}
		}

		// Update best match if this score is higher
		if score > bestScore {
			bestScore = score
			bestMatch = &support
		}
	}

	if bestMatch == nil {
		result := &GametoraSearchResult{
			Found: false,
			Query: query,
		}
		c.setCache(cacheKey, result)
		return result
	}

	// The supports.json already contains all the data we need
	// We don't need to make another API call since the core data is already in the list
	// Just create the support card from the list data
	supportCard := &GametoraSupportCard{
		URLName:     bestMatch.URLName,
		Character:   bestMatch.CharName,
		CharacterEn: bestMatch.CharName,
		Title:       bestMatch.NameJp,
		TitleEn:     bestMatch.NameJp, // Using Japanese name as English name since API doesn't provide English name
		Rarity:      fmt.Sprintf("%d", bestMatch.Rarity),
		Type:        bestMatch.Type,
	}

	// Copy effects data
	for _, effect := range bestMatch.Effects {
		supportCard.Effects = append(supportCard.Effects, struct {
			Name          string `json:"name"`
			NameEn        string `json:"nameEn"`
			Description   string `json:"description"`
			DescriptionEn string `json:"descriptionEn"`
			Type          string `json:"type"`
		}{
			Name:          fmt.Sprintf("Level %d", effect[0]),
			NameEn:        fmt.Sprintf("Level %d", effect[0]),
			Description:   fmt.Sprintf("Effect: %d", effect[1]),
			DescriptionEn: fmt.Sprintf("Effect: %d", effect[1]),
			Type:          fmt.Sprintf("Level %d", effect[0]),
		})
	}

	// Copy hints data
	for _, hint := range bestMatch.Hints.HintSkills {
		supportCard.Hints = append(supportCard.Hints, struct {
			Title         string `json:"title"`
			TitleEn       string `json:"titleEn"`
			Description   string `json:"description"`
			DescriptionEn string `json:"descriptionEn"`
		}{
			Title:         fmt.Sprintf("Skill %d", hint.ID),
			TitleEn:       hint.NameEn,
			Description:   fmt.Sprintf("Type: %v", hint.Type),
			DescriptionEn: fmt.Sprintf("Type: %v", hint.Type),
		})
	}
	for _, hint := range bestMatch.Hints.HintOthers {
		supportCard.Hints = append(supportCard.Hints, struct {
			Title         string `json:"title"`
			TitleEn       string `json:"titleEn"`
			Description   string `json:"description"`
			DescriptionEn string `json:"descriptionEn"`
		}{
			Title:         fmt.Sprintf("Hint Type %d", hint.HintType),
			TitleEn:       fmt.Sprintf("Hint Type %d", hint.HintType),
			Description:   fmt.Sprintf("Value: %d", hint.HintValue),
			DescriptionEn: fmt.Sprintf("Value: %d", hint.HintValue),
		})
	}

	// Copy events data
	for _, event := range bestMatch.EventSkills {
		supportCard.Events = append(supportCard.Events, struct {
			Title       string `json:"title"`
			TitleEn     string `json:"titleEn"`
			Description string `json:"description"`
			Choices     []struct {
				Text    string   `json:"text"`
				TextEn  string   `json:"textEn"`
				Effects []string `json:"effects"`
			} `json:"choices"`
		}{
			Title:       fmt.Sprintf("Event Skill %d", event.ID),
			TitleEn:     event.NameEn,
			Description: fmt.Sprintf("Type: %v, Rarity: %d", event.Type, event.Rarity),
			Choices: []struct {
				Text    string   `json:"text"`
				TextEn  string   `json:"textEn"`
				Effects []string `json:"effects"`
			}{
				{Text: "Description not available", TextEn: "Description not available"},
				{Text: "Description not available", TextEn: "Description not available"},
			},
		})
	}

	// Copy unique data
	if bestMatch.Unique != nil {
		supportCard.Levels = append(supportCard.Levels, struct {
			Level int `json:"level"`
			Stats struct {
				Speed   int `json:"speed,omitempty"`
				Stamina int `json:"stamina,omitempty"`
				Power   int `json:"power,omitempty"`
				Guts    int `json:"guts,omitempty"`
				Wisdom  int `json:"wisdom,omitempty"`
			} `json:"stats"`
		}{
			Level: bestMatch.Unique.Level,
			Stats: struct {
				Speed   int `json:"speed,omitempty"`
				Stamina int `json:"stamina,omitempty"`
				Power   int `json:"power,omitempty"`
				Guts    int `json:"guts,omitempty"`
				Wisdom  int `json:"wisdom,omitempty"`
			}{
				Speed:   bestMatch.Unique.Effects[0].Value,
				Stamina: bestMatch.Unique.Effects[0].Value1,
				Power:   bestMatch.Unique.Effects[0].Value2,
				Guts:    bestMatch.Unique.Effects[0].Value3,
				Wisdom:  bestMatch.Unique.Effects[0].Value4,
			},
		})
	}

	result := &GametoraSearchResult{
		Found:       true,
		SupportCard: supportCard,
		Query:       query,
	}

	c.setCache(cacheKey, result)
	return result
}

// getFromCache retrieves an item from cache
func (c *GametoraClient) getFromCache(key string) interface{} {
	c.cacheMutex.RLock()
	defer c.cacheMutex.RUnlock()

	if entry, exists := c.cache[key]; exists && !entry.IsExpired() {
		return entry.Data
	}

	return nil
}

// setCache stores an item in cache
func (c *GametoraClient) setCache(key string, data interface{}) {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()

	c.cache[key] = &CacheEntry{
		Data:      data,
		Timestamp: time.Now(),
		TTL:       c.cacheTTL,
	}
}

// GetSupportCardImageURL generates the image URL for a support card based on its URL name
func (c *GametoraClient) GetSupportCardImageURL(urlName string) string {
	// Extract the ID from the URL name (e.g., "10001-special-week" -> "10001")
	parts := strings.Split(urlName, "-")
	if len(parts) > 0 {
		return fmt.Sprintf("https://gametora.com/images/umamusume/supports/tex_support_card_%s.png", parts[0])
	}
	return ""
}

// DebugSearchSupportCard is a debug function that prints detailed information about the search process
func (c *GametoraClient) DebugSearchSupportCard(query string) {
	fmt.Printf("🔍 Debugging search for: %s\n", query)

	// Get build ID
	buildID, err := c.GetBuildID()
	if err != nil {
		fmt.Printf("❌ Failed to get build ID: %v\n", err)
		return
	}
	fmt.Printf("📦 Build ID: %s\n", buildID)

	// Fetch supports list
	supportsURL := fmt.Sprintf("%s/%s/umamusume/supports.json", c.baseURL, buildID)
	fmt.Printf("🌐 Fetching from: %s\n", supportsURL)

	resp, err := c.httpClient.Get(supportsURL)
	if err != nil {
		fmt.Printf("❌ Failed to fetch supports: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("❌ API returned status: %d\n", resp.StatusCode)
		return
	}

	var supportsResp GametoraSupportsResponse
	if err := json.NewDecoder(resp.Body).Decode(&supportsResp); err != nil {
		fmt.Printf("❌ Failed to decode response: %v\n", err)
		return
	}

	fmt.Printf("📊 Found %d support cards\n", len(supportsResp.PageProps.SupportData))

	// Show first few cards for reference
	fmt.Printf("\n📋 First 5 support cards:\n")
	for i, support := range supportsResp.PageProps.SupportData {
		if i >= 5 {
			break
		}
		fmt.Printf("  %d. %s (%s) - %s\n", i+1, support.NameJp, support.NameJp, support.CharName)
	}

	// Search for the query
	query = strings.ToLower(strings.TrimSpace(query))
	fmt.Printf("\n🔎 Searching for: '%s'\n", query)

	var matches []struct {
		Support struct {
			URLName     string  `json:"url_name"`
			SupportID   int     `json:"support_id"`
			CharID      int     `json:"char_id"`
			CharName    string  `json:"char_name"`
			NameJp      string  `json:"name_jp"`
			NameKo      string  `json:"name_ko"`
			NameTw      string  `json:"name_tw"`
			Rarity      int     `json:"rarity"`
			Type        string  `json:"type"`
			Obtained    string  `json:"obtained"`
			Release     string  `json:"release"`
			ReleaseKo   string  `json:"release_ko,omitempty"`
			ReleaseZhTw string  `json:"release_zh_tw,omitempty"`
			ReleaseEn   string  `json:"release_en,omitempty"`
			Effects     [][]int `json:"effects"`
			Hints       struct {
				HintSkills []struct {
					ID     int      `json:"id"`
					Type   []string `json:"type"`
					NameEn string   `json:"name_en"`
					IconID int      `json:"iconid"`
				} `json:"hint_skills"`
				HintOthers []struct {
					HintType  int `json:"hint_type"`
					HintValue int `json:"hint_value"`
				} `json:"hint_others"`
			} `json:"hints"`
			EventSkills []struct {
				ID     int      `json:"id"`
				Type   []string `json:"type"`
				NameEn string   `json:"name_en"`
				Rarity int      `json:"rarity"`
				IconID int      `json:"iconid"`
			} `json:"event_skills"`
			Unique *struct {
				Level   int `json:"level"`
				Effects []struct {
					Type   int `json:"type"`
					Value  int `json:"value"`
					Value1 int `json:"value_1,omitempty"`
					Value2 int `json:"value_2,omitempty"`
					Value3 int `json:"value_3,omitempty"`
					Value4 int `json:"value_4,omitempty"`
				} `json:"effects"`
			} `json:"unique,omitempty"`
		}
		Score  int
		Reason string
	}

	for _, support := range supportsResp.PageProps.SupportData {
		urlName := strings.ToLower(support.URLName)
		charName := strings.ToLower(support.CharName)

		score := 0
		reason := ""

		// Exact matches
		if urlName == query || charName == query {
			score = 100
			reason = "exact match"
		} else if strings.HasPrefix(urlName, query) || strings.HasPrefix(charName, query) {
			score = 80
			reason = "prefix match"
		} else if strings.Contains(urlName, query) || strings.Contains(charName, query) {
			score = 60
			reason = "contains match"
		} else {
			// Word-by-word matching
			queryWords := strings.Fields(query)
			for _, word := range queryWords {
				if len(word) > 2 {
					if strings.Contains(urlName, word) || strings.Contains(charName, word) {
						score += 10
						reason = "word match"
					}
				}
			}
		}

		if score > 0 {
			matches = append(matches, struct {
				Support struct {
					URLName     string  `json:"url_name"`
					SupportID   int     `json:"support_id"`
					CharID      int     `json:"char_id"`
					CharName    string  `json:"char_name"`
					NameJp      string  `json:"name_jp"`
					NameKo      string  `json:"name_ko"`
					NameTw      string  `json:"name_tw"`
					Rarity      int     `json:"rarity"`
					Type        string  `json:"type"`
					Obtained    string  `json:"obtained"`
					Release     string  `json:"release"`
					ReleaseKo   string  `json:"release_ko,omitempty"`
					ReleaseZhTw string  `json:"release_zh_tw,omitempty"`
					ReleaseEn   string  `json:"release_en,omitempty"`
					Effects     [][]int `json:"effects"`
					Hints       struct {
						HintSkills []struct {
							ID     int      `json:"id"`
							Type   []string `json:"type"`
							NameEn string   `json:"name_en"`
							IconID int      `json:"iconid"`
						} `json:"hint_skills"`
						HintOthers []struct {
							HintType  int `json:"hint_type"`
							HintValue int `json:"hint_value"`
						} `json:"hint_others"`
					} `json:"hints"`
					EventSkills []struct {
						ID     int      `json:"id"`
						Type   []string `json:"type"`
						NameEn string   `json:"name_en"`
						Rarity int      `json:"rarity"`
						IconID int      `json:"iconid"`
					} `json:"event_skills"`
					Unique *struct {
						Level   int `json:"level"`
						Effects []struct {
							Type   int `json:"type"`
							Value  int `json:"value"`
							Value1 int `json:"value_1,omitempty"`
							Value2 int `json:"value_2,omitempty"`
							Value3 int `json:"value_3,omitempty"`
							Value4 int `json:"value_4,omitempty"`
						} `json:"effects"`
					} `json:"unique,omitempty"`
				}
				Score  int
				Reason string
			}{
				Support: struct {
					URLName     string  `json:"url_name"`
					SupportID   int     `json:"support_id"`
					CharID      int     `json:"char_id"`
					CharName    string  `json:"char_name"`
					NameJp      string  `json:"name_jp"`
					NameKo      string  `json:"name_ko"`
					NameTw      string  `json:"name_tw"`
					Rarity      int     `json:"rarity"`
					Type        string  `json:"type"`
					Obtained    string  `json:"obtained"`
					Release     string  `json:"release"`
					ReleaseKo   string  `json:"release_ko,omitempty"`
					ReleaseZhTw string  `json:"release_zh_tw,omitempty"`
					ReleaseEn   string  `json:"release_en,omitempty"`
					Effects     [][]int `json:"effects"`
					Hints       struct {
						HintSkills []struct {
							ID     int      `json:"id"`
							Type   []string `json:"type"`
							NameEn string   `json:"name_en"`
							IconID int      `json:"iconid"`
						} `json:"hint_skills"`
						HintOthers []struct {
							HintType  int `json:"hint_type"`
							HintValue int `json:"hint_value"`
						} `json:"hint_others"`
					} `json:"hints"`
					EventSkills []struct {
						ID     int      `json:"id"`
						Type   []string `json:"type"`
						NameEn string   `json:"name_en"`
						Rarity int      `json:"rarity"`
						IconID int      `json:"iconid"`
					} `json:"event_skills"`
					Unique *struct {
						Level   int `json:"level"`
						Effects []struct {
							Type   int `json:"type"`
							Value  int `json:"value"`
							Value1 int `json:"value_1,omitempty"`
							Value2 int `json:"value_2,omitempty"`
							Value3 int `json:"value_3,omitempty"`
							Value4 int `json:"value_4,omitempty"`
						} `json:"effects"`
					} `json:"unique,omitempty"`
				}{
					URLName:     support.URLName,
					SupportID:   support.SupportID,
					CharID:      support.CharID,
					CharName:    support.CharName,
					NameJp:      support.NameJp,
					NameKo:      support.NameKo,
					NameTw:      support.NameTw,
					Rarity:      support.Rarity,
					Type:        support.Type,
					Obtained:    support.Obtained,
					Release:     support.Release,
					ReleaseKo:   support.ReleaseKo,
					ReleaseZhTw: support.ReleaseZhTw,
					ReleaseEn:   support.ReleaseEn,
					Effects:     support.Effects,
					Hints: struct {
						HintSkills []struct {
							ID     int      `json:"id"`
							Type   []string `json:"type"`
							NameEn string   `json:"name_en"`
							IconID int      `json:"iconid"`
						} `json:"hint_skills"`
						HintOthers []struct {
							HintType  int `json:"hint_type"`
							HintValue int `json:"hint_value"`
						} `json:"hint_others"`
					}{
						HintSkills: support.Hints.HintSkills,
						HintOthers: support.Hints.HintOthers,
					},
					EventSkills: support.EventSkills,
					Unique:      support.Unique,
				},
				Score:  score,
				Reason: reason,
			})
		}
	}

	if len(matches) == 0 {
		fmt.Printf("❌ No matches found for '%s'\n", query)
	} else {
		fmt.Printf("✅ Found %d potential matches:\n", len(matches))
		for i, match := range matches {
			fmt.Printf("  %d. %s (%s) - Score: %d (%s)\n",
				i+1, match.Support.NameJp, match.Support.NameJp, match.Score, match.Reason)
		}
	}
}
