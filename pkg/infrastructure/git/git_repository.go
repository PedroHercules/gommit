// Package git provides infrastructure implementations for Git operations.
// This package implements the GitRepository interface using actual Git commands.
package git

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/PedroHercules/gommit/pkg/domain/entities"
)

// CommandGitRepository implements the GitRepository interface using Git CLI commands.
// This implementation executes actual Git commands to interact with the repository.
type CommandGitRepository struct {
	workingDir string // The working directory for Git commands
}

// NewCommandGitRepository creates a new Git repository implementation.
// If workingDir is empty, it uses the current working directory.
func NewCommandGitRepository(workingDir string) (*CommandGitRepository, error) {
	if workingDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get current working directory: %w", err)
		}
		workingDir = cwd
	}

	// Validate that the directory exists
	if _, err := os.Stat(workingDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("working directory does not exist: %s", workingDir)
	}

	return &CommandGitRepository{
		workingDir: workingDir,
	}, nil
}

// GetStagedDiff returns the diff of staged changes.
func (r *CommandGitRepository) GetStagedDiff() (*entities.GitDiff, error) {
	output, err := r.runGitCommand("diff", "--cached")
	if err != nil {
		return nil, fmt.Errorf("failed to get staged diff: %w", err)
	}

	return entities.NewGitDiff(output)
}

// GetWorkingDiff returns the diff of working directory changes.
func (r *CommandGitRepository) GetWorkingDiff() (*entities.GitDiff, error) {
	output, err := r.runGitCommand("diff")
	if err != nil {
		return nil, fmt.Errorf("failed to get working diff: %w", err)
	}

	return entities.NewGitDiff(output)
}

// GetDiffBetweenBranches returns the diff between two branches.
func (r *CommandGitRepository) GetDiffBetweenBranches(branch1, branch2 string) (*entities.GitDiff, error) {
	output, err := r.runGitCommand("log", branch1+".."+branch2)
	if err != nil {
		return nil, fmt.Errorf("failed to get diff between branches: %w", err)
	}

	return entities.NewGitDiff(output)
}

// Commit creates a new commit with the given message.
func (r *CommandGitRepository) Commit(message string) (string, error) {
	if message == "" {
		return "", errors.New("commit message cannot be empty")
	}

	// Execute git commit command
	_, err := r.runGitCommand("commit", "-m", message)
	if err != nil {
		return "", fmt.Errorf("failed to commit: %w", err)
	}

	// Get the commit hash
	hash, err := r.runGitCommand("rev-parse", "HEAD")
	if err != nil {
		return "", fmt.Errorf("failed to get commit hash: %w", err)
	}

	return strings.TrimSpace(hash), nil
}

// HasStagedChanges checks if there are any staged changes.
func (r *CommandGitRepository) HasStagedChanges() (bool, error) {
	output, err := r.runGitCommand("diff", "--cached", "--name-only")
	if err != nil {
		return false, fmt.Errorf("failed to check staged changes: %w", err)
	}

	return strings.TrimSpace(output) != "", nil
}

// HasWorkingChanges checks if there are any working directory changes.
func (r *CommandGitRepository) HasWorkingChanges() (bool, error) {
	output, err := r.runGitCommand("diff", "--name-only")
	if err != nil {
		return false, fmt.Errorf("failed to check working changes: %w", err)
	}

	return strings.TrimSpace(output) != "", nil
}

// GetCurrentBranch returns the name of the current branch.
func (r *CommandGitRepository) GetCurrentBranch() (string, error) {
	output, err := r.runGitCommand("branch", "--show-current")
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}

	branch := strings.TrimSpace(output)
	if branch == "" {
		// Fallback for older Git versions or detached HEAD
		output, err := r.runGitCommand("rev-parse", "--abbrev-ref", "HEAD")
		if err != nil {
			return "unknown", nil
		}
		branch = strings.TrimSpace(output)
	}

	return branch, nil
}

// GetLastCommitMessage returns the message of the last commit.
func (r *CommandGitRepository) GetLastCommitMessage() (string, error) {
	output, err := r.runGitCommand("log", "-1", "--pretty=format:%s")
	if err != nil {
		return "", fmt.Errorf("failed to get last commit message: %w", err)
	}

	return strings.TrimSpace(output), nil
}

// IsGitRepository checks if the current directory is a Git repository.
func (r *CommandGitRepository) IsGitRepository() bool {
	_, err := r.runGitCommand("rev-parse", "--git-dir")
	return err == nil
}

// GetRepositoryRoot returns the root directory of the Git repository.
func (r *CommandGitRepository) GetRepositoryRoot() (string, error) {
	output, err := r.runGitCommand("rev-parse", "--show-toplevel")
	if err != nil {
		return "", fmt.Errorf("failed to get repository root: %w", err)
	}

	return strings.TrimSpace(output), nil
}

// runGitCommand executes a Git command and returns its output.
func (r *CommandGitRepository) runGitCommand(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = r.workingDir

	// Capture both stdout and stderr
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Include the Git error message in our error
		gitError := strings.TrimSpace(string(output))
		if gitError != "" {
			return "", fmt.Errorf("git command failed: %s", gitError)
		}
		return "", fmt.Errorf("git command failed: %w", err)
	}

	return string(output), nil
}

// GetWorkingDir returns the working directory used for Git commands.
func (r *CommandGitRepository) GetWorkingDir() string {
	return r.workingDir
}

// SetWorkingDir changes the working directory for Git commands.
func (r *CommandGitRepository) SetWorkingDir(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return fmt.Errorf("directory does not exist: %s", dir)
	}

	r.workingDir = dir
	return nil
}

// GetGitVersion returns the version of Git being used.
func (r *CommandGitRepository) GetGitVersion() (string, error) {
	output, err := r.runGitCommand("--version")
	if err != nil {
		return "", fmt.Errorf("failed to get Git version: %w", err)
	}

	return strings.TrimSpace(output), nil
}

// ValidateGitInstallation checks if Git is properly installed and accessible.
func (r *CommandGitRepository) ValidateGitInstallation() error {
	// Check if git command is available
	_, err := exec.LookPath("git")
	if err != nil {
		return errors.New("Git is not installed or not in PATH")
	}

	// Try to get Git version
	_, err = r.GetGitVersion()
	if err != nil {
		return fmt.Errorf("Git installation appears to be corrupted: %w", err)
	}

	return nil
}
