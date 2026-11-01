package cli

import (
	"testing"

	"github.com/PedroHercules/gommit/pkg/domain/repositories"
	"github.com/gdamore/tcell/v2"
)

type stagingFake struct {
	files                 []repositories.FileStatus
	staged, unstaged, all int
	stagedPaths           []string
	unstagedPath          string
}

func (f *stagingFake) ListChanges() ([]repositories.FileStatus, error) { return f.files, nil }
func (f *stagingFake) StageFiles(paths []string) error {
	f.staged++
	f.stagedPaths = append([]string(nil), paths...)
	return nil
}
func (f *stagingFake) StageAll() error { f.all++; return nil }
func (f *stagingFake) UnstageFile(file repositories.FileStatus) error {
	f.unstaged++
	f.unstagedPath = file.Path
	return nil
}
func TestStagingPickerStagesAndCancels(t *testing.T) {
	screen := tcell.NewSimulationScreen("")
	if err := screen.Init(); err != nil {
		t.Fatal(err)
	}
	defer screen.Fini()
	screen.SetSize(80, 15)
	repo := &stagingFake{files: []repositories.FileStatus{{Path: "file", Index: '?', Worktree: '?'}}}
	for _, event := range []*tcell.EventKey{tcell.NewEventKey(tcell.KeyRune, ' ', 0), tcell.NewEventKey(tcell.KeyEsc, 0, 0)} {
		if err := screen.PostEvent(event); err != nil {
			t.Fatal(err)
		}
	}
	proceed, err := pickStagingOnScreen(screen, repo)
	if err != nil || proceed || repo.all != 1 {
		t.Fatalf("proceed=%v all=%d err=%v", proceed, repo.all, err)
	}
}

func TestStagingRowsShowBothSidesOfPartiallyStagedFile(t *testing.T) {
	rows := stagingRows([]repositories.FileStatus{{Path: "example", Index: 'M', Worktree: 'M'}})
	var staged, unstaged bool
	for _, row := range rows {
		if row.action == "unstage" {
			staged = true
		}
		if row.action == "stage" {
			unstaged = true
		}
	}
	if !staged || !unstaged {
		t.Fatalf("missing staged or unstaged entry: %+v", rows)
	}
}

func TestStagingPickerStagesAndUnstagesIndividualFiles(t *testing.T) {
	screen := tcell.NewSimulationScreen("")
	if err := screen.Init(); err != nil {
		t.Fatal(err)
	}
	defer screen.Fini()
	screen.SetSize(80, 15)
	repo := &stagingFake{files: []repositories.FileStatus{{Path: "prepared", Index: 'M', Worktree: ' '}, {Path: "new", Index: '?', Worktree: '?'}}}
	keys := []*tcell.EventKey{tcell.NewEventKey(tcell.KeyDown, 0, 0), tcell.NewEventKey(tcell.KeyDown, 0, 0), tcell.NewEventKey(tcell.KeyRune, ' ', 0), tcell.NewEventKey(tcell.KeyDown, 0, 0), tcell.NewEventKey(tcell.KeyDown, 0, 0), tcell.NewEventKey(tcell.KeyRune, ' ', 0), tcell.NewEventKey(tcell.KeyEsc, 0, 0)}
	for _, event := range keys {
		if err := screen.PostEvent(event); err != nil {
			t.Fatal(err)
		}
	}
	proceed, err := pickStagingOnScreen(screen, repo)
	if err != nil || proceed || repo.unstaged != 1 || repo.staged != 1 || repo.all != 0 || len(repo.stagedPaths) != 1 || repo.stagedPaths[0] != "new" || repo.unstagedPath != "prepared" {
		t.Fatalf("proceed=%v staged=%d unstaged=%d all=%d err=%v", proceed, repo.staged, repo.unstaged, repo.all, err)
	}
}
