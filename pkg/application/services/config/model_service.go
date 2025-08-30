package config_services

import (
	"github.com/PedroHercules/gommit/pkg/domain/repositories"
	"github.com/PedroHercules/gommit/pkg/domain/usecases"
)

type SetupModelRequest struct {
	Model        string
	ValidateModel bool
}

type SetupModelResponse struct {
	Success      bool
	Message      string
	ErrorMessage string
	ModelInfo    *repositories.LLMModel
}

type GetDefaultModelResponse struct {
	Configured   bool
	Model        string
	ErrorMessage string
}

type GetAvailableModelsResponse struct {
	Models       []repositories.LLMModel
	ErrorMessage string
}

type modelService struct {
	configUC *usecases.ConfigUseCase
}

func newModelService(configUC *usecases.ConfigUseCase) *modelService {
	return &modelService{
		configUC: configUC,
	}
}

func (s *modelService) SetupDefaultModel(req SetupModelRequest) (*SetupModelResponse, error) {
	response := &SetupModelResponse{}

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

func (s *modelService) GetDefaultModelInfo() (*GetDefaultModelResponse, error) {
	ucResp, err := s.configUC.GetDefaultModel()
	if err != nil {
		return &GetDefaultModelResponse{
			ErrorMessage: err.Error(),
		}, nil
	}

	return &GetDefaultModelResponse{
		Configured: ucResp.Configured,
		Model:      ucResp.Model,
	}, nil
}

func (s *modelService) RemoveDefaultModel() error {
	return s.configUC.RemoveDefaultModel()
}

func (s *modelService) GetAvailableModels() (*GetAvailableModelsResponse, error) {
	ucResp, err := s.configUC.GetAvailableModels()
	if err != nil {
		return &GetAvailableModelsResponse{
			ErrorMessage: err.Error(),
		}, nil
	}

	return &GetAvailableModelsResponse{
		Models: ucResp.Models,
	}, nil
}