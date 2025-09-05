// Package cli provides the command-line interface for the gommit application.
// This package implements the interface layer of the Clean Architecture,
// handling user input and presenting output.
package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	commit_services "github.com/PedroHercules/gommit/pkg/application/services/commit"
	config_services "github.com/PedroHercules/gommit/pkg/application/services/config"
	pr_services "github.com/PedroHercules/gommit/pkg/application/services/pull-request"
)

// CLI represents the command-line interface.
// It coordinates between user input and application services.
type CLI struct {
	commitService *commit_services.CommitService
	configService *config_services.ConfigService
	prService     *pr_services.ServiceContainer
}

// NewCLI creates a new CLI instance.
func NewCLI(commitService *commit_services.CommitService, configService *config_services.ConfigService, prService *pr_services.ServiceContainer) *CLI {
	return &CLI{
		commitService: commitService,
		configService: configService,
		prService:     prService,
	}
}

// Run executes the CLI with the provided arguments.
// This is the main entry point for command processing.
func (c *CLI) Run(args []string) error {
	if len(args) < 2 {
		return c.showHelp()
	}

	command := args[1]
	switch command {
	case "help", "-h", "--help":
		return c.showHelp()
	case "commit":
		return c.handleCommit(args[2:])
	case "config":
		return c.handleConfig(args[2:])
	case "version":
		return c.showVersion()
	case "pr":
		return c.handlePr(args[2:])
	default:
		// If no subcommand is provided, default to commit
		if strings.HasPrefix(command, "-") {
			return c.showHelp()
		}
		// Treat the first argument as part of commit command
		return c.handleCommit(args[1:])
	}
}

// handleCommit processes commit-related commands.
func (c *CLI) handleCommit(args []string) error {
	fmt.Println("🔍 Analyzing staged changes...")

	// Parse commit flags
	var model string
	dryRun := false
	force := false
	autoCommit := false

	// Simple flag parsing
	for i, arg := range args {
		switch arg {
		case "--model", "-m":
			if i+1 < len(args) {
				model = args[i+1]
			}
		case "--commit":
			autoCommit = true
		case "--dry-run":
			dryRun = true
		case "--force":
			force = true
		}
	}

	// Generate commit message
	req := commit_services.GenerateCommitPreviewDTO{
		Model:  model,
		DryRun: dryRun,
		Force:  force,
	}

	fmt.Println("🤖 Generating commit message with AI...")
	resp, err := c.commitService.GenerateCommitPreview(req)
	if err != nil {
		return fmt.Errorf("failed to generate commit: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("commit generation failed: %s", resp.ErrorMessage)
	}

	// Display results
	fmt.Printf("\n📝 Generated commit message:\n%s\n\n", resp.CommitMessage)
	fmt.Printf("📊 Changes: %s\n", resp.ChangesSummary)
	fmt.Printf("🤖 Model: %s\n", resp.Model)
	fmt.Printf("🔢 Tokens used: %d\n", resp.TokensUsed)

	// Show warnings if any
	if len(resp.Warnings) > 0 {
		fmt.Println("\n⚠️  Warnings:")
		for _, warning := range resp.Warnings {
			fmt.Printf("   • %s\n", warning)
		}
	}

	// If dry run, just show the message and exit
	if dryRun {
		fmt.Println("\n🧪 Dry run completed - no changes were committed")
		return nil
	}

	// If auto-commit was requested and successful, show result
	if autoCommit {
		confirmCommitResponse, confirmCommitErr := c.commitService.ConfirmCommit(commit_services.ConfirmCommitDTO{
			Message: resp.CommitMessage,
			DryRun:  dryRun,
		})

		if confirmCommitErr != nil {
			return fmt.Errorf("failed to confirm commit: %w", confirmCommitErr)
		}

		if !confirmCommitResponse.Success {
			return fmt.Errorf("commit confirmation failed: %s", confirmCommitResponse.ErrorMessage)
		}

		fmt.Printf("\n✅ Successfully committed with hash: %s\n", confirmCommitResponse.CommitHash)
	}

	// Ask user if they want to commit (only when --commit flag was not used)
	fmt.Print("\n❓ Do you want to commit these changes? (y/N): ")
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read user input: %w", err)
	}

	response = strings.TrimSpace(strings.ToLower(response))
	if response == "y" || response == "yes" {
		// Perform the actual commit
		confirmCommitResponse, confirmCommitErr := c.commitService.ConfirmCommit(commit_services.ConfirmCommitDTO{
			Message: resp.CommitMessage,
			DryRun:  false,
		})
		if confirmCommitErr != nil {
			return fmt.Errorf("failed to commit: %w", confirmCommitErr)
		}

		if !confirmCommitResponse.Success {
			return fmt.Errorf("commit failed: %s", confirmCommitResponse.ErrorMessage)
		}

		fmt.Printf("\n✅ Successfully committed with hash: %s\n", confirmCommitResponse.CommitHash)
	} else {
		fmt.Println("\n💡 Commit cancelled. To commit later, run: git commit -m \"" + resp.CommitMessage + "\"")
	}

	return nil
}

