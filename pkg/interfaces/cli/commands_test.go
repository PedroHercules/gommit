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
	pushCall    int
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

func (f *fakeCommitCommandService) PushCurrentBranch() error {
	f.pushCall++
	return nil
}

func TestParseCommitArgsRejectsUnsupportedOptionsAndForce(t *testing.T) {
	for _, args := range [][]string{{"--force"}, {"--no-such-option"}, {"--model"}} {
		if _, _, _, err := parseCommitArgs(args); err == nil {
			t.Errorf("parseCommitArgs(%q) expected an error", args)
		}
	}
}

func TestParsePRArgsDefaultsBaseAndPassesModel(t *testing.T) {
	options, err := parsePRArgs([]string{"--model", "grok-4.1-fast"})
	if err != nil {
		t.Fatal(err)
	}
	if options.baseBranch != "main" || options.model != "grok-4.1-fast" {
		t.Fatalf("got base=%q model=%q, want main and grok-4.1-fast", options.baseBranch, options.model)
	}
	if _, err := parsePRArgs([]string{"--base-branch"}); err == nil {
		t.Fatal("missing base branch value should fail")
	}
	if _, err := parsePRArgs([]string{"--unknown"}); err == nil {
		t.Fatal("unsupported PR option should fail")
	}
}

func TestParsePRArgsCreationOptions(t *testing.T) {
	options, err := parsePRArgs([]string{"--create", "--title", "Custom title", "--reviewer", "alice, team/reviewers", "--reviewer", "bob", "--draft"})
	if err != nil {
		t.Fatal(err)
	}
	if !options.create || !options.draft || !options.draftSet || !options.reviewersSet {
		t.Fatalf("creation options not parsed: %+v", options)
	}
	if options.title != "Custom title" || strings.Join(options.reviewers, ",") != "alice,team/reviewers,bob" {
		t.Fatalf("title/reviewers parsed incorrectly: %+v", options)
	}
}

func TestSplitPRTitleAndBody(t *testing.T) {
	title, body := splitPRTitleAndBody("# Generated title\n\n## Description\nDetails\n", "")
	if title != "Generated title" || body != "## Description\nDetails" {
		t.Fatalf("got title=%q body=%q", title, body)
	}
	title, body = splitPRTitleAndBody("# Generated title\n\nBody", "Override")
	if title != "Override" || body != "Body" {
		t.Fatalf("override got title=%q body=%q", title, body)
	}
	title, body = splitPRTitleAndBody("Description without heading", "")
	if title != "" || body != "Description without heading" {
		t.Fatalf("missing heading got title=%q body=%q", title, body)
	}
}

type fakeGitHubPRClient struct {
	available     bool
	authErr       error
	loginErr      error
	branch        string
	localHead     string
	remoteHead    string
	remoteErr     error
	reviewers     []string
	listErr       error
	pushErr       error
	createErr     error
	loginCalls    int
	pushCalls     int
	createCall    *githubPRCreateOptions
	createdBranch string
}

func (f *fakeGitHubPRClient) Available() bool   { return f.available }
func (f *fakeGitHubPRClient) AuthStatus() error { return f.authErr }
func (f *fakeGitHubPRClient) Login() error {
	f.loginCalls++
	if f.loginErr == nil {
		f.authErr = nil
	}
	return f.loginErr
}
func (f *fakeGitHubPRClient) CurrentBranch() (string, error)    { return f.branch, nil }
func (f *fakeGitHubPRClient) LocalHead() (string, error)        { return f.localHead, nil }
func (f *fakeGitHubPRClient) RemoteHead(string) (string, error) { return f.remoteHead, f.remoteErr }
func (f *fakeGitHubPRClient) ListReviewers() ([]string, error)  { return f.reviewers, f.listErr }
func (f *fakeGitHubPRClient) PushBranch(string) error {
	f.pushCalls++
	if f.pushErr == nil {
		f.remoteHead = f.localHead
	}
	return f.pushErr
}
func (f *fakeGitHubPRClient) Create(options githubPRCreateOptions, branch string) (string, error) {
	f.createCall = &options
	f.createdBranch = branch
	return "https://github.com/acme/project/pull/42", f.createErr
}

