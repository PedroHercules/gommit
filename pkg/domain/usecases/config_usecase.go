// Package usecases contains the business logic use cases.
package usecases

import (
	"fmt"
	"strings"

	"github.com/PedroHercules/gommit/pkg/domain/entities"
	"github.com/PedroHercules/gommit/pkg/domain/repositories"
)

// ConfigUseCase handles all configuration-related business logic.
// This use case manages API keys, model preferences, and other settings.
type ConfigUseCase struct {
	configRepo repositories.ConfigRepository
	llmRepo    repositories.LLMRepository
}

// NewConfigUseCase creates a new instance of ConfigUseCase.
func NewConfigUseCase(
	configRepo repositories.ConfigRepository,
	llmRepo repositories.LLMRepository,
) *ConfigUseCase {
	return &ConfigUseCase{
		configRepo: configRepo,
		llmRepo:    llmRepo,
	}
}

// SetAPIKeyRequest represents the input for setting an API key.
type SetAPIKeyRequest struct {
	APIKey string
}

// SetAPIKeyResponse represents the output of setting an API key.
type SetAPIKeyResponse struct {
	Success      bool
	Message      string
	ErrorMessage string
}

// SetAPIKey stores the API key securely and validates it.
func (uc *ConfigUseCase) SetAPIKey(req SetAPIKeyRequest) (*SetAPIKeyResponse, error) {
	provider, err := uc.configRepo.LoadActiveProvider()
	if err != nil || provider == "" {
		provider = "openrouter"
	}
	return uc.SetProviderAPIKey(provider, req.APIKey)
}

func (uc *ConfigUseCase) SetProviderAPIKey(provider, apiKey string) (*SetAPIKeyResponse, error) {
	response := &SetAPIKeyResponse{}
	apiKey = strings.TrimSpace(apiKey)
	if err := validateProviderAPIKey(provider, apiKey); err != nil {
		response.ErrorMessage = err.Error()
		return response, nil
	}
	if err := uc.configureLLM(provider, "api_key"); err != nil {
		response.ErrorMessage = err.Error()
		return response, nil
	}

	// Validate the key against the selected provider before storing it.
	isValid, err := uc.llmRepo.TestConnection(apiKey)
	if err != nil {
		response.ErrorMessage = fmt.Sprintf("Failed to validate API key: %v", err)
		return response, nil
	}

	if !isValid {
		response.ErrorMessage = "Invalid API key or service unavailable"
		return response, nil
	}

	// Step 3: Save the API key
	err = uc.configRepo.SaveProviderAPIKey(provider, apiKey)
	if err != nil {
		response.ErrorMessage = fmt.Sprintf("Failed to save API key: %v", err)
		return response, nil
	}
	if err := uc.configRepo.SaveProviderAuthMethod(provider, "api_key"); err != nil {
		response.ErrorMessage = fmt.Sprintf("Failed to save authentication method: %v", err)
		return response, nil
	}

	response.Success = true
	response.Message = "API key saved successfully"
	return response, nil
}

// GetAPIKeyResponse represents the output of getting an API key.
type GetAPIKeyResponse struct {
	APIKey       string
	MaskedAPIKey string
	Configured   bool
	ErrorMessage string
}

// GetAPIKey retrieves the current API key.
func (uc *ConfigUseCase) GetAPIKey() (*GetAPIKeyResponse, error) {
	response := &GetAPIKeyResponse{}
	provider, err := uc.configRepo.LoadActiveProvider()
	if err != nil {
		provider = "openrouter"
	}
	authMethod, _ := uc.configRepo.LoadProviderAuthMethod(provider)
	if provider == "grok" && authMethod == "oauth" {
		response.Configured = true
		response.MaskedAPIKey = "Authenticated through Grok CLI"
		return response, nil
	}
	apiKey, err := uc.configRepo.LoadProviderAPIKey(provider)
	if err != nil {
		apiKey = ""
	}

	if apiKey == "" {
		response.Configured = false
		response.MaskedAPIKey = "Not configured"
		return response, nil
	}

	config, _ := entities.NewProviderConfig(provider, authMethod, apiKey, "")
	response.APIKey = apiKey
	response.MaskedAPIKey = config.GetMaskedAPIKey()
	response.Configured = true

	return response, nil
}

