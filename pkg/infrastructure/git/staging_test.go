package git

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStagingTracksChangesFromSubdirectory(t *testing.T) {
	dir := t.TempDir()
	mustRunGit(t, "", "init", "-b", "main", dir)
	mustRunGit(t, dir, "config", "user.name", "Test")
	mustRunGit(t, dir, "config", "user.email", "test@example.test")
	write := func(path, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, path), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write("old.txt", "old\n")
	mustRunGit(t, dir, "add", "old.txt")
	mustRunGit(t, dir, "commit", "-m", "initial")
	if err := os.Mkdir(filepath.Join(dir, "sub"), 0700); err != nil {
		t.Fatal(err)
	}
	write("new file.txt", "new\n")
	write("old.txt", "changed\n")
	repo, err := NewCommandGitRepository(filepath.Join(dir, "sub"))
	if err != nil {
		t.Fatal(err)
	}
	files, err := repo.ListChanges()
	if err != nil || len(files) != 2 {
		t.Fatalf("status: %+v, %v", files, err)
	}
	if err := repo.StageFiles([]string{"new file.txt"}); err != nil {
		t.Fatal(err)
	}
	files, err = repo.ListChanges()
	if err != nil {
		t.Fatal(err)
	}
	var staged bool
	for _, f := range files {
		if f.Path == "new file.txt" && f.Index == 'A' {
			staged = true
			if err := repo.UnstageFile(f); err != nil {
				t.Fatal(err)
			}
		}
	}
	if !staged {
		t.Fatal("new file was not staged")
	}
	if err := repo.StageAll(); err != nil {
		t.Fatal(err)
	}
	files, err = repo.ListChanges()
	if err != nil || len(files) != 2 {
		t.Fatalf("staged status: %+v, %v", files, err)
	}
	for _, f := range files {
		if f.Worktree != ' ' {
			t.Fatalf("not staged: %+v", f)
		}
	}
}

func TestUnstageWithoutInitialCommit(t *testing.T) {
	dir := t.TempDir()
	mustRunGit(t, "", "init", "-b", "main", dir)
	if err := os.WriteFile(filepath.Join(dir, "new"), []byte("data"), 0600); err != nil {
		t.Fatal(err)
	}
	repo, _ := NewCommandGitRepository(dir)
	if err := repo.StageAll(); err != nil {
		t.Fatal(err)
	}
	files, err := repo.ListChanges()
	if err != nil || len(files) != 1 {
		t.Fatalf("status: %+v %v", files, err)
	}
	if err := repo.UnstageFile(files[0]); err != nil {
		t.Fatal(err)
	}
	files, err = repo.ListChanges()
	if err != nil || len(files) != 1 || files[0].Index != '?' {
		t.Fatalf("status: %+v %v", files, err)
	}
}
