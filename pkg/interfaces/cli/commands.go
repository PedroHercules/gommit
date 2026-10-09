// Package cli provides the command-line interface for the gommit application.
// This package implements the interface layer of the Clean Architecture,
// handling user input and presenting output.
package cli

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	commit_services "github.com/PedroHercules/gommit/pkg/application/services/commit"
	config_services "github.com/PedroHercules/gommit/pkg/application/services/config"
	pr_services "github.com/PedroHercules/gommit/pkg/application/services/pull-request"
	"golang.org/x/term"
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
	fmt.Println("Analyzing staged changes...")

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

	fmt.Println("Generating commit message with AI...")
	resp, err := c.commitService.GenerateCommitPreview(req)
	if err != nil {
		return fmt.Errorf("failed to generate commit: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("commit generation failed: %s", resp.ErrorMessage)
	}

	// Display results
	fmt.Printf("\nGenerated commit message:\n%s\n\n", resp.CommitMessage)
	fmt.Printf("Changes summary: %s\n", resp.ChangesSummary)
	fmt.Printf("Model used: %s\n", resp.Model)
	fmt.Printf("Tokens consumed: %d\n", resp.TokensUsed)

	// Show warnings if any
	if len(resp.Warnings) > 0 {
		fmt.Println("\nWarnings:")
		for _, warning := range resp.Warnings {
			fmt.Printf("  - %s\n", warning)
		}
	}

	// If dry run, just show the message and exit
	if dryRun {
		fmt.Println("\nDry run completed - no changes were committed")
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

		fmt.Printf("\nSuccessfully committed with hash: %s\n", confirmCommitResponse.CommitHash)
	}

	// Ask user if they want to commit (only when --commit flag was not used)
	fmt.Print("\nDo you want to commit these changes? (y/N): ")
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

		fmt.Printf("\nSuccessfully committed with hash: %s\n", confirmCommitResponse.CommitHash)
	} else {
		fmt.Println("\nCommit cancelled. To commit later, run: git commit -m \"" + resp.CommitMessage + "\"")
	}

	return nil
}

