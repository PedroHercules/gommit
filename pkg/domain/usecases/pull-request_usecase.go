package usecases

import (
	"errors"
	"fmt"

	"github.com/PedroHercules/gommit/pkg/domain/entities"
	"github.com/PedroHercules/gommit/pkg/domain/repositories"
)

type PullRequestUseCase struct {
	gitRepo    repositories.GitRepository
	llmRepo    repositories.LLMRepository
	configRepo repositories.ConfigRepository
}

type GeneratePullRequestRequest struct {
	Model      string // Optional: specific model to use
	Force      bool   // Whether to force generation even with warnings
	BaseBranch string
}

type GeneratePrResponse struct {
	PullRequest  *entities.PullRequest
	LLMResponse  *repositories.LLMResponse
	Diff         *entities.GitDiff
	Warnings     []string
	Success      bool
	ErrorMessage string
}

func NewPullRequestUseCase(gitRepo repositories.GitRepository, llmRepo repositories.LLMRepository, configRepo repositories.ConfigRepository) *PullRequestUseCase {
	return &PullRequestUseCase{
		gitRepo:    gitRepo,
		llmRepo:    llmRepo,
		configRepo: configRepo,
	}
}

func (uc *PullRequestUseCase) GeneratePRPreview(req *GeneratePullRequestRequest) (*GeneratePrResponse, error) {
	response := &GeneratePrResponse{
		Warnings: []string{},
	}

	// Step 1: Pegar diferenças entre branch atual e branch base
	diff, err := uc.gitRepo.GetDiffBetweenBranches("HEAD", req.BaseBranch)
	if err != nil {
		return nil, fmt.Errorf("failed to get diff: %w", err)
	}

	// Step 5: Load configuration
	config, err := uc.configRepo.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// Step 6: Validate API key
	if !config.HasAPIKey() {
		return nil, errors.New("API key not configured. Run 'gommit config set-key <your-api-key>'")
	}

	// Step 7: Determine which model to use
	modelToUse := req.Model
	if modelToUse == "" {
		modelToUse = config.DefaultModel
	}

	// Step 8: Set API key for LLM repository
	uc.llmRepo.SetAPIKey(config.APIKey)

	// Step 2: Gerar preview do PR
	llmResponse, err := uc.llmRepo.GenerateCommitMessage(diff, modelToUse)
	if err != nil {
		return nil, fmt.Errorf("failed to generate PR preview: %w", err)
	}

	response.LLMResponse = llmResponse

	if !llmResponse.Success {
		return nil, fmt.Errorf("LLM generation failed: %s", llmResponse.Error)
	}

	pullRequest := &entities.PullRequest{
		Title: "Teste",
		Body:  llmResponse.Message,
	}

	response.PullRequest = pullRequest

	response.Success = true

	return response, nil
}
