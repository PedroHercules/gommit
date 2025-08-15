// Package entities contains the core business entities of the application.
// These entities represent the fundamental business objects and rules.
package entities

import (
	"errors"
	"strings"
	"time"
)

// Commit represents a git commit with its message and metadata.
// This is a core domain entity that encapsulates the business rules for commits.
type Commit struct {
	Message     string    // The commit message following conventional commits format
	Author      string    // The author of the commit
	Timestamp   time.Time // When the commit was created
	FilesChanged []string  // List of files that were changed
}

// NewCommit creates a new commit entity with validation.
// It ensures the commit message follows basic business rules.
func NewCommit(message, author string, filesChanged []string) (*Commit, error) {
	if strings.TrimSpace(message) == "" {
		return nil, errors.New("commit message cannot be empty")
	}

	if strings.TrimSpace(author) == "" {
		return nil, errors.New("commit author cannot be empty")
	}

	return &Commit{
		Message:      strings.TrimSpace(message),
		Author:       strings.TrimSpace(author),
		Timestamp:    time.Now(),
		FilesChanged: filesChanged,
	}, nil
}

// IsConventional checks if the commit message follows conventional commits format.
// This implements a business rule for commit message formatting.
func (c *Commit) IsConventional() bool {
	// Basic check for conventional commit format: type(scope): description
	conventionalPrefixes := []string{"feat", "fix", "docs", "style", "refactor", "test", "chore"}
	
	for _, prefix := range conventionalPrefixes {
		if strings.HasPrefix(c.Message, prefix+":") || strings.HasPrefix(c.Message, prefix+"(") {
			return true
		}
	}
	
	return false
}

// GetSummary returns the first line of the commit message.
// This is useful for displaying commit summaries in logs.
func (c *Commit) GetSummary() string {
	lines := strings.Split(c.Message, "\n")
	if len(lines) > 0 {
		return lines[0]
	}
	return c.Message
}

// GetDescription returns the detailed description (everything after the first line).
// This separates the summary from the detailed description.
func (c *Commit) GetDescription() string {
	lines := strings.Split(c.Message, "\n")
	if len(lines) > 1 {
		return strings.Join(lines[1:], "\n")
	}
	return ""
}