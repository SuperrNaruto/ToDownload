package tgutil

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/log"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/pkg/ai"
	"github.com/krau/SaveAny-Bot/pkg/tfile"
)

// GenerateAlbumFilenames generates uniform filenames for album files using AI when available
func GenerateAlbumFilenames(ctx context.Context, files []tfile.TGFileMessage) ([]string, error) {
	if len(files) == 0 {
		return nil, fmt.Errorf("no files provided")
	}

	logger := log.FromContext(ctx).With("component", "album_rename")

	// Get the message content from the first file (all files in an album should have the same message)
	firstMessage := files[0].Message()
	messageContent := firstMessage.GetMessage()

	// Initialize base filename
	var baseFilename string

	// Try AI renaming if service is available and initialized
	if IsRenameServiceInitialized() {
		renameService := GetRenameService()
		if renameService != nil && renameService.IsEnabled() {
			// Create a timeout context for AI call
			ctxWithTimeout, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()

			// Call AI rename service for album
			aiBaseName, err := renameService.RenameAlbum(ctxWithTimeout, messageContent)
			if err == nil && aiBaseName != "" {
				// 额外的安全验证
				if strings.ContainsAny(aiBaseName, "/\\:*?\"<>|") {
					logger.Warnf("AI generated album name contains unsafe characters: %s, sanitizing", aiBaseName)
					aiBaseName = ai.SanitizeFilename(aiBaseName)
				}
				baseFilename = aiBaseName
				logger.Info("AI album rename successful", "base_name", baseFilename, "file_count", len(files))
			} else {
				logger.Warn("AI album rename failed, using fallback", "error", err)
			}
		}
	}

	// Fallback to message-based naming if AI is not available or fails
	if baseFilename == "" {
		if messageContent != "" {
			// Use first 50 characters of message content and sanitize
			name := messageContent
			if len(name) > 50 {
				name = name[:50]
			}
			baseFilename = fallbackFileNaming("", messageContent, true)
		} else {
			// Last resort: timestamp-based name
			baseFilename = "album_" + time.Now().Format("20060102_150405")
		}
	}

	// Generate individual filenames with sequence numbers
	filenames := make([]string, len(files))

	// Calculate padding for sequence numbers
	padding := len(fmt.Sprintf("%d", len(files)))
	if padding < 2 {
		padding = 2
	}

	for i, file := range files {
		ext := filepath.Ext(file.Name())
		sequence := fmt.Sprintf("%0*d", padding, i+1)
		filenames[i] = fmt.Sprintf("%s_%s%s", baseFilename, sequence, ext)
	}

	return filenames, nil
}

// GetAlbumBaseDirectory generates a directory name for album files
func GetAlbumBaseDirectory(ctx context.Context, files []tfile.TGFileMessage) (string, error) {
	if len(files) == 0 {
		return "", fmt.Errorf("no files provided")
	}

	// Get the message content from the first file
	firstMessage := files[0].Message()
	messageContent := firstMessage.GetMessage()

	// Try AI renaming if service is available and initialized
	if IsRenameServiceInitialized() {
		renameService := GetRenameService()
		if renameService != nil && renameService.IsEnabled() {
			// Create a timeout context for AI call
			ctxWithTimeout, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()

			// Call AI rename service for album directory
			aiBaseName, err := renameService.RenameAlbum(ctxWithTimeout, messageContent)
			if err == nil && aiBaseName != "" {
				// 额外的安全验证
				if strings.ContainsAny(aiBaseName, "/\\:*?\"<>|") {
					aiBaseName = ai.SanitizeFilename(aiBaseName)
				}
				return aiBaseName, nil
			}
		}
	}

	// Fallback to message-based naming
	return fallbackFileNaming("", messageContent, true), nil
}

// isAlbumMessage checks if a message is part of an album/grouped media
func isAlbumMessage(message *tg.Message) bool {
	groupID, isGroup := message.GetGroupedID()
	return isGroup && groupID != 0
}
