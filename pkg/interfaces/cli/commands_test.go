package cli

import (
	"io"
	"os"
	"strings"
	"testing"

	commit_services "github.com/PedroHercules/gommit/pkg/application/services/commit"
)

type fakeCommitCommandService struct {
	preview     *commit_services.GenerateCommitPreviewEntity
	previewDTO  commit_services.GenerateCommitPreviewDTO
	confirm     *commit_services.ConfirmCommitEntity
	confirmCall int
}

func (f *fakeCommitCommandService) GenerateCommitPreview(dto commit_services.GenerateCommitPreviewDTO) (*commit_services.GenerateCommitPreviewEntity, error) {
	f.previewDTO = dto
	return f.preview, nil
}

func (f *fakeCommitCommandService) ConfirmCommit(commit_services.ConfirmCommitDTO) (*commit_services.ConfirmCommitEntity, error) {
	f.confirmCall++
	return f.confirm, nil
}

func (f *fakeCommitCommandService) ValidateCommitMessage(string) (*commit_services.ValidateCommitMessageResult, error) {
	return &commit_services.ValidateCommitMessageResult{Valid: true}, nil
}

func TestParseCommitArgsRejectsUnsupportedOptionsAndForce(t *testing.T) {
	for _, args := range [][]string{{"--force"}, {"--no-such-option"}, {"--model"}} {
		if _, _, _, err := parseCommitArgs(args); err == nil {
			t.Errorf("parseCommitArgs(%q) expected an error", args)
		}
	}
}

func TestParsePRArgsDefaultsBaseAndPassesModel(t *testing.T) {
	base, model, err := parsePRArgs([]string{"--model", "grok-4.1-fast"})
	if err != nil {
		t.Fatal(err)
	}
	if base != "main" || model != "grok-4.1-fast" {
		t.Fatalf("got base=%q model=%q, want main and grok-4.1-fast", base, model)
	}
	if _, _, err := parsePRArgs([]string{"--base-branch"}); err == nil {
		t.Fatal("missing base branch value should fail")
	}
	if _, _, err := parsePRArgs([]string{"--unknown"}); err == nil {
		t.Fatal("unsupported PR option should fail")
	}
}

func TestCommitFlagCommitsOnceAndReturnsWithoutPrompt(t *testing.T) {
	service := &fakeCommitCommandService{
		preview: &commit_services.GenerateCommitPreviewEntity{Success: true, CommitMessage: "feat: add test", Model: "model"},
		confirm: &commit_services.ConfirmCommitEntity{Success: true, CommitHash: "abc123"},
	}
	cli := &CLI{commitService: service}
	oldStdin := os.Stdin
	stdin, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	_ = writer.Close()
	os.Stdin = stdin
	defer func() { os.Stdin = oldStdin; _ = stdin.Close() }()

	if err := cli.handleCommit([]string{"--commit"}); err != nil {
		t.Fatalf("handleCommit() error = %v", err)
	}
	if service.confirmCall != 1 {
		t.Fatalf("ConfirmCommit calls = %d, want 1", service.confirmCall)
	}
}

func TestDryRunDoesNotCommit(t *testing.T) {
	service := &fakeCommitCommandService{preview: &commit_services.GenerateCommitPreviewEntity{Success: true, CommitMessage: "feat: add test"}}
	cli := &CLI{commitService: service}
	if err := cli.handleCommit([]string{"--dry-run"}); err != nil {
		t.Fatalf("handleCommit() error = %v", err)
	}
	if service.confirmCall != 0 {
		t.Fatalf("ConfirmCommit calls = %d, want 0", service.confirmCall)
	}
	if !service.previewDTO.DryRun {
		t.Fatal("dry-run request was not marked as dry-run")
	}
}

func TestTopLevelValidateRoutesMessageToValidator(t *testing.T) {
	service := &fakeCommitCommandService{}
	cli := &CLI{commitService: service}
	// This command prints to stdout; discard that output for this routing check.
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	defer func() { os.Stdout = oldStdout; _ = r.Close() }()
	if err := cli.Run([]string{"gmit", "validate", "feat:", "add", "test"}); err != nil {
		t.Fatal(err)
	}
	_ = w.Close()
	output, _ := io.ReadAll(r)
	if !strings.Contains(string(output), "non-empty") {
		t.Fatalf("validate output = %q", output)
	}
}
