package ai

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/log"
)

// RenameService provides AI-powered file renaming functionality
type RenameService struct {
	client   *Client
	logger   *log.Logger
	enabled  bool
	fallback func(string, string, bool) string // fallback function when AI fails
}

// NewRenameService creates a new rename service
func NewRenameService(client *Client, logger *log.Logger, enabled bool) *RenameService {
	return &RenameService{
		client:  client,
		logger:  logger,
		enabled: enabled,
	}
}

// SetFallback sets the fallback function to use when AI renaming fails
func (s *RenameService) SetFallback(fallback func(string, string, bool) string) {
	s.fallback = fallback
}

// IsEnabled returns whether the AI rename service is enabled
func (s *RenameService) IsEnabled() bool {
	return s.enabled && s.client != nil
}

// SetEnabled enables or disables the AI rename service
func (s *RenameService) SetEnabled(enabled bool) {
	s.enabled = enabled
}

// RenameFile generates a new filename using AI based on the original filename and message content
func (s *RenameService) RenameFile(ctx context.Context, originalFilename, messageContent string) (string, error) {
	if !s.IsEnabled() {
		return s.useFallback(originalFilename, messageContent, false), nil
	}

	req := RenameRequest{
		OriginalFilename: originalFilename,
		MessageContent:   messageContent,
		IsAlbum:          false,
	}

	return s.renameWithAI(ctx, req)
}

// RenameAlbum generates a base filename for an album using AI based on the message content
func (s *RenameService) RenameAlbum(ctx context.Context, messageContent string) (string, error) {
	if !s.IsEnabled() {
		return s.useFallback("", messageContent, true), nil
	}

	req := RenameRequest{
		OriginalFilename: "",
		MessageContent:   messageContent,
		IsAlbum:          true,
	}

	return s.renameWithAI(ctx, req)
}

// renameWithAI performs the actual AI-powered renaming
func (s *RenameService) renameWithAI(ctx context.Context, req RenameRequest) (string, error) {
	s.logger.Debug("Starting AI rename", "isAlbum", req.IsAlbum, "original", req.OriginalFilename)

	// Build the prompt
	prompt := BuildPrompt(req)

	// Create context with timeout
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Call AI API
	result, err := s.client.GenerateText(ctxWithTimeout, prompt)
	if err != nil {
		s.logger.Error("AI rename failed", "error", err)
		return s.useFallback(req.OriginalFilename, req.MessageContent, req.IsAlbum), fmt.Errorf("AI rename failed: %w", err)
	}

	// Clean and validate the result
	cleanResult := strings.TrimSpace(result)
	cleanResult = SanitizeFilename(cleanResult)

	if !ValidateFilename(cleanResult) {
		s.logger.Warn("AI generated invalid filename", "result", cleanResult)
		return s.useFallback(req.OriginalFilename, req.MessageContent, req.IsAlbum), fmt.Errorf("AI generated invalid filename: %s", cleanResult)
	}

	// 额外的路径安全检查（防止AI生成包含路径分隔符的文件名）
	if strings.ContainsAny(cleanResult, "/\\") {
		s.logger.Warn("AI generated filename contains path separators", "result", cleanResult)
		cleanResult = strings.ReplaceAll(cleanResult, "/", "_")
		cleanResult = strings.ReplaceAll(cleanResult, "\\", "_")
		s.logger.Info("Cleaned path separators from AI result", "cleaned", cleanResult)
	}

	// 最终安全验证
	if cleanResult == "" || cleanResult == "." || cleanResult == ".." {
		s.logger.Warn("AI generated unsafe filename", "result", cleanResult)
		return s.useFallback(req.OriginalFilename, req.MessageContent, req.IsAlbum), fmt.Errorf("AI generated unsafe filename: %s", cleanResult)
	}

	s.logger.Info("AI rename successful", "original", req.OriginalFilename, "result", cleanResult, "isAlbum", req.IsAlbum)
	return cleanResult, nil
}

// useFallback uses the fallback function when AI is disabled or fails
func (s *RenameService) useFallback(originalFilename, messageContent string, isAlbum bool) string {
	if s.fallback != nil {
		return s.fallback(originalFilename, messageContent, isAlbum)
	}

	// Default fallback: return original filename without extension or use message-based name
	if originalFilename != "" {
		// Remove extension from original filename
		if idx := strings.LastIndex(originalFilename, "."); idx > 0 {
			return originalFilename[:idx]
		}
		return originalFilename
	}

	// If no original filename, generate a simple name from message
	if messageContent != "" {
		// Take first 50 characters of message content and sanitize
		name := messageContent
		if len(name) > 50 {
			name = name[:50]
		}
		return SanitizeFilename(name)
	}

	// Last resort: timestamp-based name
	return fmt.Sprintf("file_%d", time.Now().Unix())
}

// GenerateAlbumFilenames generates individual filenames for album files
func (s *RenameService) GenerateAlbumFilenames(baseFilename string, count int) []string {
	filenames := make([]string, count)

	// Calculate padding for sequence numbers
	padding := len(fmt.Sprintf("%d", count))
	if padding < 2 {
		padding = 2
	}

	for i := 0; i < count; i++ {
		sequence := fmt.Sprintf("%0*d", padding, i+1)
		filenames[i] = fmt.Sprintf("%s_%s", baseFilename, sequence)
	}

	return filenames
}

// BatchRenameFiles renames multiple files with the same message content (for optimization)
func (s *RenameService) BatchRenameFiles(ctx context.Context, requests []RenameRequest) ([]string, error) {
	results := make([]string, len(requests))
	var err error

	for i, req := range requests {
		if req.IsAlbum {
			results[i], err = s.RenameAlbum(ctx, req.MessageContent)
		} else {
			results[i], err = s.RenameFile(ctx, req.OriginalFilename, req.MessageContent)
		}

		if err != nil {
			s.logger.Error("Batch rename failed for item", "index", i, "error", err)
			// Continue with other files even if one fails
		}
	}

	return results, nil
}
