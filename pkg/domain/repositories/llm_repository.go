// Package repositories defines the interfaces for data access.
package repositories

import (
	"github.com/PedroHercules/gommit/pkg/domain/entities"
)

// LLMModel represents information about an available LLM model.
type LLMModel struct {
	ID          string  // Model identifier (e.g., "openai/gpt-4o-mini")
	Name        string  // Human-readable name
	Provider    string  // Provider name (e.g., "OpenAI", "Anthropic")
	ContextSize int     // Maximum context size in tokens
	CostPer1K   float64 // Cost per 1K tokens (for selection logic)
	Available   bool    // Whether the model is currently available
}

// LLMResponse represents the response from an LLM service.
type LLMResponse struct {
	Message     string // The generated commit message
	Model       string // The model that was used
	TokensUsed  int    // Number of tokens consumed
	ContextSize int    // Context size used
	Success     bool   // Whether the request was successful
	Error       string // Error message if any
}

// LLMRepository defines the interface for LLM operations.
// This interface abstracts the LLM service and allows for different
// implementations (OpenRouter, direct API calls, etc.).
type LLMRepository interface {
	// GenerateCommitMessage generates a commit message based on the git diff.
	// It uses the configured model or selects the best available one.
	GenerateCommitMessage(diff *entities.GitDiff, model string) (*LLMResponse, error)

	// GeneratePRDescription generates a pull request description based on the git diff.
	// It uses the configured model or selects the best available one.
	GeneratePRDescription(diff *entities.GitDiff, model string) (*LLMResponse, error)

	// GetAvailableModels returns a list of available LLM models.
	// This can be used for model selection and validation.
	GetAvailableModels() ([]LLMModel, error)

	// ValidateModel checks if a model ID is valid and available.
	ValidateModel(modelID string) (bool, error)

	// GetBestModel returns the best available model based on criteria
	// such as cost, performance, and availability.
	GetBestModel() (*LLMModel, error)

	// TestConnection tests the connection to the LLM service.
	// Returns true if the service is reachable and API key is valid.
	TestConnection(apiKey string) (bool, error)

	// SetAPIKey configures the API key for the LLM service.
	SetAPIKey(apiKey string)

	// GetModelInfo returns detailed information about a specific model.
	GetModelInfo(modelID string) (*LLMModel, error)
}
