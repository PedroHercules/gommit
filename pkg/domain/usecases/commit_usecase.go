// Package usecases contains the business logic use cases.
package usecases

import (
	"errors"
	"fmt"
	"strings"

	"github.com/PedroHercules/gommit/pkg/domain/entities"
	"github.com/PedroHercules/gommit/pkg/domain/repositories"
)

// CommitUseCase handles the business logic for Git commit operations.
// This use case manages the actual committing of changes to the repository.
type CommitUseCase struct {
	gitRepo repositories.GitRepository
}

// NewCommitUseCase creates a new instance of CommitUseCase.
func NewCommitUseCase(gitRepo repositories.GitRepository) *CommitUseCase {
	return &CommitUseCase{
		gitRepo: gitRepo,
	}
}

// CommitChangesRequest represents the input for committing changes.
type CommitChangesRequest struct {
	Message string // The commit message to use
	DryRun  bool   // Whether to perform a dry run (don't actually commit)
}

// CommitChangesResponse represents the output of committing changes.
type CommitChangesResponse struct {
	CommitHash   string
	Success      bool
	Message      string
	ErrorMessage string
}

// CommitChanges performs the actual Git commit operation.
func (uc *CommitUseCase) CommitChanges(req CommitChangesRequest) (*CommitChangesResponse, error) {
	response := &CommitChangesResponse{}

	// Step 1: Validate we're in a Git repository
	if !uc.gitRepo.IsGitRepository() {
		return nil, errors.New("not in a Git repository")
	}

	// Step 2: Validate commit message
	if req.Message == "" {
		response.ErrorMessage = "commit message cannot be empty"
		return response, nil
	}

	// Step 3: Check for staged changes
	hasStagedChanges, err := uc.gitRepo.HasStagedChanges()
	if err != nil {
		response.ErrorMessage = fmt.Sprintf("failed to check staged changes: %v", err)
		return response, nil
	}

	if !hasStagedChanges {
		response.ErrorMessage = "no staged changes to commit"
		return response, nil
	}

	// Step 4: If dry run, just return success without committing
	if req.DryRun {
		response.Success = true
		response.Message = "Dry run: commit would be successful"
		return response, nil
	}

	// Step 5: Perform the actual commit
	commitHash, err := uc.gitRepo.Commit(req.Message)
	if err != nil {
		response.ErrorMessage = fmt.Sprintf("failed to commit changes: %v", err)
		return response, nil
	}

	response.Success = true
	response.CommitHash = commitHash
	response.Message = fmt.Sprintf("Successfully committed changes with hash: %s", commitHash)

	return response, nil
}

// GetStatusResponse represents the output of getting repository status.
type GetStatusResponse struct {
	StagedFiles   []string
	UnstagedFiles []string
	CurrentBranch string
	LastCommit    string
	IsClean       bool
	ErrorMessage  string
}

// GetStatus returns the current status of the Git repository.
func (uc *CommitUseCase) GetStatus() (*GetStatusResponse, error) {
	response := &GetStatusResponse{}

	// Step 1: Validate we're in a Git repository
	if !uc.gitRepo.IsGitRepository() {
		return nil, errors.New("not in a Git repository")
	}

	// Step 2: Get current branch
	branch, err := uc.gitRepo.GetCurrentBranch()
	if err != nil {
		response.ErrorMessage = fmt.Sprintf("failed to get current branch: %v", err)
		return response, nil
	}
	response.CurrentBranch = branch

	// Step 3: Get last commit message
	lastCommit, err := uc.gitRepo.GetLastCommitMessage()
	if err != nil {
		// This might fail if there are no commits yet, which is okay
		response.LastCommit = "No commits yet"
	} else {
		response.LastCommit = lastCommit
	}

	// Step 4: Check for staged changes
	hasStagedChanges, err := uc.gitRepo.HasStagedChanges()
	if err != nil {
		response.ErrorMessage = fmt.Sprintf("failed to check staged changes: %v", err)
		return response, nil
	}

	// Step 5: Check for working directory changes
	hasWorkingChanges, err := uc.gitRepo.HasWorkingChanges()
	if err != nil {
		response.ErrorMessage = fmt.Sprintf("failed to check working changes: %v", err)
		return response, nil
	}

	// Step 6: Get staged diff to extract file names
	if hasStagedChanges {
		stagedDiff, err := uc.gitRepo.GetStagedDiff()
		if err == nil {
			response.StagedFiles = stagedDiff.Files
		}
	}

	// Step 7: Get working diff to extract file names
	if hasWorkingChanges {
		workingDiff, err := uc.gitRepo.GetWorkingDiff()
		if err == nil {
			response.UnstagedFiles = workingDiff.Files
		}
	}

	// Step 8: Determine if repository is clean
	response.IsClean = !hasStagedChanges && !hasWorkingChanges

	return response, nil
}

// ValidateCommitMessageRequest represents the input for validating a commit message.
type ValidateCommitMessageRequest struct {
	Message string
}

// ValidateCommitMessageResponse represents the output of validating a commit message.
type ValidateCommitMessageResponse struct {
	Valid            bool
	IsConventional   bool
	Warnings         []string
	Suggestions      []string
	ErrorMessage     string
}

// ValidateCommitMessage validates a commit message against best practices.
func (uc *CommitUseCase) ValidateCommitMessage(req ValidateCommitMessageRequest) (*ValidateCommitMessageResponse, error) {
	response := &ValidateCommitMessageResponse{
		Warnings:    []string{},
		Suggestions: []string{},
	}

	// Step 1: Create a commit entity to leverage its validation logic
	commit, err := entities.NewCommit(req.Message, "validator", []string{})
	if err != nil {
		response.ErrorMessage = err.Error()
		return response, nil
	}

	response.Valid = true
	response.IsConventional = commit.IsConventional()

	// Step 2: Check for conventional commits format
	if !response.IsConventional {
		response.Warnings = append(response.Warnings, "Message doesn't follow conventional commits format")
		response.Suggestions = append(response.Suggestions, "Consider using format: type(scope): description (e.g., 'feat: add user authentication')")
	}

	// Step 3: Check message length
	summary := commit.GetSummary()
	if len(summary) > 72 {
		response.Warnings = append(response.Warnings, "Summary line is longer than 72 characters")
		response.Suggestions = append(response.Suggestions, "Keep the summary line under 72 characters for better readability")
	}

	if len(summary) < 10 {
		response.Warnings = append(response.Warnings, "Summary line is very short")
		response.Suggestions = append(response.Suggestions, "Provide a more descriptive summary of the changes")
	}

	// Step 4: Check for imperative mood
	if !uc.isImperativeMood(summary) {
		response.Warnings = append(response.Warnings, "Summary should use imperative mood")
		response.Suggestions = append(response.Suggestions, "Use imperative mood: 'add feature' instead of 'added feature' or 'adds feature'")
	}

	return response, nil
}

// isImperativeMood performs a basic check for imperative mood.
// This is a simplified implementation and could be enhanced with more sophisticated NLP.
func (uc *CommitUseCase) isImperativeMood(message string) bool {
	// Simple heuristic: check for common non-imperative patterns
	nonImperativePatterns := []string{
		"added", "fixed", "updated", "changed", "removed", "deleted",
		"adds", "fixes", "updates", "changes", "removes", "deletes",
		"adding", "fixing", "updating", "changing", "removing", "deleting",
	}

	messageLower := strings.ToLower(message)
	for _, pattern := range nonImperativePatterns {
		if strings.HasPrefix(messageLower, pattern+" ") {
			return false
		}
	}

	return true
}