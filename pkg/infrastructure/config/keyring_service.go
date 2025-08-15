// Package config provides keyring services for secure credential storage.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/99designs/keyring"
)

// KeyringService defines the interface for secure credential storage.
type KeyringService interface {
	Set(service, user, password string) error
	Get(service, user string) (string, error)
	Delete(service, user string) error
}

// SimpleKeyringService implements a basic file-based keyring.
// This is a fallback implementation for systems without proper keyring support.
type SimpleKeyringService struct {
	configDir string
}

// NewSimpleKeyringService creates a new simple keyring service.
func NewSimpleKeyringService(configDir string) *SimpleKeyringService {
	return &SimpleKeyringService{
		configDir: configDir,
	}
}

// Set stores a credential in the simple keyring.
func (s *SimpleKeyringService) Set(service, user, password string) error {
	filePath := s.getCredentialPath(service, user)
	dir := filepath.Dir(filePath)

	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create keyring directory: %w", err)
	}

	cred := map[string]string{
		"service":  service,
		"user":     user,
		"password": password,
	}

	data, err := json.Marshal(cred)
	if err != nil {
		return fmt.Errorf("failed to marshal credential: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0600); err != nil {
		return fmt.Errorf("failed to write credential file: %w", err)
	}

	return nil
}

// Get retrieves a credential from the simple keyring.
func (s *SimpleKeyringService) Get(service, user string) (string, error) {
	filePath := s.getCredentialPath(service, user)

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("credential not found")
		}
		return "", fmt.Errorf("failed to read credential file: %w", err)
	}

	var cred map[string]string
	if err := json.Unmarshal(data, &cred); err != nil {
		return "", fmt.Errorf("failed to unmarshal credential: %w", err)
	}

	password, exists := cred["password"]
	if !exists {
		return "", fmt.Errorf("password not found in credential")
	}

	return password, nil
}

// Delete removes a credential from the simple keyring.
func (s *SimpleKeyringService) Delete(service, user string) error {
	filePath := s.getCredentialPath(service, user)

	if err := os.Remove(filePath); err != nil {
		if os.IsNotExist(err) {
			return nil // Already deleted
		}
		return fmt.Errorf("failed to delete credential file: %w", err)
	}

	return nil
}

// getCredentialPath returns the file path for a credential.
func (s *SimpleKeyringService) getCredentialPath(service, user string) string {
	filename := fmt.Sprintf("%s_%s.json", service, user)
	return filepath.Join(s.configDir, "keyring", filename)
}

// SystemKeyringService wraps the system keyring with fallback to simple keyring.
type SystemKeyringService struct {
	keyring  keyring.Keyring
	fallback KeyringService
}

// NewSystemKeyringService creates a new system keyring service.
func NewSystemKeyringService(configDir string) KeyringService {
	// Try to open system keyring
	kr, err := keyring.Open(keyring.Config{
		ServiceName: "gommit",
	})

	fallback := NewSimpleKeyringService(configDir)

	if err != nil {
		// If system keyring fails, use fallback only
		return fallback
	}

	return &SystemKeyringService{
		keyring:  kr,
		fallback: fallback,
	}
}

// Set stores a credential using the system keyring or fallback.
func (s *SystemKeyringService) Set(service, user, password string) error {
	key := fmt.Sprintf("%s_%s", service, user)

	// Try system keyring first
	if s.keyring != nil {
		if err := s.keyring.Set(keyring.Item{
			Key:  key,
			Data: []byte(password),
		}); err == nil {
			return nil
		}
	}

	// Fall back to simple keyring
	return s.fallback.Set(service, user, password)
}

// Get retrieves a credential using the system keyring or fallback.
func (s *SystemKeyringService) Get(service, user string) (string, error) {
	key := fmt.Sprintf("%s_%s", service, user)

	// Try system keyring first
	if s.keyring != nil {
		if item, err := s.keyring.Get(key); err == nil {
			return string(item.Data), nil
		}
	}

	// Fall back to simple keyring
	return s.fallback.Get(service, user)
}

// Delete removes a credential using the system keyring or fallback.
func (s *SystemKeyringService) Delete(service, user string) error {
	key := fmt.Sprintf("%s_%s", service, user)

	// Try system keyring first
	if s.keyring != nil {
		if err := s.keyring.Remove(key); err == nil {
			return nil
		}
	}

	// Fall back to simple keyring
	return s.fallback.Delete(service, user)
}