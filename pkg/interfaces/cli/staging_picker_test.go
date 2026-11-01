package cli

import (
	"testing"

	"github.com/PedroHercules/gommit/pkg/domain/repositories"
	"github.com/gdamore/tcell/v2"
)

type stagingFake struct {
	files                 []repositories.FileStatus
	staged, unstaged, all int
}

func (f *stagingFake) ListChanges() ([]repositories.FileStatus, error) { return f.files, nil }
func (f *stagingFake) StageFiles([]string) error                       { f.staged++; return nil }
func (f *stagingFake) StageAll() error                                 { f.all++; return nil }
func (f *stagingFake) UnstageFile(repositories.FileStatus) error       { f.unstaged++; return nil }
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
