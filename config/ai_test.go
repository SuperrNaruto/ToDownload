package config

import (
	"testing"
	"time"
)

func TestAIConfig_Validate(t *testing.T) {
	tests := []struct {
		name      string
		config    AIConfig
		wantError bool
	}{
		{
			name: "Valid enabled config",
			config: AIConfig{
				Enable:     true,
				BaseURL:    "https://api.openai.com/v1",
				APIKey:     "sk-test123",
				Model:      "gpt-3.5-turbo",
				Timeout:    30,
				MaxRetries: 3,
			},
			wantError: false,
		},
		{
			name: "Disabled config (should pass validation)",
			config: AIConfig{
				Enable: false,
			},
			wantError: false,
		},
		{
			name: "Missing base URL",
			config: AIConfig{
				Enable:  true,
				BaseURL: "",
				APIKey:  "sk-test123",
				Model:   "gpt-3.5-turbo",
			},
			wantError: true,
		},
		{
			name: "Invalid base URL",
			config: AIConfig{
				Enable:  true,
				BaseURL: "not-a-url",
				APIKey:  "sk-test123",
				Model:   "gpt-3.5-turbo",
			},
			wantError: true,
		},
		{
			name: "Missing API key",
			config: AIConfig{
				Enable:  true,
				BaseURL: "https://api.openai.com/v1",
				APIKey:  "",
				Model:   "gpt-3.5-turbo",
			},
			wantError: true,
		},
		{
			name: "Missing model",
			config: AIConfig{
				Enable:  true,
				BaseURL: "https://api.openai.com/v1",
				APIKey:  "sk-test123",
				Model:   "",
			},
			wantError: true,
		},
		{
			name: "Negative timeout",
			config: AIConfig{
				Enable:  true,
				BaseURL: "https://api.openai.com/v1",
				APIKey:  "sk-test123",
				Model:   "gpt-3.5-turbo",
				Timeout: -1,
			},
			wantError: true,
		},
		{
			name: "Negative max retries",
			config: AIConfig{
				Enable:     true,
				BaseURL:    "https://api.openai.com/v1",
				APIKey:     "sk-test123",
				Model:      "gpt-3.5-turbo",
				MaxRetries: -1,
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantError {
				t.Errorf("AIConfig.Validate() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestAIConfig_GetTimeout(t *testing.T) {
	tests := []struct {
		name     string
		timeout  int
		expected time.Duration
	}{
		{
			name:     "Positive timeout",
			timeout:  60,
			expected: 60 * time.Second,
		},
		{
			name:     "Zero timeout (use default)",
			timeout:  0,
			expected: 30 * time.Second,
		},
		{
			name:     "Negative timeout (use default)",
			timeout:  -10,
			expected: 30 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := AIConfig{Timeout: tt.timeout}
			result := config.GetTimeout()
			if result != tt.expected {
				t.Errorf("AIConfig.GetTimeout() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestAIConfig_GetMaxRetries(t *testing.T) {
	tests := []struct {
		name       string
		maxRetries int
		expected   int
	}{
		{
			name:       "Positive max retries",
			maxRetries: 5,
			expected:   5,
		},
		{
			name:       "Zero max retries (use default)",
			maxRetries: 0,
			expected:   3,
		},
		{
			name:       "Negative max retries (use default)",
			maxRetries: -1,
			expected:   3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := AIConfig{MaxRetries: tt.maxRetries}
			result := config.GetMaxRetries()
			if result != tt.expected {
				t.Errorf("AIConfig.GetMaxRetries() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestAIConfig_SetDefaults(t *testing.T) {
	config := AIConfig{}
	config.SetDefaults()

	if config.BaseURL != "https://api.openai.com/v1" {
		t.Errorf("SetDefaults() BaseURL = %s, expected https://api.openai.com/v1", config.BaseURL)
	}

	if config.Model != "gpt-3.5-turbo" {
		t.Errorf("SetDefaults() Model = %s, expected gpt-3.5-turbo", config.Model)
	}

	if config.Timeout != 30 {
		t.Errorf("SetDefaults() Timeout = %d, expected 30", config.Timeout)
	}

	if config.MaxRetries != 3 {
		t.Errorf("SetDefaults() MaxRetries = %d, expected 3", config.MaxRetries)
	}
}

func TestAIConfig_IsEnabled(t *testing.T) {
	tests := []struct {
		name     string
		config   AIConfig
		expected bool
	}{
		{
			name: "Fully configured and enabled",
			config: AIConfig{
				Enable:  true,
				BaseURL: "https://api.openai.com/v1",
				APIKey:  "sk-test123",
			},
			expected: true,
		},
		{
			name: "Disabled",
			config: AIConfig{
				Enable:  false,
				BaseURL: "https://api.openai.com/v1",
				APIKey:  "sk-test123",
			},
			expected: false,
		},
		{
			name: "Enabled but missing BaseURL",
			config: AIConfig{
				Enable:  true,
				BaseURL: "",
				APIKey:  "sk-test123",
			},
			expected: false,
		},
		{
			name: "Enabled but missing API key",
			config: AIConfig{
				Enable:  true,
				BaseURL: "https://api.openai.com/v1",
				APIKey:  "",
			},
			expected: false,
		},
		{
			name: "Enabled but whitespace-only BaseURL",
			config: AIConfig{
				Enable:  true,
				BaseURL: "   ",
				APIKey:  "sk-test123",
			},
			expected: false,
		},
		{
			name: "Enabled but whitespace-only API key",
			config: AIConfig{
				Enable:  true,
				BaseURL: "https://api.openai.com/v1",
				APIKey:  "   ",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.IsEnabled()
			if result != tt.expected {
				t.Errorf("AIConfig.IsEnabled() = %v, expected %v", result, tt.expected)
			}
		})
	}
}