package cli

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type githubPRCreateOptions struct {
	BaseBranch string
	Title      string
	Body       string
	Reviewers  []string
	Draft      bool
}

type githubPRClient interface {
	Available() bool
	AuthStatus() error
	Login() error
	CurrentBranch() (string, error)
	LocalHead() (string, error)
	RemoteHead(branch string) (string, error)
	ListReviewers() ([]string, error)
	PushBranch(branch string) error
	Create(options githubPRCreateOptions, branch string) (string, error)
}

type commandGitHubPRClient struct {
	lookPath func(string) (string, error)
	command  func(binary string, interactive bool, stdin string, args ...string) (string, error)
}

func newCommandGitHubPRClient() *commandGitHubPRClient {
	return &commandGitHubPRClient{
		lookPath: exec.LookPath,
		command:  runPRCommand,
	}
}

func (c *commandGitHubPRClient) Available() bool {
	_, err := c.lookPath("gh")
	return err == nil
}

func (c *commandGitHubPRClient) AuthStatus() error {
	_, err := c.command("gh", false, "", "auth", "status", "--active")
	return err
}

func (c *commandGitHubPRClient) Login() error {
	_, err := c.command("gh", true, "", "auth", "login")
	return err
}

func (c *commandGitHubPRClient) CurrentBranch() (string, error) {
	value, err := c.command("git", false, "", "rev-parse", "--abbrev-ref", "HEAD")
	return strings.TrimSpace(value), err
}

func (c *commandGitHubPRClient) LocalHead() (string, error) {
	value, err := c.command("git", false, "", "rev-parse", "HEAD")
	return strings.TrimSpace(value), err
}

func (c *commandGitHubPRClient) RemoteHead(branch string) (string, error) {
	value, err := c.command("git", false, "", "ls-remote", "--heads", "origin", "refs/heads/"+branch)
	if err != nil {
		return "", err
	}
	fields := strings.Fields(value)
	if len(fields) == 0 {
		return "", nil
	}
	return fields[0], nil
}

func (c *commandGitHubPRClient) ListReviewers() ([]string, error) {
	output, err := c.command("gh", false, "", "api", "repos/{owner}/{repo}/collaborators?per_page=100", "--paginate", "--jq", ".[].login")
	if err != nil {
		return nil, err
	}
	return uniqueReviewerLogins(strings.Fields(output)), nil
}

func (c *commandGitHubPRClient) PushBranch(branch string) error {
	_, err := c.command("git", true, "", "push", "--set-upstream", "origin", branch)
	return err
}

func (c *commandGitHubPRClient) Create(options githubPRCreateOptions, branch string) (string, error) {
	args := []string{"pr", "create", "--base", options.BaseBranch, "--head", branch, "--title", options.Title, "--body-file", "-"}
	for _, reviewer := range options.Reviewers {
		args = append(args, "--reviewer", reviewer)
	}
	if options.Draft {
		args = append(args, "--draft")
	}
	output, err := c.command("gh", false, options.Body, args...)
	if err != nil {
		return "", err
	}
	for _, field := range strings.Fields(output) {
		if strings.HasPrefix(field, "https://") || strings.HasPrefix(field, "http://") {
			return strings.TrimRight(field, ".,;"), nil
		}
	}
	return "", nil
}

func runPRCommand(binary string, interactive bool, stdin string, args ...string) (string, error) {
	cmd := exec.Command(binary, args...)
	var stdout, stderr bytes.Buffer
	if interactive {
		cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
		if err := cmd.Run(); err != nil {
			return "", err
		}
		return "", nil
	}
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		detail := strings.TrimSpace(stderr.String())
		if detail == "" {
			detail = strings.TrimSpace(stdout.String())
		}
		if detail != "" {
			return stdout.String(), fmt.Errorf("%w: %s", err, detail)
		}
		return stdout.String(), err
	}
	return stdout.String(), nil
}
