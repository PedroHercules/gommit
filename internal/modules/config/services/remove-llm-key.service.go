package config

import (
	"fmt"
)

func RemoveLlmKey() error {
	service, err := NewConfigService()
	if err != nil {
		return fmt.Errorf("failed to initialize config service: %w", err)
	}

	return service.RemoveLlmKey()
}