package config_services

import "github.com/PedroHercules/gommit/pkg/domain/usecases"

type ServiceContainer struct {
	apiKeyService    *apiKeyService
	modelService     *modelService
	validationService *validationService
}

type ConfigService struct {
	services *ServiceContainer
}

func NewConfigService(configUC *usecases.ConfigUseCase) *ConfigService {
	container := &ServiceContainer{
		apiKeyService:    newAPIKeyService(configUC),
		modelService:     newModelService(configUC),
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

func (c *ConfigService) GetConfigSummary() (*ConfigSummaryResponse, error) {
	return c.services.validationService.GetConfigSummary()
}

func (c *ConfigService) ValidateConfiguration() (*ValidateConfigurationResponse, error) {
	return c.services.validationService.ValidateConfiguration()
}