# Uma Musume Integration Package

This package provides integration with the umapyoi.net API to fetch Uma Musume character information.

## Features

- **Character Search**: Search for characters by name with partial matching
- **Caching**: In-memory LRU cache with 5-minute TTL to reduce API calls
- **Error Handling**: Graceful handling of API errors and missing data
- **Rate Limiting**: Respectful API usage with built-in timeouts

## Usage

### Basic Character Search

```go
client := uma.NewClient()
result := client.SearchCharacter("Oguri Cap")

if result.Found {
    fmt.Printf("Found: %s (Rarity: %s)\n", result.Character.Name, result.Character.Rarity)
} else {
    fmt.Printf("Character not found: %s\n", result.Query)
}
```

### Command Integration

The package is integrated into the Discord bot with the command:

```
!uma char <character name>
```

Examples:
- `!uma char Oguri Cap`
- `!uma char oguri`
- `!uma char Special Week`

## API Endpoints

- **Base URL**: `https://umapyoi.net/api`
- **Characters**: `GET /v1/character/list`

## Data Structures

### Character
```go
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
```

### CharacterSearchResult
```go
type CharacterSearchResult struct {
    Found     bool
    Character *Character
    Error     error
    Query     string
}
```

## Caching

The client implements an in-memory cache with:
- **TTL**: 5 minutes
- **Key Format**: `char_search_<lowercase_query>`
- **Thread-safe**: Uses RWMutex for concurrent access

## Error Handling

The package handles various error scenarios:
- Network timeouts (10-second timeout)
- API errors (non-200 status codes)
- JSON parsing errors
- Missing or empty data

## Future Enhancements

- Support for support cards
- Event information
- News integration
- Global vs JP content filtering 