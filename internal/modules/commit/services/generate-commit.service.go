package commit

import (
	"fmt"
	"os"
	"os/exec"
)

func runGitDiff() (string, error) {
	cmd := exec.Command("git", "diff", "--cached")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	
	if len(output) == 0 {
		cmd = exec.Command("git", "diff")
		output, err = cmd.Output()
		if err != nil {
			return "", err
		}
	}
	
	return string(output), nil
}

func runGitCommit(message string) error {
	cmd := exec.Command("git", "commit", "-m", message)
	output, err := cmd.Output()
	if err != nil {
		return err
	}
	fmt.Println(string(output))
	return nil
}

func GenerateCommit() {
	diff, err := runGitDiff()
	if err != nil {
		fmt.Println("Error running git diff:", err)
		os.Exit(1)
	}

	if diff == "" {
		fmt.Println("No changes detected. Make sure you have staged changes or unstaged modifications.")
		return
	}

	fmt.Println("Git diff output:", diff)
	
	commitMessage := "chore: update dependencies"
	err = runGitCommit(commitMessage)
	if err != nil {
		fmt.Println("Error running git commit:", err)
		os.Exit(1)
	}

	fmt.Println("Git commit message:", commitMessage)
}
