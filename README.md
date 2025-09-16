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

## Configuration

```bash
# Configure your OpenRouter API key
gmit config set-key sk-or-v1-your-key-here

# (Optional) Set a default model
gmit config set-model claude-3-sonnet

# Check configuration
gmit config summary
```

## 💰 API Costs and Usage

**Important**: Each user is responsible for their own OpenRouter API costs. Please monitor your usage carefully:

- **Free Models Available**: Gmit is configured to use free models by default (like `meta-llama/llama-3.2-3b-instruct:free`)
- **Cost Monitoring**: Check your usage at [OpenRouter Dashboard](https://openrouter.ai/activity)
- **Rate Limits**: Free models have usage limits - monitor to avoid interruptions
- **Paid Models**: If you choose paid models, costs vary from $0.001 to $0.10+ per request
- **Usage Tips**:
  - Use `--dry-run` to preview without committing
  - Prefer free models for regular use
  - Monitor your monthly usage
  - Set up billing alerts in OpenRouter

> ⚠️ **Disclaimer**: The developers of Gmit are not responsible for any API costs incurred. Users should monitor their own usage and set appropriate limits.

## Usage

### Commit Generation

```bash
# Generate and commit automatically
gmit commit --commit

# Only generate the message (dry run)
gmit commit --dry-run

# Use a specific model
gmit commit --model claude-3-sonnet

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

# Use specific AI model for PR generation
gmit pr --model claude-3-haiku --base-branch develop
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

Gmit supports the following AI models by default:

- **deepseek/deepseek-chat-v3.1:free** (default) - 32k context, stable and accurate
- **nvidia/llama-3.1-nemotron-70b-instruct** (alternative) - High-performance model optimized for code understanding

To use other models available through OpenRouter, set them as default using:
```bash
gmit config set-model <model-name>
```

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
