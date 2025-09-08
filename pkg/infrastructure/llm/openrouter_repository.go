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
	Model     string              `json:"model"`
	Messages  []openRouterMessage `json:"messages"`
	Stream    bool                `json:"stream"`
	MaxTokens int                 `json:"max_tokens,omitempty"`
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

	// Prepare the request - using only user message to avoid "developer instruction" issues
	request := openRouterRequest{
		Model: modelToUse,
		Messages: []openRouterMessage{
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

// GeneratePRDescription generates a PR description based on the git diff.
func (r *OpenRouterRepository) GeneratePRDescription(diff *entities.GitDiff, model string) (*repositories.LLMResponse, error) {
	if r.apiKey == "" {
		return nil, errors.New("API key not configured")
	}

	if !diff.HasChanges() {
		return nil, errors.New("no changes to generate PR description for")
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

	// Create the prompt for PR description generation
	prompt := r.createPRPrompt(diff)

	// Prepare the request
	request := openRouterRequest{
		Model: modelToUse,
		Messages: []openRouterMessage{
			{
				Role:    "system",
				Content: "You are an expert developer who writes clear, concise PR descriptions. Generate a PR description based on the provided git diff.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Stream:    false,
		MaxTokens: 300, // PR descriptions can be longer
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

	// Extract and clean the PR description
	prDescription := strings.TrimSpace(response.Choices[0].Message.Content)
	prDescription = r.cleanPRDescription(prDescription)

	return &repositories.LLMResponse{
		Message:     prDescription,
		Model:       response.Model,
		TokensUsed:  response.Usage.TotalTokens,
		ContextSize: 0, // OpenRouter doesn't provide this in the response
		Success:     true,
	}, nil
}

// GetAvailableModels returns a list of available LLM models.
func (r *OpenRouterRepository) GetAvailableModels() ([]repositories.LLMModel, error) {
	// Return predefined models - stable and reliable models first
	models := []repositories.LLMModel{
		{
			ID:          "deepseek/deepseek-chat-v3.1:free",
			Name:        "DeepSeek Chat v3.1 (Free)",
			Provider:    "DeepSeek",
			ContextSize: 32768,
			Available:   true,
		},
		{
			ID:          "moonshotai/kimi-k2:free",
			Name:        "Moonshot AI Kimi-K2 (Free)",
			Provider:    "Moonshot AI",
			ContextSize: 200000,
			Available:   true,
		},
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

	// Preferred models in order of preference - stable and reliable models first
	preferredModels := []string{
		"deepseek/deepseek-chat-v3.1:free",
		"moonshotai/kimi-k2:free",
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
	// Format the files list with proper prefix
	filesList := ""
	for _, file := range diff.Files {
		filesList += "- " + file + "\n"
	}
	
	prompt := fmt.Sprintf(`You MUST generate a commit message using EXACTLY this template:

type(scope): description
- Added/Updated/Fixed/Removed [generic description]
- Added/Updated/Fixed/Removed [generic description]
- Added/Updated/Fixed/Removed [generic description]

Changed files:
%s
CRITICAL: Your output MUST include the "Changed files:" section exactly as shown above.

EXAMPLE of correct format:
feat(config): update model settings
- Updated configuration parameters
- Enhanced model selection logic
- Fixed default model assignment

Changed files:
- pkg/config/models.go
- README.md

RULES:
1) Copy the "Changed files:" section EXACTLY as provided above
2) Use types: feat, fix, docs, style, refactor, test, chore
3) Keep description under 50 chars, lowercase, no period
4) Use verbs: Added, Updated, Fixed, Removed, Enhanced, Implemented
5) Be generic with terms like 'configuration', 'implementation', 'functionality'
6) Output ONLY the commit message - nothing else
7) The "Changed files:" section is MANDATORY - do not omit it

Git diff:
%s`,
		filesList,
		diff.Content)

	return prompt
}

// createPRPrompt creates a prompt for pull request description generation.
func (r *OpenRouterRepository) createPRPrompt(diff *entities.GitDiff) string {
	// Create a list of modified files with markdown formatting
	modifiedFiles := ""
	for _, file := range diff.Files {
		modifiedFiles += fmt.Sprintf("- `%s`\n", file)
	}
	prompt := fmt.Sprintf(`You are a pull request description generator. Analyze the git diff and generate a clear, concise pull request description following this format EXACTLY:

# [Title: Brief description of the main purpose of the changes - only capitalize the first letter of the sentence]

## Description
[A paragraph that provides an overview of the changes, including their purpose, scope, and impact. Explain what was improved, added, or fixed.]

### Key Changes
- **Category/Feature 1:**
  - [Detailed bullet point about specific change]
  - [Detailed bullet point about specific change]

- **Category/Feature 2:**
  - [Detailed bullet point about specific change]
  - [Detailed bullet point about specific change]

### Modified Files
<!-- The list below is automatically generated from the git diff -->
%s

CRITICAL RULES - FOLLOW EXACTLY:
1) Follow the format EXACTLY as shown above
2) Do NOT add any additional explanations or notes outside the template
3) Do NOT include any text like "Here's the PR description" or "I've analyzed the diff"
4) Return ONLY the PR description using the template format
5) Do NOT add any signature, comments, or other text after the PR description
6) CAREFULLY analyze the git diff to identify EXACTLY what was added, removed, or modified
7) ONLY include changes that are actually present in the diff
8) Pay close attention to the + and - symbols in the diff to accurately determine additions and removals
9) Ensure each bullet point corresponds to a real change in the code
10) DO NOT invent or assume changes that are not explicitly shown in the diff
11) DO NOT add features or functionality that are not clearly visible in the code changes
12) STRICTLY base your description on the actual code modifications shown
13) If unsure about a change, describe it generically rather than making assumptions
14) NEVER hallucinate or create fictional details about the implementation
15) Only describe what you can directly observe in the code changes

Git diff:
%s`,
		modifiedFiles,
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

func (r *OpenRouterRepository) cleanPRDescription(description string) string {
	// Remove common prefixes that LLMs might add
	description = strings.TrimPrefix(description, "Pull request description: ")
	description = strings.TrimPrefix(description, "pr description: ")
	description = strings.TrimPrefix(description, "PR description: ")

	// Remove quotes if the entire message is quoted
	if strings.HasPrefix(description, `"`) && strings.HasSuffix(description, `"`) {
		description = strings.Trim(description, `"`)
	}

	return strings.TrimSpace(description)
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
