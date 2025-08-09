package main

import (
	"flag"
	"fmt"
	"os"

	commit "github.com/PedroHercules/gommit/internal/modules/commit/services"
)

func runCommit() error {
	commit.GenerateCommit()
	return nil
}

func main() {
	command := flag.String("c", "commit", "Command to run (commit or pr)")
	flag.Parse()

	switch *command {
	case "commit":
		fmt.Println("Running commit command")
		runCommit()
	case "pr":
		fmt.Println("PR command not implemented yet")
	default:
		fmt.Printf("Unknown command: %s. Available commands: commit, pr\n", *command)
		os.Exit(1)
	}
}
