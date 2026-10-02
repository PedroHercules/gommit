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
	"sort"
	"strconv"
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
	Model               string              `json:"model"`
	Messages            []openRouterMessage `json:"messages"`
	Stream              bool                `json:"stream"`
	MaxCompletionTokens int                 `json:"max_completion_tokens,omitempty"`
}

// openRouterMessage represents a message in the conversation.
type openRouterMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// openRouterResponse represents the response from OpenRouter API.
type openRouterResponse struct {
	Choices []struct {
		Message      openRouterCompletionMessage `json:"message"`
		Error        *openRouterAPIError         `json:"error,omitempty"`
		FinishReason string                      `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		TotalTokens int `json:"total_tokens"`
	} `json:"usage"`
	Model string              `json:"model"`
	Error *openRouterAPIError `json:"error,omitempty"`
}

type openRouterCompletionMessage struct {
	Content json.RawMessage `json:"content"`
	Refusal string          `json:"refusal,omitempty"`
}

type openRouterAPIError struct {
	Code     json.RawMessage `json:"code,omitempty"`
	Message  string          `json:"message"`
	Metadata struct {
		ProviderName string          `json:"provider_name"`
		Raw          json.RawMessage `json:"raw,omitempty"`
	} `json:"metadata,omitempty"`
}

func (e *openRouterAPIError) Error() string {
	if e == nil {
		return "OpenRouter returned an unspecified error"
	}

	var details []string
	if len(e.Code) > 0 && string(e.Code) != "null" {
		details = append(details, "code "+string(e.Code))
	}
	if e.Metadata.ProviderName != "" {
		details = append(details, "provider "+e.Metadata.ProviderName)
	}
	if e.Message != "" {
		details = append(details, strings.TrimSpace(e.Message))
	}
	if len(e.Metadata.Raw) > 0 && string(e.Metadata.Raw) != "null" {
		var rawMessage string
		if err := json.Unmarshal(e.Metadata.Raw, &rawMessage); err != nil {
			rawMessage = string(e.Metadata.Raw)
		}
		if rawMessage = truncateErrorBody([]byte(rawMessage), 512); rawMessage != "" {
			details = append(details, "upstream details: "+rawMessage)
		}
	}
	if len(details) == 0 {
		return "OpenRouter returned an unspecified error"
	}
	return "OpenRouter error: " + strings.Join(details, "; ")
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
		Stream: false,
		// Leave room for model reasoning and the required changed-files list.
		MaxCompletionTokens: 1024,
	}

	return r.generate(request, modelToUse, r.cleanCommitMessage), nil
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
		Stream:              false,
		MaxCompletionTokens: 300, // PR descriptions can be longer
	}

	return r.generate(request, modelToUse, r.cleanPRDescription), nil
}

func (r *OpenRouterRepository) generate(request openRouterRequest, requestedModel string, clean func(string) string) *repositories.LLMResponse {
	fail := func(err error) *repositories.LLMResponse {
		return failedLLMResponse(fmt.Errorf("model %q: %w", requestedModel, err))
	}

	response, err := r.makeAPICall(request)
	if err != nil {
		return fail(err)
	}
	if response.Error != nil {
		return fail(response.Error)
	}
	if len(response.Choices) == 0 {
		return fail(errors.New("OpenRouter returned no completion choices"))
	}

	choice := response.Choices[0]
	if choice.Error != nil {
		return fail(choice.Error)
	}
	if choice.Message.Refusal != "" {
		return fail(fmt.Errorf("model refused the request: %s", strings.TrimSpace(choice.Message.Refusal)))
	}
	if strings.EqualFold(choice.FinishReason, "length") || strings.EqualFold(choice.FinishReason, "max_tokens") {
		return fail(errors.New("model response was truncated because it reached the token limit"))
	}

	content, err := extractCompletionText(choice.Message.Content)
	if err != nil {
		return fail(err)
	}
	message := clean(content)
	if message == "" {
		return fail(errors.New("OpenRouter returned an empty completion"))
	}

	model := response.Model
	if model == "" {
		model = requestedModel
	}
	return &repositories.LLMResponse{
		Message:     message,
		Model:       model,
		TokensUsed:  response.Usage.TotalTokens,
		ContextSize: 0,
		Success:     true,
	}
}

func failedLLMResponse(err error) *repositories.LLMResponse {
	return &repositories.LLMResponse{Success: false, Error: err.Error()}
}

func extractCompletionText(content json.RawMessage) (string, error) {
	content = bytes.TrimSpace(content)
	if len(content) == 0 || bytes.Equal(content, []byte("null")) {
		return "", nil
	}

	var text string
	if err := json.Unmarshal(content, &text); err == nil {
		return text, nil
	}

	var parts []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(content, &parts); err == nil {
		texts := make([]string, 0, len(parts))
		for _, part := range parts {
			if part.Text != "" && (part.Type == "" || part.Type == "text") {
				texts = append(texts, part.Text)
			}
		}
		return strings.Join(texts, "\n"), nil
	}

	var part struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(content, &part); err == nil && part.Text != "" && (part.Type == "" || part.Type == "text") {
		return part.Text, nil
	}

	return "", errors.New("OpenRouter returned an unsupported completion content format")
}

// GetAvailableModels returns a list of available LLM models.
func (r *OpenRouterRepository) GetAvailableModels() ([]repositories.LLMModel, error) {
	requestURL := r.baseURL + "/models"
	req, err := http.NewRequest(http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create models request: %w", err)
	}
	if r.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+r.apiKey)
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch OpenRouter models: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read OpenRouter models response: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("OpenRouter models request failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var catalog openRouterModelsResponse
	if err := json.Unmarshal(body, &catalog); err != nil {
		return nil, fmt.Errorf("failed to parse OpenRouter models response: %w", err)
	}

	models := make([]repositories.LLMModel, 0, len(catalog.Data))
	for _, item := range catalog.Data {
		if item.ID == "" {
			continue
		}
		promptPrice, promptOK := parseModelPrice(item.Pricing.Prompt)
		completionPrice, completionOK := parseModelPrice(item.Pricing.Completion)
		model := repositories.LLMModel{
			ID:               item.ID,
			Name:             item.Name,
			Provider:         modelProvider(item.ID),
			ContextSize:      item.Context,
			Available:        true,
			PricingAvailable: promptOK && completionOK,
		}
		if model.PricingAvailable {
			model.PromptCostPer1K = promptPrice * 1000
			model.CompletionCostPer1K = completionPrice * 1000
			model.CostPer1K = (model.PromptCostPer1K + model.CompletionCostPer1K) / 2
			model.Free = promptPrice == 0 && completionPrice == 0
		}
		models = append(models, model)
	}
	sort.Slice(models, func(i, j int) bool { return models[i].ID < models[j].ID })

	// Cache the models
	r.models = models
	return models, nil
}

func parseModelPrice(value string) (float64, bool) {
	if value == "" {
		return 0, false
	}
	price, err := strconv.ParseFloat(value, 64)
	if err != nil || price < 0 {
		return 0, false
	}
	return price, true
}

func modelProvider(modelID string) string {
	provider, _, found := strings.Cut(modelID, "/")
	if !found {
		return ""
	}
	return provider
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
		"nvidia/nemotron-nano-9b-v2:free",
	}

	for _, preferred := range preferredModels {
		for _, model := range models {
			if model.ID == preferred && model.Available && model.Free {
				return &model, nil
			}
		}
	}

	// Fall back to another free model, keeping automatic selection from
	// unexpectedly choosing a model that may incur charges.
	for _, model := range models {
		if model.Available && model.Free {
			return &model, nil
		}
	}

	return nil, errors.New("no free models available; choose a model explicitly with --model or config set-model, and check its OpenRouter pricing")
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
	message = cleanGeneratedText(message, "commit message:", "commit:", "generated commit message:")

	// Remove any trailing periods from the summary line
	lines := strings.Split(message, "\n")
	if len(lines) > 0 {
		lines[0] = strings.TrimSuffix(lines[0], ".")
		message = strings.Join(lines, "\n")
	}

	return strings.TrimSpace(message)
}

func (r *OpenRouterRepository) cleanPRDescription(description string) string {
	return cleanGeneratedText(description, "pull request description:", "pr description:", "generated pr description:")
}

func cleanGeneratedText(text string, labels ...string) string {
	text = strings.TrimSpace(text)
	text = stripWrappingQuotes(text)
	if strings.HasPrefix(text, "```") {
		if lineEnd := strings.IndexByte(text, '\n'); lineEnd >= 0 {
			text = text[lineEnd+1:]
		} else {
			text = strings.TrimPrefix(text, "```")
		}
		if fenceEnd := strings.LastIndex(text, "```"); fenceEnd >= 0 {
			text = text[:fenceEnd]
		}
		text = strings.TrimSpace(text)
	}

	for _, label := range labels {
		if len(text) >= len(label) && strings.EqualFold(text[:len(label)], label) {
			text = strings.TrimSpace(text[len(label):])
			break
		}
	}
	return strings.TrimSpace(stripWrappingQuotes(text))
}

func stripWrappingQuotes(text string) string {
	if len(text) >= 2 && strings.HasPrefix(text, `"`) && strings.HasSuffix(text, `"`) {
		return strings.TrimSpace(strings.Trim(text, `"`))
	}
	return text
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
		if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
			return nil, fmt.Errorf("OpenRouter returned HTTP %d with an unreadable response: %s", resp.StatusCode, truncateErrorBody(body, 512))
		}
		return nil, fmt.Errorf("failed to parse OpenRouter response: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		if response.Error != nil {
			return nil, fmt.Errorf("OpenRouter returned HTTP %d: %w", resp.StatusCode, response.Error)
		}
		return nil, fmt.Errorf("OpenRouter returned HTTP %d: %s", resp.StatusCode, truncateErrorBody(body, 512))
	}

	return &response, nil
}

func truncateErrorBody(body []byte, limit int) string {
	text := strings.TrimSpace(string(body))
	runes := []rune(text)
	if len(runes) > limit {
		return string(runes[:limit]) + "…"
	}
	return text
}
