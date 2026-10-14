// Package cli provides the command-line interface for the gommit application.
// This package implements the interface layer of the Clean Architecture,
// handling user input and presenting output.
package cli

import (
	"fmt"
	"io"
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
	commitService      commitCommandService
	configService      *config_services.ConfigService
	prService          *pr_services.ServiceContainer
	selectCommitAction func() (string, error)
	updates            *updateManager
	updateOutput       io.Writer
}

type commitCommandService interface {
	GenerateCommitPreview(commit_services.GenerateCommitPreviewDTO) (*commit_services.GenerateCommitPreviewEntity, error)
	ConfirmCommit(commit_services.ConfirmCommitDTO) (*commit_services.ConfirmCommitEntity, error)
	ValidateCommitMessage(string) (*commit_services.ValidateCommitMessageResult, error)
	PushCurrentBranch() error
}

const (
	commitAndPushAction = "1. Commit and push"
	commitOnlyAction    = "2. Commit only"
	cancelCommitAction  = "3. Cancel"
)

// NewCLI creates a new CLI instance.
func NewCLI(commitService *commit_services.CommitService, configService *config_services.ConfigService, prService *pr_services.ServiceContainer) *CLI {
	return &CLI{
		commitService: commitService,
		configService: configService,
		prService:     prService,
		updates:       newUpdateManager(version),
		updateOutput:  os.Stderr,
	}
}

// Run executes the CLI with the provided arguments.
// This is the main entry point for command processing.
func (c *CLI) Run(args []string) error {
	update := c.checkForUpdate()
	if len(args) < 2 {
		return c.handleCommit(nil)
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
	case "update":
		return c.handleUpdate(update)
	case "pr":
		return c.handlePr(args[2:])
	case "validate":
		return c.handleValidate(args[2:])
	default:
		if strings.HasPrefix(command, "-") {
			return c.handleCommit(args[1:])
		}
		return fmt.Errorf("unknown command %q; run `gmit help` to see available commands", command)
	}
}

func (c *CLI) checkForUpdate() updateNotice {
	if c.updates == nil {
		return updateNotice{}
	}
	notice, err := c.updates.check()
	if err != nil {
		return updateNotice{}
	}
	if notice.LatestVersion != "" && notice.ReleaseURL != "" {
		notice.printTo(c.updateOutput)
	}
	return notice
}

func (c *CLI) handleUpdate(notice updateNotice) error {
	if notice.LatestVersion != "" && notice.CurrentVersion != "" {
		newer, err := isNewerVersion(notice.LatestVersion, notice.CurrentVersion)
		if err == nil && !newer {
			fmt.Printf("✓ Gmit is already up to date (v%s).\n", notice.CurrentVersion)
			return nil
		}
	}
	if c.updates == nil {
		return fmt.Errorf("npm update is unavailable in this build")
	}
	fmt.Println("› Updating Gmit to the latest npm release...")
	if err := c.updates.update(); err != nil {
		return err
	}
	fmt.Println("✓ Gmit update completed. Run `gmit version` to verify the installed version.")
	return nil
}

// handleCommit processes commit-related commands.
func (c *CLI) handleCommit(args []string) error {
	model, dryRun, autoCommit, err := parseCommitArgs(args)
	if err != nil {
		return err
	}
	if dryRun && autoCommit {
		return fmt.Errorf("--dry-run and --commit cannot be used together")
	}
	if !dryRun && !autoCommit && c.selectCommitAction == nil &&
		(!term.IsTerminal(int(os.Stdin.Fd())) || !term.IsTerminal(int(os.Stdout.Fd()))) {
		return fmt.Errorf("an interactive terminal is required to choose a commit action; use --commit to commit without pushing or --dry-run to preview")
	}
	req := commit_services.GenerateCommitPreviewDTO{Model: model, DryRun: dryRun}

	var resp *commit_services.GenerateCommitPreviewEntity
	err = withSpinner("Drafting commit message", func() error {
		var callErr error
		resp, callErr = c.commitService.GenerateCommitPreview(req)
		if callErr != nil {
			return fmt.Errorf("failed to generate commit: %w", callErr)
		}
		if !resp.Success {
			return fmt.Errorf("commit generation failed: %s", resp.ErrorMessage)
		}
		return nil
	})
	if err != nil {
		return err
	}

	fmt.Printf("\nCommit preview\n%s\n\n", resp.CommitMessage)
	fmt.Printf("Changes  %s\n", resp.ChangesSummary)
	fmt.Printf("Model    %s\n", resp.Model)
	fmt.Printf("Tokens   %d\n", resp.TokensUsed)

	if len(resp.Warnings) > 0 {
		fmt.Println("\n! Review these notes:")
		for _, warning := range resp.Warnings {
			fmt.Printf("  - %s\n", warning)
		}
	}

	if dryRun {
		fmt.Println("\n✓ Preview complete; no changes were committed.")
		return nil
	}

	if autoCommit {
		return c.commitGeneratedMessage(resp.CommitMessage, false)
	}
	action, err := c.chooseCommitAction(resp.CommitMessage)
	if err != nil {
		return err
	}
	switch action {
	case commitAndPushAction:
		return c.commitGeneratedMessage(resp.CommitMessage, true)
	case commitOnlyAction:
		return c.commitGeneratedMessage(resp.CommitMessage, false)
	case cancelCommitAction, "":
		fmt.Println("\n› Commit cancelled. Your changes are still staged.")
		return nil
	default:
		return fmt.Errorf("unknown commit action %q", action)
	}
}