func (c *CLI) handlePr(args []string) error {
	fmt.Println("🔍 Analyzing staged changes...")

	// Parse commit flags
	var baseBranch string

	// Simple flag parsing
	for i, arg := range args {
		switch arg {
		case "--base-branch":
			if i+1 < len(args) {
				baseBranch = args[i+1]
			}
		}
	}

	// Generate PR message
	req := &pr_services.GeneratePRPreviewRequest{
		BaseBranch: baseBranch,
	}

	fmt.Println("🤖 Generating PR message with AI...")
	resp, err := c.prService.GeneratePRPreview(req)
	if err != nil {
		return fmt.Errorf("failed to generate PR message: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("PR message generation failed: %s", resp.ErrorMessage)
	}

	// Display results
	fmt.Printf("📝 Generated PR title:\n%s\n\n", resp.PullRequest.Title)
	fmt.Printf("\n📝 Generated PR message:\n%s\n\n", resp.PullRequest.Body)

	return nil
}

// handleConfig processes configuration-related commands.
func (c *CLI) handleConfig(args []string) error {
	if len(args) == 0 {
		return c.showConfigHelp()
	}

	subcommand := args[0]
	switch subcommand {
	case "set-key":
		return c.handleSetAPIKey(args[1:])
	case "get-key":
		return c.handleGetAPIKey()
	case "remove-key":
		return c.handleRemoveAPIKey()
	case "set-model":
		return c.handleSetModel(args[1:])
	case "get-model":
		return c.handleGetModel()
	case "remove-model":
		return c.handleRemoveModel()
	case "list-models":
		return c.handleListModels()
	case "summary":
		return c.handleConfigSummary()
	case "validate":
		return c.handleConfigValidate()
	default:
		return c.showConfigHelp()
	}
}

// handleSetAPIKey processes the set-key command.
func (c *CLI) handleSetAPIKey(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("API key is required. Usage: gommit config set-key <your-api-key>")
	}

	fmt.Println("🔐 Storing API key securely...")

	req := config_services.SetupAPIKeyRequest{
		APIKey:      args[0],
		ValidateKey: true,
	}

	resp, err := c.configService.SetupAPIKey(req)
	if err != nil {
		return fmt.Errorf("failed to set API key: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("failed to set API key: %s", resp.ErrorMessage)
	}

	fmt.Printf("✅ %s\n", resp.Message)
	if resp.KeyMasked != "" {
		fmt.Printf("🔑 API Key: %s\n", resp.KeyMasked)
	}

	return nil
}

// handleGetAPIKey processes the get-key command.
func (c *CLI) handleGetAPIKey() error {
	resp, err := c.configService.GetAPIKeyInfo()
	if err != nil {
		return fmt.Errorf("failed to get API key: %w", err)
	}

	if !resp.Configured {
		fmt.Println("❌ API key is not configured")
		fmt.Println("💡 Run 'gommit config set-key <your-api-key>' to configure it")
		return nil
	}

	fmt.Printf("🔑 API Key: %s\n", resp.MaskedAPIKey)
	return nil
}

// handleRemoveAPIKey processes the remove-key command.
func (c *CLI) handleRemoveAPIKey() error {
	fmt.Println("🗑️ Removing API key...")

	err := c.configService.RemoveAPIKey()
	if err != nil {
		return fmt.Errorf("failed to remove API key: %w", err)
	}

	fmt.Println("✅ API key removed successfully")
	return nil
}

// handleSetModel processes the set-model command.
func (c *CLI) handleSetModel(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("model is required. Usage: gommit config set-model <model-id>")
	}

	fmt.Println("⚙️ Setting default model...")

	req := config_services.SetupModelRequest{
		Model:         args[0],
		ValidateModel: true,
	}

	resp, err := c.configService.SetupDefaultModel(req)
	if err != nil {
		return fmt.Errorf("failed to set model: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("failed to set model: %s", resp.ErrorMessage)
	}

	fmt.Printf("✅ %s\n", resp.Message)
	return nil
}

// handleGetModel processes the get-model command.
func (c *CLI) handleGetModel() error {
	resp, err := c.configService.GetDefaultModelInfo()
	if err != nil {
		return fmt.Errorf("failed to get model: %w", err)
	}

	if !resp.Configured {
		fmt.Println("❌ Default model is not configured")
		fmt.Println("💡 Run 'gommit config set-model <model-id>' to configure it")
		return nil
	}

	fmt.Printf("🤖 Default Model: %s\n", resp.Model)
	return nil
}

// handleRemoveModel processes the remove-model command.
func (c *CLI) handleRemoveModel() error {
	fmt.Println("🗑️ Removing default model...")

	err := c.configService.RemoveDefaultModel()
	if err != nil {
		return fmt.Errorf("failed to remove model: %w", err)
	}

	fmt.Println("✅ Default model removed successfully")
	return nil
}

// handleListModels processes the list-models command.
func (c *CLI) handleListModels() error {
	fmt.Println("📋 Fetching available models...")

	resp, err := c.configService.GetAvailableModels()
	if err != nil {
		return fmt.Errorf("failed to get models: %w", err)
	}

	if resp.ErrorMessage != "" {
		return fmt.Errorf("failed to get models: %s", resp.ErrorMessage)
	}

	if len(resp.Models) == 0 {
		fmt.Println("❌ No models available")
		return nil
	}

	fmt.Printf("\n🤖 Available Models (%d):\n\n", len(resp.Models))
	for _, model := range resp.Models {
		fmt.Printf("  • %s\n", model.ID)
		if model.Name != "" {
			fmt.Printf("    Name: %s\n", model.Name)
		}
		if model.Provider != "" {
			fmt.Printf("    Provider: %s\n", model.Provider)
		}
		if model.ContextSize > 0 {
			fmt.Printf("    Context: %d tokens\n", model.ContextSize)
		}
		fmt.Println()
	}

	return nil
}

// handleConfigSummary processes the config summary command.
func (c *CLI) handleConfigSummary() error {
	resp, err := c.configService.GetConfigSummary()
	if err != nil {
		return fmt.Errorf("failed to get config summary: %w", err)
	}

	if resp.ErrorMessage != "" {
		return fmt.Errorf("failed to get config summary: %s", resp.ErrorMessage)
	}

	fmt.Println("\n⚙️ Configuration Summary:")

	// API Key status
	if resp.APIKeyConfigured {
		fmt.Printf("🔑 API Key: %s\n", resp.APIKeyMasked)
	} else {
		fmt.Println("🔑 API Key: ❌ Not configured")
	}

	// Default model status
	if resp.DefaultModelSet {
		fmt.Printf("🤖 Default Model: %s\n", resp.DefaultModel)
	} else {
		fmt.Println("🤖 Default Model: ❌ Not configured")
	}

	// Available models count
	if resp.AvailableModelsCount > 0 {
		fmt.Printf("📋 Available Models: %d\n", resp.AvailableModelsCount)
	} else {
		fmt.Println("📋 Available Models: ❌ Unable to fetch")
	}

	return nil
}

// handleConfigValidate processes the config validate command.
func (c *CLI) handleConfigValidate() error {
	fmt.Println("🔍 Validating configuration...")

	resp, err := c.configService.ValidateConfiguration()
	if err != nil {
		return fmt.Errorf("failed to validate configuration: %w", err)
	}

	if resp.ErrorMessage != "" {
		return fmt.Errorf("validation failed: %s", resp.ErrorMessage)
	}

	if resp.Valid {
		fmt.Println("✅ Configuration is valid and ready to use")
	} else {
		fmt.Println("❌ Configuration has issues")
	}

	if len(resp.Issues) > 0 {
		fmt.Println("\n🚨 Issues:")
		for _, issue := range resp.Issues {
			fmt.Printf("   • %s\n", issue)
		}
	}

	if len(resp.Recommendations) > 0 {
		fmt.Println("\n💡 Recommendations:")
		for _, rec := range resp.Recommendations {
			fmt.Printf("   • %s\n", rec)
		}
	}

	return nil
}
