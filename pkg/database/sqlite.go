package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/latoulicious/HKTM/pkg/uma"
	_ "github.com/mattn/go-sqlite3"
)

// Database represents the SQLite database for caching UMA data
type Database struct {
	db *sql.DB
}

// CacheEntry represents a cached item in the database
type DBCacheEntry struct {
	ID        int64
	Key       string
	Data      string
	CreatedAt time.Time
	ExpiresAt time.Time
	Type      string
}

// NewDatabase creates a new database instance
func NewDatabase(dbPath string) (*Database, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}

	// Initialize the database with required tables
	if err := initDatabase(db); err != nil {
		return nil, fmt.Errorf("failed to initialize database: %v", err)
	}

	return &Database{db: db}, nil
}

// initDatabase creates the necessary tables
func initDatabase(db *sql.DB) error {
	// Create cache table
	createCacheTable := `
	CREATE TABLE IF NOT EXISTS uma_cache (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		cache_key TEXT UNIQUE NOT NULL,
		data TEXT NOT NULL,
		type TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		expires_at DATETIME NOT NULL
	);
	`

	// Create character search cache table
	createCharacterSearchTable := `
	CREATE TABLE IF NOT EXISTS character_search_cache (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		query TEXT UNIQUE NOT NULL,
		character_id INTEGER NOT NULL,
		character_data TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		expires_at DATETIME NOT NULL
	);
	`

	// Create character images cache table
	createCharacterImagesTable := `
	CREATE TABLE IF NOT EXISTS character_images_cache (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		character_id INTEGER UNIQUE NOT NULL,
		images_data TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		expires_at DATETIME NOT NULL
	);
	`

	// Create support card search cache table
	createSupportCardSearchTable := `
	CREATE TABLE IF NOT EXISTS support_card_search_cache (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		query TEXT UNIQUE NOT NULL,
		support_cards_data TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		expires_at DATETIME NOT NULL
	);
	`

	// Create support card list cache table
	createSupportCardListTable := `
	CREATE TABLE IF NOT EXISTS support_card_list_cache (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		list_data TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		expires_at DATETIME NOT NULL
	);
	`

	// Create Gametora skills cache table
	createGametoraSkillsTable := `
	CREATE TABLE IF NOT EXISTS gametora_skills_cache (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		query TEXT UNIQUE NOT NULL,
		skills_data TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		expires_at DATETIME NOT NULL
	);
	`

	// Create indexes for better performance
	createIndexes := `
	CREATE INDEX IF NOT EXISTS idx_uma_cache_key ON uma_cache(cache_key);
	CREATE INDEX IF NOT EXISTS idx_uma_cache_expires ON uma_cache(expires_at);
	CREATE INDEX IF NOT EXISTS idx_character_search_query ON character_search_cache(query);
	CREATE INDEX IF NOT EXISTS idx_character_search_expires ON character_search_cache(expires_at);
	CREATE INDEX IF NOT EXISTS idx_character_images_id ON character_images_cache(character_id);
	CREATE INDEX IF NOT EXISTS idx_character_images_expires ON character_images_cache(expires_at);
	CREATE INDEX IF NOT EXISTS idx_support_card_search_query ON support_card_search_cache(query);
	CREATE INDEX IF NOT EXISTS idx_support_card_search_expires ON support_card_search_cache(expires_at);
	CREATE INDEX IF NOT EXISTS idx_support_card_list_expires ON support_card_list_cache(expires_at);
	CREATE INDEX IF NOT EXISTS idx_gametora_skills_query ON gametora_skills_cache(query);
	CREATE INDEX IF NOT EXISTS idx_gametora_skills_expires ON gametora_skills_cache(expires_at);
	`

	queries := []string{
		createCacheTable,
		createCharacterSearchTable,
		createCharacterImagesTable,
		createSupportCardSearchTable,
		createSupportCardListTable,
		createGametoraSkillsTable,
		createIndexes,
	}

	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			return fmt.Errorf("failed to execute query: %v", err)
		}
	}

	return nil
}

// Close closes the database connection
func (d *Database) Close() error {
	return d.db.Close()
}

// CleanExpiredCache removes expired cache entries
func (d *Database) CleanExpiredCache() error {
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
		if _, err := d.db.Exec(query, now); err != nil {
			return fmt.Errorf("failed to clean expired cache: %v", err)
		}
	}

	return nil
}

// CacheCharacterSearch caches a character search result
func (d *Database) CacheCharacterSearch(query string, result *uma.CharacterSearchResult, ttl time.Duration) error {
	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal character search result: %v", err)
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

	_, err = d.db.Exec(sqlQuery, query, characterID, string(data), expiresAt)
	return err
}

// GetCachedCharacterSearch retrieves a cached character search result
func (d *Database) GetCachedCharacterSearch(query string) (*uma.CharacterSearchResult, error) {
	sqlQuery := `
	SELECT character_data FROM character_search_cache 
	WHERE query = ? AND expires_at > ?
	`

	var data string
	err := d.db.QueryRow(sqlQuery, query, time.Now()).Scan(&data)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No cache found
		}
		return nil, fmt.Errorf("failed to get cached character search: %v", err)
	}

	var result uma.CharacterSearchResult
	if err := json.Unmarshal([]byte(data), &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cached character search: %v", err)
	}

	return &result, nil
}