func parseCommitArgs(args []string) (model string, dryRun, autoCommit bool, err error) {
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--model", "-m":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				return "", false, false, fmt.Errorf("%s requires a model ID", args[i])
			}
			i++
			model = args[i]
		case "--commit":
			autoCommit = true
		case "--dry-run":
			dryRun = true
		case "--force":
			return "", false, false, fmt.Errorf("--force is not supported; stage the changes you want to include and review the generated message")
		default:
			return "", false, false, fmt.Errorf("unsupported commit argument %q; run `gmit help`", args[i])
		}
	}
	return model, dryRun, autoCommit, nil
}

func (c *CLI) chooseCommitAction(message string) (string, error) {
	if c.selectCommitAction != nil {
		return c.selectCommitAction()
	}
	return promptChoiceWithMessage("Select a commit action", message, []string{
		commitAndPushAction,
		commitOnlyAction,
		cancelCommitAction,
	})
}

func (c *CLI) commitGeneratedMessage(message string, push bool) error {
	result, err := c.commitService.ConfirmCommit(commit_services.ConfirmCommitDTO{Message: message})
	if err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}
	if !result.Success {
		return fmt.Errorf("commit failed: %s", result.ErrorMessage)
	}
	fmt.Printf("\n✓ Commit created: %s\n", result.CommitHash)
	if push {
		if err := c.commitService.PushCurrentBranch(); err != nil {
			return fmt.Errorf("commit %s succeeded, but push failed: %w", result.CommitHash, err)
		}
		fmt.Println("✓ Pushed current branch to origin.")
	}
	return nil
}

func (c *CLI) handlePr(args []string) error {
	baseBranch, model, err := parsePRArgs(args)
	if err != nil {
		return err
	}
	req := &pr_services.GeneratePRPreviewRequest{BaseBranch: baseBranch, Model: model}
	var resp *pr_services.GeneratePRPreviewResponse
	err = withSpinner("Building pull request preview", func() error {
		var callErr error
		resp, callErr = c.prService.GeneratePRPreview(req)
		if callErr != nil {
			return fmt.Errorf("failed to generate PR description: %w", callErr)
		}
		if !resp.Success {
			return fmt.Errorf("PR description generation failed: %s", resp.ErrorMessage)
		}
		return nil
	})
	if err != nil {
		return err
	}
	fmt.Printf("\nPull request preview\n%s\n", resp.PullRequest.Body)
	return nil
}

func parsePRArgs(args []string) (baseBranch, model string, err error) {
	baseBranch = "main"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--base-branch":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				return "", "", fmt.Errorf("--base-branch requires a branch name")
			}
			i++
			baseBranch = args[i]
		case "--model", "-m":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				return "", "", fmt.Errorf("%s requires a model ID", args[i])
			}
			i++
			model = args[i]
		default:
			return "", "", fmt.Errorf("unsupported PR argument %q; run `gmit help`", args[i])
		}
	}
	return baseBranch, model, nil
}

