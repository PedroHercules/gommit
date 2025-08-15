// Package services provides application services that orchestrate use cases.
// These services act as a facade for the domain layer and handle cross-cutting concerns.
package services

import (
	"fmt"

	"github.com/PedroHercules/gommit/pkg/domain/usecases"
)

// CommitService provides high-level operations for commit management.
// This service orchestrates multiple use cases and handles application-level logic.
type CommitService struct {
	generateCommitUC *usecases.GenerateCommitUseCase
	commitUC         *usecases.CommitUseCase
	configUC         *usecases.ConfigUseCase
}

// NewCommitService creates a new commit service.
func NewCommitService(
	generateCommitUC *usecases.GenerateCommitUseCase,
	commitUC *usecases.CommitUseCase,
	configUC *usecases.ConfigUseCase,
) *CommitService {
	return &CommitService{
		generateCommitUC: generateCommitUC,
		commitUC:         commitUC,
		configUC:         configUC,
	}
}

// GenerateAndCommitRequest represents the input for generating and committing.
type GenerateAndCommitRequest struct {
	Model       string // Optional: specific model to use
	AutoCommit  bool   // Whether to automatically commit after generation
	DryRun      bool   // Whether to perform a dry run
	Force       bool   // Whether to force generation even with warnings
}

// GenerateAndCommitResponse represents the output of the generate and commit operation.
type GenerateAndCommitResponse struct {
	CommitMessage    string
	CommitHash       string
	Model            string
	TokensUsed       int
	Warnings         []string
	Success          bool
	Committed        bool
	ErrorMessage     string
	ChangesSummary   string
}

// GenerateAndCommit generates a commit message and optionally commits the changes.
// This is the main workflow for the application.
func (s *CommitService) GenerateAndCommit(req GenerateAndCommitRequest) (*GenerateAndCommitResponse, error) {
	response := &GenerateAndCommitResponse{
		Warnings: []string{},
	}

	// Step 1: Generate the commit message
	generateReq := usecases.GenerateCommitRequest{
		Model: req.Model,
		Force: req.Force,
	}

	generateResp, err := s.generateCommitUC.Execute(generateReq)
	if err != nil {
		response.ErrorMessage = err.Error()
		return response, nil
	}

	if !generateResp.Success {
		response.ErrorMessage = generateResp.ErrorMessage
		return response, nil
	}

	// Populate response with generation results
	response.CommitMessage = generateResp.Commit.Message
	response.Model = generateResp.LLMResponse.Model
	response.TokensUsed = generateResp.LLMResponse.TokensUsed
	response.Warnings = generateResp.Warnings
	response.ChangesSummary = generateResp.Diff.GetChangesSummary()
	response.Success = true

	// Step 2: If auto-commit is enabled, commit the changes
	if req.AutoCommit && !req.DryRun {
		commitReq := usecases.CommitChangesRequest{
			Message: generateResp.Commit.Message,
			DryRun:  req.DryRun,
		}

		commitResp, err := s.commitUC.CommitChanges(commitReq)
		if err != nil {
			response.ErrorMessage = fmt.Sprintf("Generated commit message but failed to commit: %v", err)
			return response, nil
		}

		if !commitResp.Success {
			response.ErrorMessage = fmt.Sprintf("Generated commit message but failed to commit: %s", commitResp.ErrorMessage)
			return response, nil
		}

		response.CommitHash = commitResp.CommitHash
		response.Committed = true
	}

	return response, nil
}

// ValidateCommitMessage validates a commit message against best practices.
func (s *CommitService) ValidateCommitMessage(message string) (*usecases.ValidateCommitMessageResponse, error) {
	req := usecases.ValidateCommitMessageRequest{
		Message: message,
	}

	return s.commitUC.ValidateCommitMessage(req)
}

// GetRepositoryStatus returns the current status of the Git repository.
func (s *CommitService) GetRepositoryStatus() (*usecases.GetStatusResponse, error) {
	return s.commitUC.GetStatus()
}

// GetCommitPreview returns a preview of what would be committed.
func (s *CommitService) GetCommitPreview() (*usecases.GetStatusResponse, error) {
	// Get repository status which includes staged files
	return s.commitUC.GetStatus()
}

// CommitWithMessage commits the staged changes with a specific message.
func (s *CommitService) CommitWithMessage(message string, dryRun bool) (*usecases.CommitChangesResponse, error) {
	req := usecases.CommitChangesRequest{
		Message: message,
		DryRun:  dryRun,
	}

	return s.commitUC.CommitChanges(req)
}