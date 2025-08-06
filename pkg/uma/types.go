package uma

import "time"

// Character represents a Uma Musume character
type Character struct {
	ID              int    `json:"id"`
	NameEn          string `json:"name_en"`
	NameJp          string `json:"name_jp"`
	NameEnInternal  string `json:"name_en_internal"`
	CategoryLabel   string `json:"category_label"`
	CategoryLabelEn string `json:"category_label_en"`
	CategoryValue   string `json:"category_value"`
	ColorMain       string `json:"color_main"`
	ColorSub        string `json:"color_sub"`
	PreferredURL    string `json:"preferred_url"`
	RowNumber       int    `json:"row_number"`
	ThumbImg        string `json:"thumb_img"`
}

// CharacterSearchResult represents the result of a character search
type CharacterSearchResult struct {
	Found     bool
	Character *Character
	Error     error
	Query     string
}

// APIResponse represents the response from umapyoi.net API
type APIResponse []Character

// CacheEntry represents a cached API response
type CacheEntry struct {
	Data      interface{}
	Timestamp time.Time
	TTL       time.Duration
}

// IsExpired checks if the cache entry has expired
func (ce *CacheEntry) IsExpired() bool {
	return time.Since(ce.Timestamp) > ce.TTL
}

// CharacterImage represents a character image from the API
type CharacterImage struct {
	Image    string `json:"image"`
	Uploaded string `json:"uploaded"`
}

// CharacterImageCategory represents a category of character images
type CharacterImageCategory struct {
	Images  []CharacterImage `json:"images"`
	Label   string           `json:"label"`
	LabelEn string           `json:"label_en"`
}

// CharacterImagesResult represents the result of fetching character images
type CharacterImagesResult struct {
	Found   bool
	Images  []CharacterImageCategory
	Error   error
	CharaID int
}
