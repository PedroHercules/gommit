# Development Guide

## Local Development Setup

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

## Building

```bash
# Build for current platform
go build -o gmit main.go

# Build for multiple platforms
# Windows
GOOS=windows GOARCH=amd64 go build -o bin/gmit-windows-amd64.exe main.go

# macOS
GOOS=darwin GOARCH=amd64 go build -o bin/gmit-darwin-amd64 main.go
GOOS=darwin GOARCH=arm64 go build -o bin/gmit-darwin-arm64 main.go

# Linux
GOOS=linux GOARCH=amd64 go build -o bin/gmit-linux-amd64 main.go
```

## Testing

```bash
# Run tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests with verbose output
go test -v ./...
```

## 🤝 Contributing

1. Fork the project
2. Create a feature branch (`git checkout -b feature/new-feature`)
3. Commit your changes (`git commit -am 'feat: add new feature'`)
4. Push to the branch (`git push origin feature/new-feature`)
5. Open a Pull Request

## Code Style

- Follow Go conventions and best practices
- Use `gofmt` to format code
- Run `go vet` to check for issues
- Follow Clean Architecture principles
- Write meaningful commit messages using Conventional Commits

## Debugging

```bash
# Enable debug mode
export DEBUG=true
gmit commit --dry-run

# Check configuration
gmit config summary

# Test API connection
gmit config test-connection
```