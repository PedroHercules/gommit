// Package llm provides infrastructure implementations for LLM operations.
// This package implements the LLMRepository interface using OpenRouter API.
package llm

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/PedroHercules/gommit/pkg/domain/entities"
	"github.com/PedroHercules/gommit/pkg/domain/repositories"
)

// OpenRouterRepository implements the LLMRepository interface using OpenRouter API.
// OpenRouter provides access to multiple LLM providers through a unified API.
type OpenRouterRepository struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
	models     []repositories.LLMModel // Cached list of available models
}

// NewOpenRouterRepository creates a new OpenRouter LLM repository.
func NewOpenRouterRepository(apiKey string) *OpenRouterRepository {
	return &OpenRouterRepository{
		apiKey:  apiKey,
		baseURL: "https://openrouter.ai/api/v1",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		models: []repositories.LLMModel{},
	}
}

// openRouterRequest represents the request structure for OpenRouter API.
type openRouterRequest struct {
	Model    string                   `json:"model"`
	Messages []openRouterMessage     `json:"messages"`
	Stream   bool                     `json:"stream"`
	MaxTokens int                     `json:"max_tokens,omitempty"`
}

// openRouterMessage represents a message in the conversation.
type openRouterMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// openRouterResponse represents the response from OpenRouter API.
type openRouterResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		TotalTokens int `json:"total_tokens"`
	} `json:"usage"`
	Model string `json:"model"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// openRouterModelsResponse represents the response from the models endpoint.
type openRouterModelsResponse struct {
	Data []struct {
		ID      string `json:"id"`
		Name    string `json:"name,omitempty"`
		Context int    `json:"context_length,omitempty"`
		Pricing struct {
			Prompt     string `json:"prompt,omitempty"`
			Completion string `json:"completion,omitempty"`
		} `json:"pricing,omitempty"`
	} `json:"data"`
}

// GenerateCommitMessage generates a commit message based on the git diff.
func (r *OpenRouterRepository) GenerateCommitMessage(diff *entities.GitDiff, model string) (*repositories.LLMResponse, error) {
	if r.apiKey == "" {
		return nil, errors.New("API key not configured")
	}

	if !diff.HasChanges() {
		return nil, errors.New("no changes to generate commit message for")
	}

	// Determine which model to use
	modelToUse := model
	if modelToUse == "" {
		// Get the best available model
		bestModel, err := r.GetBestModel()
		if err != nil {
			return nil, fmt.Errorf("failed to get best model: %w", err)
		}
		modelToUse = bestModel.ID
	}

	// Create the prompt for commit message generation
	prompt := r.createCommitPrompt(diff)

	// Prepare the request
	request := openRouterRequest{
		Model: modelToUse,
		Messages: []openRouterMessage{
			{
				Role:    "system",
				Content: "You are an expert developer who writes clear, concise commit messages following conventional commits format. Generate a commit message based on the provided git diff.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Stream:    false,
		MaxTokens: 150, // Commit messages should be concise
	}

	// Make the API call
	response, err := r.makeAPICall(request)
	if err != nil {
		return &repositories.LLMResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	if response.Error != nil {
		return &repositories.LLMResponse{
			Success: false,
			Error:   response.Error.Message,
		}, nil
	}

	if len(response.Choices) == 0 {
		return &repositories.LLMResponse{
			Success: false,
			Error:   "no response from LLM",
		}, nil
	}

	// Extract and clean the commit message
	commitMessage := strings.TrimSpace(response.Choices[0].Message.Content)
	commitMessage = r.cleanCommitMessage(commitMessage)

	return &repositories.LLMResponse{
		Message:     commitMessage,
		Model:       response.Model,
		TokensUsed:  response.Usage.TotalTokens,
		ContextSize: 0, // OpenRouter doesn't provide this in the response
		Success:     true,
	}, nil
}

// GetAvailableModels returns a list of available LLM models.
func (r *OpenRouterRepository) GetAvailableModels() ([]repositories.LLMModel, error) {
	// Return cached models if available
	if len(r.models) > 0 {
		return r.models, nil
	}

	// Fetch models from API
	req, err := http.NewRequest("GET", r.baseURL+"/models", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+r.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch models: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var modelsResp openRouterModelsResponse
	if err := json.Unmarshal(body, &modelsResp); err != nil {
		return nil, fmt.Errorf("failed to parse models response: %w", err)
	}

	// Convert to our model format
	models := make([]repositories.LLMModel, 0, len(modelsResp.Data))
	for _, model := range modelsResp.Data {
		// Extract provider from model ID
		provider := "Unknown"
		if parts := strings.Split(model.ID, "/"); len(parts) > 0 {
			provider = strings.Title(parts[0])
		}

		models = append(models, repositories.LLMModel{
			ID:          model.ID,
			Name:        model.Name,
			Provider:    provider,
			ContextSize: model.Context,
			Available:   true,
		})
	}

	// Cache the models
	r.models = models
	return models, nil
}

// ValidateModel checks if a model ID is valid and available.
func (r *OpenRouterRepository) ValidateModel(modelID string) (bool, error) {
	models, err := r.GetAvailableModels()
	if err != nil {
		return false, err
	}

	for _, model := range models {
		if model.ID == modelID {
			return model.Available, nil
		}
	}

	return false, nil
}

// GetBestModel returns the best available model based on criteria.
func (r *OpenRouterRepository) GetBestModel() (*repositories.LLMModel, error) {
	models, err := r.GetAvailableModels()
	if err != nil {
		return nil, err
	}

	if len(models) == 0 {
		return nil, errors.New("no models available")
	}

	// Prefer specific models known to work well for commit messages
	preferredModels := []string{
		"openai/gpt-4o-mini",
		"openai/gpt-3.5-turbo",
		"anthropic/claude-3-haiku",
		"meta-llama/llama-3.1-8b-instruct",
	}

	for _, preferred := range preferredModels {
		for _, model := range models {
			if model.ID == preferred && model.Available {
				return &model, nil
			}
		}
	}

	// Fallback to the first available model
	for _, model := range models {
		if model.Available {
			return &model, nil
		}
	}

	return nil, errors.New("no available models found")
}

// TestConnection tests the connection to the LLM service.
func (r *OpenRouterRepository) TestConnection(apiKey string) (bool, error) {
	// Temporarily set the API key for testing
	originalKey := r.apiKey
	r.apiKey = apiKey
	defer func() { r.apiKey = originalKey }()

	// Try to fetch models as a connection test
	_, err := r.GetAvailableModels()
	return err == nil, err
}

// SetAPIKey configures the API key for the LLM service.
func (r *OpenRouterRepository) SetAPIKey(apiKey string) {
	r.apiKey = apiKey
	// Clear cached models when API key changes
	r.models = []repositories.LLMModel{}
}

// GetModelInfo returns detailed information about a specific model.
func (r *OpenRouterRepository) GetModelInfo(modelID string) (*repositories.LLMModel, error) {
	models, err := r.GetAvailableModels()
	if err != nil {
		return nil, err
	}

	for _, model := range models {
		if model.ID == modelID {
			return &model, nil
		}
	}

	return nil, fmt.Errorf("model not found: %s", modelID)
}

// createCommitPrompt creates a prompt for commit message generation.
func (r *OpenRouterRepository) createCommitPrompt(diff *entities.GitDiff) string {
	prompt := fmt.Sprintf(`Generate a concise commit message for the following git diff. Follow conventional commits format (type(scope): description).

