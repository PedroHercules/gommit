# Gmit — AI Git Commit Message & Pull Request CLI

[![CI](https://github.com/PedroHercules/gommit/actions/workflows/ci-cd.yml/badge.svg)](https://github.com/PedroHercules/gommit/actions/workflows/ci-cd.yml)
[![npm version](https://img.shields.io/npm/v/gmit)](https://www.npmjs.com/package/gmit)
[![License: MIT](https://img.shields.io/github/license/PedroHercules/gommit)](LICENSE)

Gmit is a cross-platform **AI-powered Git CLI** that drafts commit messages from staged changes and pull request descriptions from branch diffs. Connect an available LLM provider, select a model, review the generated text, then commit or push from the terminal. Current providers include **OpenRouter** and **xAI Grok**. Gmit also includes a lightweight Conventional Commits style check.

## Why Gmit

- Generate Conventional Commits-style messages from staged Git changes.
- Draft pull request descriptions from your branch changes.
- Use OpenRouter or xAI Grok with interactive provider and model setup.
- Review generated commit text before choosing to commit, push, or cancel.
- Run on Windows, Linux, macOS, and WSL on supported x64 and ARM64 systems.

Install the latest release from npm:

```sh
npm install --global gmit
```

Then configure a provider and generate a preview:

```sh
gmit config
git add src/file.go
gmit --dry-run
```

See the sections below for requirements, all commands, provider billing details, and known limitations.

Gmit checks npm for a newer stable version whenever you run a command. If one is available, it shows the installed and available versions, release highlights, and a link to the GitHub release notes. This check contacts the npm registry on each command and fetches release notes for a new version from GitHub, caching them locally; it does not send your repository data or provider credentials. The check has a short timeout and is skipped silently when the network is unavailable. Gmit never updates itself automatically. Release notes are generated and published with each automated release from the Conventional Commit history.

## Requirements

- Git installed and available as `git` in your terminal.
- An OpenRouter account/API key or an xAI account/API key.
- For Grok OAuth only: the official `grok` CLI. During setup, Gmit checks for it and, if it is missing, offers to install it globally with npm after asking you. You can also follow [xAI's current installation instructions](https://docs.x.ai/build/overview) yourself.
- For creating GitHub pull requests only: the official [GitHub CLI (`gh`)](https://github.com/cli/cli#installation), installed and authenticated with `gh auth login`. Gmit can still generate a PR description without `gh`.
- A terminal for interactive setup and model selection. Most commands also work non-interactively.

## Supported platforms

The npm package includes binaries for Windows, Linux, and macOS on x64 and ARM64. The release build cross-compiles all six targets. GitHub Actions currently runs the Go test suite on Ubuntu; Windows and macOS runtime behavior is not covered by native CI tests. WSL uses the Linux binary for your WSL architecture.

Install Gmit globally with npm:

```sh
npm install --global gmit
gmit help
```

### Test changes from a local checkout

From the repository directory, use the `dev` Make target to run the source in this checkout. It does not replace the globally installed `gmit` command:

```sh
make dev ARGS="help"
make dev ARGS="commit --dry-run"
make dev ARGS="config list-models"
```

The Go toolchain and project dependencies must be available. The command runs Gmit with your normal user configuration, so provider requests can still incur charges.

## Quick start

Run the interactive setup. Select OpenRouter or Grok, configure authentication, then search for and select a model.

```sh
gmit config
gmit config validate
```

Run `gmit config validate` after setup to check that Gmit can read your credentials and fetch the provider's model catalog before you generate a message.

After setup, stage the changes you want Gmit to analyze. Running `gmit` by itself is an alias for `gmit commit`:

```sh
git add src/file.go
gmit --dry-run
```

`--dry-run` prevents a Git commit, but Gmit still sends a model request. Provider charges may apply. By default, after generation a single-choice selector displays the generated message above **Commit and push**, **Commit only**, and **Cancel**. Use Page Up/Page Down to scroll long messages, the arrow keys to select one action, and Enter to confirm. A push sends the current branch to `origin` and sets its upstream. To create a commit automatically without the selector or push, use `gmit commit --commit`.

## Commands

### `gmit commit`

Generates a commit message from the **staged diff only**. Unstaged changes are not included. Stage the exact files you intend to include before running the command.

```sh
git add src/auth.go
gmit                                 # Same as gmit commit
gmit commit                         # Generate, then choose push, commit, or cancel
gmit commit --dry-run               # Generate and show; do not commit
gmit commit --commit                # Generate and create one commit
gmit commit --model provider/model-id # Use a specific configured-provider model
```

Options:

- `--model <id>` or `-m <id>` uses that model instead of the configured default.
- `--commit` creates a commit with the generated message and exits without pushing. It requires staged changes.
- `--dry-run` skips the Git commit. It still contacts the provider and may incur charges.
- The default post-generation selector is single-choice. **Commit and push** runs one commit, then pushes the current branch to `origin` with upstream tracking. If the push fails, the commit remains created and Gmit reports the push error.
- `--force` is unsupported. Stage the intended changes and review the output instead.

### `gmit pr`

Generates a Markdown pull request description from the code diff between your current `HEAD` and the merge base with a remote-tracking branch. By default it fetches `origin` and compares against `origin/main`. In an interactive terminal, Gmit opens a full-screen preview that you can scroll through with Up/Down, Page Up/Page Down, or jump with Home/End before choosing to create a GitHub PR or keep the preview only. Press Enter to continue or Esc to cancel. Use `--create` to request creation directly.

```sh
gmit pr
gmit pr --base-branch develop
gmit pr --base-branch main --model provider/model-id
gmit pr --create --reviewer alice --reviewer my-team --draft
gmit pr --create --title "Add account recovery" --reviewer alice
```

Options:

- `--base-branch <name>` selects the target branch on `origin` (default: `main`).
- `--model <id>` or `-m <id>` uses a model from the configured provider.
- `--create` creates the GitHub PR after generating the preview. In a terminal, Gmit asks for reviewers and whether it should be a draft unless you provide those options.
- `--title <text>` overrides the title extracted from the generated Markdown heading. If the model did not return a heading, Gmit asks for a title in an interactive terminal; otherwise provide `--title`.
- `--reviewer <login>` requests a review from a GitHub username or team. Repeat the option or separate multiple handles with commas.
- `--draft` creates the PR as a draft. Without it, non-interactive creation creates a PR ready for review.

Creating a PR requires the official GitHub CLI (`gh`) installed and authenticated. If it is missing, Gmit prints the [official installation guide](https://github.com/cli/cli#installation); installation methods vary by operating system. If `gh` is installed but not authenticated, Gmit offers to start `gh auth login` in an interactive terminal. You can also run `gh auth login` yourself.

When creating interactively, Gmit loads repository collaborators into a searchable, multi-select list. Use the arrow keys to navigate, type to filter, Space or Enter to toggle a reviewer, then choose **Done**. The **Add username or team manually** option accepts GitHub logins and team slugs separated by commas. Loading collaborators requires GitHub API access for the repository; if GitHub denies the request or returns no users, Gmit explains that and keeps manual entry available. The [GitHub collaborators endpoint](https://docs.github.com/en/rest/collaborators/collaborators) requires sufficient repository access, and classic tokens may need `repo` and `read:org` scopes.

If the current branch is not published at the same commit on `origin`, Gmit asks before running `git push --set-upstream origin <branch>`. In non-interactive use, publish the branch yourself and rerun the command. Gmit does not force-push. If `origin` or the named base branch is unavailable, it stops with an error instead of generating a description from incomplete data. It fetches from `origin`, but does not checkout branches or pull into your current branch.

### `gmit validate <message>`

Checks that a message is non-empty and reports whether it resembles Conventional Commits. It also gives heuristic warnings about summary length and imperative wording. This is a lightweight advisory check, not a complete validator or guarantee that a repository's commit policy will accept the message.

```sh
gmit validate "feat(auth): add token refresh"
gmit validate "fixed the login bug"
```

The command exits successfully for any non-empty message; style issues are reported as warnings. An empty message is an error.

### `gmit config`

Run `gmit config` in an interactive terminal to select a provider, set up authentication, and choose a default model.

```sh
gmit config
gmit config list-models
gmit config set-model provider/model-id
gmit config summary
gmit config validate
gmit config get-model
```

`config list-models` fetches the active provider's current catalog. In a terminal it opens a searchable picker: type to filter, use arrow keys to move, press Enter to save the selected model, or Esc to cancel. In non-interactive output it prints the catalog. A model listed by a provider can still be unavailable, rate-limited, or incompatible with a request by the time it is used.

`config set-key <key>` saves a key for the active provider. `get-key` displays a masked key, and `remove-key` removes it. Keys are stored using the operating system keyring. `config validate` checks the current provider configuration; it is different from `gmit validate <message>`.

For Grok OAuth, Gmit uses the official xAI Grok CLI and invokes its login flow. The CLI must remain installed and authenticated for this mode. Alternatively, choose xAI API-key authentication. See xAI's [Grok CLI guide](https://docs.x.ai/build/overview) and [CLI reference](https://docs.x.ai/build/cli/reference) for current details.

### Other commands

```sh
gmit help
gmit config help
gmit version
gmit update
```

`gmit update` installs the latest release globally with npm. It requires npm and the same permissions used for the original global installation. Release highlights link to the automatically generated [GitHub release notes](https://github.com/PedroHercules/gommit/releases).

Unknown commands and unsupported command options return an error. The old `--force` commit option is not supported.

## Models, pricing, and billing

Gmit sends prompts and diffs to the provider and model you select. OpenRouter and xAI control model availability, pricing, and billing; Gmit does not set prices, bill your account, or guarantee free use or cap charges. **You are responsible for charges incurred through your provider account.** A free model may have quotas, rate limits, change availability, or stop being free. Model catalogs and prices can change, so check current information before use.

- OpenRouter: [model pricing](https://openrouter.ai/pricing/), [free models](https://openrouter.ai/collections/free-models), and [billing support](https://openrouter.ai/support).
- xAI: [API pricing](https://docs.x.ai/developers/pricing) and [billing documentation](https://docs.x.ai/developers/faq/billing).

`--dry-run` only skips creating a local Git commit. It does **not** skip model generation or provider billing. For lower cost, choose a model marked free when available and verify its current price in the provider's own catalog.

## Availability and known limitations

Gmit depends on provider APIs, model catalogs, authentication, network access, and response formats. Any of these can change or be temporarily unavailable; requests can fail because of provider outages, quotas, rate limits, unsupported models, or response limits. Gmit cannot guarantee that every listed model will work for every task.

PR generation requires a successful fetch and `origin/<base-branch>`. It compares code changes from the merge base to `HEAD`; it does not include unstaged working-tree edits. Commit generation analyzes staged changes only. The commit-message validator uses simple heuristics and may report false positives or miss repository-specific rules.

## Contributing and support

Contributions are welcome, but opening an issue or pull request does not guarantee that a change will be accepted. Read the [contribution guidelines](CONTRIBUTING.md) before proposing a change. Report bugs and suggest features through the [GitHub issue tracker](https://github.com/PedroHercules/gommit/issues). To report a security vulnerability, follow [SECURITY.md](SECURITY.md) and do not post exploit details in a public issue.

Gmit is released under the MIT License.
