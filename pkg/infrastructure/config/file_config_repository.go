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
	configDir  string         // Directory where config files are stored
	configFile string         // Path to the main config file
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
	configDir := filepath.Join(homeDir, ".gmit")
	configFile := filepath.Join(configDir, "config.json")

	// Ensure config directory exists with proper permissions
	if err := ensureDirectoryWithPermissions(configDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to setup config directory %s: %w", configDir, err)
	}

	// Verify directory is writable
	testFile := filepath.Join(configDir, ".test_write")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		return nil, fmt.Errorf("config directory %s is not writable: %w", configDir, err)
	}
	os.Remove(testFile) // Clean up test file

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
	ActiveProvider string            `json:"active_provider,omitempty"`
	AuthMethods    map[string]string `json:"auth_methods,omitempty"`
	DefaultModels  map[string]string `json:"default_models,omitempty"`
	DefaultModel   string            `json:"default_model,omitempty"`
	Version        string            `json:"version,omitempty"`
}

// Save persists the configuration to the file system.
func (r *FileConfigRepository) Save(config *entities.Config) error {
	provider := config.Provider
	if provider == "" {
		provider = "openrouter"
	}
	if err := r.SaveActiveProvider(provider); err != nil {
		return err
	}
	if config.APIKey != "" {
		if err := r.SaveProviderAPIKey(provider, config.APIKey); err != nil {
			return fmt.Errorf("failed to save API key: %w", err)
		}
	}
	if config.AuthMethod != "" {
		if err := r.SaveProviderAuthMethod(provider, config.AuthMethod); err != nil {
			return err
		}
	}
	if config.DefaultModel != "" {
		if err := r.SaveProviderDefaultModel(provider, config.DefaultModel); err != nil {
			return fmt.Errorf("failed to save default model: %w", err)
		}
	}

	return nil
}

// Load retrieves the configuration from storage.
func (r *FileConfigRepository) Load() (*entities.Config, error) {
	provider, err := r.LoadActiveProvider()
	if err != nil {
		provider = "openrouter"
	}
	authMethod, err := r.LoadProviderAuthMethod(provider)
	if err != nil {
		authMethod = "api_key"
	}
	apiKey, _ := r.LoadProviderAPIKey(provider)
	defaultModel, _ := r.LoadProviderDefaultModel(provider)

	config, err := entities.NewProviderConfig(provider, authMethod, apiKey, defaultModel)
	if err != nil {
		return nil, fmt.Errorf("failed to create config entity: %w", err)
	}

	return config, nil
}

// SaveAPIKey stores the API key securely in the system keyring.
func (r *FileConfigRepository) SaveAPIKey(apiKey string) error {
	return r.SaveProviderAPIKey("openrouter", apiKey)
}

// LoadAPIKey retrieves the API key from the system keyring.
func (r *FileConfigRepository) LoadAPIKey() (string, error) {
	return r.LoadProviderAPIKey("openrouter")
}

// DeleteAPIKey removes the API key from the system keyring.
func (r *FileConfigRepository) DeleteAPIKey() error {
	return r.DeleteProviderAPIKey("openrouter")
}

// SaveDefaultModel stores the default model preference in the config file.
func (r *FileConfigRepository) SaveDefaultModel(model string) error {
	provider, err := r.LoadActiveProvider()
	if err != nil {
		provider = "openrouter"
	}
	return r.SaveProviderDefaultModel(provider, model)
}

func (r *FileConfigRepository) SaveActiveProvider(provider string) error {
	data, err := r.loadConfigFile()
	if err != nil {
		return err
	}
	data.ActiveProvider = provider
	data.Version = "2.0"
	return r.saveConfigFile(data)
}

func (r *FileConfigRepository) LoadActiveProvider() (string, error) {
	data, err := r.loadConfigFile()
	if err != nil {
		return "", err
	}
	if data.ActiveProvider != "" {
		return data.ActiveProvider, nil
	}
	return "openrouter", nil
}

func (r *FileConfigRepository) SaveProviderAuthMethod(provider, method string) error {
	data, err := r.loadConfigFile()
	if err != nil {
		return err
	}
	if data.AuthMethods == nil {
		data.AuthMethods = make(map[string]string)
	}
	data.AuthMethods[provider] = method
	data.Version = "2.0"
	return r.saveConfigFile(data)
}