func (c *CLI) handleValidate(args []string) error {
	message := strings.TrimSpace(strings.Join(args, " "))
	if message == "" {
		return fmt.Errorf("commit message is required; usage: gmit validate \"feat: add feature\"")
	}
	result, err := c.commitService.ValidateCommitMessage(message)
	if err != nil {
		return fmt.Errorf("failed to validate commit message: %w", err)
	}
	if !result.Valid {
		if result.ErrorMessage != "" {
			return fmt.Errorf("invalid commit message: %s", result.ErrorMessage)
		}
		return fmt.Errorf("invalid commit message")
	}
	fmt.Println("✓ Commit message is not empty.")
	if result.IsConventional {
		fmt.Println("✓ Conventional Commits format detected.")
	} else {
		fmt.Println("! Conventional Commits format not detected.")
	}
	for i, warning := range result.Warnings {
		fmt.Printf("! %s\n", warning)
		if i < len(result.Suggestions) {
			fmt.Printf("› Suggestion: %s\n", result.Suggestions[i])
		}
	}
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
	providerChoice, err := promptChoice("Select a provider", []string{"OpenRouter", "Grok"})
	if err != nil {
		return err
	}
	if providerChoice == "" {
		fmt.Println("› Setup cancelled.")
		return nil
	}
	provider := "openrouter"
	authMethod := "api_key"
	apiKey := ""

	if providerChoice == "OpenRouter" {
		fmt.Println("! OpenRouter controls pricing and billing; Gmit cannot cap charges. Free models are recommended.")
		apiKey, err = promptAPIKey("OpenRouter API key")
		if err != nil {
			return err
		}
	} else {
		provider = "grok"
		authChoice, choiceErr := promptChoice("Choose how to connect to Grok", []string{"OAuth login", "xAI API key"})
		if choiceErr != nil {
			return choiceErr
		}
		if authChoice == "" {
			fmt.Println("› Setup cancelled.")
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
			fmt.Println("! xAI controls pricing and billing; Gmit cannot cap charges. Check prices before use.")
			apiKey, err = promptAPIKey("xAI API key")
			if err != nil {
				return err
			}
		}
	}

	var modelsResponse *config_services.GetAvailableModelsResponse
	err = withSpinner("Loading provider models", func() error {
		var callErr error
		modelsResponse, callErr = c.configService.GetAvailableModelsFor(provider, authMethod, apiKey)
		if callErr != nil {
			return fmt.Errorf("could not load %s models: %w", provider, callErr)
		}
		if modelsResponse.ErrorMessage != "" {
			return fmt.Errorf("could not load %s models: %s", provider, modelsResponse.ErrorMessage)
		}
		return nil
	})
	if err != nil {
		return err
	}
	model, err := pickModel(modelsResponse.Models)
	if err != nil {
		return err
	}
	if model == "" {
		fmt.Println("› Setup cancelled.")
		return nil
	}
	err = withSpinner("Verifying provider connection", func() error {
		return c.configService.ConfigureProvider(provider, authMethod, apiKey)
	})
	if err != nil {
		return fmt.Errorf("could not save %s configuration: %w", provider, err)
	}
	var modelResponse *config_services.SetupModelResponse
	err = withSpinner("Validating and saving selected model", func() error {
		var callErr error
		modelResponse, callErr = c.configService.SetProviderDefaultModel(provider, model)
		if callErr != nil {
			return fmt.Errorf("could not save the selected model: %w", callErr)
		}
		if !modelResponse.Success {
			return fmt.Errorf("could not save the selected model: %s", modelResponse.ErrorMessage)
		}
		return nil
	})
	if err != nil {
		return err
	}
	fmt.Printf("✓ %s is ready with model %s.\n", providerChoice, model)
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
	fmt.Println("! Grok OAuth requires the official Grok CLI.")
	fmt.Printf("Install guide: %s\n", docsURL)
	fmt.Printf("Package: %s — Gmit can install it globally with: npm install --global %s\n", installPackage, installPackage)
	confirmed, err := promptYesNo("Have you reviewed the guide and want Gmit to install it globally? (y/N): ")
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

	req := config_services.SetupAPIKeyRequest{
		APIKey:      args[0],
		ValidateKey: true,
	}

	var resp *config_services.SetupAPIKeyResponse
	err := withSpinner("Validating and saving API key", func() error {
		var callErr error
		resp, callErr = c.configService.SetupAPIKey(req)
		if callErr != nil {
			return fmt.Errorf("failed to set API key: %w", callErr)
		}
		if !resp.Success {
			return fmt.Errorf("failed to set API key: %s", resp.ErrorMessage)
		}
		return nil
	})
	if err != nil {
		return err
	}

	fmt.Printf("✓ %s\n", resp.Message)
	if resp.KeyMasked != "" {
		fmt.Printf("API key  %s\n", resp.KeyMasked)
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
		fmt.Println("! No API key is configured.")
		fmt.Println("› Run `gmit config set-key <your-api-key>` to add one.")
		return nil
	}

	fmt.Printf("API key  %s\n", resp.MaskedAPIKey)
	return nil
}

// handleRemoveAPIKey processes the remove-key command.
func (c *CLI) handleRemoveAPIKey() error {
	fmt.Println("› Removing API key...")

	err := c.configService.RemoveAPIKey()
	if err != nil {
		return fmt.Errorf("failed to remove API key: %w", err)
	}

	fmt.Println("✓ API key removed.")
	return nil
}

