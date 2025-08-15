package main

import (
	"fmt"
	"os"

	commit "github.com/PedroHercules/gommit/internal/modules/commit/services"
	config "github.com/PedroHercules/gommit/internal/modules/config/services"
	llm_provider "github.com/PedroHercules/gommit/internal/providers/llm"
)

func runCommit() error {
	openRouterProvider := llm_provider.NewOpenRouterProvider()
	commitResponse := commit.GenerateCommit(openRouterProvider)
	if commitResponse.IsFailure() {
		fmt.Println(commitResponse.GetError())
		return commitResponse.GetError()
	}

	fmt.Println(commitResponse.GetData().Message)

	return nil
}

func runConfigSetKey(key string) error {
	err := config.AddLlmKey(key)
	if err != nil {
		fmt.Printf("Error storing API key: %v\n", err)
		return err
	}
	fmt.Println("API key stored securely")
	return nil
}

func runConfigGetKey() error {
	key, err := config.GetLlmKey()
	if err != nil {
		fmt.Printf("Error retrieving API key: %v\n", err)
		return err
	}
	if key == "" {
		fmt.Println("No API key found")
		return nil
	}
	fmt.Printf("API key: %s\n", key)
	return nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: gommit <command> [options]")
		fmt.Println("Commands:")
		fmt.Println("  commit                 - Generate and create commit")
		fmt.Println("  config set-key <key>   - Store API key securely")
		fmt.Println("  config get-key         - Retrieve stored API key")
		fmt.Println("  pr                     - Generate PR (not implemented)")
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "commit":
		fmt.Println("Running commit command")
		runCommit()
	case "config":
		if len(os.Args) < 3 {
			fmt.Println("Usage: gommit config <subcommand>")
			fmt.Println("Subcommands:")
			fmt.Println("  set-key <key>  - Store API key securely")
			fmt.Println("  get-key        - Retrieve stored API key")
			os.Exit(1)
		}
		subcommand := os.Args[2]
		switch subcommand {
		case "set-key":
			if len(os.Args) < 4 {
				fmt.Println("Usage: gommit config set-key <api-key>")
				os.Exit(1)
			}
			apiKey := os.Args[3]
			runConfigSetKey(apiKey)
		case "get-key":
			runConfigGetKey()
		default:
			fmt.Printf("Unknown config subcommand: %s\n", subcommand)
			os.Exit(1)
		}
	case "pr":
		fmt.Println("PR command not implemented yet")
	default:
		fmt.Printf("Unknown command: %s. Available commands: commit, config, pr\n", command)
		os.Exit(1)
	}
}