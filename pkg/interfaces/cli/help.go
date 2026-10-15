package cli

import "fmt"

var version = "dev"

func (c *CLI) showHelp() error {
	fmt.Println(`
GMIT  /  AI-assisted Git workflow

USAGE
    gmit [command] [options]

COMMANDS
    commit              Generate a message from staged changes (default)
    pr                  Generate a pull request description
    validate <message>  Check a commit message
    config              Configure a provider, credentials, and model
    version             Show version
    update              Update the npm installation to the latest release
    help                Show help

COMMIT OPTIONS
    --model, -m <id>    Use a model from the configured provider
    --commit            Create one Git commit without showing the selector
    --dry-run           Generate and display a message without committing

PR OPTIONS
    --base-branch <name>  Compare with origin/<name> (default: main)
    --model, -m <id>      Use a model from the configured provider
    --create              Create the PR after generating its preview
    --title <text>        Override the title extracted from the generated preview
    --reviewer <login>    Request a review (repeatable; accepts comma-separated logins/teams)
    --draft               Create the pull request as a draft

CONFIG COMMANDS
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

EXAMPLES
    gmit
    gmit config
    gmit config list-models
    gmit commit --dry-run
    gmit commit --model provider/model-id
    gmit commit --commit
    gmit pr --base-branch main --model provider/model-id
    gmit pr --create --reviewer alice --reviewer my-team --draft
    gmit validate "feat: add user authentication"
    gmit update

QUICK NOTES
    › Commit generation uses staged changes only. Stage files with git add first.
    › Review the full preview, then choose Commit and push, Commit only, or Cancel.
    › --dry-run still sends a provider request and may incur charges.
    ! OpenRouter and xAI control pricing and billing. Gmit cannot cap charges.
    › Prefer free models and verify current prices with your provider.
    › Review the full PR preview with ↑/↓ or PgUp/PgDn; Enter continues and Esc cancels.
    › PR generation fetches origin/<base>; it does not switch branches or pull changes.
    › Creating a PR requires GitHub CLI (gh) installed and authenticated with gh auth login.
    › Search collaborators; use Space/Enter to select, or enter usernames/team slugs manually.
    › Gmit asks before pushing an unpublished branch to origin.
    › Gmit checks npm for updates on every launch and shows release notes when available.
    › ` + "`gmit update`" + ` installs the latest release using npm; it never updates automatically.
    › Run ` + "`gmit config help`" + ` for configuration details.

Repository: https://github.com/PedroHercules/gommit`)
	return nil
}

func (c *CLI) showConfigHelp() error {
	fmt.Println(`
GMIT  /  CONFIGURATION

USAGE
    gmit config [COMMAND]

Run ` + "`gmit config`" + ` in a terminal to select OpenRouter or Grok, authenticate,
and choose a model. For Grok OAuth, Gmit uses the official Grok CLI; if it is
missing, Gmit offers to install it globally after asking you.

COMMANDS
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

API keys are stored in your system keyring. The default model is used
when --model is omitted. Provider catalogs and authentication can change or be
temporarily unavailable. Providers control all pricing and billing; Gmit cannot
limit charges. Review current provider prices and prefer free models when available.`)
	return nil
}

func (c *CLI) showVersion() error {
	fmt.Printf("Gmit v%s\nAI-assisted Git commit message and PR description generator\nRepository: https://github.com/PedroHercules/gommit\nLicense: MIT\n", version)
	return nil
}