Files changed: %s
Changes summary: %s

Git diff:
%s

Generate only the commit message, nothing else. Keep it under 72 characters for the summary line.`,
		strings.Join(diff.Files, ", "),
		diff.GetChangesSummary(),
		diff.Content)

	return prompt
}

// cleanCommitMessage cleans and formats the generated commit message.
func (r *OpenRouterRepository) cleanCommitMessage(message string) string {
	// Remove common prefixes that LLMs might add
	message = strings.TrimPrefix(message, "Commit message: ")
	message = strings.TrimPrefix(message, "commit: ")
	message = strings.TrimPrefix(message, "Commit: ")
	
	// Remove quotes if the entire message is quoted
	if strings.HasPrefix(message, `"`) && strings.HasSuffix(message, `"`) {
		message = strings.Trim(message, `"`)
	}
	
	// Remove any trailing periods from the summary line
	lines := strings.Split(message, "\n")
	if len(lines) > 0 {
		lines[0] = strings.TrimSuffix(lines[0], ".")
		message = strings.Join(lines, "\n")
	}
	
	return strings.TrimSpace(message)
}

// makeAPICall makes an HTTP request to the OpenRouter API.
func (r *OpenRouterRepository) makeAPICall(request openRouterRequest) (*openRouterResponse, error) {
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", r.baseURL+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+r.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("HTTP-Referer", "https://github.com/PedroHercules/gommit")
	req.Header.Set("X-Title", "Gommit - AI Commit Message Generator")

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make API call: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var response openRouterResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}