func TestCreateGitHubPullRequestPushesOnlyAfterApprovalAndPassesOptions(t *testing.T) {
	client := &fakeGitHubPRClient{available: true, branch: "feature/login", localHead: "local-sha"}
	cli := &CLI{
		githubPR: client,
		confirmPRPush: func(branch string) (bool, error) {
			if branch != "feature/login" {
				t.Fatalf("confirmation branch = %q", branch)
			}
			return true, nil
		},
	}
	options := githubPRCreateOptions{BaseBranch: "develop", Title: "Add login", Body: "## Description\nDetails", Reviewers: []string{"alice", "team/reviewers"}, Draft: true}
	if err := cli.createGitHubPullRequest(options); err != nil {
		t.Fatal(err)
	}
	if client.pushCalls != 1 || client.createCall == nil {
		t.Fatalf("pushes=%d create=%+v", client.pushCalls, client.createCall)
	}
	if client.createdBranch != "feature/login" || client.createCall.Title != options.Title || client.createCall.BaseBranch != options.BaseBranch || !client.createCall.Draft || strings.Join(client.createCall.Reviewers, ",") != "alice,team/reviewers" {
		t.Fatalf("created PR options = %+v on %q", client.createCall, client.createdBranch)
	}
}

func TestCreateGitHubPullRequestCancellationDoesNotPushOrCreate(t *testing.T) {
	client := &fakeGitHubPRClient{available: true, branch: "feature/login", localHead: "local-sha"}
	cli := &CLI{githubPR: client, confirmPRPush: func(string) (bool, error) { return false, nil }}
	if err := cli.createGitHubPullRequest(githubPRCreateOptions{BaseBranch: "main", Title: "Add login"}); err != nil {
		t.Fatal(err)
	}
	if client.pushCalls != 0 || client.createCall != nil {
		t.Fatalf("pushes=%d create=%+v; cancellation must have no side effects", client.pushCalls, client.createCall)
	}
}

func TestCreateGitHubPullRequestMissingCLIAndAuth(t *testing.T) {
	missing := &fakeGitHubPRClient{}
	if err := (&CLI{githubPR: missing}).createGitHubPullRequest(githubPRCreateOptions{}); err == nil || !strings.Contains(err.Error(), "install `gh`") {
		t.Fatalf("missing gh error = %v", err)
	}
	client := &fakeGitHubPRClient{available: true, authErr: os.ErrPermission, branch: "feature", localHead: "same", remoteHead: "same"}
	cli := &CLI{githubPR: client, confirmGitHubLogin: func() (bool, error) { return true, nil }}
	if err := cli.createGitHubPullRequest(githubPRCreateOptions{BaseBranch: "main", Title: "Title"}); err != nil {
		t.Fatal(err)
	}
	if client.loginCalls != 1 || client.createCall == nil {
		t.Fatalf("login calls=%d create=%+v", client.loginCalls, client.createCall)
	}
}

func TestCreateGitHubPullRequestNonInteractiveRequiresPrePushedBranch(t *testing.T) {
	client := &fakeGitHubPRClient{available: true, branch: "feature", localHead: "local", remoteHead: ""}
	err := (&CLI{githubPR: client}).createGitHubPullRequest(githubPRCreateOptions{BaseBranch: "main", Title: "Title"})
	if err == nil || !strings.Contains(err.Error(), "must be pushed") {
		t.Fatalf("error = %v", err)
	}
	if client.pushCalls != 0 || client.createCall != nil {
		t.Fatalf("pushes=%d create=%+v", client.pushCalls, client.createCall)
	}
}

