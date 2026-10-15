package cli

import (
	"os/exec"
	"reflect"
	"strings"
	"testing"
)

func TestCommandGitHubPRClientBuildsCreateCommand(t *testing.T) {
	var gotBinary string
	var gotArgs []string
	var gotInteractive bool
	var gotStdin string
	client := &commandGitHubPRClient{
		lookPath: func(binary string) (string, error) {
			if binary != "gh" {
				t.Fatalf("looked up %q, want gh", binary)
			}
			return "/usr/bin/gh", nil
		},
		command: func(binary string, interactive bool, stdin string, args ...string) (string, error) {
			gotBinary, gotInteractive, gotStdin, gotArgs = binary, interactive, stdin, args
			return "https://github.com/acme/project/pull/42\n", nil
		},
	}
	if !client.Available() {
		t.Fatal("Available() = false, want true")
	}
	options := githubPRCreateOptions{
		BaseBranch: "develop",
		Title:      "Add login",
		Body:       "## Description\nAdds login.",
		Reviewers:  []string{"alice", "team/reviewers"},
		Draft:      true,
	}
	url, err := client.Create(options, "feature/login")
	if err != nil {
		t.Fatal(err)
	}
	wantArgs := []string{
		"pr", "create", "--base", "develop", "--head", "feature/login",
		"--title", "Add login", "--body-file", "-",
		"--reviewer", "alice", "--reviewer", "team/reviewers", "--draft",
	}
	if gotBinary != "gh" || gotInteractive || gotStdin != options.Body || !reflect.DeepEqual(gotArgs, wantArgs) {
		t.Fatalf("command = %s interactive=%t stdin=%q args=%q", gotBinary, gotInteractive, gotStdin, gotArgs)
	}
	if url != "https://github.com/acme/project/pull/42" {
		t.Fatalf("Create() URL = %q", url)
	}
}

func TestCommandGitHubPRClientReadsRemoteHead(t *testing.T) {
	client := &commandGitHubPRClient{command: func(binary string, interactive bool, _ string, args ...string) (string, error) {
		if binary != "git" || interactive || !reflect.DeepEqual(args, []string{"ls-remote", "--heads", "origin", "refs/heads/feature/login"}) {
			t.Fatalf("unexpected command: %s interactive=%t args=%q", binary, interactive, args)
		}
		return "abc123\trefs/heads/feature/login\n", nil
	}}
	sha, err := client.RemoteHead("feature/login")
	if err != nil || sha != "abc123" {
		t.Fatalf("RemoteHead() = %q, %v", sha, err)
	}
}

func TestCommandGitHubPRClientListsCollaboratorsWithPagination(t *testing.T) {
	client := &commandGitHubPRClient{command: func(binary string, interactive bool, stdin string, args ...string) (string, error) {
		if binary != "gh" || interactive || stdin != "" {
			t.Fatalf("unexpected command invocation: binary=%s interactive=%t stdin=%q", binary, interactive, stdin)
		}
		want := []string{"api", "repos/{owner}/{repo}/collaborators?per_page=100", "--paginate", "--jq", ".[].login"}
		if !reflect.DeepEqual(args, want) {
			t.Fatalf("ListReviewers() args = %q, want %q", args, want)
		}
		return "zoe\nalice\nzoe\n", nil
	}}
	got, err := client.ListReviewers()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, []string{"alice", "zoe"}) {
		t.Fatalf("ListReviewers() = %#v", got)
	}
}

func TestRunPRCommandIncludesCommandOutputInError(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("shell-based command output test requires sh")
	}
	_, err := runPRCommand("sh", false, "", "-c", "echo denied >&2; exit 1")
	if err == nil {
		t.Fatalf("runPRCommand() error = %v, want command error", err)
	}
	if !strings.Contains(err.Error(), "denied") {
		t.Fatalf("command error omitted stderr: %q", err)
	}
}
