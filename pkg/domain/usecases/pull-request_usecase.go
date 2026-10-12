package usecases

import (
	"fmt"
	"log"
	"strings"

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
	if req == nil || strings.TrimSpace(req.BaseBranch) == "" {
		return nil, fmt.Errorf("base branch is required")
	}
	response := &GeneratePrResponse{
		Warnings: []string{},
	}

	// Step 1: Fetch latest changes from remote
	log.Printf("Fetching latest changes from remote repository...")
	err := uc.gitRepo.FetchRemote()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch origin; PR generation requires origin/%s: %w", req.BaseBranch, err)
	}
	log.Printf("Fetched latest changes from origin")

	// Compare against the fetched remote-tracking ref without checking out or pulling.
	baseRef := "origin/" + req.BaseBranch
	log.Printf("Analyzing code changes between '%s' and current branch...", baseRef)
	diff, err := uc.gitRepo.GetDiffBetweenBranches(baseRef, "HEAD")
	if err != nil {
		return nil, fmt.Errorf("failed to get diff: %w", err)
	}

	// Step 4: Load configuration
	config, err := uc.configRepo.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	if err := configureLLMForConfig(uc.llmRepo, config); err != nil {
		return nil, err
	}

	// Step 6: Determine which model to use
	modelToUse := req.Model
	if modelToUse == "" {
		modelToUse = config.DefaultModel
	}

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
