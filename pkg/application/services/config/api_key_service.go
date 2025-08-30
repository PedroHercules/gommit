package config_services

import "github.com/PedroHercules/gommit/pkg/domain/usecases"

type SetupAPIKeyRequest struct {
	APIKey      string
	ValidateKey bool
}

type SetupAPIKeyResponse struct {
	Success      bool
	Message      string
	ErrorMessage string
	KeyMasked    string
}

type GetAPIKeyResponse struct {
	Configured   bool
	MaskedAPIKey string
	ErrorMessage string
}

type apiKeyService struct {
	configUC *usecases.ConfigUseCase
}

func newAPIKeyService(configUC *usecases.ConfigUseCase) *apiKeyService {
	return &apiKeyService{
		configUC: configUC,
	}
}

func (s *apiKeyService) SetupAPIKey(req SetupAPIKeyRequest) (*SetupAPIKeyResponse, error) {
	response := &SetupAPIKeyResponse{}

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

	getKeyResp, err := s.configUC.GetAPIKey()
	if err == nil && getKeyResp.Configured {
		response.KeyMasked = getKeyResp.MaskedAPIKey
	}

	response.Success = true
	response.Message = setKeyResp.Message

	return response, nil
}

func (s *apiKeyService) GetAPIKeyInfo() (*GetAPIKeyResponse, error) {
	ucResp, err := s.configUC.GetAPIKey()
	if err != nil {
		return &GetAPIKeyResponse{
			ErrorMessage: err.Error(),
		}, nil
	}

	return &GetAPIKeyResponse{
		Configured:   ucResp.Configured,
		MaskedAPIKey: ucResp.MaskedAPIKey,
	}, nil
}

func (s *apiKeyService) RemoveAPIKey() error {
	return s.configUC.RemoveAPIKey()
}