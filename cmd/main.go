package main

import (
	"fmt"
	"os"

	commit "github.com/PedroHercules/gommit/internal/modules/commit/services"
	config "github.com/PedroHercules/gommit/internal/modules/config/services"
	llm_provider "github.com/PedroHercules/gommit/internal/providers/llm"
)

func runCommit() error {
	fmt.Println("🔍 Analyzing staged changes...")
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
	fmt.Println("🔐 Storing API key securely...")
	err := config.AddLlmKey(key)
	if err != nil {
		fmt.Printf("Error storing API key: %v\n", err)
		return err
	}
	fmt.Println("✅ API key stored securely")
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

func runConfigSetModel(model string) error {
	fmt.Println("⚙️ Setting default model...")
	err := config.SetDefaultModel(model)
	if err != nil {
		fmt.Printf("Error storing default model: %v\n", err)
		return err
	}
	fmt.Printf("✅ Default model set to: %s\n", model)
	return nil
}

func runConfigGetModel() error {
	model, err := config.GetDefaultModel()
	if err != nil {
		fmt.Printf("Error retrieving default model: %v\n", err)
		return err
	}
	if model == "" {
		fmt.Println("No default model configured")
		return nil
	}
	fmt.Printf("Default model: %s\n", model)
	return nil
}

func runConfigRemoveModel() error {
	fmt.Println("🗑️ Removing default model...")
	err := config.RemoveDefaultModel()
	if err != nil {
		fmt.Printf("Error removing default model: %v\n", err)
		return err
	}
	fmt.Println("✅ Default model removed")
	return nil
}

func runConfigRemoveKey() error {
	fmt.Println("🗑️ Removing API key...")
	err := config.RemoveLlmKey()
	if err != nil {
		fmt.Printf("Error removing API key: %v\n", err)
		return err
	}
	fmt.Println("✅ API key removed")
	return nil
}

func runHelp() {
	fmt.Println("🚀 Gommit - AI-powered Git commit message generator")
	fmt.Println("")
	fmt.Println("USAGE:")
	fmt.Println("  gommit <command> [options]")
	fmt.Println("")
	fmt.Println("COMMANDS:")
	fmt.Println("")
	fmt.Println("  commit                     Generate and create commit message")
	fmt.Println("                             Analyzes staged changes and creates a conventional commit")
	fmt.Println("")
	fmt.Println("  help, --help, -h           Show this help message")
	fmt.Println("")
	fmt.Println("CONFIGURATION:")
	fmt.Println("")
	fmt.Println("  config set-key <key>       Store OpenRouter API key securely")
	fmt.Println("                             Required for AI-powered commit generation")
	fmt.Println("                             Get your key at: https://openrouter.ai/keys")
	fmt.Println("")
	fmt.Println("  config get-key             Display currently stored API key")
	fmt.Println("                             Shows the API key (masked for security)")
	fmt.Println("")
	fmt.Println("  config remove-key          Remove stored API key")
	fmt.Println("                             Clears the API key from secure storage")
	fmt.Println("")
	fmt.Println("MODEL MANAGEMENT:")
	fmt.Println("")
	fmt.Println("  config set-model <model>   Set preferred LLM model")
	fmt.Println("                             When set, gommit will always use this model")
	fmt.Println("                             Disables automatic model selection")
	fmt.Println("                             Example: openai/gpt-4o-mini")
	fmt.Println("")
	fmt.Println("  config get-model           Show currently configured default model")
	fmt.Println("                             Displays the model ID if one is set")
	fmt.Println("")
	fmt.Println("  config remove-model        Remove default model configuration")
	fmt.Println("                             Returns to automatic model selection")
	fmt.Println("                             Gommit will choose the best available free model")
	fmt.Println("")
	fmt.Println("MODEL SELECTION BEHAVIOR:")
	fmt.Println("")
	fmt.Println("  • AUTOMATIC MODE (default): Gommit automatically selects the best")
	fmt.Println("    available free model based on context length and capabilities")
	fmt.Println("")
	fmt.Println("  • MANUAL MODE: When you set a default model, gommit will always")
	fmt.Println("    try to use that specific model first, with fallback to other models")
	fmt.Println("")
	fmt.Println("EXAMPLES:")
	fmt.Println("")
	fmt.Println("  # First time setup")
	fmt.Println("  gommit config set-key sk-or-v1-...")
	fmt.Println("")
	fmt.Println("  # Generate commit (automatic model selection)")
	fmt.Println("  git add .")
	fmt.Println("  gommit commit")
	fmt.Println("")
	fmt.Println("  # Set preferred model")
	fmt.Println("  gommit config set-model openai/gpt-4o-mini")
	fmt.Println("")
	fmt.Println("  # Return to automatic selection")
	fmt.Println("  gommit config remove-model")
	fmt.Println("")
	fmt.Println("SUPPORTED MODELS:")
	fmt.Println("")
	fmt.Println("  • openai/gpt-4o-mini (recommended)")
	fmt.Println("  • openai/gpt-3.5-turbo")
	fmt.Println("  • anthropic/claude-3-haiku")
	fmt.Println("  • meta-llama/llama-3.1-8b-instruct:free")
	fmt.Println("  • google/gemma-7b-it:free")
	fmt.Println("  • And many more at: https://openrouter.ai/models")
	fmt.Println("")
	fmt.Println("For more information, visit: https://github.com/PedroHercules/gommit")
}

func main() {
	if len(os.Args) < 2 {
		runHelp()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "help", "--help", "-h":
		runHelp()
	case "commit":
		fmt.Println("Running commit command")
		runCommit()
	case "config":
		if len(os.Args) < 3 {
			fmt.Println("Usage: gommit config <subcommand>")
			fmt.Println("Subcommands:")
			fmt.Println("  set-key <key>      - Store API key securely")
			fmt.Println("  get-key            - Retrieve stored API key")
			fmt.Println("  remove-key         - Remove stored API key")
			fmt.Println("  set-model <model>  - Set default LLM model")
			fmt.Println("  get-model          - Get current default model")
			fmt.Println("  remove-model       - Remove default model")
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
		case "remove-key":
			runConfigRemoveKey()
		case "set-model":
			if len(os.Args) < 4 {
				fmt.Println("Usage: gommit config set-model <model-id>")
				fmt.Println("Example: gommit config set-model openai/gpt-4o-mini")
				os.Exit(1)
			}
			model := os.Args[3]
			runConfigSetModel(model)
		case "get-model":
			runConfigGetModel()
		case "remove-model":
			runConfigRemoveModel()
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