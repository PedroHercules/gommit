// Package repositories defines the interfaces for data access.
// These interfaces are part of the domain layer and define contracts
// that must be implemented by the infrastructure layer.
package repositories

import (
	"github.com/PedroHercules/gommit/pkg/domain/entities"
)

// ConfigRepository defines the interface for configuration persistence.
// This interface follows the Repository pattern and allows for different
// implementations (file-based, database, etc.) without changing the domain logic.
type ConfigRepository interface {
	// Save persists the configuration to storage.
	// Returns an error if the operation fails.
	Save(config *entities.Config) error

	// Load retrieves the configuration from storage.
	// Returns a default configuration if none exists.
	Load() (*entities.Config, error)

	// SaveAPIKey stores the API key securely.
	// This method handles secure storage (keyring, etc.).
	SaveAPIKey(apiKey string) error

	// LoadAPIKey retrieves the API key from secure storage.
	// Returns empty string if no key is found.
	LoadAPIKey() (string, error)

	// DeleteAPIKey removes the API key from secure storage.
	DeleteAPIKey() error

	// SaveDefaultModel stores the default model preference.
	SaveDefaultModel(model string) error

	// LoadDefaultModel retrieves the default model preference.
	// Returns empty string if no model is configured.
	LoadDefaultModel() (string, error)

	// DeleteDefaultModel removes the default model preference.
	DeleteDefaultModel() error

	SaveActiveProvider(provider string) error
	LoadActiveProvider() (string, error)
	SaveProviderAuthMethod(provider, method string) error
	LoadProviderAuthMethod(provider string) (string, error)
	SaveProviderAPIKey(provider, apiKey string) error
	LoadProviderAPIKey(provider string) (string, error)
	DeleteProviderAPIKey(provider string) error
	SaveProviderDefaultModel(provider, model string) error
	LoadProviderDefaultModel(provider string) (string, error)
	DeleteProviderDefaultModel(provider string) error

	// Exists checks if a configuration file exists.
	Exists() bool
}
