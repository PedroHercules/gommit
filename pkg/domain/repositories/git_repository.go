// Package repositories defines the interfaces for data access.
package repositories

import (
	"github.com/PedroHercules/gommit/pkg/domain/entities"
)

// GitRepository defines the interface for Git operations.
// This interface abstracts Git commands and allows for testing
// and different implementations of Git operations.
type GitRepository interface {
	// GetStagedDiff returns the diff of staged changes.
	// This is equivalent to running 'git diff --cached'.
	GetStagedDiff() (*entities.GitDiff, error)

	// GetWorkingDiff returns the diff of working directory changes.
	// This is equivalent to running 'git diff'.
	GetWorkingDiff() (*entities.GitDiff, error)

	// Commit creates a new commit with the given message.
	// Returns the commit hash if successful.
	Commit(message string) (string, error)

	// HasStagedChanges checks if there are any staged changes.
	// Returns true if there are files in the staging area.
	HasStagedChanges() (bool, error)

	// HasWorkingChanges checks if there are any working directory changes.
	// Returns true if there are modified files in the working directory.
	HasWorkingChanges() (bool, error)

	// GetCurrentBranch returns the name of the current branch.
	GetCurrentBranch() (string, error)

	// GetLastCommitMessage returns the message of the last commit.
	// This can be useful for context when generating new commit messages.
	GetLastCommitMessage() (string, error)

	// IsGitRepository checks if the current directory is a Git repository.
	// Returns true if we're inside a Git repository.
	IsGitRepository() bool

	// GetRepositoryRoot returns the root directory of the Git repository.
	GetRepositoryRoot() (string, error)
}