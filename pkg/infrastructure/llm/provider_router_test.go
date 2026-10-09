package llm

import (
	"testing"

	"github.com/PedroHercules/gommit/pkg/domain/entities"
	"github.com/PedroHercules/gommit/pkg/domain/repositories"
)

type routerTestRepository struct{ name string }

func (r *routerTestRepository) GenerateCommitMessage(_ *entities.GitDiff, model string) (*repositories.LLMResponse, error) {
	return &repositories.LLMResponse{Success: true, Model: r.name + ":" + model}, nil
}
func (r *routerTestRepository) GeneratePRDescription(_ *entities.GitDiff, model string) (*repositories.LLMResponse, error) {
	return &repositories.LLMResponse{Success: true, Model: r.name + ":" + model}, nil
}
func (r *routerTestRepository) GetAvailableModels() ([]repositories.LLMModel, error) {
	return []repositories.LLMModel{{ID: r.name, Available: true}}, nil
}
func (r *routerTestRepository) ValidateModel(model string) (bool, error) { return model == r.name, nil }
func (r *routerTestRepository) GetBestModel() (*repositories.LLMModel, error) {
	return &repositories.LLMModel{ID: r.name}, nil
}
func (r *routerTestRepository) TestConnection(string) (bool, error) { return true, nil }
func (r *routerTestRepository) SetAPIKey(string)                    {}
func (r *routerTestRepository) GetModelInfo(model string) (*repositories.LLMModel, error) {
	return &repositories.LLMModel{ID: model}, nil
}

func TestProviderRouterSelectsGrokAuthBackend(t *testing.T) {
	openRouter := &routerTestRepository{name: "openrouter"}
	grokAPI := &routerTestRepository{name: "grok-api"}
	grokOAuth := &routerTestRepository{name: "grok-oauth"}
	router := &ProviderRouter{providers: map[string]map[string]repositories.LLMRepository{
		"openrouter": {"api_key": openRouter},
		"grok":       {"api_key": grokAPI, "oauth": grokOAuth},
	}, active: openRouter}

	if err := router.ConfigureProvider("grok", "oauth"); err != nil {
		t.Fatalf("ConfigureProvider() error = %v", err)
	}
	response, err := router.GenerateCommitMessage(&entities.GitDiff{}, "grok-4.7")
	if err != nil {
		t.Fatalf("GenerateCommitMessage() error = %v", err)
	}
	if response.Model != "grok-oauth:grok-4.7" {
		t.Fatalf("generated with %q, want OAuth backend", response.Model)
	}
}