// RemoveAPIKey deletes the stored API key.
func (uc *ConfigUseCase) RemoveAPIKey() error {
	provider, err := uc.configRepo.LoadActiveProvider()
	if err != nil {
		provider = "openrouter"
	}
	return uc.configRepo.DeleteProviderAPIKey(provider)
}

func (uc *ConfigUseCase) SetActiveProvider(provider string) error {
	if provider != "openrouter" && provider != "grok" {
		return fmt.Errorf("unsupported provider %q", provider)
	}
	authMethod, err := uc.configRepo.LoadProviderAuthMethod(provider)
	if err != nil {
		authMethod = "api_key"
	}
	if err := uc.configureLLM(provider, authMethod); err != nil {
		return err
	}
	return uc.configRepo.SaveActiveProvider(provider)
}

func (uc *ConfigUseCase) GetProviderInfo() (string, string, error) {
	provider, err := uc.configRepo.LoadActiveProvider()
	if err != nil || provider == "" {
		provider = "openrouter"
	}
	authMethod, err := uc.configRepo.LoadProviderAuthMethod(provider)
	if err != nil || authMethod == "" {
		authMethod = "api_key"
	}
	return provider, authMethod, nil
}

func (uc *ConfigUseCase) ConfigureProvider(provider, authMethod, apiKey string) error {
	if err := uc.configureLLM(provider, authMethod); err != nil {
		return err
	}
	if authMethod == "api_key" {
		apiKey = strings.TrimSpace(apiKey)
		if err := validateProviderAPIKey(provider, apiKey); err != nil {
			return err
		}
		valid, err := uc.llmRepo.TestConnection(apiKey)
		if err != nil {
			return err
		}
		if !valid {
			return fmt.Errorf("invalid API key or %s is unavailable", provider)
		}
		if err := uc.configRepo.SaveProviderAPIKey(provider, strings.TrimSpace(apiKey)); err != nil {
			return err
		}
	}
	if authMethod == "oauth" {
		valid, err := uc.llmRepo.TestConnection("")
		if err != nil {
			return err
		}
		if !valid {
			return fmt.Errorf("Grok OAuth session is not available; run `grok login` and try again")
		}
	}
	if err := uc.configRepo.SaveProviderAuthMethod(provider, authMethod); err != nil {
		return err
	}
	return uc.configRepo.SaveActiveProvider(provider)
}

func validateProviderAPIKey(provider, apiKey string) error {
	if apiKey == "" {
		return fmt.Errorf("API key cannot be empty")
	}
	switch provider {
	case "openrouter":
		if _, err := entities.NewConfig(apiKey, ""); err != nil {
			return err
		}
	case "grok":
		if !strings.HasPrefix(apiKey, "xai-") || len(apiKey) <= 20 {
			return fmt.Errorf("xAI API keys should start with xai- and be complete")
		}
	default:
		return fmt.Errorf("unsupported provider %q", provider)
	}
	return nil
}

func (uc *ConfigUseCase) GetAvailableModelsFor(provider, authMethod, apiKey string) (*GetAvailableModelsResponse, error) {
	response := &GetAvailableModelsResponse{}
	if err := uc.configureLLM(provider, authMethod); err != nil {
		response.ErrorMessage = err.Error()
		return response, nil
	}
	uc.llmRepo.SetAPIKey(apiKey)
	models, err := uc.llmRepo.GetAvailableModels()
	if err != nil {
		response.ErrorMessage = err.Error()
		return response, nil
	}
	response.Models = models
	return response, nil
}

func (uc *ConfigUseCase) configureLLM(provider, authMethod string) error {
	if configurable, ok := uc.llmRepo.(repositories.ProviderConfigurableLLMRepository); ok {
		return configurable.ConfigureProvider(provider, authMethod)
	}
	if provider != "openrouter" || authMethod != "api_key" {
		return fmt.Errorf("configured LLM repository does not support %s/%s", provider, authMethod)
	}
	return nil
}

