package config

import (
	"fmt"
)

func SetDefaultModel(model string) error {
	service, err := NewConfigService()
	if err != nil {
		return fmt.Errorf("failed to initialize config service: %w", err)
	}

	return service.SetDefaultModel(model)
}

func GetDefaultModel() (string, error) {
	service, err := NewConfigService()
	if err != nil {
		return "", fmt.Errorf("failed to initialize config service: %w", err)
	}

	return service.GetDefaultModel(), nil
}

func RemoveDefaultModel() error {
	service, err := NewConfigService()
	if err != nil {
		return fmt.Errorf("failed to initialize config service: %w", err)
	}

	return service.RemoveDefaultModel()
}