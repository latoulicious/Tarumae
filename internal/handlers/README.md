# Slash Commands Implementation

This document explains how to use the new global slash command system for the Tarumae bot.

## Overview

The bot now supports both traditional prefix commands (`!play`, `!queue`, etc.) and modern Discord slash commands. Slash commands provide a better user experience with autocomplete, parameter validation, and native Discord integration.

### Key Features

- **Global Registration**: Commands are registered globally, available in all servers
- **Autocomplete Support**: Built-in autocomplete for command parameters
- **Compatibility**: Slash commands work alongside existing prefix commands
- **Error Handling**: Robust error handling and user feedback
- **Permission System**: Maintains existing permission checks

## Available Slash Commands

### Music Commands
- `/play <url>` - Add a song to the queue and play it
- `/queue add <url>` - Add a song to the queue
- `/queue list` - Show the current queue
- `/queue remove <index>` - Remove a song from the queue
- `/clear` - Clear the entire queue
- `/skip` - Skip the current song
- `/stop` - Stop playback and clear the queue
- `/pause` - Pause the current playback
- `/resume` - Resume paused playback
- `/nowplaying` - Show what's currently playing

### Information Commands
- `/help` - Show help information
- `/about` - Show bot information, uptime, and stats
- `/servers` - Show server information (Bot Owner Only)

### Fun Commands
- `/gremlin` - Post a random gremlin image

## Managing Slash Commands

### Registering Commands

To register all slash commands globally:

```bash
go run debug/slash_manager.go -action register
```

### Deleting Commands

To delete all slash commands:

```bash
go run debug/slash_manager.go -action delete-all
```

To delete a specific command:

```bash
go run debug/slash_manager.go -action delete-specific -command play
```

To check currently registered commands:

```bash
go run debug/slash_manager.go -action check
```

## Implementation Details

### Files Created/Modified

1. **`internal/handlers/slash.go`** - Slash command interaction handler
2. **`internal/commands/slash.go`** - Slash command registration and management
3. **`cmd/main.go`** - Updated to handle slash command interactions
4. **`debug/slash_manager.go`** - Command-line tool for managing slash commands
5. **`Makefile`** - Build system with clean commands

### Technical Implementation

The slash command system:

1. **Registers commands globally** using Discord's Application Commands API via the slash manager tool
2. **Handles interactions** through the `InteractionCreate` event in main.go
3. **Converts slash interactions** to compatible message format for existing command logic
4. **Provides immediate feedback** with deferred responses for long-running operations
5. **Supports autocomplete** for better user experience
6. **Separates concerns** - registration is handled by tools, runtime handling by main.go

## Migration from Guild Commands

If you previously had guild-specific slash commands, you can delete them using the Discord Developer Portal or the provided tools:

1. **Delete old guild commands** using the Discord Developer Portal
2. **Register new global commands** using the slash manager tool
3. **Test the new commands** in your servers

## Troubleshooting

### Commands Not Appearing
- Ensure the bot has the `applications.commands` scope
- Check that the bot has proper permissions in the server
- Verify the bot token is correct

### Permission Issues
- The bot needs `applications.commands` scope for slash commands
- Server members need `Use Application Commands` permission
- Bot owner commands still require the `BOT_OWNER_ID` environment variable

### Command Registration Errors
- Check the bot's permissions and scopes
- Ensure the bot is properly connected to Discord
- Verify the command structure is valid

## Usage Examples

```bash
# Register all slash commands
go run debug/slash_manager.go -action register

# Delete all slash commands
go run debug/slash_manager.go -action delete-all

# Delete specific command
go run debug/slash_manager.go -action delete-specific -command play

# Check currently registered commands
go run debug/slash_manager.go -action check
```

The slash command system provides a modern, user-friendly interface while maintaining compatibility with the existing prefix command system. 