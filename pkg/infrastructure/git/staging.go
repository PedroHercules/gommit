package git

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/PedroHercules/gommit/pkg/domain/repositories"
)

// stagingCommand runs from the repository root so paths from status are unambiguous.
func (r *CommandGitRepository) stagingCommand(args ...string) (string, error) {
	root, err := r.GetRepositoryRoot()
	if err != nil {
		return "", err
	}
	cmd := exec.Command("git", append([]string{"--literal-pathspecs"}, args...)...)
	cmd.Dir = root
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s failed: %s: %w", args[0], strings.TrimSpace(string(output)), err)
	}
	return string(output), nil
}

func (r *CommandGitRepository) ListChanges() ([]repositories.FileStatus, error) {
	out, err := r.stagingCommand("status", "--porcelain=v1", "-z", "--untracked-files=all", "--ignore-submodules=none")
	if err != nil {
		return nil, err
	}
	parts := strings.Split(out, "\x00")
	var result []repositories.FileStatus
	for i := 0; i < len(parts); i++ {
		entry := parts[i]
		if entry == "" {
			continue
		}
		if len(entry) < 4 || entry[2] != ' ' {
			return nil, fmt.Errorf("unexpected git status record")
		}
		item := repositories.FileStatus{Path: entry[3:], Index: entry[0], Worktree: entry[1]}
		if item.Index == 'R' || item.Index == 'C' || item.Worktree == 'R' || item.Worktree == 'C' {
			i++
			if i >= len(parts) || parts[i] == "" {
				return nil, fmt.Errorf("incomplete git rename status")
			}
			item.OriginalPath = parts[i]
		}
		result = append(result, item)
	}
	return result, nil
}

func (r *CommandGitRepository) StageFiles(paths []string) error {
	if len(paths) == 0 {
		return nil
	}
	_, err := r.stagingCommand(append([]string{"add", "--"}, paths...)...)
	return err
}
func (r *CommandGitRepository) StageAll() error {
	_, err := r.stagingCommand("add", "-A", "--", ".")
	return err
}
func (r *CommandGitRepository) UnstageFile(file repositories.FileStatus) error {
	paths := []string{file.Path}
	if file.OriginalPath != "" {
		paths = append(paths, file.OriginalPath)
	}
	if _, err := r.stagingCommand("rev-parse", "--verify", "--quiet", "HEAD^{commit}"); err != nil {
		_, err = r.stagingCommand(append([]string{"rm", "--cached", "-r", "--ignore-unmatch", "--"}, paths...)...)
		return err
	}
	_, err := r.stagingCommand(append([]string{"restore", "--source=HEAD", "--staged", "--"}, paths...)...)
	return err
}
