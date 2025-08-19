# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

SaveAny-Bot is a Telegram bot written in Go that saves files from Telegram to various storage endpoints. The bot supports multiple storage types (local, Alist, WebDAV, MinIO/S3, Telegram), multi-user access control, rule-based file organization, and streaming downloads.

## Development Commands

### Building and Running
```bash
# Build the binary
go build -o saveany-bot .

# Run directly with Go
go run main.go

# Run with configuration (requires config.toml in working directory)
./saveany-bot

# Build and run with Docker
docker build -t saveany-bot .
docker run -v ./config.toml:/app/config.toml saveany-bot
```

### Testing
```bash
# Run specific tests
go test ./pkg/queue
go test ./storage/webdav

# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...
```

### Development Setup
```bash
# Install dependencies
go mod download

# Tidy dependencies
go mod tidy

# Generate i18n files (requires geni18n tool)
go run cmd/geni18n/main.go
```

## Architecture Overview

### Core Components

1. **Command Layer** (`cmd/`): Cobra-based CLI with main entry point and run logic
2. **Core Engine** (`core/`): Task queue management and worker pool execution
3. **Client Layer** (`client/`): 
   - Bot client (`client/bot/`) - Telegram bot handling
   - User client (`client/user/`) - User authentication and management
4. **Storage Abstraction** (`storage/`): Pluggable storage backends with unified interface
5. **Configuration** (`config/`): TOML-based configuration with storage-specific settings
6. **Database** (`database/`): SQLite-based persistence for users, chats, rules, and directories

### Key Design Patterns

- **Interface-based Storage**: All storage backends implement the `Storage` interface for consistent handling
- **Worker Pool**: Configurable number of workers process download tasks concurrently
- **Rule System**: Generic rule matching system for automatic file organization
- **Task Queue**: Priority-based task execution with cancellation support
- **Middleware Chain**: Request processing pipeline for bot handlers

### Important Interfaces

- `Storage` interface (`storage/storage.go`): Defines storage backend contract
- `Exectable` interface (`core/core.go`): Defines task execution contract
- `RuleClass` interface (`pkg/rule/rule.go`): Defines rule matching contract

### Task Types and Execution

The system supports three main task types:
- **TfTask**: Standard file downloads with progress tracking
- **BatchTfTask**: Batch processing of multiple files
- **TphTask**: Telegraph content processing

Tasks are queued through `core.AddTask()` and executed by worker goroutines with configurable concurrency.

### Configuration Structure

Configuration uses TOML format with these main sections:
- `[telegram]`: Bot token, API credentials, proxy settings
- `[[storages]]`: Array of storage endpoint configurations
- `[[users]]`: Array of user access control definitions
- Global settings: workers, retry limits, stream mode

### Storage Implementation

Each storage type has its own package under `storage/`:
- `local/`: Local filesystem storage
- `alist/`: Alist API integration
- `webdav/`: WebDAV protocol support
- `minio/`: S3-compatible object storage
- `telegram/`: Telegram file upload backend

Storage instances are created via factory pattern in `storage/load.go`.

### Bot Handler Structure

Bot handlers are organized in `client/bot/handlers/`:
- Command handlers for each bot command (`/start`, `/save`, `/dir`, etc.)
- Middleware for authentication, flood control, and recovery
- Message utilities and callback handling

## Important Development Notes

### Database Schema
The application uses GORM with SQLite. Main models are in `database/model.go`:
- User management with storage permissions
- Chat tracking and directory management  
- Rule-based file organization
- Task progress tracking

### Internationalization
Uses `go-i18n` for multi-language support:
- Keys defined in `common/i18n/i18nk/keys.go`
- Translations in `common/i18n/locale/`
- Generate translations with `go run cmd/geni18n/main.go`

### Error Handling
- Uses `go-faster/errors` for error wrapping
- Structured logging with `charmbracelet/log`
- Context-based cancellation throughout

### Stream Mode
When enabled, files stream directly to storage without local caching:
- Reduces disk usage but increases failure rates
- Not supported by all storage backends
- Disables multi-threaded downloads

### Testing Approach
- Unit tests for queue management (`pkg/queue/queue_test.go`)
- Integration tests for WebDAV client (`storage/webdav/client_test.go`)
- Tests use standard Go testing framework

### AI-Powered File Renaming
The bot includes AI-powered intelligent file renaming functionality:
- **Configuration**: AI section in `config.toml` with OpenAI-compatible API settings
- **Implementation**: Located in `pkg/ai/` with client, prompt, and rename logic
- **Integration**: Automatic renaming during file processing via `tgutil.ai_rename.go`
- **Bot Commands**: `/ai_status`, `/ai_test`, `/ai_help` for management and testing
- **Naming Patterns**: 
  - Normal files: `名称.作者.时间.要点` format
  - Album files: `统一名称_序号` format
- **Fallback**: Automatic fallback to original naming if AI service fails

## Configuration Example

Copy `config.example.toml` to `config.toml` and configure:
- Telegram bot token from BotFather
- Storage endpoint credentials
- User access permissions
- Worker and retry settings
- AI service settings (optional, for intelligent file renaming)

## Task Master AI Instructions
**Import Task Master's development workflow commands and guidelines, treat as if import is in the main CLAUDE.md file.**
@./.taskmaster/CLAUDE.md