func (c *CLI) handlePr(args []string) error {
	fmt.Println("Analyzing repository changes...")

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

	fmt.Println("Generating pull request description with AI...")
	resp, err := c.prService.GeneratePRPreview(req)
	if err != nil {
		return fmt.Errorf("failed to generate PR message: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("PR message generation failed: %s", resp.ErrorMessage)
	}

	// Display results
	fmt.Printf("Generated PR title:\n%s\n\n", resp.PullRequest.Title)
	fmt.Printf("\nGenerated PR description:\n%s\n\n", resp.PullRequest.Body)

	return nil
}

// handleConfig processes configuration-related commands.
func (c *CLI) handleConfig(args []string) error {
	if len(args) == 0 {
		if !term.IsTerminal(int(os.Stdin.Fd())) || !term.IsTerminal(int(os.Stdout.Fd())) {
			return fmt.Errorf("interactive setup requires a terminal; use `gmit config help` for configuration commands")
		}
		return c.handleProviderSetup()
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

func (c *CLI) handleProviderSetup() error {
	providerChoice, err := promptChoice("Choose a provider", []string{"OpenRouter", "Grok"})
	if err != nil {
		return err
	}
	if providerChoice == "" {
		fmt.Println("Setup cancelled")
		return nil
	}
	provider := "openrouter"
	authMethod := "api_key"
	apiKey := ""

	if providerChoice == "OpenRouter" {
		fmt.Println("OpenRouter controls model pricing and billing. Gommit cannot cap charges; free models are recommended.")
		apiKey, err = promptAPIKey("OpenRouter API key")
		if err != nil {
			return err
		}
	} else {
		provider = "grok"
		authChoice, choiceErr := promptChoice("Connect to Grok using", []string{"OAuth login", "xAI API key"})
		if choiceErr != nil {
			return choiceErr
		}
		if authChoice == "" {
			fmt.Println("Setup cancelled")
			return nil
		}
		if authChoice == "OAuth login" {
			if err := ensureGrokCLI(); err != nil {
				return err
			}
			if err := runGrokLogin(); err != nil {
				return err
			}
			authMethod = "oauth"
		} else {
			fmt.Println("xAI API usage may incur charges based on the selected model; Gommit cannot cap them.")
			apiKey, err = promptAPIKey("xAI API key")
			if err != nil {
				return err
			}
		}
	}

	modelsResponse, err := c.configService.GetAvailableModelsFor(provider, authMethod, apiKey)
	if err != nil {
		return fmt.Errorf("failed to list %s models: %w", provider, err)
	}
	if modelsResponse.ErrorMessage != "" {
		return fmt.Errorf("failed to list %s models: %s", provider, modelsResponse.ErrorMessage)
	}
	model, err := pickModel(modelsResponse.Models)
	if err != nil {
		return err
	}
	if model == "" {
		fmt.Println("Setup cancelled")
		return nil
	}
	if err := c.configService.ConfigureProvider(provider, authMethod, apiKey); err != nil {
		return fmt.Errorf("failed to configure %s: %w", provider, err)
	}
	modelResponse, err := c.configService.SetProviderDefaultModel(provider, model)
	if err != nil {
		return fmt.Errorf("failed to save model: %w", err)
	}
	if !modelResponse.Success {
		return fmt.Errorf("failed to save model: %s", modelResponse.ErrorMessage)
	}
	fmt.Printf("Configured %s with model %s\n", providerChoice, model)
	return nil
}

func promptAPIKey(label string) (string, error) {
	fmt.Printf("%s: ", label)
	key, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		return "", fmt.Errorf("could not read API key: %w", err)
	}
	if strings.TrimSpace(string(key)) == "" {
		return "", fmt.Errorf("API key cannot be empty")
	}
	return strings.TrimSpace(string(key)), nil
}

func readTerminalLine() (string, error) {
	var line strings.Builder
	var char [1]byte
	for {
		n, err := os.Stdin.Read(char[:])
		if err != nil {
			return "", err
		}
		if n == 0 {
			continue
		}
		if char[0] == '\n' || char[0] == '\r' {
			return line.String(), nil
		}
		line.WriteByte(char[0])
	}
}

func ensureGrokCLI() error {
	if _, err := exec.LookPath("grok"); err == nil {
		return nil
	}
	const docsURL = "https://docs.x.ai/build/enterprise"
	const installPackage = "@xai-official/grok"
	fmt.Println("OAuth requires the official Grok CLI.")
	fmt.Printf("Check the current install instructions at %s\n", docsURL)
	fmt.Printf("The documented npm package is %s. After checking the docs, Gommit can run: npm install --global %s\n", installPackage, installPackage)
	confirmed, err := promptYesNo("Did you check the official docs and want Gommit to install this package globally? (y/N): ")
	if err != nil {
		return err
	}
	if !confirmed {
		return fmt.Errorf("install the official Grok CLI using the current instructions at %s, then run `gmit config` again", docsURL)
	}
	if _, err := exec.LookPath("npm"); err != nil {
		return fmt.Errorf("npm was not found; install the official Grok CLI manually using %s", docsURL)
	}
	cmd := exec.Command("npm", "install", "--global", installPackage)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Grok CLI installation failed; check %s and retry: %w", docsURL, err)
	}
	if _, err := exec.LookPath("grok"); err != nil {
		return fmt.Errorf("installation completed but `grok` was not found on PATH; check %s", docsURL)
	}
	return nil
}

func runGrokLogin() error {
	cmd := exec.Command("grok", "login")
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Grok login failed: %w", err)
	}
	return nil
}

func promptYesNo(prompt string) (bool, error) {
	fmt.Print(prompt)
	answer, err := readTerminalLine()
	if err != nil {
		return false, err
	}
	return strings.EqualFold(strings.TrimSpace(answer), "y") || strings.EqualFold(strings.TrimSpace(answer), "yes"), nil
}

// handleSetAPIKey processes the set-key command.
func (c *CLI) handleSetAPIKey(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("API key is required. Usage: gommit config set-key <your-api-key>")
	}

	fmt.Println("Storing API key securely...")

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

	fmt.Printf("Success: %s\n", resp.Message)
	if resp.KeyMasked != "" {
		fmt.Printf("API Key: %s\n", resp.KeyMasked)
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
		fmt.Println("API key is not configured")
		fmt.Println("Run 'gommit config set-key <your-api-key>' to configure it")
		return nil
	}

	fmt.Printf("API Key: %s\n", resp.MaskedAPIKey)
	return nil
}

// handleRemoveAPIKey processes the remove-key command.
func (c *CLI) handleRemoveAPIKey() error {
	fmt.Println("Removing API key...")

	err := c.configService.RemoveAPIKey()
	if err != nil {
		return fmt.Errorf("failed to remove API key: %w", err)
	}

	fmt.Println("API key removed successfully")
	return nil
}