func (r *FileConfigRepository) LoadProviderAuthMethod(provider string) (string, error) {
	data, err := r.loadConfigFile()
	if err != nil {
		return "", err
	}
	if method := data.AuthMethods[provider]; method != "" {
		return method, nil
	}
	return "api_key", nil
}

func (r *FileConfigRepository) SaveProviderAPIKey(provider, apiKey string) error {
	return r.keyring.Set("gmit", provider+"_api_key", apiKey)
}

func (r *FileConfigRepository) LoadProviderAPIKey(provider string) (string, error) {
	apiKey, err := r.keyring.Get("gmit", provider+"_api_key")
	if err == nil || provider != "openrouter" {
		return apiKey, err
	}
	return r.keyring.Get("gmit", "api_key")
}

func (r *FileConfigRepository) DeleteProviderAPIKey(provider string) error {
	if err := r.keyring.Delete("gmit", provider+"_api_key"); err != nil {
		return err
	}
	if provider == "openrouter" {
		return r.keyring.Delete("gmit", "api_key")
	}
	return nil
}

func (r *FileConfigRepository) SaveProviderDefaultModel(provider, model string) error {
	data, err := r.loadConfigFile()
	if err != nil {
		data = &configFileData{}
	}
	if data.DefaultModels == nil {
		data.DefaultModels = make(map[string]string)
	}
	data.DefaultModels[provider] = model
	if provider == "openrouter" {
		data.DefaultModel = model
	}
	data.Version = "2.0"
	return r.saveConfigFile(data)
}

func (r *FileConfigRepository) LoadDefaultModel() (string, error) {
	provider, err := r.LoadActiveProvider()
	if err != nil {
		return "", err
	}
	return r.LoadProviderDefaultModel(provider)
}

func (r *FileConfigRepository) LoadProviderDefaultModel(provider string) (string, error) {
	data, err := r.loadConfigFile()
	if err != nil {
		return "", err
	}
	if model := data.DefaultModels[provider]; model != "" {
		return model, nil
	}
	if provider == "openrouter" {
		return data.DefaultModel, nil
	}
	return "", nil
}

func (r *FileConfigRepository) DeleteDefaultModel() error {
	provider, err := r.LoadActiveProvider()
	if err != nil {
		return err
	}
	return r.DeleteProviderDefaultModel(provider)
}

func (r *FileConfigRepository) DeleteProviderDefaultModel(provider string) error {
	data, err := r.loadConfigFile()
	if err != nil {
		return nil
	}
	delete(data.DefaultModels, provider)
	if provider == "openrouter" {
		data.DefaultModel = ""
	}
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
	// Ensure parent directory has proper permissions
	dir := filepath.Dir(r.configFile)
	if err := ensureDirectoryWithPermissions(dir, 0755); err != nil {
		return fmt.Errorf("failed to ensure config directory permissions: %w", err)
	}

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

// ensureDirectoryWithPermissions creates a directory with proper permissions
// and fixes permissions if the directory already exists.
func ensureDirectoryWithPermissions(dir string, perm os.FileMode) error {
	// Try to create directory with desired permissions
	if err := os.MkdirAll(dir, perm); err != nil {
		// If creation fails, try with more permissive settings
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory even with fallback permissions: %w", err)
		}
	}

	// Check if directory exists and verify/fix permissions
	info, err := os.Stat(dir)
	if err != nil {
		return fmt.Errorf("failed to stat directory: %w", err)
	}

	// Ensure directory has at least the minimum required permissions (0755)
	currentPerm := info.Mode().Perm()
	minPerm := os.FileMode(0755)

	// If current permissions are too restrictive, try to fix them
	if currentPerm&minPerm != minPerm {
		if err := os.Chmod(dir, perm); err != nil {
			// If chmod fails, try with minimum permissions
			if err := os.Chmod(dir, minPerm); err != nil {
				// If still fails, at least warn but don't fail completely
				// The directory exists, so basic operations might still work
				return fmt.Errorf("warning: could not set optimal directory permissions (directory exists but may have restricted access): %w", err)
			}
		}
	}

	return nil
}
