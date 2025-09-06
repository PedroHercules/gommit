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

### Via NPM (Recommended)

```bash
npm install -g gmit
```

### Manual Installation

```bash
# Clone the repository
git clone https://github.com/PedroHercules/gommit.git
cd gommit

# Install dependencies
go mod tidy

# Build the project
go build -o gmit main.go
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

## Architecture

This project follows **Clean Architecture** principles, providing a clean, testable, and maintainable structure. It's ideal for learning Go and software architecture patterns.

### 📁 Project Structure

```
gmit/
├── main.go                      # Application entry point
├── pkg/
│   ├── domain/                  # Domain Layer (business rules)
│   │   ├── entities/            # Domain entities
│   │   │   ├── commit.go        # Commit entity
│   │   │   ├── config.go        # Config entity
│   │   │   └── git_diff.go      # GitDiff entity
│   │   ├── repositories/        # Repository interfaces
│   │   │   ├── config_repository.go
│   │   │   ├── git_repository.go
│   │   │   └── llm_repository.go
│   │   └── usecases/            # Use cases (business logic)
│   │       ├── commit_usecase.go
│   │       ├── config_usecase.go
│   │       ├── generate_commit_usecase.go
│   │       └── pull_request_usecase.go
│   ├── infrastructure/          # Infrastructure Layer
│   │   ├── config/              # Configuration implementation
│   │   │   ├── file_config_repository.go
│   │   │   └── keyring_service.go
│   │   ├── git/                 # Git implementation
│   │   │   └── git_repository.go
│   │   └── llm/                 # LLM implementation
│   │       └── openrouter_repository.go
│   ├── application/             # Application Layer
│   │   └── services/            # Application services
│   │       ├── commit/          # Commit services
│   │       ├── config/          # Configuration services
│   │       └── pull-request/    # Pull request services
│   └── interfaces/              # Interface Layer
│       └── cli/                 # Command line interface
│           ├── commands.go      # Command processing
│           └── help.go          # Help system
├── bin/                         # Compiled binaries
├── install.js                   # Installation script
├── package.json                 # Node.js configuration
└── README.md                    # This file
```

## Dependencies

The project uses the following main dependencies:

- **github.com/99designs/keyring**: Secure credential storage
- **github.com/joho/godotenv**: Environment variable loading
- **Go 1.21+**: Minimum Go version

```bash
# Install dependencies
go mod tidy

# Verify dependencies
go mod verify

# Update dependencies
go get -u ./...
```

## Supported AI Models

Gmit supports various AI models through OpenRouter:

- **Claude 3 Sonnet** (recommended for quality)
- **Claude 3 Haiku** (faster, cost-effective)
- **GPT-4** (OpenAI)
- **GPT-3.5 Turbo** (faster alternative)
- And many more available through OpenRouter

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

1. Fork the project
2. Create a feature branch (`git checkout -b feature/new-feature`)
3. Commit your changes (`git commit -am 'feat: add new feature'`)
4. Push to the branch (`git push origin feature/new-feature`)
5. Open a Pull Request

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
