package entities

import (
	"testing"
)

func TestNewGitDiff(t *testing.T) {
	tests := []struct {
		name        string
		rawDiff     string
		expectEmpty bool
	}{
		{
			name:        "valid diff",
			rawDiff:     "diff --git a/file.go b/file.go\n+added line\n-removed line",
			expectEmpty: false,
		},
		{
			name:        "empty diff",
			rawDiff:     "",
			expectEmpty: true,
		},
		{
			name:        "whitespace only diff",
			rawDiff:     "   \n\t  ",
			expectEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff, err := NewGitDiff(tt.rawDiff)

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if diff.IsEmpty != tt.expectEmpty {
				t.Errorf("expected IsEmpty %v, got %v", tt.expectEmpty, diff.IsEmpty)
			}

			if !tt.expectEmpty && diff.Content != tt.rawDiff {
				t.Errorf("expected Content %q, got %q", tt.rawDiff, diff.Content)
			}
		})
	}
}

func TestGitDiff_HasChanges(t *testing.T) {
	tests := []struct {
		name     string
		rawDiff  string
		isEmpty  bool
		expected bool
	}{
		{
			name:     "non-empty diff",
			rawDiff:  "diff --git a/file.go b/file.go\n+added line",
			isEmpty:  false,
			expected: true,
		},
		{
			name:     "empty diff",
			rawDiff:  "",
			isEmpty:  true,
			expected: false,
		},
		{
			name:     "whitespace only diff",
			rawDiff:  "   \n\t  ",
			isEmpty:  true,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff := &GitDiff{Content: tt.rawDiff, IsEmpty: tt.isEmpty}
			result := diff.HasChanges()
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestGitDiff_GetChangesSummary(t *testing.T) {
	tests := []struct {
		name      string
		gitDiff   *GitDiff
		expected  string
	}{
		{
			name: "simple diff with changes",
			gitDiff: &GitDiff{
				Content:   "diff --git a/main.go b/main.go\n+added line",
				Files:     []string{"main.go"},
				Additions: 1,
				Deletions: 0,
				IsEmpty:   false,
			},
			expected: "1 file changed, 1 additions, 0 deletions",
		},
		{
			name: "multiple files",
			gitDiff: &GitDiff{
				Content:   "diff --git a/file1.go b/file1.go\n+added\ndiff --git a/file2.go b/file2.go\n-removed",
				Files:     []string{"file1.go", "file2.go"},
				Additions: 1,
				Deletions: 1,
				IsEmpty:   false,
			},
			expected: "2 files changed, 1 additions, 1 deletions",
		},
		{
			name: "empty diff",
			gitDiff: &GitDiff{
				Content:   "",
				Files:     []string{},
				Additions: 0,
				Deletions: 0,
				IsEmpty:   true,
			},
			expected: "No changes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.gitDiff.GetChangesSummary()
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestGitDiff_GetFileCount(t *testing.T) {
	tests := []struct {
		name     string
		files    []string
		expected int
	}{
		{
			name:     "single file",
			files:    []string{"main.go"},
			expected: 1,
		},
		{
			name:     "multiple files",
			files:    []string{"file1.go", "file2.go"},
			expected: 2,
		},
		{
			name:     "no files",
			files:    []string{},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff := &GitDiff{Files: tt.files}
			result := diff.GetFileCount()
			if result != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestGitDiff_ExtractFileNames(t *testing.T) {
	tests := []struct {
		name     string
		rawDiff  string
		expected []string
	}{
		{
			name:     "single file",
			rawDiff:  "diff --git a/main.go b/main.go\n+added line",
			expected: []string{"main.go"},
		},
		{
			name:     "multiple files",
			rawDiff:  "diff --git a/file1.go b/file1.go\n+added\ndiff --git a/file2.go b/file2.go\n-removed",
			expected: []string{"file1.go", "file2.go"},
		},
		{
			name:     "file with spaces",
			rawDiff:  "diff --git a/file with spaces.go b/file with spaces.go\n+added line",
			expected: []string{"file with spaces.go"},
		},
		{
			name:     "file with spaces from +++ line",
			rawDiff:  "--- a/old file.go\n+++ b/new file with spaces.go\n@@ -1,3 +1,3 @@\n-old\n+new",
			expected: []string{"new file with spaces.go"},
		},
		{
			name:     "long path file",
			rawDiff:  "diff --git a/pkg/domain/repositories/llm_repository.go b/pkg/domain/repositories/llm_repository.go\n+added line",
			expected: []string{"pkg/domain/repositories/llm_repository.go"},
		},
		{
			name:     "empty diff",
			rawDiff:  "",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff, err := NewGitDiff(tt.rawDiff)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if len(diff.Files) != len(tt.expected) {
				t.Errorf("expected %d files, got %d", len(tt.expected), len(diff.Files))
				return
			}

			for i, file := range tt.expected {
				if i >= len(diff.Files) || diff.Files[i] != file {
					t.Errorf("expected file %q at position %d, got %q", file, i, diff.Files[i])
				}
			}
		})
	}

}
