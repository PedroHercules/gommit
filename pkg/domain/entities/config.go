// Package entities contains the core business entities of the application.
package entities

import (
	"errors"
	"strings"
)

// Config represents the application configuration.
// This entity encapsulates all configuration-related business rules.
type Config struct {
	Provider     string // Active provider: openrouter or grok
	AuthMethod   string // api_key or oauth (Grok only)
	APIKey       string // API key for the active provider
	DefaultModel string // Preferred model for the active provider
}

// NewConfig creates a new configuration entity with validation.
// It ensures the configuration follows business rules.
func NewConfig(apiKey, defaultModel string) (*Config, error) {
	return NewProviderConfig("openrouter", "api_key", apiKey, defaultModel)
}

func NewProviderConfig(provider, authMethod, apiKey, defaultModel string) (*Config, error) {
	config := &Config{
		Provider:     strings.TrimSpace(provider),
		AuthMethod:   strings.TrimSpace(authMethod),
		APIKey:       strings.TrimSpace(apiKey),
		DefaultModel: strings.TrimSpace(defaultModel),
	}

	if config.Provider == "openrouter" && config.APIKey != "" && !config.IsValidAPIKey() {
		return nil, errors.New("invalid API key format")
	}

	return config, nil
}

// IsValidAPIKey checks if the API key follows the expected format.
// This implements a business rule for API key validation.
func (c *Config) IsValidAPIKey() bool {
	if c.APIKey == "" {
		return false
	}

	if c.Provider == "grok" {
		return strings.HasPrefix(c.APIKey, "xai-") && len(c.APIKey) > 20
	}
	return strings.HasPrefix(c.APIKey, "sk-or-v1-") && len(c.APIKey) > 20
}

// HasAPIKey returns true if a valid API key is configured.
func (c *Config) HasAPIKey() bool {
	return c.APIKey != "" && c.IsValidAPIKey()
}

// HasDefaultModel returns true if a default model is configured.
func (c *Config) HasDefaultModel() bool {
	return c.DefaultModel != ""
}

// SetAPIKey updates the API key with validation.
func (c *Config) SetAPIKey(apiKey string) error {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey != "" && c.Provider == "openrouter" && !strings.HasPrefix(apiKey, "sk-or-v1-") {
		return errors.New("invalid API key format")
	}
	c.APIKey = apiKey
	return nil
}

// SetDefaultModel updates the default model.
func (c *Config) SetDefaultModel(model string) {
	c.DefaultModel = strings.TrimSpace(model)
}

// ClearAPIKey removes the API key from configuration.
func (c *Config) ClearAPIKey() {
	c.APIKey = ""
}

// ClearDefaultModel removes the default model from configuration.
func (c *Config) ClearDefaultModel() {
	c.DefaultModel = ""
}

// GetMaskedAPIKey returns a masked version of the API key for display purposes.
// This is useful for showing the key without exposing the full value.
func (c *Config) GetMaskedAPIKey() string {
	if c.APIKey == "" {
		return "Not configured"
	}

	if len(c.APIKey) < 10 {
		return "***"
	}

	// Show first 8 characters and last 4 characters
	return c.APIKey[:8] + "..." + c.APIKey[len(c.APIKey)-4:]
}
