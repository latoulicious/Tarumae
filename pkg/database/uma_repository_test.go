package database

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/latoulicious/HKTM/pkg/uma"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestUMARepository(t *testing.T) (UMARepository, func()) {
	// Create temporary database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	db, err := sql.Open("sqlite3", dbPath)
	require.NoError(t, err)

	repo, err := NewUMARepository(db)
	require.NoError(t, err)

	cleanup := func() {
		db.Close()
	}

	return repo, cleanup
}

func TestUMARepository_CharacterSearch(t *testing.T) {
	repo, cleanup := setupTestUMARepository(t)
	defer cleanup()

	// Test data
	query := "test character"
	result := &uma.CharacterSearchResult{
		Found: true,
		Character: &uma.Character{
			ID:     123,
			NameEn: "Test Character",
		},
	}
	ttl := 1 * time.Hour

	// Test caching
	err := repo.CacheCharacterSearch(query, result, ttl)
	assert.NoError(t, err)

	// Test retrieval
	cached, err := repo.GetCachedCharacterSearch(query)
	assert.NoError(t, err)
	assert.NotNil(t, cached)
	assert.Equal(t, result.Found, cached.Found)
	assert.Equal(t, result.Character.ID, cached.Character.ID)
	assert.Equal(t, result.Character.NameEn, cached.Character.NameEn)

	// Test non-existent query
	notFound, err := repo.GetCachedCharacterSearch("non-existent")
	assert.NoError(t, err)
	assert.Nil(t, notFound)
}

func TestUMARepository_CharacterImages(t *testing.T) {
	repo, cleanup := setupTestUMARepository(t)
	defer cleanup()

	// Test data
	characterID := 123
	result := &uma.CharacterImagesResult{
		CharaID: characterID,
		Images: []uma.CharacterImageCategory{
			{
				Label: "portrait",
				Images: []uma.CharacterImage{
					{
						Image:    "https://example.com/image1.jpg",
						Uploaded: "2023-01-01",
					},
				},
			},
		},
	}
	ttl := 1 * time.Hour

	// Test caching
	err := repo.CacheCharacterImages(characterID, result, ttl)
	assert.NoError(t, err)

	// Test retrieval
	cached, err := repo.GetCachedCharacterImages(characterID)
	assert.NoError(t, err)
	assert.NotNil(t, cached)
	assert.Equal(t, result.CharaID, cached.CharaID)
	assert.Len(t, cached.Images, 1)
	assert.Equal(t, result.Images[0].Images[0].Image, cached.Images[0].Images[0].Image)

	// Test non-existent character
	notFound, err := repo.GetCachedCharacterImages(999)
	assert.NoError(t, err)
	assert.Nil(t, notFound)
}

func TestUMARepository_SupportCardSearch(t *testing.T) {
	repo, cleanup := setupTestUMARepository(t)
	defer cleanup()

	// Test data
	query := "test support card"
	result := &uma.SupportCardSearchResult{
		Found: true,
		SupportCards: []uma.SupportCard{
			{
				ID:      456,
				TitleEn: "Test Support Card",
			},
		},
	}
	ttl := 1 * time.Hour

	// Test caching
	err := repo.CacheSupportCardSearch(query, result, ttl)
	assert.NoError(t, err)

	// Test retrieval
	cached, err := repo.GetCachedSupportCardSearch(query)
	assert.NoError(t, err)
	assert.NotNil(t, cached)
	assert.Equal(t, result.Found, cached.Found)
	assert.Len(t, cached.SupportCards, 1)
	assert.Equal(t, result.SupportCards[0].ID, cached.SupportCards[0].ID)
	assert.Equal(t, result.SupportCards[0].TitleEn, cached.SupportCards[0].TitleEn)

	// Test non-existent query
	notFound, err := repo.GetCachedSupportCardSearch("non-existent")
	assert.NoError(t, err)
	assert.Nil(t, notFound)
}

