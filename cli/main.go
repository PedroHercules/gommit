package main

import (
	"flag"
	"fmt"
	"os"

	commit "github.com/PedroHercules/gommit/internal/modules/commit/services"
	llm_provider "github.com/PedroHercules/gommit/internal/providers/llm"
)

func runCommit() error {
	openRouterProvider := llm_provider.NewOpenRouterProvider()
	commitResponse := commit.GenerateCommit(openRouterProvider)
	if commitResponse.IsFailure() {
		fmt.Println(commitResponse.GetError())
	}

	fmt.Println(commitResponse.GetData().Message)

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
