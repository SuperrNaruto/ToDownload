package config

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

// AIConfig represents the AI service configuration
type AIConfig struct {
	Enable     bool   `toml:"enable" mapstructure:"enable" json:"enable"`
	BaseURL    string `toml:"base_url" mapstructure:"base_url" json:"base_url"`
	APIKey     string `toml:"api_key" mapstructure:"api_key" json:"api_key"`
	Model      string `toml:"model" mapstructure:"model" json:"model"`
	Timeout    int    `toml:"timeout" mapstructure:"timeout" json:"timeout"` // timeout in seconds
	MaxRetries int    `toml:"max_retries" mapstructure:"max_retries" json:"max_retries"`
}

// GetTimeout returns the timeout as time.Duration
func (a *AIConfig) GetTimeout() time.Duration {
	if a.Timeout <= 0 {
		return 30 * time.Second // default timeout
	}
	return time.Duration(a.Timeout) * time.Second
}

// GetMaxRetries returns the maximum number of retries, with a sensible default
func (a *AIConfig) GetMaxRetries() int {
	if a.MaxRetries <= 0 {
		return 3 // default max retries
	}
	return a.MaxRetries
}

// Validate validates the AI configuration
func (a *AIConfig) Validate() error {
	if !a.Enable {
		return nil // No validation needed if disabled
	}

	// Validate base URL
	if strings.TrimSpace(a.BaseURL) == "" {
		return fmt.Errorf("ai.base_url is required when AI is enabled")
	}

	baseURL := strings.TrimSpace(a.BaseURL)
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return fmt.Errorf("ai.base_url is not a valid URL: %w", err)
	}

	// Check if URL has scheme (http/https)
	if parsedURL.Scheme == "" || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		return fmt.Errorf("ai.base_url must be a valid HTTP/HTTPS URL")
	}

	// Validate API key
	if strings.TrimSpace(a.APIKey) == "" {
		return fmt.Errorf("ai.api_key is required when AI is enabled")
	}

	// Validate model
	if strings.TrimSpace(a.Model) == "" {
		return fmt.Errorf("ai.model is required when AI is enabled")
	}

	// Validate timeout
	if a.Timeout < 0 {
		return fmt.Errorf("ai.timeout must be non-negative")
	}

	// Validate max retries
	if a.MaxRetries < 0 {
		return fmt.Errorf("ai.max_retries must be non-negative")
	}

	return nil
}

// SetDefaults sets default values for AI configuration
func (a *AIConfig) SetDefaults() {
	if a.BaseURL == "" {
		a.BaseURL = "https://api.openai.com/v1"
	}
	if a.Model == "" {
		a.Model = "gpt-3.5-turbo"
	}
	if a.Timeout <= 0 {
		a.Timeout = 30
	}
	if a.MaxRetries <= 0 {
		a.MaxRetries = 3
	}
}

// IsEnabled returns true if AI functionality is enabled and properly configured
func (a *AIConfig) IsEnabled() bool {
	return a.Enable && strings.TrimSpace(a.BaseURL) != "" && strings.TrimSpace(a.APIKey) != ""
}
