# 🚀 Gommit

AI-powered Git commit message generator using OpenRouter LLM models.

## ✨ Features

- 🤖 **AI-Generated Commits**: Uses OpenRouter's LLM models to generate meaningful commit messages
- 📝 **Conventional Commits**: Follows conventional commit format automatically
- 🔄 **Smart Fallback**: Multiple model fallback system for reliability
- ⚙️ **Configurable**: Set your preferred LLM model
- 🔐 **Secure**: API keys stored securely in system keyring
- 🌍 **Cross-Platform**: Works on Windows, macOS, and Linux

## 📦 Installation

### NPM (Recommended)

```bash
npm install -g gommit
```

### Manual Installation

Download the latest binary from [GitHub Releases](https://github.com/PedroHercules/gommit/releases).

## 🚀 Quick Start

1. **Set your OpenRouter API key:**
   ```bash
   gommit config set-key
   ```

2. **Generate a commit message:**
   ```bash
   # Stage your changes first
   git add .
   
   # Generate and commit
   gommit commit
   ```

## ⚙️ Configuration

### API Key Management
```bash
# Set API key
gommit config set-key

# View current key
gommit config get-key

# Remove key
gommit config remove-key
```

### Default Model Configuration
```bash
# Set preferred model
gommit config set-model openai/gpt-4o-mini

# View current model
gommit config get-model

# Remove model (use automatic selection)
gommit config remove-model
```

## 🤖 Supported Models

Gommit works with any OpenRouter model, including:

- `openai/gpt-4o-mini` (recommended)
- `openai/gpt-3.5-turbo`
- `anthropic/claude-3-haiku`
- `meta-llama/llama-3.1-8b-instruct:free`
- `google/gemma-7b-it:free`

See [OpenRouter Models](https://openrouter.ai/models) for the complete list.

## 📋 Commands

```bash
gommit commit              # Generate and create commit
gommit config set-key      # Set OpenRouter API key
gommit config get-key      # Show current API key
gommit config remove-key   # Remove API key
gommit config set-model    # Set default model
gommit config get-model    # Show current model
gommit config remove-model # Remove default model
gommit --help             # Show help
gommit --version          # Show version
```

## 🔧 How It Works

1. **Analyzes** your staged Git changes
2. **Sends** the diff to OpenRouter's LLM API
3. **Generates** a conventional commit message
4. **Creates** the commit automatically

## 🛡️ Privacy & Security

- API keys are stored securely in your system's keyring
- Only staged changes are analyzed
- No data is stored or logged by gommit
- All communication is encrypted (HTTPS)

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## 📄 License

MIT License - see [LICENSE](LICENSE) file for details.

## 🔗 Links

- [GitHub Repository](https://github.com/PedroHercules/gommit)
- [OpenRouter](https://openrouter.ai/)
- [Conventional Commits](https://www.conventionalcommits.org/)

---

**Made with ❤️ for developers who love clean commit messages**