func TestUMARepository_SupportCardList(t *testing.T) {
	repo, cleanup := setupTestUMARepository(t)
	defer cleanup()

	// Test data
	result := &uma.SupportCardListResult{
		SupportCards: []uma.SupportCard{
			{
				ID:      789,
				TitleEn: "Test Support Card List",
			},
		},
	}
	ttl := 1 * time.Hour

	// Test caching
	err := repo.CacheSupportCardList(result, ttl)
	assert.NoError(t, err)

	// Test retrieval
	cached, err := repo.GetCachedSupportCardList()
	assert.NoError(t, err)
	assert.NotNil(t, cached)
	assert.Len(t, cached.SupportCards, 1)
	assert.Equal(t, result.SupportCards[0].ID, cached.SupportCards[0].ID)
	assert.Equal(t, result.SupportCards[0].TitleEn, cached.SupportCards[0].TitleEn)
}

func TestUMARepository_GametoraSkills(t *testing.T) {
	repo, cleanup := setupTestUMARepository(t)
	defer cleanup()

	// Test data
	query := "test skill"
	result := &uma.SimplifiedGametoraSearchResult{
		Found: true,
		SupportCards: []*uma.SimplifiedSupportCard{
			{
				SupportID: 101,
				CharName:  "Test Skill",
			},
		},
	}
	ttl := 1 * time.Hour

	// Test caching
	err := repo.CacheGametoraSkills(query, result, ttl)
	assert.NoError(t, err)

	// Test retrieval
	cached, err := repo.GetCachedGametoraSkills(query)
	assert.NoError(t, err)
	assert.NotNil(t, cached)
	assert.Equal(t, result.Found, cached.Found)
	assert.Len(t, cached.SupportCards, 1)
	assert.Equal(t, result.SupportCards[0].SupportID, cached.SupportCards[0].SupportID)
	assert.Equal(t, result.SupportCards[0].CharName, cached.SupportCards[0].CharName)

	// Test non-existent query
	notFound, err := repo.GetCachedGametoraSkills("non-existent")
	assert.NoError(t, err)
	assert.Nil(t, notFound)
}

func TestUMARepository_CleanExpiredCache(t *testing.T) {
	repo, cleanup := setupTestUMARepository(t)
	defer cleanup()

	// Add some test data with short TTL
	query := "test character"
	result := &uma.CharacterSearchResult{
		Found: true,
		Character: &uma.Character{
			ID:     123,
			NameEn: "Test Character",
		},
	}

	// Cache with very short TTL
	err := repo.CacheCharacterSearch(query, result, 1*time.Millisecond)
	assert.NoError(t, err)

	// Wait for expiration
	time.Sleep(10 * time.Millisecond)

	// Clean expired cache
	err = repo.CleanExpiredCache()
	assert.NoError(t, err)

	// Verify data is gone
	cached, err := repo.GetCachedCharacterSearch(query)
	assert.NoError(t, err)
	assert.Nil(t, cached)
}

func TestUMARepository_GetCacheStats(t *testing.T) {
	repo, cleanup := setupTestUMARepository(t)
	defer cleanup()

	// Add some test data
	query := "test character"
	result := &uma.CharacterSearchResult{
		Found: true,
		Character: &uma.Character{
			ID:     123,
			NameEn: "Test Character",
		},
	}
	ttl := 1 * time.Hour

	err := repo.CacheCharacterSearch(query, result, ttl)
	assert.NoError(t, err)

	// Get stats
	stats, err := repo.GetCacheStats()
	assert.NoError(t, err)
	assert.NotNil(t, stats)

	// Verify stats contain expected data
	assert.Contains(t, stats, "character_search")
	assert.Contains(t, stats, "character_images")
	assert.Contains(t, stats, "support_card_search")
	assert.Contains(t, stats, "support_card_list")
	assert.Contains(t, stats, "gametora_skills")
	assert.Contains(t, stats, "total_cache")

	// At least one character search should be cached
	assert.GreaterOrEqual(t, stats["character_search"], 1)
}

func TestNewUMARepository_NilDB(t *testing.T) {
	repo, err := NewUMARepository(nil)
	assert.Error(t, err)
	assert.Nil(t, repo)
	assert.Contains(t, err.Error(), "database connection is nil")
}