// SetDefaultModelRequest represents the input for setting a default model.
type SetDefaultModelRequest struct {
	Model string
}

// SetDefaultModelResponse represents the output of setting a default model.
type SetDefaultModelResponse struct {
	Success      bool
	Message      string
	ErrorMessage string
}

// SetDefaultModel sets the default LLM model to use.
func (uc *ConfigUseCase) SetDefaultModel(req SetDefaultModelRequest) (*SetDefaultModelResponse, error) {
	provider, err := uc.configRepo.LoadActiveProvider()
	if err != nil {
		provider = "openrouter"
	}
	return uc.SetProviderDefaultModel(provider, req.Model)
}

func (uc *ConfigUseCase) SetProviderDefaultModel(provider, model string) (*SetDefaultModelResponse, error) {
	response := &SetDefaultModelResponse{}
	authMethod, err := uc.configRepo.LoadProviderAuthMethod(provider)
	if err != nil {
		authMethod = "api_key"
	}
	apiKey, _ := uc.configRepo.LoadProviderAPIKey(provider)
	if err := uc.configureLLM(provider, authMethod); err != nil {
		response.ErrorMessage = err.Error()
		return response, nil
	}
	uc.llmRepo.SetAPIKey(apiKey)

	// Step 1: Validate the model exists
	isValid, err := uc.llmRepo.ValidateModel(model)
	if err != nil {
		response.ErrorMessage = fmt.Sprintf("Failed to validate model: %v", err)
		return response, nil
	}

	if !isValid {
		response.ErrorMessage = fmt.Sprintf("Model '%s' is not available", model)
		return response, nil
	}

	// Step 2: Save the default model
	err = uc.configRepo.SaveProviderDefaultModel(provider, model)
	if err != nil {
		response.ErrorMessage = fmt.Sprintf("Failed to save default model: %v", err)
		return response, nil
	}

	response.Success = true
	response.Message = fmt.Sprintf("Default model set to '%s'", model)
	return response, nil
}

// GetDefaultModelResponse represents the output of getting the default model.
type GetDefaultModelResponse struct {
	Model        string
	Configured   bool
	ErrorMessage string
}

// GetDefaultModel retrieves the current default model.
func (uc *ConfigUseCase) GetDefaultModel() (*GetDefaultModelResponse, error) {
	response := &GetDefaultModelResponse{}
	provider, err := uc.configRepo.LoadActiveProvider()
	if err != nil {
		provider = "openrouter"
	}
	model, err := uc.configRepo.LoadProviderDefaultModel(provider)
	if err != nil {
		response.ErrorMessage = fmt.Sprintf("Failed to load default model: %v", err)
		return response, nil
	}

	if model == "" {
		response.Configured = false
		return response, nil
	}

	response.Model = model
	response.Configured = true
	return response, nil
}

// RemoveDefaultModel removes the default model configuration.
func (uc *ConfigUseCase) RemoveDefaultModel() error {
	provider, err := uc.configRepo.LoadActiveProvider()
	if err != nil {
		provider = "openrouter"
	}
	return uc.configRepo.DeleteProviderDefaultModel(provider)
}

// GetAvailableModelsResponse represents the output of getting available models.
type GetAvailableModelsResponse struct {
	Models       []repositories.LLMModel
	ErrorMessage string
}

// GetAvailableModels retrieves all available LLM models.
func (uc *ConfigUseCase) GetAvailableModels() (*GetAvailableModelsResponse, error) {
	response := &GetAvailableModelsResponse{}
	config, err := uc.configRepo.Load()
	if err != nil {
		response.ErrorMessage = err.Error()
		return response, nil
	}
	if err := uc.configureLLM(config.Provider, config.AuthMethod); err != nil {
		response.ErrorMessage = err.Error()
		return response, nil
	}
	uc.llmRepo.SetAPIKey(config.APIKey)
	models, err := uc.llmRepo.GetAvailableModels()
	if err != nil {
		response.ErrorMessage = fmt.Sprintf("Failed to get available models: %v", err)
		return response, nil
	}

	response.Models = models
	return response, nil
}
