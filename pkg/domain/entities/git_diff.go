// Package entities contains the core business entities of the application.
package entities

import (
	"errors"
	"fmt"
	"strings"
)

// GitDiff represents the changes in a git repository.
// This entity encapsulates the git diff information and related business rules.
type GitDiff struct {
	Content     string   // The raw diff content
	Files       []string // List of files that were changed
	Additions   int      // Number of lines added
	Deletions   int      // Number of lines deleted
	IsEmpty     bool     // Whether there are any changes
}

// NewGitDiff creates a new GitDiff entity from raw diff content.
// It parses the diff and extracts relevant information.
func NewGitDiff(content string) (*GitDiff, error) {
	content = strings.TrimSpace(content)
	
	if content == "" {
		return &GitDiff{
			Content: "",
			Files:   []string{},
			IsEmpty: true,
		}, nil
	}

	diff := &GitDiff{
		Content: content,
		IsEmpty: false,
	}

	// Parse the diff to extract file information
	diff.parseFiles()
	diff.parseStats()

	return diff, nil
}

// parseFiles extracts the list of changed files from the diff content.
func (g *GitDiff) parseFiles() {
	lines := strings.Split(g.Content, "\n")
	files := make(map[string]bool) // Use map to avoid duplicates

	for _, line := range lines {
		// Look for diff --git lines or +++ lines to identify files
		if strings.HasPrefix(line, "diff --git") {
			// Extract file path from "diff --git a/file b/file"
			// Format is typically: diff --git a/path/to/file b/path/to/file
			// We need to find the b/ part which may contain spaces
			parts := strings.SplitN(line, " a/", 2)
			if len(parts) == 2 {
				// Now parts[1] contains: path/to/file b/path/to/file
				// Split at the first occurrence of " b/"
				fileParts := strings.SplitN(parts[1], " b/", 2)
				if len(fileParts) == 2 {
					// fileParts[1] now contains the full path
					files[fileParts[1]] = true
				}
			}
		} else if strings.HasPrefix(line, "+++") && !strings.Contains(line, "/dev/null") {
			// Extract file path from "+++ b/file"
			// Handle the case where the path might contain spaces
			parts := strings.SplitN(line, "+++ b/", 2)
			if len(parts) == 2 && parts[1] != "" {
				files[parts[1]] = true
			}
		}
	}

	// Convert map keys to slice
	for file := range files {
		g.Files = append(g.Files, file)
	}
}

// parseStats calculates the number of additions and deletions.
func (g *GitDiff) parseStats() {
	lines := strings.Split(g.Content, "\n")

	for _, line := range lines {
		if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			g.Additions++
		} else if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
			g.Deletions++
		}
	}
}

// HasChanges returns true if there are any changes in the diff.
func (g *GitDiff) HasChanges() bool {
	return !g.IsEmpty && g.Content != ""
}

// GetFileCount returns the number of files changed.
func (g *GitDiff) GetFileCount() int {
	return len(g.Files)
}

// GetChangesSummary returns a human-readable summary of the changes.
func (g *GitDiff) GetChangesSummary() string {
	if g.IsEmpty {
		return "No changes"
	}

	fileCount := g.GetFileCount()
	fileText := "file"
	if fileCount != 1 {
		fileText = "files"
	}

	return fmt.Sprintf("%d %s changed, %d additions, %d deletions", 
		fileCount, fileText, g.Additions, g.Deletions)
}

// Validate checks if the GitDiff is valid for commit generation.
func (g *GitDiff) Validate() error {
	if g.IsEmpty {
		return errors.New("no changes found in stage. Run 'git add <file>' to stage files before generating commit")
	}

	if g.Content == "" {
		return errors.New("diff content is empty")
	}

	return nil
}