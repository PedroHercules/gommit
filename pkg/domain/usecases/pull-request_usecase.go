package usecases

import (
	"errors"
	"fmt"
	"log"

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

	// Step 1: Fetch latest changes from remote
	log.Printf("Fetching latest changes from remote repository...")
	err := uc.gitRepo.FetchRemote()
	if err != nil {
		log.Printf("Warning: Failed to fetch from remote: %v", err)
		response.Warnings = append(response.Warnings, fmt.Sprintf("Could not fetch latest changes: %v", err))
	} else {
		log.Printf("Successfully fetched latest changes from remote")
	}

	// Step 2: Update base branch with latest changes
	log.Printf("Updating base branch '%s' with latest changes...", req.BaseBranch)
	err = uc.gitRepo.UpdateBaseBranch(req.BaseBranch)
	if err != nil {
		log.Printf("Warning: Failed to update base branch: %v", err)
		response.Warnings = append(response.Warnings, fmt.Sprintf("Could not update base branch '%s': %v", req.BaseBranch, err))
	} else {
		log.Printf("Successfully updated base branch '%s' with latest changes", req.BaseBranch)
	}

	// Step 3: Get differences between current branch and updated base branch
	log.Printf("Analyzing differences between '%s' and current branch...", req.BaseBranch)
	diff, err := uc.gitRepo.GetDiffBetweenBranches(req.BaseBranch, "HEAD")
	if err != nil {
		return nil, fmt.Errorf("failed to get diff: %w", err)
	}

	// Step 4: Load configuration
	config, err := uc.configRepo.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// Step 5: Validate API key
	if !config.HasAPIKey() {
		return nil, errors.New("API key not configured. Run 'gommit config set-key <your-api-key>'")
	}

	// Step 6: Determine which model to use
	modelToUse := req.Model
	if modelToUse == "" {
		modelToUse = config.DefaultModel
	}

	// Step 7: Set API key for LLM repository
	uc.llmRepo.SetAPIKey(config.APIKey)

	// Step 8: Generate PR preview with AI
	log.Printf("Generating PR description using AI model: %s", modelToUse)
	llmResponse, err := uc.llmRepo.GeneratePRDescription(diff, modelToUse)
	if err != nil {
		return nil, fmt.Errorf("failed to generate PR preview: %w", err)
	}

	response.LLMResponse = llmResponse

	if !llmResponse.Success {
		return nil, fmt.Errorf("LLM generation failed: %s", llmResponse.Error)
	}

	log.Printf("Successfully generated PR description")

	// Extract title and body from LLM response
	body := llmResponse.Message

	pullRequest := &entities.PullRequest{
		Body: body,
	}

	response.PullRequest = pullRequest

	response.Success = true

	return response, nil
}
