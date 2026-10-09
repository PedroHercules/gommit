package config_services

import "github.com/PedroHercules/gommit/pkg/domain/usecases"

type ServiceContainer struct {
	apiKeyService     *apiKeyService
	modelService      *modelService
	validationService *validationService
}

type ConfigService struct {
	services *ServiceContainer
}

func NewConfigService(configUC *usecases.ConfigUseCase) *ConfigService {
	container := &ServiceContainer{
		apiKeyService:     newAPIKeyService(configUC),
		modelService:      newModelService(configUC),
		validationService: newValidationService(configUC),
	}

	return &ConfigService{
		services: container,
	}
}

func (c *ConfigService) SetupAPIKey(req SetupAPIKeyRequest) (*SetupAPIKeyResponse, error) {
	return c.services.apiKeyService.SetupAPIKey(req)
}

func (c *ConfigService) GetAPIKeyInfo() (*GetAPIKeyResponse, error) {
	return c.services.apiKeyService.GetAPIKeyInfo()
}

func (c *ConfigService) RemoveAPIKey() error {
	return c.services.apiKeyService.RemoveAPIKey()
}

func (c *ConfigService) SetupDefaultModel(req SetupModelRequest) (*SetupModelResponse, error) {
	return c.services.modelService.SetupDefaultModel(req)
}

func (c *ConfigService) GetDefaultModelInfo() (*GetDefaultModelResponse, error) {
	return c.services.modelService.GetDefaultModelInfo()
}

func (c *ConfigService) RemoveDefaultModel() error {
	return c.services.modelService.RemoveDefaultModel()
}

func (c *ConfigService) GetAvailableModels() (*GetAvailableModelsResponse, error) {
	return c.services.modelService.GetAvailableModels()
}

func (c *ConfigService) GetAvailableModelsFor(provider, authMethod, apiKey string) (*GetAvailableModelsResponse, error) {
	response, err := c.services.apiKeyService.configUC.GetAvailableModelsFor(provider, authMethod, apiKey)
	if err != nil {
		return nil, err
	}
	return &GetAvailableModelsResponse{Models: response.Models, ErrorMessage: response.ErrorMessage}, nil
}

func (c *ConfigService) ConfigureProvider(provider, authMethod, apiKey string) error {
	return c.services.apiKeyService.configUC.ConfigureProvider(provider, authMethod, apiKey)
}

func (c *ConfigService) SetActiveProvider(provider string) error {
	return c.services.apiKeyService.configUC.SetActiveProvider(provider)
}

func (c *ConfigService) SetProviderDefaultModel(provider, model string) (*SetupModelResponse, error) {
	ucResponse, err := c.services.apiKeyService.configUC.SetProviderDefaultModel(provider, model)
	if err != nil {
		return nil, err
	}
	return &SetupModelResponse{Success: ucResponse.Success, Message: ucResponse.Message, ErrorMessage: ucResponse.ErrorMessage}, nil
}

func (c *ConfigService) GetConfigSummary() (*ConfigSummaryResponse, error) {
	return c.services.validationService.GetConfigSummary()
}

func (c *ConfigService) ValidateConfiguration() (*ValidateConfigurationResponse, error) {
	return c.services.validationService.ValidateConfiguration()
}
