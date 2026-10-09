package config_services

import (
	"fmt"
	"github.com/PedroHercules/gommit/pkg/domain/usecases"
)

type ConfigSummaryResponse struct {
	Provider             string
	AuthMethod           string
	APIKeyConfigured     bool
	APIKeyMasked         string
	DefaultModelSet      bool
	DefaultModel         string
	AvailableModelsCount int
	ErrorMessage         string
}

type ValidateConfigurationResponse struct {
	Valid           bool
	Issues          []string
	Recommendations []string
	ErrorMessage    string
}

type validationService struct {
	configUC *usecases.ConfigUseCase
}

func newValidationService(configUC *usecases.ConfigUseCase) *validationService {
	return &validationService{
		configUC: configUC,
	}
}

func (s *validationService) GetConfigSummary() (*ConfigSummaryResponse, error) {
	response := &ConfigSummaryResponse{}
	response.Provider, response.AuthMethod, _ = s.configUC.GetProviderInfo()

	apiKeyResp, err := s.configUC.GetAPIKey()
	if err != nil {
		response.ErrorMessage = fmt.Sprintf("Failed to get API key info: %v", err)
		return response, nil
	}

	response.APIKeyConfigured = apiKeyResp.Configured
	response.APIKeyMasked = apiKeyResp.MaskedAPIKey

	modelResp, err := s.configUC.GetDefaultModel()
	if err != nil {
		response.ErrorMessage = fmt.Sprintf("Failed to get default model info: %v", err)
		return response, nil
	}

	response.DefaultModelSet = modelResp.Configured
	response.DefaultModel = modelResp.Model

	modelsResp, err := s.configUC.GetAvailableModels()
	if err == nil {
		response.AvailableModelsCount = len(modelsResp.Models)
	}

	return response, nil
}

func (s *validationService) ValidateConfiguration() (*ValidateConfigurationResponse, error) {
	response := &ValidateConfigurationResponse{
		Issues:          []string{},
		Recommendations: []string{},
	}

	apiKeyResp, err := s.configUC.GetAPIKey()
	if err != nil {
		response.ErrorMessage = fmt.Sprintf("Failed to validate API key: %v", err)
		return response, nil
	}

	provider, authMethod, _ := s.configUC.GetProviderInfo()
	if !apiKeyResp.Configured {
		response.Issues = append(response.Issues, "API key is not configured")
		if provider == "grok" && authMethod == "oauth" {
			response.Recommendations = append(response.Recommendations, "Run 'grok login' to authenticate with Grok")
		} else {
			response.Recommendations = append(response.Recommendations, "Run 'gmit config' to configure a provider and API key")
		}
	}

	modelResp, err := s.configUC.GetDefaultModel()
	if err != nil {
		response.ErrorMessage = fmt.Sprintf("Failed to validate default model: %v", err)
		return response, nil
	}

	if !modelResp.Configured {
		response.Recommendations = append(response.Recommendations, "Consider setting a default model with 'gommit config set-model <model-id>' for faster commit generation")
	}

	modelsResp, err := s.configUC.GetAvailableModels()
	if err != nil {
		response.Issues = append(response.Issues, "Cannot fetch available models - check your API key and internet connection")
	} else if len(modelsResp.Models) == 0 {
		response.Issues = append(response.Issues, "No models available - check your API key")
	}

	response.Valid = len(response.Issues) == 0

	return response, nil
}
