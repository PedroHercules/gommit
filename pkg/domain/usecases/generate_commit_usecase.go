// Package usecases contains the business logic use cases.
// Use cases orchestrate the flow of data to and from entities,
// and direct those entities to use their business rules to achieve
// the goals of the use case.
package usecases

import (
	"errors"
	"fmt"

	"github.com/PedroHercules/gommit/pkg/domain/entities"
	"github.com/PedroHercules/gommit/pkg/domain/repositories"
)

// GenerateCommitUseCase handles the business logic for generating commit messages.
// This use case orchestrates the interaction between Git operations,
// LLM services, and configuration management.
type GenerateCommitUseCase struct {
	gitRepo    repositories.GitRepository
	llmRepo    repositories.LLMRepository
	configRepo repositories.ConfigRepository
}

// NewGenerateCommitUseCase creates a new instance of GenerateCommitUseCase.
// This constructor follows the dependency injection pattern.
func NewGenerateCommitUseCase(
	gitRepo repositories.GitRepository,
	llmRepo repositories.LLMRepository,
	configRepo repositories.ConfigRepository,
) *GenerateCommitUseCase {
	return &GenerateCommitUseCase{
		gitRepo:    gitRepo,
		llmRepo:    llmRepo,
		configRepo: configRepo,
	}
}

// GenerateCommitRequest represents the input for generating a commit.
type GenerateCommitRequest struct {
	Model string // Optional: specific model to use
	Force bool   // Whether to force generation even with warnings
}

// GenerateCommitResponse represents the output of commit generation.
type GenerateCommitResponse struct {
	Commit       *entities.Commit
	LLMResponse  *repositories.LLMResponse
	Diff         *entities.GitDiff
	Warnings     []string
	Success      bool
	ErrorMessage string
}

// Execute performs the commit message generation use case.
// This method implements the main business logic for generating commits.
func (uc *GenerateCommitUseCase) Execute(req GenerateCommitRequest) (*GenerateCommitResponse, error) {
	response := &GenerateCommitResponse{
		Warnings: []string{},
	}

	// Step 1: Validate we're in a Git repository
	if !uc.gitRepo.IsGitRepository() {
		return nil, errors.New("not in a Git repository")
	}

	// Step 2: Check for staged changes
	hasStagedChanges, err := uc.gitRepo.HasStagedChanges()
	if err != nil {
		return nil, fmt.Errorf("failed to check staged changes: %w", err)
	}

	if !hasStagedChanges {
		return nil, errors.New("no staged changes found. Run 'git add <file>' to stage files")
	}

	// Step 3: Get the staged diff
	diff, err := uc.gitRepo.GetStagedDiff()
	if err != nil {
		return nil, fmt.Errorf("failed to get staged diff: %w", err)
	}

	response.Diff = diff

	// Step 4: Validate the diff
	if validateErr := diff.Validate(); validateErr != nil {
		return nil, validateErr
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

	// Step 9: Generate commit message using LLM
	llmResponse, err := uc.llmRepo.GenerateCommitMessage(diff, modelToUse)
	if err != nil {
		return nil, fmt.Errorf("failed to generate commit message: %w", err)
	}

	response.LLMResponse = llmResponse

	if !llmResponse.Success {
		return nil, fmt.Errorf("LLM generation failed: %s", llmResponse.Error)
	}

	// Step 10: Create commit entity
	commit, err := entities.NewCommit(llmResponse.Message, "gommit", diff.Files)
	if err != nil {
		return nil, fmt.Errorf("failed to create commit entity: %w", err)
	}

	response.Commit = commit

	// Step 11: Add warnings if applicable
	if !commit.IsConventional() {
		response.Warnings = append(response.Warnings, "Generated commit message doesn't follow conventional commits format")
	}

	if diff.GetFileCount() > 10 {
		response.Warnings = append(response.Warnings, "Large number of files changed - consider splitting into smaller commits")
	}

	response.Success = true
	return response, nil
}

// GetCommitPreview returns a preview of what the commit would look like.
// This is useful for showing the user what will be committed before confirmation.
func (uc *GenerateCommitUseCase) GetCommitPreview() (*entities.GitDiff, error) {
	if !uc.gitRepo.IsGitRepository() {
		return nil, errors.New("not in a Git repository")
	}

	return uc.gitRepo.GetStagedDiff()
}
