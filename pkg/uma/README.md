# Uma Musume API Client

This Go package provides a convenient client for interacting with the [umapyoi.net](https://umapyoi.net) API to fetch Uma Musume character and support card data.

## Features

- **Character Search**: Search characters by English or Japanese name (with partial matching)
- **Support Card Search**: Search support cards by English/Japanese title or Gametora ID
- **Character Images**: Fetch and paginate through character images
- **Caching**: In-memory LRU cache with 5-minute TTL to reduce API calls
- **Error Handling**: Graceful handling of timeouts, API errors, and invalid input
- **Discord Bot Integration**: Ready-to-use with Discord bot commands


## Usage

### Character Search

```go
client := uma.NewClient()
result := client.SearchCharacter("Oguri Cap")

if result.Found {
    fmt.Printf("Found: %s (Rarity: %s)\n", result.Character.NameEn, result.Character.Rarity)
} else {
    fmt.Printf("Character not found: %s\n", result.Query)
}
```

### Support Card Search

```go
client := uma.NewClient()
result := client.SearchSupportCard("daring tact")

if result.Found {
    fmt.Printf("Found support card: %s\n", result.SupportCard.TitleEn)
    fmt.Printf("Rarity: %s\n", result.SupportCard.RarityString)
    fmt.Printf("Type: %s\n", result.SupportCard.Type)
} else {
    fmt.Printf("Support card not found: %s\n", result.Query)
}
```


## Discord Command Integration

### Character Lookup

```
!uma char <character name>
```

Examples:

- `!uma char Oguri Cap`
- `!uma char Special Week`

### Support Card Lookup

```
!uma support <support card name>
```

Examples:

- `!uma support daring tact`
- `!uma support 10001-special-week`


## API Endpoints Used

| Endpoint                            | Description                           |
| ----------------------------------- | ------------------------------------- |
| `GET /api/v1/character/list`        | List all characters                   |
| `GET /api/v1/character/images/{id}` | Retrieve images for a character       |
| `GET /api/v1/support`               | List all support cards                |
| `GET /api/v1/support/{id}`          | Get detailed support card information |


## Caching

- **Mechanism**: In-memory LRU cache
- **TTL**: 5 minutes
- **Thread-safe**: Uses `sync.RWMutex` for concurrency
- **Key Format**: `char_search_<query>` or `support_search_<query>`


## Error Handling

The client handles:

- **Network Timeouts**: Defaults to a 10-second timeout
- **Invalid API Responses**: Non-200 status codes
- **Invalid JSON**: Parsing errors
- **Empty Results**: User-friendly error messages for unknown input


## Data Structures

### `Character`

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

### `CharacterSearchResult`

```go
type CharacterSearchResult struct {
    Found     bool
    Character *Character
    Error     error
    Query     string
}
```


## Roadmap / Future Enhancements

- Detailed support card embed rendering (image, skill, stat bonuses)
- Event data support
- News integration
- Global vs JP content toggle


## Disclaimer

This is a personal project intended for educational or non-commercial purposes. Not affiliated with Cygames, Umamusume, or umapyoi.net. Use responsibly and respect upstream API rate limits.