// CacheCharacterImages caches character images
func (d *Database) CacheCharacterImages(characterID int, result *uma.CharacterImagesResult, ttl time.Duration) error {
	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal character images result: %v", err)
	}

	expiresAt := time.Now().Add(ttl)

	query := `
	INSERT OR REPLACE INTO character_images_cache (character_id, images_data, expires_at)
	VALUES (?, ?, ?)
	`

	_, err = d.db.Exec(query, characterID, string(data), expiresAt)
	return err
}

// GetCachedCharacterImages retrieves cached character images
func (d *Database) GetCachedCharacterImages(characterID int) (*uma.CharacterImagesResult, error) {
	query := `
	SELECT images_data FROM character_images_cache 
	WHERE character_id = ? AND expires_at > ?
	`

	var data string
	err := d.db.QueryRow(query, characterID, time.Now()).Scan(&data)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No cache found
		}
		return nil, fmt.Errorf("failed to get cached character images: %v", err)
	}

	var result uma.CharacterImagesResult
	if err := json.Unmarshal([]byte(data), &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cached character images: %v", err)
	}

	return &result, nil
}

// CacheSupportCardSearch caches a support card search result
func (d *Database) CacheSupportCardSearch(query string, result *uma.SupportCardSearchResult, ttl time.Duration) error {
	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal support card search result: %v", err)
	}

	expiresAt := time.Now().Add(ttl)

	sqlQuery := `
	INSERT OR REPLACE INTO support_card_search_cache (query, support_cards_data, expires_at)
	VALUES (?, ?, ?)
	`

	_, err = d.db.Exec(sqlQuery, query, string(data), expiresAt)
	return err
}

// GetCachedSupportCardSearch retrieves a cached support card search result
func (d *Database) GetCachedSupportCardSearch(query string) (*uma.SupportCardSearchResult, error) {
	sqlQuery := `
	SELECT support_cards_data FROM support_card_search_cache 
	WHERE query = ? AND expires_at > ?
	`

	var data string
	err := d.db.QueryRow(sqlQuery, query, time.Now()).Scan(&data)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No cache found
		}
		return nil, fmt.Errorf("failed to get cached support card search: %v", err)
	}

	var result uma.SupportCardSearchResult
	if err := json.Unmarshal([]byte(data), &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cached support card search: %v", err)
	}

	return &result, nil
}

// CacheSupportCardList caches the support card list
func (d *Database) CacheSupportCardList(result *uma.SupportCardListResult, ttl time.Duration) error {
	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal support card list result: %v", err)
	}

	expiresAt := time.Now().Add(ttl)

	query := `
	INSERT OR REPLACE INTO support_card_list_cache (list_data, expires_at)
	VALUES (?, ?)
	`

	_, err = d.db.Exec(query, string(data), expiresAt)
	return err
}

// GetCachedSupportCardList retrieves the cached support card list
func (d *Database) GetCachedSupportCardList() (*uma.SupportCardListResult, error) {
	query := `
	SELECT list_data FROM support_card_list_cache 
	WHERE expires_at > ?
	ORDER BY created_at DESC
	LIMIT 1
	`

	var data string
	err := d.db.QueryRow(query, time.Now()).Scan(&data)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No cache found
		}
		return nil, fmt.Errorf("failed to get cached support card list: %v", err)
	}

	var result uma.SupportCardListResult
	if err := json.Unmarshal([]byte(data), &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cached support card list: %v", err)
	}

	return &result, nil
}

// CacheGametoraSkills caches a Gametora skills search result
func (d *Database) CacheGametoraSkills(query string, result *uma.SimplifiedGametoraSearchResult, ttl time.Duration) error {
	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal Gametora skills result: %v", err)
	}

	expiresAt := time.Now().Add(ttl)

	sqlQuery := `
	INSERT OR REPLACE INTO gametora_skills_cache (query, skills_data, expires_at)
	VALUES (?, ?, ?)
	`

	_, err = d.db.Exec(sqlQuery, query, string(data), expiresAt)
	return err
}

// GetCachedGametoraSkills retrieves a cached Gametora skills search result
func (d *Database) GetCachedGametoraSkills(query string) (*uma.SimplifiedGametoraSearchResult, error) {
	sqlQuery := `
	SELECT skills_data FROM gametora_skills_cache 
	WHERE query = ? AND expires_at > ?
	`

	var data string
	err := d.db.QueryRow(sqlQuery, query, time.Now()).Scan(&data)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No cache found
		}
		return nil, fmt.Errorf("failed to get cached Gametora skills: %v", err)
	}

	var result uma.SimplifiedGametoraSearchResult
	if err := json.Unmarshal([]byte(data), &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cached Gametora skills: %v", err)
	}

	return &result, nil
}

// GetCacheStats returns cache statistics
func (d *Database) GetCacheStats() (map[string]int, error) {
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
		err := d.db.QueryRow(query, now).Scan(&count)
		if err != nil {
			return nil, fmt.Errorf("failed to get cache stats for %s: %v", name, err)
		}
		stats[name] = count
	}

	return stats, nil
}

// StartCacheCleanup starts a background goroutine to clean expired cache entries
func (d *Database) StartCacheCleanup(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			if err := d.CleanExpiredCache(); err != nil {
				log.Printf("Failed to clean expired cache: %v", err)
			}
		}
	}()
}
