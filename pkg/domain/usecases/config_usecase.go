// Package usecases contains the business logic use cases.
package usecases

import (
	"fmt"

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
	response := &SetAPIKeyResponse{}

	// Step 1: Validate the API key format
	_, err := entities.NewConfig(req.APIKey, "")
	if err != nil {
		response.ErrorMessage = err.Error()
		return response, nil
	}

	// Step 2: Test the API key with the LLM service
	isValid, err := uc.llmRepo.TestConnection(req.APIKey)
	if err != nil {
		response.ErrorMessage = fmt.Sprintf("Failed to validate API key: %v", err)
		return response, nil
	}

	if !isValid {
		response.ErrorMessage = "Invalid API key or service unavailable"
		return response, nil
	}

	// Step 3: Save the API key
	err = uc.configRepo.SaveAPIKey(req.APIKey)
	if err != nil {
		response.ErrorMessage = fmt.Sprintf("Failed to save API key: %v", err)
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

	apiKey, err := uc.configRepo.LoadAPIKey()
	if err != nil {
		response.ErrorMessage = fmt.Sprintf("Failed to load API key: %v", err)
		return response, nil
	}

	if apiKey == "" {
		response.Configured = false
		response.MaskedAPIKey = "Not configured"
		return response, nil
	}

	config, _ := entities.NewConfig(apiKey, "")
	response.APIKey = apiKey
	response.MaskedAPIKey = config.GetMaskedAPIKey()
	response.Configured = true

	return response, nil
}

// RemoveAPIKey deletes the stored API key.
func (uc *ConfigUseCase) RemoveAPIKey() error {
	return uc.configRepo.DeleteAPIKey()
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
	response := &SetDefaultModelResponse{}

	// Step 1: Validate the model exists
	isValid, err := uc.llmRepo.ValidateModel(req.Model)
	if err != nil {
		response.ErrorMessage = fmt.Sprintf("Failed to validate model: %v", err)
		return response, nil
	}

	if !isValid {
		response.ErrorMessage = fmt.Sprintf("Model '%s' is not available", req.Model)
		return response, nil
	}

	// Step 2: Save the default model
	err = uc.configRepo.SaveDefaultModel(req.Model)
	if err != nil {
		response.ErrorMessage = fmt.Sprintf("Failed to save default model: %v", err)
		return response, nil
	}

	response.Success = true
	response.Message = fmt.Sprintf("Default model set to '%s'", req.Model)
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

	model, err := uc.configRepo.LoadDefaultModel()
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
	return uc.configRepo.DeleteDefaultModel()
}

// GetAvailableModelsResponse represents the output of getting available models.
type GetAvailableModelsResponse struct {
	Models       []repositories.LLMModel
	ErrorMessage string
}

// GetAvailableModels retrieves all available LLM models.
func (uc *ConfigUseCase) GetAvailableModels() (*GetAvailableModelsResponse, error) {
	response := &GetAvailableModelsResponse{}

	models, err := uc.llmRepo.GetAvailableModels()
	if err != nil {
		response.ErrorMessage = fmt.Sprintf("Failed to get available models: %v", err)
		return response, nil
	}

	response.Models = models
	return response, nil
}