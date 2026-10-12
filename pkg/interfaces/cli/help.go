package cli

import "fmt"

var version = "dev"

func (c *CLI) showHelp() error {
	fmt.Println(`
Gmit - AI-assisted Git commit messages and PR descriptions

USAGE:
    gmit <command> [options]

COMMANDS:
    commit              Generate a message from staged changes
    pr                  Generate a pull request description
    validate <message>  Check a commit message
    config              Configure a provider, credentials, and model
    version             Show version
    help                Show help

COMMIT OPTIONS:
    --model, -m <id>    Use a model from the configured provider
    --commit            Create one Git commit without showing the selector
    --dry-run           Generate and display a message without committing

PR OPTIONS:
    --base-branch <name>  Compare with origin/<name> (default: main)
    --model, -m <id>      Use a model from the configured provider

CONFIG COMMANDS:
    config                          Open interactive provider setup
    config set-key <key>            Save a key for the active provider
    config get-key                  Show the saved key (masked)
    config remove-key               Remove the saved key
    config set-model <model-id>     Set the default model
    config get-model                Show the default model
    config remove-model             Remove the default model
    config list-models              Search/select or print available models
    config summary                  Show configuration summary
    config validate                 Check provider configuration
    config help                     Show configuration help

EXAMPLES:
    gmit config
    gmit config list-models
    gmit commit --dry-run
    gmit commit --model provider/model-id
    gmit commit --commit
    gmit pr --base-branch main --model provider/model-id
    gmit validate "feat: add user authentication"

NOTES:
    • Commit generation uses staged changes only. Stage intended files with git add first.
    • After generation, choose Commit and push, Commit only, or Cancel with the arrow keys and Enter.
    • --dry-run still sends a request to your provider and may incur charges.
    • OpenRouter and xAI control model availability, prices, and billing. Gmit cannot cap spending.
    • Prefer models marked free and check provider pricing before use.
    • PR generation fetches origin/<base>; it does not switch branches or pull into your branch.
    • Run ` + "`gmit config help`" + ` for configuration details.

Repository: https://github.com/PedroHercules/gommit`)
	return nil
}

func (c *CLI) showConfigHelp() error {
	fmt.Println(`
Gmit configuration

USAGE:
    gmit config [COMMAND]

Run ` + "`gmit config`" + ` in a terminal to select OpenRouter or Grok, authenticate,
and choose a model. For Grok OAuth, Gmit uses the official Grok CLI; if it is
missing, Gmit offers to install it globally after asking you.

COMMANDS:
    set-key <key>       Save a key for the active provider
    get-key             Show the saved key (masked)
    remove-key          Remove the saved key
    set-model <id>      Set the default model
    get-model           Show the default model
    remove-model        Remove the default model
    list-models         Search/select models in a terminal; print catalog otherwise
    summary             Show provider and model settings
    validate            Check provider configuration

EXAMPLES:
    gmit config
    gmit config set-key <api-key>
    gmit config list-models
    gmit config set-model provider/model-id
    gmit config summary
    gmit config validate

API keys are stored in the operating system keyring. The default model is used
when --model is omitted. Provider catalogs and authentication can change or be
temporarily unavailable. Providers control all pricing and billing; Gmit cannot
limit charges. Review current provider prices and prefer free models when available.`)
	return nil
}

func (c *CLI) showVersion() error {
	fmt.Printf("Gmit v%s\nAI-assisted Git commit message and PR description generator\nRepository: https://github.com/PedroHercules/gommit\nLicense: MIT\n", version)
	return nil
}
