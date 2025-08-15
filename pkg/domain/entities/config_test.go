package entities

import (
	"strings"
	"testing"
)

func TestNewConfig(t *testing.T) {
	tests := []struct {
		name         string
		apiKey       string
		defaultModel string
		expectError  bool
		errorMessage string
	}{
		{
			name:         "valid config with both fields",
			apiKey:       "sk-or-v1-1234567890abcdef1234567890abcdef",
			defaultModel: "gpt-3.5-turbo",
			expectError:  false,
		},
		{
			name:         "valid config with only api key",
			apiKey:       "sk-or-v1-1234567890abcdef1234567890abcdef",
			defaultModel: "",
			expectError:  false,
		},
		{
			name:         "empty api key",
			apiKey:       "",
			defaultModel: "gpt-3.5-turbo",
			expectError:  false,
			errorMessage: "API key cannot be empty",
		},
		{
			name:         "whitespace only api key",
			apiKey:       "   ",
			defaultModel: "gpt-3.5-turbo",
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := NewConfig(tt.apiKey, tt.defaultModel)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if err.Error() != tt.errorMessage {
					t.Errorf("expected error message %q, got %q", tt.errorMessage, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			expectedAPIKey := strings.TrimSpace(tt.apiKey)
			if config.APIKey != expectedAPIKey {
				t.Errorf("expected APIKey %q, got %q", expectedAPIKey, config.APIKey)
			}

			if config.DefaultModel != tt.defaultModel {
				t.Errorf("expected DefaultModel %q, got %q", tt.defaultModel, config.DefaultModel)
			}
		})
	}
}

func TestConfig_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		config   *Config
		expected bool
	}{
		{
			name: "valid config",
			config: &Config{
				APIKey:       "sk-or-v1-1234567890abcdef1234567890abcdef",
				DefaultModel: "gpt-3.5-turbo",
			},
			expected: true,
		},
		{
			name: "empty api key",
			config: &Config{
				APIKey:       "",
				DefaultModel: "claude-3-sonnet",
			},
			expected: false,
		},
		{
			name: "whitespace api key",
			config: &Config{
				APIKey:       "   ",
				DefaultModel: "claude-3-sonnet",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.IsValidAPIKey()
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestConfig_GetMaskedAPIKey(t *testing.T) {
	tests := []struct {
		name     string
		apiKey   string
		expected string
	}{
		{
			name:     "normal_key",
			apiKey:   "sk-or-v1-1234567890abcdef1234567890abcdef",
			expected: "sk-or-v1...cdef",
		},
		{
			name:     "short_key",
			apiKey:   "sk-or-v1-123",
			expected: "sk-or-v1...-123",
		},
		{
			name:     "very_short_key",
			apiKey:   "abc",
			expected: "***",
		},
		{
			name:     "empty_key",
			apiKey:   "",
			expected: "Not configured",
		},
	}

	for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				config := &Config{APIKey: tt.apiKey}
				result := config.GetMaskedAPIKey()
				if result != tt.expected {
					t.Errorf("expected %q, got %q", tt.expected, result)
				}
			})
		}
}
