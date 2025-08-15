package usecases

import (
	"errors"
	"testing"

	"github.com/PedroHercules/gommit/pkg/domain/entities"
	"github.com/PedroHercules/gommit/pkg/domain/repositories"
)

// Mock implementations for testing
type mockConfigRepository struct {
	config   *entities.Config
	saveErr  error
	loadErr  error
	deleteErr error
	exists   bool
}

func (m *mockConfigRepository) Save(config *entities.Config) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	m.config = config
	return nil
}

func (m *mockConfigRepository) Load() (*entities.Config, error) {
	if m.loadErr != nil {
		return nil, m.loadErr
	}
	if m.config == nil {
		return &entities.Config{}, nil
	}
	return m.config, nil
}

func (m *mockConfigRepository) Exists() bool {
	return m.exists
}

func (m *mockConfigRepository) SaveAPIKey(apiKey string) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	if m.config == nil {
		m.config = &entities.Config{}
	}
	m.config.APIKey = apiKey
	return nil
}

func (m *mockConfigRepository) LoadAPIKey() (string, error) {
	if m.loadErr != nil {
		return "", m.loadErr
	}
	if m.config == nil {
		return "", nil
	}
	return m.config.APIKey, nil
}

func (m *mockConfigRepository) DeleteAPIKey() error {
	return m.deleteErr
}

func (m *mockConfigRepository) SaveDefaultModel(model string) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	if m.config == nil {
		m.config = &entities.Config{}
	}
	m.config.DefaultModel = model
	return nil
}

func (m *mockConfigRepository) LoadDefaultModel() (string, error) {
	if m.loadErr != nil {
		return "", m.loadErr
	}
	if m.config == nil {
		return "", nil
	}
	return m.config.DefaultModel, nil
}

func (m *mockConfigRepository) DeleteDefaultModel() error {
	return m.deleteErr
}

type mockLLMRepository struct {
	testConnectionResult bool
	testConnectionErr    error
	models              []repositories.LLMModel
	getModelsErr        error
}

func (m *mockLLMRepository) GenerateCommitMessage(diff *entities.GitDiff, model string) (*repositories.LLMResponse, error) {
	return nil, nil
}

func (m *mockLLMRepository) GetAvailableModels() ([]repositories.LLMModel, error) {
	return m.models, m.getModelsErr
}

func (m *mockLLMRepository) ValidateModel(modelID string) (bool, error) {
	return true, nil
}

func (m *mockLLMRepository) GetBestModel() (*repositories.LLMModel, error) {
	if len(m.models) > 0 {
		return &m.models[0], nil
	}
	return nil, errors.New("no models available")
}

func (m *mockLLMRepository) TestConnection(apiKey string) (bool, error) {
	if m.testConnectionErr != nil {
		return false, m.testConnectionErr
	}
	return m.testConnectionResult, nil
}

func (m *mockLLMRepository) SetAPIKey(apiKey string) {
	// Mock implementation
}

func (m *mockLLMRepository) GetModelInfo(modelID string) (*repositories.LLMModel, error) {
	for _, model := range m.models {
		if model.ID == modelID {
			return &model, nil
		}
	}
	return nil, errors.New("model not found")
}

func TestConfigUseCase_SetAPIKey(t *testing.T) {
	tests := []struct {
		name           string
		apiKey         string
		mockConfig     *mockConfigRepository
		mockLLM        *mockLLMRepository
		expectSuccess  bool
		expectError    string
	}{
		{
			name:   "valid api key",
			apiKey: "sk-or-v1-1234567890abcdef1234567890abcdef",
			mockConfig: &mockConfigRepository{},
			mockLLM: &mockLLMRepository{
				testConnectionResult: true,
			},
			expectSuccess: true,
		},
		{
			name:   "empty api key",
			apiKey: "",
			mockConfig: &mockConfigRepository{},
			mockLLM:    &mockLLMRepository{
				testConnectionResult: false,
			},
			expectSuccess: false,
			expectError:   "Invalid API key or service unavailable",
		},
		{
			name:   "connection test fails",
			apiKey: "sk-invalid-key",
			mockConfig: &mockConfigRepository{},
			mockLLM: &mockLLMRepository{
				testConnectionErr: errors.New("invalid api key"),
			},
			expectSuccess: false,
			expectError:   "invalid API key format",
		},
		{
			name:   "save fails",
			apiKey: "sk-or-v1-1234567890abcdef1234567890abcdef",
			mockConfig: &mockConfigRepository{
				saveErr: errors.New("save failed"),
			},
			mockLLM: &mockLLMRepository{
				testConnectionResult: true,
			},
			expectSuccess: false,
			expectError:   "Failed to save API key: save failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := NewConfigUseCase(tt.mockConfig, tt.mockLLM)
			req := SetAPIKeyRequest{APIKey: tt.apiKey}

			resp, err := uc.SetAPIKey(req)

			if tt.expectSuccess {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
					return
				}
				if !resp.Success {
					t.Errorf("expected success but got failure: %s", resp.ErrorMessage)
				}
			} else {
				if err == nil && resp.Success {
					t.Errorf("expected failure but got success")
					return
				}
				if tt.expectError != "" {
					if err != nil && err.Error() != tt.expectError {
						t.Errorf("expected error %q, got %q", tt.expectError, err.Error())
					} else if err == nil && resp.ErrorMessage != tt.expectError {
						t.Errorf("expected error message %q, got %q", tt.expectError, resp.ErrorMessage)
					}
				}
			}
		})
	}
}

func TestConfigUseCase_GetAPIKey(t *testing.T) {
	tests := []struct {
		name           string
		mockConfig     *mockConfigRepository
		expectConfigured bool
		expectMaskedKey string
	}{
		{
			name: "api key exists",
			mockConfig: &mockConfigRepository{
				config: &entities.Config{APIKey: "sk-or-v1-1234567890abcdef1234567890abcdef"},
			},
			expectConfigured: true,
			expectMaskedKey:   "sk-or-v1...cdef",
		},
		{
			name: "api key not found",
			mockConfig: &mockConfigRepository{},
			expectConfigured: false,
			expectMaskedKey: "Not configured",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := NewConfigUseCase(tt.mockConfig, &mockLLMRepository{})

			resp, err := uc.GetAPIKey()

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if resp.Configured != tt.expectConfigured {
				t.Errorf("expected Configured %v, got %v", tt.expectConfigured, resp.Configured)
			}

			if resp.MaskedAPIKey != tt.expectMaskedKey {
				t.Errorf("expected MaskedAPIKey %q, got %q", tt.expectMaskedKey, resp.MaskedAPIKey)
			}
		})
	}
}