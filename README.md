# Gmit - AI-Powered Git Commit Message & Pull Request Generator

Gmit is a command-line tool that uses artificial intelligence to automatically generate Git commit messages and Pull Request descriptions, following best practices and the Conventional Commits format. The tool also provides functionality for commit message validation and flexible AI model configuration.

## Key Features

- **Automatic Commit Generation**: AI analyzes your changes and generates messages following Conventional Commits
- **Pull Request Descriptions**: Creates detailed descriptions for your PRs automatically with smart branch comparison
- **Message Validation**: Verifies that your messages follow best practices
- **Flexible Configuration**: Support for multiple AI models and custom configurations
- **Secure Storage**: API keys stored securely in the system keyring
- **Intuitive Interface**: Simple and easy-to-use CLI with contextual help
- **Cross-Platform**: Works on Windows, macOS, and Linux

## 🚀 Installation

```bash
npm install -g gmit
```

## Maintainer releases

Pull requests to `main` run the Go test suite. A push to `main` with Conventional Commit changes automatically calculates the next SemVer version, builds binaries for Linux, macOS, and Windows on amd64 and arm64, updates `package.json`, publishes the package to npm, and creates a GitHub release. `feat` creates a minor release, `fix` and `perf` create patch releases, and `BREAKING CHANGE` creates a major release; documentation and chore commits do not publish a new version.

Before the first automated release, configure npm authentication using one of these options:

- **Trusted Publishing (recommended):** in the `gmit` package settings on npmjs.com, add GitHub Actions with organization/user `PedroHercules`, repository `gommit`, workflow filename `ci-cd.yml`, and no GitHub environment. The workflow grants `id-token: write` and uses OIDC.
- **Token fallback:** create an npm publishing token and save it as the `NPM_TOKEN` Actions secret in the GitHub repository settings. The workflow passes this secret to semantic-release as a fallback if OIDC authentication is unavailable.

The `404 OIDC token exchange error - package not found` message means npm did not match the OIDC request to a trusted publisher for this package. Check that the publisher is configured on the `gmit` package and that the repository and workflow filename match exactly. On its first run, the workflow verifies that the repository version matches npm's `latest` and initializes the matching Git release tag as a baseline.

## Configuration

```bash
# Configure your OpenRouter API key
gmit config set-key sk-or-v1-your-key-here

# (Optional) List models, then set any model ID from OpenRouter
gmit config list-models
gmit config set-model <model-id>

# Check configuration
gmit config summary
```

## 💰 API Costs and Usage

**Important**: Model requests are billed by OpenRouter according to the selected model and its current pricing. Gommit does not control provider pricing, charge your account, or limit your usage. You are responsible for all charges and should check pricing before using a model.

- **Recommended**: Choose free models when configuring Gommit; these are commonly marked with `:free` and are labeled in `gmit config list-models` when OpenRouter reports zero pricing.
- **Cost Monitoring**: Check your usage at [OpenRouter Dashboard](https://openrouter.ai/activity)
- **Rate Limits**: Free models have usage limits - monitor to avoid interruptions
- **Paid Models**: Costs vary by model and can change. Review OpenRouter's current pricing before using a paid model.
- **Usage Tips**:
  - Use `--dry-run` to preview without committing
  - Prefer free models for regular use
  - Monitor your monthly usage
  - Set up billing alerts in OpenRouter

> ⚠️ **Disclaimer**: Gommit does not control or cap OpenRouter costs. Users are responsible for checking prices, monitoring usage, and setting any available limits in their OpenRouter account.

## Usage

### Commit Generation

```bash
# Generate and commit automatically
gmit commit --commit

# Only generate the message (dry run)
gmit commit --dry-run

# Use a model from the OpenRouter catalog
gmit commit --model <model-id>

# Interactive mode with confirmation
gmit commit
```

### Pull Request Generation

The PR functionality is one of Gmit's most powerful features, providing intelligent analysis of your branch changes:

```bash
# Generate PR description for current branch
gmit pr

# Generate PR comparing with specific base branch
gmit pr --base-branch develop

# Generate PR with custom target branch
gmit pr --base-branch main

# Use a model from the OpenRouter catalog for PR generation
gmit pr --model <model-id> --base-branch develop
```

#### How PR Generation Works

1. **Branch Analysis**: Gmit automatically detects your current branch and compares it with the target branch (default: main)
2. **Commit History**: Analyzes all commits between your branch and the base branch
3. **Code Changes**: Reviews the actual code differences (git diff)
4. **Smart Summarization**: Uses AI to create a comprehensive PR description including:
   - Clear title following conventional commit format
   - Detailed description of changes
   - List of modified files and their purposes
   - Breaking changes (if any)
   - Testing recommendations

#### PR Description Format

Generated PR descriptions follow this structure:

```markdown
## Description
[AI-generated summary of changes]

## Changes Made
- [List of key changes]
- [Feature additions]
- [Bug fixes]

## Files Modified
- `file1.go` - [Purpose of changes]
- `file2.js` - [What was modified]

## Breaking Changes
[If applicable]

## Testing
[Suggested testing approach]
```

### Other Commands

```bash
# Validate a commit message
gmit validate "feat: add new feature"

# Show help
gmit help

# Show version
gmit version

# Configuration management
gmit config help
```



## Supported AI Models

Gmit fetches the current model catalog from OpenRouter. Use `gmit config list-models` to see available model IDs and published pricing, then choose any listed ID:

```bash
gmit config set-model <model-id>
```

In an interactive terminal, `gmit config list-models` opens a searchable list. Type to filter by model ID, name, or provider, use the arrow keys to move through results, then press Enter to save the selection as the default model. Without an interactive terminal, the command prints the full catalog.

Prefer free models. OpenRouter controls pricing and billing, and Gommit cannot limit charges. The catalog's prices may change; check OpenRouter before selecting a paid model.

## Examples

### Commit Message Examples

```bash
# For a new feature
gmit commit
# Output: "feat: add user authentication system"

# For a bug fix
gmit commit
# Output: "fix: resolve memory leak in data processing"

# For documentation
gmit commit
# Output: "docs: update API documentation with new endpoints"
```

### Pull Request Examples

```bash
# Feature branch to main
gmit pr
# Generates comprehensive PR description with:
# - Feature overview
# - Implementation details
# - Files changed
# - Testing suggestions

# Hotfix branch to develop
gmit pr --base-branch develop
# Focuses on bug fixes and urgent changes
```

## 🤝 Contributing

See [DEVELOPMENT.md](DEVELOPMENT.md) for development setup and contribution guidelines.

## 📄 License

MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- [OpenRouter](https://openrouter.ai/) for the AI API
- [Conventional Commits](https://www.conventionalcommits.org/) for the message format standard
- [99designs/keyring](https://github.com/99designs/keyring) for secure storage
- [joho/godotenv](https://github.com/joho/godotenv) for environment management
- Go community for best practices and architecture patterns

## Support

If you encounter any issues or have questions:

1. Check the [Issues](https://github.com/PedroHercules/gommit/issues) page
2. Create a new issue with detailed information
3. Use `gmit help` for command-specific help

---

**Made with ❤️ by Pedro Hercules**
