package tgutil

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/log"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/pkg/ai"
)

var (
	globalRenameService *ai.RenameService
	renameServiceMutex  sync.RWMutex
	serviceInitialized  bool
)

// InitAIRenameService initializes the global AI rename service
func InitAIRenameService(ctx context.Context, cfg *config.Config) error {
	renameServiceMutex.Lock()
	defer renameServiceMutex.Unlock()

	logger := log.FromContext(ctx).With("component", "ai_rename")

	// If AI is disabled, set a disabled service
	if !cfg.AI.IsEnabled() {
		logger.Info("AI rename service disabled")
		globalRenameService = ai.NewRenameService(nil, logger, false)
		serviceInitialized = true
		return nil
	}

	// Create AI client with timeout
	client := ai.NewClient(cfg.AI.BaseURL, cfg.AI.APIKey, cfg.AI.Model, cfg.AI.GetTimeout())

	// Create rename service
	renameService := ai.NewRenameService(client, logger, true)

	// Set fallback function that uses the original naming logic
	renameService.SetFallback(fallbackFileNaming)

	globalRenameService = renameService
	serviceInitialized = true

	logger.Info("AI rename service initialized",
		"base_url", cfg.AI.BaseURL,
		"model", cfg.AI.Model,
		"timeout", cfg.AI.GetTimeout())

	return nil
}

// GetRenameService returns the global AI rename service
func GetRenameService() *ai.RenameService {
	renameServiceMutex.RLock()
	defer renameServiceMutex.RUnlock()
	return globalRenameService
}

// IsRenameServiceInitialized returns whether the AI rename service has been initialized
func IsRenameServiceInitialized() bool {
	renameServiceMutex.RLock()
	defer renameServiceMutex.RUnlock()
	return serviceInitialized
}

// fallbackFileNaming provides the original file naming logic as fallback
func fallbackFileNaming(originalFilename, messageContent string, isAlbum bool) string {
	// For albums, we don't have original filename, so use message content
	if isAlbum {
		if messageContent != "" {
			// Take first 50 characters of message content and sanitize
			name := messageContent
			if len(name) > 50 {
				name = name[:50]
			}
			return ai.SanitizeFilename(name)
		}
		// Last resort for albums: timestamp-based name
		return "album_" + time.Now().Format("20060102_150405")
	}

	// For regular files, prefer original filename without extension
	if originalFilename != "" {
		// Remove extension from original filename
		if idx := strings.LastIndex(originalFilename, "."); idx > 0 {
			return originalFilename[:idx]
		}
		return originalFilename
	}

	// If no original filename, use message content
	if messageContent != "" {
		// Take first 50 characters of message content and sanitize
		name := messageContent
		if len(name) > 50 {
			name = name[:50]
		}
		return ai.SanitizeFilename(name)
	}

	// Last resort: timestamp-based name
	return "file_" + time.Now().Format("20060102_150405")
}