// handleSetModel processes the set-model command.
func (c *CLI) handleSetModel(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("model is required. Usage: gommit config set-model <model-id>")
	}

	fmt.Println("Setting default model...")

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

	fmt.Printf("Success: %s\n", resp.Message)
	fmt.Println("Cost notice: the provider controls model pricing and billing; Gommit cannot limit charges. Check pricing before using paid models.")
	return nil
}

// handleGetModel processes the get-model command.
func (c *CLI) handleGetModel() error {
	resp, err := c.configService.GetDefaultModelInfo()
	if err != nil {
		return fmt.Errorf("failed to get model: %w", err)
	}

	if !resp.Configured {
		fmt.Println("Default model is not configured")
		fmt.Println("Run 'gommit config set-model <model-id>' to configure it")
		return nil
	}

	fmt.Printf("Default Model: %s\n", resp.Model)
	return nil
}

// handleRemoveModel processes the remove-model command.
func (c *CLI) handleRemoveModel() error {
	fmt.Println("Removing default model...")

	err := c.configService.RemoveDefaultModel()
	if err != nil {
		return fmt.Errorf("failed to remove model: %w", err)
	}

	fmt.Println("Default model removed successfully")
	return nil
}

// handleListModels processes the list-models command.
func (c *CLI) handleListModels() error {
	fmt.Println("Fetching available models...")

	resp, err := c.configService.GetAvailableModels()
	if err != nil {
		return fmt.Errorf("failed to get models: %w", err)
	}

	if resp.ErrorMessage != "" {
		return fmt.Errorf("failed to get models: %s", resp.ErrorMessage)
	}

	if len(resp.Models) == 0 {
		fmt.Println("No models available")
		return nil
	}

	stdinFD := int(os.Stdin.Fd())
	stdoutFD := int(os.Stdout.Fd())
	if !term.IsTerminal(stdinFD) || !term.IsTerminal(stdoutFD) {
		return printModels(os.Stdout, resp.Models)
	}

	selectedModel, pickerErr := pickModel(resp.Models)
	if pickerErr != nil {
		return fmt.Errorf("model search failed: %w", pickerErr)
	}
	if selectedModel == "" {
		fmt.Println("Model selection cancelled")
		return nil
	}
	return c.handleSetModel([]string{selectedModel})
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

	fmt.Println("\nConfiguration Summary:")
	fmt.Printf("Provider: %s (%s)\n", resp.Provider, resp.AuthMethod)

	// Authentication status
	if resp.AuthMethod == "oauth" && resp.APIKeyConfigured {
		fmt.Printf("Authentication: %s\n", resp.APIKeyMasked)
	} else if resp.APIKeyConfigured {
		fmt.Printf("API Key: %s\n", resp.APIKeyMasked)
	} else {
		fmt.Println("Authentication: Not configured")
	}

	// Default model status
	if resp.DefaultModelSet {
		fmt.Printf("Default Model: %s\n", resp.DefaultModel)
	} else {
		fmt.Println("Default Model: Not configured")
	}

	// Available models count
	if resp.AvailableModelsCount > 0 {
		fmt.Printf("Available Models: %d\n", resp.AvailableModelsCount)
	} else {
		fmt.Println("Available Models: Unable to fetch")
	}

	return nil
}

// handleConfigValidate processes the config validate command.
func (c *CLI) handleConfigValidate() error {
	fmt.Println("Validating configuration...")

	resp, err := c.configService.ValidateConfiguration()
	if err != nil {
		return fmt.Errorf("failed to validate configuration: %w", err)
	}

	if resp.ErrorMessage != "" {
		return fmt.Errorf("validation failed: %s", resp.ErrorMessage)
	}

	if resp.Valid {
		fmt.Println("Configuration is valid and ready to use")
	} else {
		fmt.Println("Configuration has issues")
	}

	if len(resp.Issues) > 0 {
		fmt.Println("\nIssues:")
		for _, issue := range resp.Issues {
			fmt.Printf("   • %s\n", issue)
		}
	}

	if len(resp.Recommendations) > 0 {
		fmt.Println("\nRecommendations:")
		for _, rec := range resp.Recommendations {
			fmt.Printf("   • %s\n", rec)
		}
	}

	return nil
}
