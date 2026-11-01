package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PedroHercules/gommit/pkg/domain/repositories"
)

func stagingFixture(t *testing.T) (string, *CommandGitRepository) {
	t.Helper()
	dir := t.TempDir()
	mustRunGit(t, "", "init", "-b", "main", dir)
	mustRunGit(t, dir, "config", "user.name", "Test")
	mustRunGit(t, dir, "config", "user.email", "test@example.test")
	mustRunGit(t, dir, "config", "core.autocrlf", "false")
	repo, err := NewCommandGitRepository(dir)
	if err != nil {
		t.Fatal(err)
	}
	return dir, repo
}
func writeStagingFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
}
func findStaging(t *testing.T, repo *CommandGitRepository, path string) repositories.FileStatus {
	t.Helper()
	files, err := repo.ListChanges()
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		if f.Path == path {
			return f
		}
	}
	t.Fatalf("%q absent from status: %+v", path, files)
	return repositories.FileStatus{}
}
func TestStagingRenameAndDeletionPreserveWorkingTree(t *testing.T) {
	dir, repo := stagingFixture(t)
	writeStagingFile(t, dir, "original.txt", "content\n")
	writeStagingFile(t, dir, "deleted.txt", "delete me\n")
	mustRunGit(t, dir, "add", ".")
	mustRunGit(t, dir, "commit", "-m", "base")
	if err := os.Rename(filepath.Join(dir, "original.txt"), filepath.Join(dir, "renamed.txt")); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(dir, "deleted.txt")); err != nil {
		t.Fatal(err)
	}
	if err := repo.StageAll(); err != nil {
		t.Fatal(err)
	}
	renamed := findStaging(t, repo, "renamed.txt")
	if renamed.Index != 'R' || renamed.OriginalPath != "original.txt" {
		t.Fatalf("rename: %+v", renamed)
	}
	if err := repo.UnstageFile(renamed); err != nil {
		t.Fatal(err)
	}
	deleted := findStaging(t, repo, "deleted.txt")
	if deleted.Index != 'D' {
		t.Fatalf("deleted: %+v", deleted)
	}
	if err := repo.UnstageFile(deleted); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "deleted.txt")); !os.IsNotExist(err) {
		t.Fatalf("deleted file restored: %v", err)
	}
	if data, err := os.ReadFile(filepath.Join(dir, "renamed.txt")); err != nil || string(data) != "content\n" {
		t.Fatalf("rename contents %q %v", data, err)
	}
	if diff := mustRunGit(t, dir, "diff", "--cached", "--name-only"); diff != "" {
		t.Fatalf("unexpected staged files: %q", diff)
	}
}
func TestStagingPartialFileAndIgnoredPaths(t *testing.T) {
	dir, repo := stagingFixture(t)
	writeStagingFile(t, dir, "tracked.txt", "base\n")
	writeStagingFile(t, dir, ".gitignore", "ignored.env\n")
	mustRunGit(t, dir, "add", ".")
	mustRunGit(t, dir, "commit", "-m", "base")
	writeStagingFile(t, dir, "tracked.txt", "staged\n")
	if err := repo.StageFiles([]string{"tracked.txt"}); err != nil {
		t.Fatal(err)
	}
	writeStagingFile(t, dir, "tracked.txt", "staged\nunstaged\n")
	writeStagingFile(t, dir, "ignored.env", "private\n")
	writeStagingFile(t, dir, "-special [é].txt", "new\n")
	f := findStaging(t, repo, "tracked.txt")
	if f.Index != 'M' || f.Worktree != 'M' {
		t.Fatalf("partial status: %+v", f)
	}
	files, err := repo.ListChanges()
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		if f.Path == "ignored.env" {
			t.Fatal("ignored file listed")
		}
	}
	if err := repo.StageFiles([]string{"-special [é].txt"}); err != nil {
		t.Fatal(err)
	}
	if got := mustRunGit(t, dir, "show", ":tracked.txt"); got != "staged\n" {
		t.Fatalf("partial index changed: %q", got)
	}
	if err := repo.UnstageFile(f); err != nil {
		t.Fatal(err)
	}
	if got, err := os.ReadFile(filepath.Join(dir, "tracked.txt")); err != nil || string(got) != "staged\nunstaged\n" {
		t.Fatalf("worktree %q %v", got, err)
	}
	if err := repo.StageAll(); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(mustRunGit(t, dir, "diff", "--cached", "--name-only"), "ignored.env") {
		t.Fatal("ignored file staged")
	}
}
func TestStagingConflictIsVisibleAndNotResolvedByListing(t *testing.T) {
	dir, repo := stagingFixture(t)
	writeStagingFile(t, dir, "conflict.txt", "base\n")
	mustRunGit(t, dir, "add", ".")
	mustRunGit(t, dir, "commit", "-m", "base")
	mustRunGit(t, dir, "checkout", "-b", "other")
	writeStagingFile(t, dir, "conflict.txt", "other\n")
	mustRunGit(t, dir, "commit", "-am", "other")
	mustRunGit(t, dir, "checkout", "main")
	writeStagingFile(t, dir, "conflict.txt", "main\n")
	mustRunGit(t, dir, "commit", "-am", "main")
	cmd := exec.Command("git", "merge", "other")
	cmd.Dir = dir
	if err := cmd.Run(); err == nil {
		t.Fatal("expected merge conflict")
	}
	f := findStaging(t, repo, "conflict.txt")
	if f.Index != 'U' || f.Worktree != 'U' {
		t.Fatalf("conflict: %+v", f)
	}
	if got := mustRunGit(t, dir, "diff", "--name-only", "--diff-filter=U"); !strings.Contains(got, "conflict.txt") {
		t.Fatal("listing resolved conflict")
	}
}
