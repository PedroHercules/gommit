// Package config provides infrastructure implementations for configuration management.
// This package implements the repository interfaces defined in the domain layer.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/PedroHercules/gommit/pkg/domain/entities"
)

// FileConfigRepository implements the ConfigRepository interface using file-based storage.
// This implementation stores configuration in JSON files and API keys in the system keyring.
type FileConfigRepository struct {
	configDir  string // Directory where config files are stored
	configFile string // Path to the main config file
	keyring    KeyringService // Service for secure API key storage
}

// NewFileConfigRepository creates a new file-based configuration repository.
// It automatically creates the configuration directory if it doesn't exist.
func NewFileConfigRepository() (*FileConfigRepository, error) {
	// Get user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}

	// Create config directory path
	configDir := filepath.Join(homeDir, ".gommit")
	configFile := filepath.Join(configDir, "config.json")

	// Create config directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	// Initialize keyring service
	keyringService := NewSystemKeyringService(configDir)

	return &FileConfigRepository{
		configDir:  configDir,
		configFile: configFile,
		keyring:    keyringService,
	}, nil
}

// configFileData represents the structure of the configuration file.
type configFileData struct {
	DefaultModel string `json:"default_model,omitempty"`
	Version      string `json:"version,omitempty"`
}

// Save persists the configuration to the file system.
func (r *FileConfigRepository) Save(config *entities.Config) error {
	// Save API key to keyring if provided
	if config.APIKey != "" {
		if err := r.SaveAPIKey(config.APIKey); err != nil {
			return fmt.Errorf("failed to save API key: %w", err)
		}
	}

	// Save other config data to file
	if config.DefaultModel != "" {
		if err := r.SaveDefaultModel(config.DefaultModel); err != nil {
			return fmt.Errorf("failed to save default model: %w", err)
		}
	}

	return nil
}

// Load retrieves the configuration from storage.
func (r *FileConfigRepository) Load() (*entities.Config, error) {
	// Load API key from keyring
	apiKey, err := r.LoadAPIKey()
	if err != nil {
		// Don't fail if API key is not found, just continue with empty key
		apiKey = ""
	}

	// Load default model from file
	defaultModel, err := r.LoadDefaultModel()
	if err != nil {
		// Don't fail if default model is not found
		defaultModel = ""
	}

	// Create config entity
	config, err := entities.NewConfig(apiKey, defaultModel)
	if err != nil {
		return nil, fmt.Errorf("failed to create config entity: %w", err)
	}

	return config, nil
}

// SaveAPIKey stores the API key securely in the system keyring.
func (r *FileConfigRepository) SaveAPIKey(apiKey string) error {
	return r.keyring.Set("gommit", "api_key", apiKey)
}

// LoadAPIKey retrieves the API key from the system keyring.
func (r *FileConfigRepository) LoadAPIKey() (string, error) {
	return r.keyring.Get("gommit", "api_key")
}

// DeleteAPIKey removes the API key from the system keyring.
func (r *FileConfigRepository) DeleteAPIKey() error {
	return r.keyring.Delete("gommit", "api_key")
}

// SaveDefaultModel stores the default model preference in the config file.
func (r *FileConfigRepository) SaveDefaultModel(model string) error {
	// Load existing config data
	data, err := r.loadConfigFile()
	if err != nil {
		// If file doesn't exist, create new data
		data = &configFileData{}
	}

	// Update default model
	data.DefaultModel = model
	data.Version = "1.0"

	// Save to file
	return r.saveConfigFile(data)
}

// LoadDefaultModel retrieves the default model preference from the config file.
func (r *FileConfigRepository) LoadDefaultModel() (string, error) {
	data, err := r.loadConfigFile()
	if err != nil {
		return "", err
	}

	return data.DefaultModel, nil
}

// DeleteDefaultModel removes the default model preference from the config file.
func (r *FileConfigRepository) DeleteDefaultModel() error {
	data, err := r.loadConfigFile()
	if err != nil {
		// If file doesn't exist, nothing to delete
		return nil
	}

	// Clear default model
	data.DefaultModel = ""

	// Save to file
	return r.saveConfigFile(data)
}

// Exists checks if the configuration file exists.
func (r *FileConfigRepository) Exists() bool {
	_, err := os.Stat(r.configFile)
	return err == nil
}

// loadConfigFile reads and parses the configuration file.
func (r *FileConfigRepository) loadConfigFile() (*configFileData, error) {
	data, err := os.ReadFile(r.configFile)
	if err != nil {
		if os.IsNotExist(err) {
			return &configFileData{}, nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config configFileData
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// saveConfigFile writes the configuration data to the file.
func (r *FileConfigRepository) saveConfigFile(data *configFileData) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config data: %w", err)
	}

	if err := os.WriteFile(r.configFile, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetConfigDir returns the configuration directory path.
// This is useful for other components that need to store files in the config directory.
func (r *FileConfigRepository) GetConfigDir() string {
	return r.configDir
}

// GetConfigFile returns the configuration file path.
func (r *FileConfigRepository) GetConfigFile() string {
	return r.configFile
}