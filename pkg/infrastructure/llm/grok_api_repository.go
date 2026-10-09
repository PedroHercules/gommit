package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/PedroHercules/gommit/pkg/domain/entities"
	"github.com/PedroHercules/gommit/pkg/domain/repositories"
)

// GrokAPIRepository calls xAI's public REST API using an API key.
type GrokAPIRepository struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

func NewGrokAPIRepository(apiKey string) *GrokAPIRepository {
	return &GrokAPIRepository{
		apiKey:  apiKey,
		baseURL: "https://api.x.ai/v1",
		httpClient: &http.Client{
			Timeout: 2 * time.Minute,
		},
	}
}

type xAIResponsesRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

type xAIResponsesResponse struct {
	ID     string `json:"id"`
	Model  string `json:"model"`
	Status string `json:"status"`
	Output []struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	} `json:"output"`
	OutputText string `json:"output_text"`
	Usage      struct {
		TotalTokens int `json:"total_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (r *GrokAPIRepository) GenerateCommitMessage(diff *entities.GitDiff, model string) (*repositories.LLMResponse, error) {
	return r.generate(model, (&OpenRouterRepository{}).createCommitPrompt(diff), (&OpenRouterRepository{}).cleanCommitMessage)
}

func (r *GrokAPIRepository) GeneratePRDescription(diff *entities.GitDiff, model string) (*repositories.LLMResponse, error) {
	return r.generate(model, (&OpenRouterRepository{}).createPRPrompt(diff), (&OpenRouterRepository{}).cleanPRDescription)
}

func (r *GrokAPIRepository) generate(model, prompt string, clean func(string) string) (*repositories.LLMResponse, error) {
	if strings.TrimSpace(r.apiKey) == "" {
		return nil, errors.New("xAI API key is not configured")
	}
	if strings.TrimSpace(model) == "" {
		return nil, errors.New("Grok model is not configured; run `gmit config` to select one")
	}

	payload, err := json.Marshal(xAIResponsesRequest{Model: model, Input: prompt})
	if err != nil {
		return nil, fmt.Errorf("encode xAI request: %w", err)
	}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, r.baseURL+"/responses", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create xAI request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+r.apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := r.httpClient.Do(req)
	if err != nil {
		return failedLLMResponse(fmt.Errorf("xAI request failed: %w", err)), nil
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return failedLLMResponse(fmt.Errorf("read xAI response: %w", err)), nil
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return failedLLMResponse(fmt.Errorf("xAI API returned HTTP %d: %s", resp.StatusCode, truncateErrorBody(body, 512))), nil
	}

	var result xAIResponsesResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return failedLLMResponse(fmt.Errorf("parse xAI response: %w", err)), nil
	}
	if result.Error != nil {
		return failedLLMResponse(fmt.Errorf("xAI API error: %s", result.Error.Message)), nil
	}
	var content strings.Builder
	if result.OutputText != "" {
		content.WriteString(result.OutputText)
	} else {
		for _, item := range result.Output {
			for _, part := range item.Content {
				if part.Type == "output_text" || part.Type == "text" || part.Type == "" {
					content.WriteString(part.Text)
				}
			}
		}
	}
	message := clean(content.String())
	if message == "" {
		return failedLLMResponse(fmt.Errorf("xAI returned no text (status %q)", result.Status)), nil
	}
	usedModel := result.Model
	if usedModel == "" {
		usedModel = model
	}
	return &repositories.LLMResponse{Message: message, Model: usedModel, TokensUsed: result.Usage.TotalTokens, Success: true}, nil
}

type xAIModelsResponse struct {
	Models []xAIModel `json:"models"`
	Data   []xAIModel `json:"data"`
}

type xAIModel struct {
	ID                       string   `json:"id"`
	Name                     string   `json:"name"`
	OwnedBy                  string   `json:"owned_by"`
	Aliases                  []string `json:"aliases"`
	ContextLength            int      `json:"context_length"`
	PromptTextTokenPrice     float64  `json:"prompt_text_token_price"`
	CompletionTextTokenPrice float64  `json:"completion_text_token_price"`
}

func (r *GrokAPIRepository) GetAvailableModels() ([]repositories.LLMModel, error) {
	if strings.TrimSpace(r.apiKey) == "" {
		return nil, errors.New("xAI API key is not configured")
	}
	req, err := http.NewRequest(http.MethodGet, r.baseURL+"/language-models", nil)
	if err != nil {
		return nil, fmt.Errorf("create xAI models request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+r.apiKey)
	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch xAI models: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read xAI models response: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("xAI models request failed with HTTP %d: %s", resp.StatusCode, truncateErrorBody(body, 512))
	}
	var catalog xAIModelsResponse
	if err := json.Unmarshal(body, &catalog); err != nil {
		return nil, fmt.Errorf("parse xAI models response: %w", err)
	}
	items := catalog.Models
	if len(items) == 0 {
		items = catalog.Data
	}
	models := make([]repositories.LLMModel, 0, len(items))
	for _, item := range items {
		if item.ID == "" {
			continue
		}
		name := item.Name
		if name == "" {
			name = item.ID
		}
		models = append(models, repositories.LLMModel{
			ID:                  item.ID,
			Name:                name,
			Provider:            "xAI",
			ContextSize:         item.ContextLength,
			PromptCostPer1K:     item.PromptTextTokenPrice / 1e7,
			CompletionCostPer1K: item.CompletionTextTokenPrice / 1e7,
			CostPer1K:           (item.PromptTextTokenPrice + item.CompletionTextTokenPrice) / 2e7,
			PricingAvailable:    item.PromptTextTokenPrice > 0 || item.CompletionTextTokenPrice > 0,
			Available:           true,
		})
	}
	sort.Slice(models, func(i, j int) bool { return models[i].ID < models[j].ID })
	if len(models) == 0 {
		return nil, errors.New("xAI returned no language models for this API key")
	}
	return models, nil
}

func (r *GrokAPIRepository) ValidateModel(modelID string) (bool, error) {
	models, err := r.GetAvailableModels()
	if err != nil {
		return false, err
	}
	for _, model := range models {
		if model.ID == modelID {
			return true, nil
		}
	}
	return false, nil
}

func (r *GrokAPIRepository) GetBestModel() (*repositories.LLMModel, error) {
	return nil, errors.New("select a Grok model with `gmit config`")
}

func (r *GrokAPIRepository) TestConnection(apiKey string) (bool, error) {
	previousKey := r.apiKey
	r.apiKey = apiKey
	defer func() { r.apiKey = previousKey }()
	_, err := r.GetAvailableModels()
	return err == nil, err
}

func (r *GrokAPIRepository) SetAPIKey(apiKey string) { r.apiKey = apiKey }

func (r *GrokAPIRepository) GetModelInfo(modelID string) (*repositories.LLMModel, error) {
	models, err := r.GetAvailableModels()
	if err != nil {
		return nil, err
	}
	for _, model := range models {
		if model.ID == modelID {
			return &model, nil
		}
	}
	return nil, fmt.Errorf("xAI model not found: %s", modelID)
}
