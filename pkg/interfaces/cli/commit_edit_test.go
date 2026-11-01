package cli

import (
	"errors"
	"strings"
	"testing"

	commitservices "github.com/PedroHercules/gommit/pkg/application/services/commit"
	"github.com/PedroHercules/gommit/pkg/domain/entities"
	"github.com/PedroHercules/gommit/pkg/domain/repositories"
)

type editableGit struct {
	fakeSecurityGit
	stagingFake
}

func TestEditStagingRegeneratesOnlyWhenDiffChanges(t *testing.T) {
	first, _ := entities.NewGitDiff(diffFixture("one.go", "+first\n"))
	second, _ := entities.NewGitDiff(diffFixture("two.go", "+second\n"))
	empty, _ := entities.NewGitDiff("")
	cases := []struct {
		name                                  string
		choices                               []*entities.GitDiff
		proceed                               []bool
		wantGenerations, wantScans, wantPicks int
	}{
		{"unchanged", []*entities.GitDiff{first}, []bool{true}, 1, 1, 1},
		{"changed", []*entities.GitDiff{second}, []bool{true}, 2, 2, 1},
		{"cancel after change", []*entities.GitDiff{second}, []bool{false}, 2, 2, 1},
		{"empty then original", []*entities.GitDiff{empty, first}, []bool{true, true}, 1, 1, 2},
		{"empty then changed", []*entities.GitDiff{empty, second}, []bool{true, true}, 2, 2, 2},
		{"empty then cancel", []*entities.GitDiff{empty}, []bool{false}, 1, 1, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			git := &editableGit{fakeSecurityGit: fakeSecurityGit{stagedDiff: first}}
			service := &fakeCommitCommandService{preview: &commitservices.GenerateCommitPreviewEntity{Success: true, CommitMessage: "feat: example"}}
			picks, scans, actions := 0, 0, 0
			cli := &CLI{commitService: service, securityGit: git,
				securityScanEnabled: func() (bool, error) { scans++; return true, nil },
				selectCommitAction: func() (string, error) {
					actions++
					if actions == 1 {
						return editStagingAction, nil
					}
					return cancelCommitAction, nil
				},
				selectStaging: func(repositories.StagingRepository) (bool, error) {
					git.stagedDiff = tc.choices[picks]
					result := tc.proceed[picks]
					picks++
					return result, nil
				},
			}
			if err := cli.handleCommit(nil); err != nil {
				t.Fatal(err)
			}
			if service.previewCall != tc.wantGenerations || scans != tc.wantScans || picks != tc.wantPicks {
				t.Fatalf("generations=%d scans=%d picks=%d", service.previewCall, scans, picks)
			}
		})
	}
}

func TestEditStagingReturnsPickerError(t *testing.T) {
	diff, _ := entities.NewGitDiff(diffFixture("one.go", "+first\n"))
	git := &editableGit{fakeSecurityGit: fakeSecurityGit{stagedDiff: diff}}
	expected := errors.New("picker failed")
	cli := &CLI{commitService: &fakeCommitCommandService{preview: &commitservices.GenerateCommitPreviewEntity{Success: true}}, securityGit: git,
		securityScanEnabled: func() (bool, error) { return false, nil },
		selectCommitAction:  func() (string, error) { return editStagingAction, nil },
		selectStaging:       func(repositories.StagingRepository) (bool, error) { return false, expected },
	}
	if err := cli.handleCommit(nil); !errors.Is(err, expected) {
		t.Fatalf("got %v", err)
	}
}

func TestEditStagingWarnsBeforeSendingNewSensitiveDiff(t *testing.T) {
	first, _ := entities.NewGitDiff(diffFixture("one.go", "+first\n"))
	sensitive, _ := entities.NewGitDiff(diffFixture("settings.env", "+SERVICE_TOKEN=synthetic-token-123456\n"))
	git := &editableGit{fakeSecurityGit: fakeSecurityGit{stagedDiff: first}}
	service := &fakeCommitCommandService{preview: &commitservices.GenerateCommitPreviewEntity{Success: true, CommitMessage: "feat: example"}}
	warnings := 0
	cli := &CLI{commitService: service, securityGit: git,
		securityScanEnabled: func() (bool, error) { return true, nil },
		selectCommitAction:  func() (string, error) { return editStagingAction, nil },
		selectStaging:       func(repositories.StagingRepository) (bool, error) { git.stagedDiff = sensitive; return true, nil },
		reviewSensitive: func(report securityReport) (bool, error) {
			warnings++
			if !report.hasFindings() {
				t.Fatal("missing findings")
			}
			if service.previewCall != 1 {
				t.Fatal("new diff reached provider before review")
			}
			return false, nil
		},
	}
	if err := cli.handleCommit(nil); err == nil {
		t.Fatal("expected cancellation")
	}
	if warnings != 1 || service.previewCall != 1 || service.confirmCall != 0 {
		t.Fatalf("warnings=%d generation=%d commits=%d", warnings, service.previewCall, service.confirmCall)
	}
}

func TestCommitRejectsStagingChangedAfterPreview(t *testing.T) {
	first, _ := entities.NewGitDiff(diffFixture("one.go", "+first\n"))
	second, _ := entities.NewGitDiff(diffFixture("two.go", "+second\n"))
	git := &editableGit{fakeSecurityGit: fakeSecurityGit{stagedDiff: first}}
	service := &fakeCommitCommandService{preview: &commitservices.GenerateCommitPreviewEntity{Success: true, CommitMessage: "feat: example"}}
	cli := &CLI{commitService: service, securityGit: git, securityScanEnabled: func() (bool, error) { return false, nil }, selectCommitAction: func() (string, error) { git.stagedDiff = second; return commitOnlyAction, nil }}
	if err := cli.handleCommit(nil); err == nil || !strings.Contains(err.Error(), "staged changes changed") {
		t.Fatalf("error=%v", err)
	}
	if service.confirmCall != 0 {
		t.Fatal("commit created using stale preview")
	}
}

func TestCommitAllStagesBeforeGenerating(t *testing.T) {
	diff, _ := entities.NewGitDiff(diffFixture("one.go", "+first\n"))
	git := &editableGit{fakeSecurityGit: fakeSecurityGit{stagedDiff: diff}}
	service := &fakeCommitCommandService{preview: &commitservices.GenerateCommitPreviewEntity{Success: true, CommitMessage: "feat: example"}}
	cli := &CLI{commitService: service, securityGit: git, securityScanEnabled: func() (bool, error) { return false, nil }}
	if err := cli.handleCommit([]string{"--all", "--dry-run"}); err != nil {
		t.Fatal(err)
	}
	if git.all != 1 || service.previewCall != 1 || service.previewDTO.Diff != diff {
		t.Fatalf("stage all=%d previews=%d", git.all, service.previewCall)
	}
}