func TestPromptForPRReviewersFallsBackToManualSelection(t *testing.T) {
	client := &fakeGitHubPRClient{listErr: os.ErrPermission}
	var pickerReceived []string
	cli := &CLI{selectPRReviewers: func(candidates, _ []string) (reviewerPickerResult, error) {
		pickerReceived = candidates
		return reviewerPickerResult{Reviewers: []string{"team/reviewers"}}, nil
	}}
	got, err := cli.promptForPRReviewers(client)
	if err != nil {
		t.Fatal(err)
	}
	if len(pickerReceived) != 0 || strings.Join(got, ",") != "team/reviewers" {
		t.Fatalf("manual fallback candidates=%q reviewers=%q", pickerReceived, got)
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

func TestDryRunUsesStatusMarkersAndPreservesGeneratedCommitMessage(t *testing.T) {
	const generatedMessage = "feat(auth): add token refresh support"
	service := &fakeCommitCommandService{preview: &commit_services.GenerateCommitPreviewEntity{
		Success:        true,
		CommitMessage:  generatedMessage,
		ChangesSummary: "1 file changed",
		Model:          "test/model",
	}}
	cli := &CLI{commitService: service}

	oldStdout := os.Stdout
	stdoutReader, stdoutWriter, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	oldStderr := os.Stderr
	stderrReader, stderrWriter, err := os.Pipe()
	if err != nil {
		_ = stdoutReader.Close()
		_ = stdoutWriter.Close()
		t.Fatal(err)
	}
	os.Stdout = stdoutWriter
	os.Stderr = stderrWriter
	if err := cli.handleCommit([]string{"--dry-run"}); err != nil {
		os.Stdout = oldStdout
		os.Stderr = oldStderr
		_ = stdoutReader.Close()
		_ = stderrReader.Close()
		_ = stdoutWriter.Close()
		_ = stderrWriter.Close()
		t.Fatalf("handleCommit() error = %v", err)
	}
	os.Stdout = oldStdout
	os.Stderr = oldStderr
	_ = stdoutWriter.Close()
	_ = stderrWriter.Close()
	output, err := io.ReadAll(stdoutReader)
	_ = stdoutReader.Close()
	if err != nil {
		t.Fatal(err)
	}
	progress, err := io.ReadAll(stderrReader)
	_ = stderrReader.Close()
	if err != nil {
		t.Fatal(err)
	}

	text := string(output)
	if !strings.Contains(string(progress), "› Drafting commit message") || !strings.Contains(string(progress), "✓ Drafting commit message complete.") {
		t.Fatalf("dry-run progress output is missing status labels: %q", progress)
	}
	if !strings.Contains(text, "Commit preview\n"+generatedMessage+"\n") {
		t.Fatalf("generated commit message was changed or omitted: %q", text)
	}
	if strings.Contains(text, "Drafting commit") || strings.Contains(text, "complete.") {
		t.Fatalf("progress output polluted stdout: %q", text)
	}
}

func TestCommitSelectorExecutesOnlySelectedAction(t *testing.T) {
	tests := []struct {
		name        string
		choice      string
		wantCommits int
		wantPushes  int
	}{
		{name: "commit and push", choice: commitAndPushAction, wantCommits: 1, wantPushes: 1},
		{name: "commit only", choice: commitOnlyAction, wantCommits: 1},
		{name: "cancel", choice: cancelCommitAction},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &fakeCommitCommandService{
				preview: &commit_services.GenerateCommitPreviewEntity{Success: true, CommitMessage: "feat: add test"},
				confirm: &commit_services.ConfirmCommitEntity{Success: true, CommitHash: "abc123"},
			}
			cli := &CLI{
				commitService: service,
				selectCommitAction: func() (string, error) {
					return tt.choice, nil
				},
			}
			if err := cli.handleCommit(nil); err != nil {
				t.Fatalf("handleCommit() error = %v", err)
			}
			if service.confirmCall != tt.wantCommits || service.pushCall != tt.wantPushes {
				t.Fatalf("commit calls=%d push calls=%d, want %d and %d", service.confirmCall, service.pushCall, tt.wantCommits, tt.wantPushes)
			}
		})
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
	if !strings.Contains(string(output), "not empty") {
		t.Fatalf("validate output = %q", output)
	}
}

func TestBareGmitRunsCommitFlow(t *testing.T) {
	service := &fakeCommitCommandService{
		preview: &commit_services.GenerateCommitPreviewEntity{Success: true, CommitMessage: "feat: add test"},
	}
	cli := &CLI{
		commitService: service,
		selectCommitAction: func() (string, error) {
			return cancelCommitAction, nil
		},
	}
	if err := cli.Run([]string{"gmit"}); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if service.preview == nil || service.confirmCall != 0 {
		t.Fatalf("bare gmit should generate a preview and not commit; preview=%v commits=%d", service.preview != nil, service.confirmCall)
	}
}

func TestBareGmitAcceptsCommitOptions(t *testing.T) {
	service := &fakeCommitCommandService{
		preview: &commit_services.GenerateCommitPreviewEntity{Success: true, CommitMessage: "feat: add test"},
	}
	cli := &CLI{commitService: service}
	if err := cli.Run([]string{"gmit", "--dry-run"}); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !service.previewDTO.DryRun || service.confirmCall != 0 {
		t.Fatalf("gmit --dry-run should preview only; dry-run=%v commits=%d", service.previewDTO.DryRun, service.confirmCall)
	}
}
