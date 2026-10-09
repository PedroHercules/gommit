package llm

import (
	"errors"
	"fmt"

	"github.com/PedroHercules/gommit/pkg/domain/entities"
	"github.com/PedroHercules/gommit/pkg/domain/repositories"
)

// ProviderRouter delegates the shared LLM interface to a configured provider.
type ProviderRouter struct {
	providers map[string]map[string]repositories.LLMRepository
	active    repositories.LLMRepository
}

func NewProviderRouter(openRouter *OpenRouterRepository, grokAPI *GrokAPIRepository, grokCLI *GrokCLIRepository) *ProviderRouter {
	openRouterRepo := repositories.LLMRepository(openRouter)
	router := &ProviderRouter{
		providers: map[string]map[string]repositories.LLMRepository{
			"openrouter": {"api_key": openRouterRepo},
			"grok":       {"api_key": grokAPI, "oauth": grokCLI},
		},
		active: openRouterRepo,
	}
	return router
}

func (r *ProviderRouter) ConfigureProvider(provider, authMethod string) error {
	methods, ok := r.providers[provider]
	if !ok {
		return fmt.Errorf("unsupported LLM provider %q", provider)
	}
	repo, ok := methods[authMethod]
	if !ok {
		return fmt.Errorf("unsupported authentication method %q for %s", authMethod, provider)
	}
	if configurable, ok := repo.(interface{ ConfigureProvider(string, string) error }); ok {
		if err := configurable.ConfigureProvider(provider, authMethod); err != nil {
			return err
		}
	}
	r.active = repo
	return nil
}

func (r *ProviderRouter) selected() (repositories.LLMRepository, error) {
	if r.active == nil {
		return nil, errors.New("no LLM provider is configured")
	}
	return r.active, nil
}

func (r *ProviderRouter) GenerateCommitMessage(diff *entities.GitDiff, model string) (*repositories.LLMResponse, error) {
	repo, err := r.selected()
	if err != nil {
		return nil, err
	}
	return repo.GenerateCommitMessage(diff, model)
}

func (r *ProviderRouter) GeneratePRDescription(diff *entities.GitDiff, model string) (*repositories.LLMResponse, error) {
	repo, err := r.selected()
	if err != nil {
		return nil, err
	}
	return repo.GeneratePRDescription(diff, model)
}

func (r *ProviderRouter) GetAvailableModels() ([]repositories.LLMModel, error) {
	repo, err := r.selected()
	if err != nil {
		return nil, err
	}
	return repo.GetAvailableModels()
}

func (r *ProviderRouter) ValidateModel(modelID string) (bool, error) {
	repo, err := r.selected()
	if err != nil {
		return false, err
	}
	return repo.ValidateModel(modelID)
}

func (r *ProviderRouter) GetBestModel() (*repositories.LLMModel, error) {
	repo, err := r.selected()
	if err != nil {
		return nil, err
	}
	return repo.GetBestModel()
}

func (r *ProviderRouter) TestConnection(apiKey string) (bool, error) {
	repo, err := r.selected()
	if err != nil {
		return false, err
	}
	return repo.TestConnection(apiKey)
}

func (r *ProviderRouter) SetAPIKey(apiKey string) {
	if repo, err := r.selected(); err == nil {
		repo.SetAPIKey(apiKey)
	}
}

func (r *ProviderRouter) GetModelInfo(modelID string) (*repositories.LLMModel, error) {
	repo, err := r.selected()
	if err != nil {
		return nil, err
	}
	return repo.GetModelInfo(modelID)
}
