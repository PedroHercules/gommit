// Package services provides application services that orchestrate use cases.
package services

import (
	"fmt"

	"github.com/PedroHercules/gommit/pkg/domain/repositories"
	"github.com/PedroHercules/gommit/pkg/domain/usecases"
)

// ConfigService provides high-level operations for configuration management.
// This service orchestrates configuration-related use cases and provides
// a simplified interface for the CLI layer.
type ConfigService struct {
	configUC *usecases.ConfigUseCase
}

// NewConfigService creates a new configuration service.
func NewConfigService(configUC *usecases.ConfigUseCase) *ConfigService {
	return &ConfigService{
		configUC: configUC,
	}
}

// SetupAPIKeyRequest represents the input for setting up an API key.
type SetupAPIKeyRequest struct {
	APIKey      string
	ValidateKey bool // Whether to validate the key with the LLM service
}

// SetupAPIKeyResponse represents the output of setting up an API key.
type SetupAPIKeyResponse struct {
	Success      bool
	Message      string
	ErrorMessage string
	KeyMasked    string // Masked version of the key for display
}

// SetupAPIKey configures the API key with validation and user feedback.
func (s *ConfigService) SetupAPIKey(req SetupAPIKeyRequest) (*SetupAPIKeyResponse, error) {
	response := &SetupAPIKeyResponse{}

	// Set up the API key
	setKeyReq := usecases.SetAPIKeyRequest{
		APIKey: req.APIKey,
	}

	setKeyResp, err := s.configUC.SetAPIKey(setKeyReq)
	if err != nil {
		response.ErrorMessage = err.Error()
		return response, nil
	}

	if !setKeyResp.Success {
		response.ErrorMessage = setKeyResp.ErrorMessage
		return response, nil
	}

	// Get the masked key for display
	getKeyResp, err := s.configUC.GetAPIKey()
	if err == nil && getKeyResp.Configured {
		response.KeyMasked = getKeyResp.MaskedAPIKey
	}

	response.Success = true
	response.Message = setKeyResp.Message

	return response, nil
}

// GetAPIKeyInfo returns information about the current API key.
func (s *ConfigService) GetAPIKeyInfo() (*usecases.GetAPIKeyResponse, error) {
	return s.configUC.GetAPIKey()
}

// RemoveAPIKey removes the stored API key.
func (s *ConfigService) RemoveAPIKey() error {
	return s.configUC.RemoveAPIKey()
}

// SetupModelRequest represents the input for setting up a default model.
type SetupModelRequest struct {
	Model        string
	ValidateModel bool // Whether to validate the model exists
}

// SetupModelResponse represents the output of setting up a default model.
type SetupModelResponse struct {
	Success      bool
	Message      string
	ErrorMessage string
	ModelInfo    *repositories.LLMModel // Information about the selected model
}

// SetupDefaultModel configures the default model with validation.
func (s *ConfigService) SetupDefaultModel(req SetupModelRequest) (*SetupModelResponse, error) {
	response := &SetupModelResponse{}

	// Set up the default model
	setModelReq := usecases.SetDefaultModelRequest{
		Model: req.Model,
	}

	setModelResp, err := s.configUC.SetDefaultModel(setModelReq)
	if err != nil {
		response.ErrorMessage = err.Error()
		return response, nil
	}

	if !setModelResp.Success {
		response.ErrorMessage = setModelResp.ErrorMessage
		return response, nil
	}

	response.Success = true
	response.Message = setModelResp.Message

	return response, nil
}

// GetDefaultModelInfo returns information about the current default model.
func (s *ConfigService) GetDefaultModelInfo() (*usecases.GetDefaultModelResponse, error) {
	return s.configUC.GetDefaultModel()
}

// RemoveDefaultModel removes the default model configuration.
func (s *ConfigService) RemoveDefaultModel() error {
	return s.configUC.RemoveDefaultModel()
}

// GetAvailableModels returns all available LLM models.
func (s *ConfigService) GetAvailableModels() (*usecases.GetAvailableModelsResponse, error) {
	return s.configUC.GetAvailableModels()
}

// ConfigSummaryResponse represents a summary of the current configuration.
type ConfigSummaryResponse struct {
	APIKeyConfigured    bool
	APIKeyMasked        string
	DefaultModelSet     bool
	DefaultModel        string
	AvailableModelsCount int
	ErrorMessage        string
}

// GetConfigSummary returns a summary of the current configuration.
// This is useful for displaying the current state to the user.
func (s *ConfigService) GetConfigSummary() (*ConfigSummaryResponse, error) {
	response := &ConfigSummaryResponse{}

	// Get API key info
	apiKeyResp, err := s.configUC.GetAPIKey()
	if err != nil {
		response.ErrorMessage = fmt.Sprintf("Failed to get API key info: %v", err)
		return response, nil
	}

	response.APIKeyConfigured = apiKeyResp.Configured
	response.APIKeyMasked = apiKeyResp.MaskedAPIKey

	// Get default model info
	modelResp, err := s.configUC.GetDefaultModel()
	if err != nil {
		response.ErrorMessage = fmt.Sprintf("Failed to get default model info: %v", err)
		return response, nil
	}

	response.DefaultModelSet = modelResp.Configured
	response.DefaultModel = modelResp.Model

	// Get available models count
	modelsResp, err := s.configUC.GetAvailableModels()
	if err == nil {
		response.AvailableModelsCount = len(modelsResp.Models)
	}

	return response, nil
}

// ValidateConfigurationResponse represents the result of configuration validation.
type ValidateConfigurationResponse struct {
	Valid            bool
	Issues           []string
	Recommendations  []string
	ErrorMessage     string
}

// ValidateConfiguration checks if the current configuration is valid and complete.
func (s *ConfigService) ValidateConfiguration() (*ValidateConfigurationResponse, error) {
	response := &ValidateConfigurationResponse{
		Issues:          []string{},
		Recommendations: []string{},
	}

	// Check API key
	apiKeyResp, err := s.configUC.GetAPIKey()
	if err != nil {
		response.ErrorMessage = fmt.Sprintf("Failed to validate API key: %v", err)
		return response, nil
	}

	if !apiKeyResp.Configured {
		response.Issues = append(response.Issues, "API key is not configured")
		response.Recommendations = append(response.Recommendations, "Run 'gommit config set-key <your-api-key>' to configure your OpenRouter API key")
	}

	// Check default model
	modelResp, err := s.configUC.GetDefaultModel()
	if err != nil {
		response.ErrorMessage = fmt.Sprintf("Failed to validate default model: %v", err)
		return response, nil
	}

	if !modelResp.Configured {
		response.Recommendations = append(response.Recommendations, "Consider setting a default model with 'gommit config set-model <model-id>' for faster commit generation")
	}

	// Check if models are available
	modelsResp, err := s.configUC.GetAvailableModels()
	if err != nil {
		response.Issues = append(response.Issues, "Cannot fetch available models - check your API key and internet connection")
	} else if len(modelsResp.Models) == 0 {
		response.Issues = append(response.Issues, "No models available - check your API key")
	}

	// Configuration is valid if there are no critical issues
	response.Valid = len(response.Issues) == 0

	return response, nil
}