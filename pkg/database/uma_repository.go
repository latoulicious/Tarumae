package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/latoulicious/HKTM/pkg/uma"
)

// umaRepository implements the UMARepository interface
type umaRepository struct {
	db *sql.DB
}

// NewUMARepository creates a new UMA repository
func NewUMARepository(db *sql.DB) (UMARepository, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}

	repo := &umaRepository{
		db: db,
	}

	// Initialize UMA-specific tables
	if err := repo.initializeTables(); err != nil {
		return nil, fmt.Errorf("failed to initialize UMA tables: %w", err)
	}

	return repo, nil
}

// initializeTables creates the UMA cache tables if they don't exist
func (r *umaRepository) initializeTables() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS uma_cache (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			cache_key TEXT UNIQUE NOT NULL,
			data TEXT NOT NULL,
			type TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			expires_at DATETIME NOT NULL
		)`,

		`CREATE TABLE IF NOT EXISTS character_search_cache (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			query TEXT UNIQUE NOT NULL,
			character_id INTEGER NOT NULL,
			character_data TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			expires_at DATETIME NOT NULL
		)`,

		`CREATE TABLE IF NOT EXISTS character_images_cache (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			character_id INTEGER UNIQUE NOT NULL,
			images_data TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			expires_at DATETIME NOT NULL
		)`,

		`CREATE TABLE IF NOT EXISTS support_card_search_cache (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			query TEXT UNIQUE NOT NULL,
			support_cards_data TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			expires_at DATETIME NOT NULL
		)`,

		`CREATE TABLE IF NOT EXISTS support_card_list_cache (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			list_data TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			expires_at DATETIME NOT NULL
		)`,

		`CREATE TABLE IF NOT EXISTS gametora_skills_cache (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			query TEXT UNIQUE NOT NULL,
			skills_data TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			expires_at DATETIME NOT NULL
		)`,

		// Create indexes for better performance
		`CREATE INDEX IF NOT EXISTS idx_uma_cache_key ON uma_cache(cache_key)`,
		`CREATE INDEX IF NOT EXISTS idx_uma_cache_expires ON uma_cache(expires_at)`,
		`CREATE INDEX IF NOT EXISTS idx_character_search_query ON character_search_cache(query)`,
		`CREATE INDEX IF NOT EXISTS idx_character_search_expires ON character_search_cache(expires_at)`,
		`CREATE INDEX IF NOT EXISTS idx_character_images_id ON character_images_cache(character_id)`,
		`CREATE INDEX IF NOT EXISTS idx_character_images_expires ON character_images_cache(expires_at)`,
		`CREATE INDEX IF NOT EXISTS idx_support_card_search_query ON support_card_search_cache(query)`,
		`CREATE INDEX IF NOT EXISTS idx_support_card_search_expires ON support_card_search_cache(expires_at)`,
		`CREATE INDEX IF NOT EXISTS idx_support_card_list_expires ON support_card_list_cache(expires_at)`,
		`CREATE INDEX IF NOT EXISTS idx_gametora_skills_query ON gametora_skills_cache(query)`,
		`CREATE INDEX IF NOT EXISTS idx_gametora_skills_expires ON gametora_skills_cache(expires_at)`,
	}

	for _, query := range queries {
		if _, err := r.db.Exec(query); err != nil {
			return fmt.Errorf("failed to execute query: %v", err)
		}
	}

	return nil
}

// CacheCharacterSearch caches a character search result
func (r *umaRepository) CacheCharacterSearch(query string, result *uma.CharacterSearchResult, ttl time.Duration) error {
	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal character search result: %w", err)
	}

	expiresAt := time.Now().Add(ttl)

	sqlQuery := `
	INSERT OR REPLACE INTO character_search_cache (query, character_id, character_data, expires_at)
	VALUES (?, ?, ?, ?)
	`

	var characterID int
	if result.Found && result.Character != nil {
		characterID = result.Character.ID
	}

	_, err = r.db.Exec(sqlQuery, query, characterID, string(data), expiresAt)
	if err != nil {
		return fmt.Errorf("failed to cache character search: %w", err)
	}

	return nil
}

// GetCachedCharacterSearch retrieves a cached character search result
func (r *umaRepository) GetCachedCharacterSearch(query string) (*uma.CharacterSearchResult, error) {
	sqlQuery := `
	SELECT character_data FROM character_search_cache 
	WHERE query = ? AND expires_at > ?
	`

	var data string
	err := r.db.QueryRow(sqlQuery, query, time.Now()).Scan(&data)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No cache found
		}
		return nil, fmt.Errorf("failed to get cached character search: %w", err)
	}

	var result uma.CharacterSearchResult
	if err := json.Unmarshal([]byte(data), &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cached character search: %w", err)
	}

	return &result, nil
}

// CacheCharacterImages caches character images
func (r *umaRepository) CacheCharacterImages(characterID int, result *uma.CharacterImagesResult, ttl time.Duration) error {
	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal character images result: %w", err)
	}

	expiresAt := time.Now().Add(ttl)

	query := `
	INSERT OR REPLACE INTO character_images_cache (character_id, images_data, expires_at)
	VALUES (?, ?, ?)
	`

	_, err = r.db.Exec(query, characterID, string(data), expiresAt)
	if err != nil {
		return fmt.Errorf("failed to cache character images: %w", err)
	}

	return nil
}

// GetCachedCharacterImages retrieves cached character images
func (r *umaRepository) GetCachedCharacterImages(characterID int) (*uma.CharacterImagesResult, error) {
	query := `
	SELECT images_data FROM character_images_cache 
	WHERE character_id = ? AND expires_at > ?
	`

	var data string
	err := r.db.QueryRow(query, characterID, time.Now()).Scan(&data)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No cache found
		}
		return nil, fmt.Errorf("failed to get cached character images: %w", err)
	}

	var result uma.CharacterImagesResult
	if err := json.Unmarshal([]byte(data), &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cached character images: %w", err)
	}

	return &result, nil
}

// CacheSupportCardSearch caches a support card search result
func (r *umaRepository) CacheSupportCardSearch(query string, result *uma.SupportCardSearchResult, ttl time.Duration) error {
	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal support card search result: %w", err)
	}

	expiresAt := time.Now().Add(ttl)

	sqlQuery := `
	INSERT OR REPLACE INTO support_card_search_cache (query, support_cards_data, expires_at)
	VALUES (?, ?, ?)
	`

	_, err = r.db.Exec(sqlQuery, query, string(data), expiresAt)
	if err != nil {
		return fmt.Errorf("failed to cache support card search: %w", err)
	}

	return nil
}

// GetCachedSupportCardSearch retrieves a cached support card search result
func (r *umaRepository) GetCachedSupportCardSearch(query string) (*uma.SupportCardSearchResult, error) {
	sqlQuery := `
	SELECT support_cards_data FROM support_card_search_cache 
	WHERE query = ? AND expires_at > ?
	`

	var data string
	err := r.db.QueryRow(sqlQuery, query, time.Now()).Scan(&data)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No cache found
		}
		return nil, fmt.Errorf("failed to get cached support card search: %w", err)
	}

	var result uma.SupportCardSearchResult
	if err := json.Unmarshal([]byte(data), &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cached support card search: %w", err)
	}

	return &result, nil
}

// CacheSupportCardList caches the support card list
func (r *umaRepository) CacheSupportCardList(result *uma.SupportCardListResult, ttl time.Duration) error {
	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal support card list result: %w", err)
	}

	expiresAt := time.Now().Add(ttl)

	query := `
	INSERT OR REPLACE INTO support_card_list_cache (list_data, expires_at)
	VALUES (?, ?)
	`

	_, err = r.db.Exec(query, string(data), expiresAt)
	if err != nil {
		return fmt.Errorf("failed to cache support card list: %w", err)
	}

	return nil
}

// GetCachedSupportCardList retrieves the cached support card list
func (r *umaRepository) GetCachedSupportCardList() (*uma.SupportCardListResult, error) {
	query := `
	SELECT list_data FROM support_card_list_cache 
	WHERE expires_at > ?
	ORDER BY created_at DESC
	LIMIT 1
	`

	var data string
	err := r.db.QueryRow(query, time.Now()).Scan(&data)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No cache found
		}
		return nil, fmt.Errorf("failed to get cached support card list: %w", err)
	}

	var result uma.SupportCardListResult
	if err := json.Unmarshal([]byte(data), &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cached support card list: %w", err)
	}

	return &result, nil
}

// CacheGametoraSkills caches a Gametora skills search result
func (r *umaRepository) CacheGametoraSkills(query string, result *uma.SimplifiedGametoraSearchResult, ttl time.Duration) error {
	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal Gametora skills result: %w", err)
	}

	expiresAt := time.Now().Add(ttl)

	sqlQuery := `
	INSERT OR REPLACE INTO gametora_skills_cache (query, skills_data, expires_at)
	VALUES (?, ?, ?)
	`

	_, err = r.db.Exec(sqlQuery, query, string(data), expiresAt)
	if err != nil {
		return fmt.Errorf("failed to cache Gametora skills: %w", err)
	}

	return nil
}

// GetCachedGametoraSkills retrieves a cached Gametora skills search result
func (r *umaRepository) GetCachedGametoraSkills(query string) (*uma.SimplifiedGametoraSearchResult, error) {
	sqlQuery := `
	SELECT skills_data FROM gametora_skills_cache 
	WHERE query = ? AND expires_at > ?
	`

	var data string
	err := r.db.QueryRow(sqlQuery, query, time.Now()).Scan(&data)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No cache found
		}
		return nil, fmt.Errorf("failed to get cached Gametora skills: %w", err)
	}

	var result uma.SimplifiedGametoraSearchResult
	if err := json.Unmarshal([]byte(data), &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cached Gametora skills: %w", err)
	}

	return &result, nil
}

// CleanExpiredCache removes expired cache entries
func (r *umaRepository) CleanExpiredCache() error {
	now := time.Now()

	queries := []string{
		"DELETE FROM uma_cache WHERE expires_at < ?",
		"DELETE FROM character_search_cache WHERE expires_at < ?",
		"DELETE FROM character_images_cache WHERE expires_at < ?",
		"DELETE FROM support_card_search_cache WHERE expires_at < ?",
		"DELETE FROM support_card_list_cache WHERE expires_at < ?",
		"DELETE FROM gametora_skills_cache WHERE expires_at < ?",
	}

	for _, query := range queries {
		if _, err := r.db.Exec(query, now); err != nil {
			return fmt.Errorf("failed to clean expired cache: %w", err)
		}
	}

	return nil
}

// GetCacheStats returns cache statistics
func (r *umaRepository) GetCacheStats() (map[string]int, error) {
	stats := make(map[string]int)

	queries := map[string]string{
		"character_search":    "SELECT COUNT(*) FROM character_search_cache WHERE expires_at > ?",
		"character_images":    "SELECT COUNT(*) FROM character_images_cache WHERE expires_at > ?",
		"support_card_search": "SELECT COUNT(*) FROM support_card_search_cache WHERE expires_at > ?",
		"support_card_list":   "SELECT COUNT(*) FROM support_card_list_cache WHERE expires_at > ?",
		"gametora_skills":     "SELECT COUNT(*) FROM gametora_skills_cache WHERE expires_at > ?",
		"total_cache":         "SELECT COUNT(*) FROM uma_cache WHERE expires_at > ?",
	}

	now := time.Now()
	for name, query := range queries {
		var count int
		err := r.db.QueryRow(query, now).Scan(&count)
		if err != nil {
			return nil, fmt.Errorf("failed to get cache stats for %s: %w", name, err)
		}
		stats[name] = count
	}

	return stats, nil
}
