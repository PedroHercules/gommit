package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestBranchDiffUsesCodeChangesWithoutChangingCheckoutOrWorktree(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	root := t.TempDir()
	seed := filepath.Join(root, "seed")
	bare := filepath.Join(root, "origin.git")
	clone := filepath.Join(root, "clone")
	mustRunGit(t, "", "init", "-b", "main", seed)
	mustRunGit(t, seed, "config", "user.name", "Gmit Test")
	mustRunGit(t, seed, "config", "user.email", "gmit@example.test")
	if err := os.WriteFile(filepath.Join(seed, "base.txt"), []byte("base\n"), 0600); err != nil {
		t.Fatal(err)
	}
	mustRunGit(t, seed, "add", "base.txt")
	mustRunGit(t, seed, "commit", "-m", "base")
	mustRunGit(t, "", "clone", "--bare", seed, bare)
	mustRunGit(t, "", "clone", bare, clone)
	mustRunGit(t, clone, "config", "user.name", "Gmit Test")
	mustRunGit(t, clone, "config", "user.email", "gmit@example.test")
	mustRunGit(t, clone, "checkout", "-b", "feature")
	if err := os.WriteFile(filepath.Join(clone, "feature.txt"), []byte("committed feature\n"), 0600); err != nil {
		t.Fatal(err)
	}
	mustRunGit(t, clone, "add", "feature.txt")
	mustRunGit(t, clone, "commit", "-m", "feature")
	if err := os.WriteFile(filepath.Join(clone, "feature.txt"), []byte("committed feature\nunstaged edit\n"), 0600); err != nil {
		t.Fatal(err)
	}

	repo, err := NewCommandGitRepository(clone)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.FetchRemote(); err != nil {
		t.Fatal(err)
	}
	diff, err := repo.GetDiffBetweenBranches("origin/main", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if len(diff.Files) != 1 || diff.Files[0] != "feature.txt" {
		t.Fatalf("diff files = %#v, want only committed feature.txt", diff.Files)
	}
	branch, err := repo.GetCurrentBranch()
	if err != nil || branch != "feature" {
		t.Fatalf("current branch = %q, err = %v; want feature", branch, err)
	}
	contents, err := os.ReadFile(filepath.Join(clone, "feature.txt"))
	if err != nil || string(contents) != "committed feature\nunstaged edit\n" {
		t.Fatalf("working file changed: contents=%q err=%v", contents, err)
	}
	status := mustRunGit(t, clone, "status", "--porcelain")
	if !strings.Contains(status, " M feature.txt") {
		t.Fatalf("unstaged worktree change was not preserved: %q", status)
	}
}

func TestBranchDiffErrorsWhenRemoteBaseIsUnavailable(t *testing.T) {
	dir := t.TempDir()
	mustRunGit(t, "", "init", "-b", "main", dir)
	repo, err := NewCommandGitRepository(dir)
	if err != nil {
		t.Fatal(err)
	}
	_, err = repo.GetDiffBetweenBranches("origin/missing", "HEAD")
	if err == nil || !strings.Contains(err.Error(), "origin/missing") {
		t.Fatalf("error = %v, want clear unavailable-base error", err)
	}
}

func mustRunGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	command := exec.Command("git", args...)
	if dir != "" {
		command.Dir = dir
	}
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, output)
	}
	return string(output)
}
