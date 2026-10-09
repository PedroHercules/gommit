package cli

import "fmt"

var version = "dev"

// showHelp displays the main help message.
func (c *CLI) showHelp() error {
	fmt.Println(`
Gommit - AI-Powered Git Commit Message Generator

USAGE:
    gommit [COMMAND] [OPTIONS]

COMMANDS:
    commit              Generate and commit with AI (default)
    pr                  Generate Pull Request description
    config              Manage configuration
    validate <message>  Validate a commit message
    version             Show version information
    help                Show this help message

COMMIT OPTIONS:
    --model, -m <model>    Use specific AI model
    --commit              Generate message and commit automatically
    --dry-run            Show what would be done without committing
    --force              Force commit even with warnings

PR OPTIONS:
    --base-branch <branch> Specify base branch for comparison (default: main)

CONFIG COMMANDS:
    config                   Open provider setup and model selector
    config set-key <key>      Set active provider API key
    config get-key           Show current API key (masked)
    config remove-key        Remove API key
    config set-model <model> Set default AI model
    config get-model         Show current default model
    config remove-model      Remove default model
    config list-models       Search and select a model for the active provider
    config summary           Show configuration summary
    config validate          Validate current configuration

EXAMPLES:
    gommit commit                           # Generate message and commit
    gommit commit --model provider/model-id # Use a model from the active provider
    gommit pr                               # Generate PR description
    gommit pr --base-branch develop         # Generate PR against develop branch
    gommit config set-key sk-xxx            # Set API key
    gommit config list-models               # Search and select a model
    gommit config summary                   # Show configuration summary   
    gommit validate "feat: add new feature"  # Validate message

NOTES:
    • Supports OpenRouter and Grok (xAI API key or official Grok CLI OAuth)
    • Works with staged Git changes
    • Supports Conventional Commits format
    • Configuration is stored securely
    • Providers control pricing and billing; Gommit cannot cap charges

For more information, visit: https://github.com/PedroHercules/gommit`)

	return nil
}

// showConfigHelp displays help for config commands.
func (c *CLI) showConfigHelp() error {
	fmt.Println(`
Gommit Configuration Commands

USAGE:
    gommit config [COMMAND] [OPTIONS]

Run gmit config without a command to select OpenRouter or Grok, authenticate, and choose a model.

COMMANDS:
    set-key <key>      Set API key for the active provider
    get-key           Show current API key (masked)
    remove-key        Remove stored API key
    set-model <model> Set default AI model
    get-model         Show current default model
    remove-model      Remove default model setting
    list-models       Search and select a model for the active provider
    summary           Show complete configuration summary
    validate          Validate current configuration

EXAMPLES:
    gmit config                                  # Interactive provider setup
    gmit config set-key <key>                    # Set active provider API key
    gommit config set-model provider/model-id    # Set default model
    gommit config list-models                    # Search and select a model
    gommit config summary                        # Check all settings
    gommit config validate                       # Test configuration

NOTES:
    • API key is stored securely in system keyring
    • Default model is used when --model is not specified
    • Configuration is validated before use
    • Some commands require valid API key
    • The provider controls pricing and billing; Gommit cannot limit charges
    • Check current prices before selecting paid models
    • In an interactive terminal, list-models filters while you type; Enter saves the selected model as default
    • Without an interactive terminal, list-models prints the full catalog`)

	return nil
}

// showVersion displays version information.
func (c *CLI) showVersion() error {
	fmt.Printf(`
Gommit v%s

AI-Powered Git Commit Message Generator

Built with:
    • Go 1.21+
    • OpenRouter and xAI APIs

Author: Pedro Hercules
License: MIT
Repository: https://github.com/PedroHercules/gommit`, version)

	return nil
}