// handleSetModel processes the set-model command.
func (c *CLI) handleSetModel(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("model is required. Usage: gmit config set-model <model-id>")
	}

	req := config_services.SetupModelRequest{
		Model:         args[0],
		ValidateModel: true,
	}

	var resp *config_services.SetupModelResponse
	err := withSpinner("Validating and saving default model", func() error {
		var callErr error
		resp, callErr = c.configService.SetupDefaultModel(req)
		if callErr != nil {
			return fmt.Errorf("failed to set model: %w", callErr)
		}
		if !resp.Success {
			return fmt.Errorf("failed to set model: %s", resp.ErrorMessage)
		}
		return nil
	})
	if err != nil {
		return err
	}

	fmt.Printf("✓ %s\n", resp.Message)
	fmt.Println("! The provider controls pricing and billing; Gmit cannot cap charges. Check prices before using paid models.")
	return nil
}

// handleGetModel processes the get-model command.
func (c *CLI) handleGetModel() error {
	resp, err := c.configService.GetDefaultModelInfo()
	if err != nil {
		return fmt.Errorf("failed to get model: %w", err)
	}

	if !resp.Configured {
		fmt.Println("! No default model is configured.")
		fmt.Println("› Run `gmit config set-model <model-id>` to choose one.")
		return nil
	}

	fmt.Printf("Default model  %s\n", resp.Model)
	return nil
}

// handleRemoveModel processes the remove-model command.
func (c *CLI) handleRemoveModel() error {
	fmt.Println("› Removing default model...")

	err := c.configService.RemoveDefaultModel()
	if err != nil {
		return fmt.Errorf("failed to remove model: %w", err)
	}

	fmt.Println("✓ Default model removed.")
	return nil
}

// handleListModels processes the list-models command.
func (c *CLI) handleListModels() error {
	var resp *config_services.GetAvailableModelsResponse
	err := withSpinner("Loading model catalog", func() error {
		var callErr error
		resp, callErr = c.configService.GetAvailableModels()
		if callErr != nil {
			return fmt.Errorf("failed to get models: %w", callErr)
		}
		if resp.ErrorMessage != "" {
			return fmt.Errorf("failed to get models: %s", resp.ErrorMessage)
		}
		return nil
	})
	if err != nil {
		return err
	}

	if len(resp.Models) == 0 {
		fmt.Println("! No models are available right now.")
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
		fmt.Println("› Model selection cancelled.")
		return nil
	}
	return c.handleSetModel([]string{selectedModel})
}

// handleConfigSummary processes the config summary command.
func (c *CLI) handleConfigSummary() error {
	var resp *config_services.ConfigSummaryResponse
	err := withSpinner("Reading configuration and model catalog", func() error {
		var callErr error
		resp, callErr = c.configService.GetConfigSummary()
		if callErr != nil {
			return fmt.Errorf("failed to get config summary: %w", callErr)
		}
		if resp.ErrorMessage != "" {
			return fmt.Errorf("failed to get config summary: %s", resp.ErrorMessage)
		}
		return nil
	})
	if err != nil {
		return err
	}

	fmt.Println("\nConfiguration")
	fmt.Printf("Provider       %s (%s)\n", resp.Provider, resp.AuthMethod)

	// Authentication status
	if resp.AuthMethod == "oauth" && resp.APIKeyConfigured {
		fmt.Printf("Authentication %s\n", resp.APIKeyMasked)
	} else if resp.APIKeyConfigured {
		fmt.Printf("API key        %s\n", resp.APIKeyMasked)
	} else {
		fmt.Println("Authentication Not configured")
	}

	// Default model status
	if resp.DefaultModelSet {
		fmt.Printf("Default model  %s\n", resp.DefaultModel)
	} else {
		fmt.Println("Default model  Not configured")
	}

	// Available models count
	if resp.AvailableModelsCount > 0 {
		fmt.Printf("Models         %d available\n", resp.AvailableModelsCount)
	} else {
		fmt.Println("Models         Unable to load")
	}

	return nil
}

// handleConfigValidate processes the config validate command.
func (c *CLI) handleConfigValidate() error {
	var resp *config_services.ValidateConfigurationResponse
	err := withSpinner("Checking provider and model availability", func() error {
		var callErr error
		resp, callErr = c.configService.ValidateConfiguration()
		if callErr != nil {
			return fmt.Errorf("failed to validate configuration: %w", callErr)
		}
		if resp.ErrorMessage != "" {
			return fmt.Errorf("validation failed: %s", resp.ErrorMessage)
		}
		return nil
	})
	if err != nil {
		return err
	}

	if resp.Valid {
		fmt.Println("✓ Configuration is ready.")
	} else {
		fmt.Println("! Configuration needs attention.")
	}

	if len(resp.Issues) > 0 {
		fmt.Println("\n! Issues")
		for _, issue := range resp.Issues {
			fmt.Printf("  - %s\n", issue)
		}
	}

	if len(resp.Recommendations) > 0 {
		fmt.Println("\n› Suggestions")
		for _, rec := range resp.Recommendations {
			fmt.Printf("  - %s\n", rec)
		}
	}

	return nil
}
