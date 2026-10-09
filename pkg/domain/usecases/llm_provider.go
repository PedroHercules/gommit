package usecases

import (
	"fmt"

	"github.com/PedroHercules/gommit/pkg/domain/entities"
	"github.com/PedroHercules/gommit/pkg/domain/repositories"
)

func configureLLMForConfig(repo repositories.LLMRepository, config *entities.Config) error {
	if configurable, ok := repo.(repositories.ProviderConfigurableLLMRepository); ok {
		if err := configurable.ConfigureProvider(config.Provider, config.AuthMethod); err != nil {
			return err
		}
	} else if config.Provider != "openrouter" {
		return fmt.Errorf("LLM provider %q is not supported", config.Provider)
	}

	if config.AuthMethod == "oauth" {
		valid, err := repo.TestConnection("")
		if err != nil {
			return fmt.Errorf("Grok OAuth is unavailable: %w", err)
		}
		if !valid {
			return fmt.Errorf("Grok OAuth session is unavailable; run `grok login`")
		}
		return nil
	}
	if !config.HasAPIKey() {
		return fmt.Errorf("%s API key is not configured; run `gmit config`", config.Provider)
	}
	repo.SetAPIKey(config.APIKey)
	return nil
}
