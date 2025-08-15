package providers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	config "github.com/PedroHercules/gommit/internal/modules/config/services"
	llm_type "github.com/PedroHercules/gommit/internal/providers/llm/types"
	"github.com/PedroHercules/gommit/internal/types"
	"github.com/joho/godotenv"
)

type OpenRouterProvider struct {
}

type ModelPricing struct {
	Prompt     string `json:"prompt"`
	Completion string `json:"completion"`
	Request    string `json:"request"`
	Image      string `json:"image"`
}

type ModelArchitecture struct {
	InputModalities  []string `json:"input_modalities"`
	OutputModalities []string `json:"output_modalities"`
	Tokenizer        string   `json:"tokenizer"`
	InstructType     *string  `json:"instruct_type"`
}

type OpenRouterModel struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	Description   string            `json:"description"`
	ContextLength int               `json:"context_length"`
	Pricing       ModelPricing      `json:"pricing"`
	Architecture  ModelArchitecture `json:"architecture"`
}

type ModelsResponse struct {
	Data []OpenRouterModel `json:"data"`
}

func NewOpenRouterProvider() *OpenRouterProvider {
	return &OpenRouterProvider{}
}

func (p *OpenRouterProvider) getModelInfo(modelID string) (*OpenRouterModel, error) {
	req, err := http.NewRequest("GET", "https://openrouter.ai/api/v1/models", nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var modelsResp ModelsResponse
	err = json.Unmarshal(respBody, &modelsResp)
	if err != nil {
		return nil, err
	}

	for _, model := range modelsResp.Data {
		if model.ID == modelID {
			return &model, nil
		}
	}

	return nil, fmt.Errorf("model not found: %s", modelID)
}

func (p *OpenRouterProvider) getBestFreeModel() (string, error) {
	req, err := http.NewRequest("GET", "https://openrouter.ai/api/v1/models", nil)
	if err != nil {
		return "", err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var modelsResp ModelsResponse
	err = json.Unmarshal(respBody, &modelsResp)
	if err != nil {
		return "", err
	}

	var freeModels []OpenRouterModel
	for _, model := range modelsResp.Data {
		if model.Pricing.Prompt == "0" && model.Pricing.Completion == "0" {
			freeModels = append(freeModels, model)
		}
	}

	if len(freeModels) == 0 {
		return "openai/gpt-oss-20b:free", nil
	}

	sort.Slice(freeModels, func(i, j int) bool {
		modelA := freeModels[i]
		modelB := freeModels[j]

		if modelA.ContextLength != modelB.ContextLength {
			return modelA.ContextLength > modelB.ContextLength
		}

		if strings.Contains(strings.ToLower(modelA.Name), "gpt") && !strings.Contains(strings.ToLower(modelB.Name), "gpt") {
			return true
		}
		if !strings.Contains(strings.ToLower(modelA.Name), "gpt") && strings.Contains(strings.ToLower(modelB.Name), "gpt") {
			return false
		}

		if strings.Contains(strings.ToLower(modelA.Name), "llama") && !strings.Contains(strings.ToLower(modelB.Name), "llama") {
			return true
		}
		if !strings.Contains(strings.ToLower(modelA.Name), "llama") && strings.Contains(strings.ToLower(modelB.Name), "llama") {
			return false
		}

		return modelA.Name < modelB.Name
	})

	return freeModels[0].ID, nil
}

func (p *OpenRouterProvider) GenerateCommitMessage(diff string) *types.ResultEntity[llm_type.LlmResponseEntity] {
	godotenv.Load()

	// Check for user-configured default model first
	var primaryModel string
	if defaultModel, err := config.GetDefaultModel(); err == nil && defaultModel != "" {
		primaryModel = defaultModel
	} else {
		// Try to get best model with fallback
		bestModel, err := p.getBestFreeModel()
		if err != nil {
			bestModel = "openai/gpt-oss-20b:free"
		}
		primaryModel = bestModel
	}

	// Fallback models in order of preference
	fallbackModels := []string{
		"openai/gpt-oss-20b:free",
		"meta-llama/llama-3.2-3b-instruct:free",
		"microsoft/phi-3-mini-128k-instruct:free",
		"google/gemma-2-9b-it:free",
	}

	// Try primary model first, then fallbacks
	modelsToTry := append([]string{primaryModel}, fallbackModels...)
	
	for _, model := range modelsToTry {
		result := p.tryGenerateWithModel(diff, model)
		if result.IsSuccess() {
			return result
		}
	}

	// If all models fail, return a default commit message
	return types.NewSuccess(llm_type.LlmResponseEntity{
		Message:     "feat: update files\n\n- Modified files based on staged changes",
		Model:       "fallback",
		TokensUsed:  0,
		ContextSize: 0,
	})
}

func (p *OpenRouterProvider) tryGenerateWithModel(diff string, model string) *types.ResultEntity[llm_type.LlmResponseEntity] {
	reqBody := map[string]interface{}{
		"model": model,
		"messages": []map[string]string{
			{
				"role":    "system",
				"content": "You are a commit message generator following Conventional Commits specification. Analyze the git diff and generate ONLY a commit message. STRICT FORMAT: type(scope): description\n\n- List of changes in bullet points\n- Each bullet explains what was added/changed/fixed\n- Use past tense for changes (Added, Enhanced, Fixed, etc.)\n\nChanged files:\n- path/to/file1\n- path/to/file2\n\nRULES: 1) Types: feat, fix, docs, style, refactor, test, chore, ci, perf, build 2) Scope: use module/component name 3) Description: present tense, lowercase, no period, max 50 chars 4) Body: bullet points with past tense verbs (Added, Enhanced, Fixed, Updated, Implemented) 5) Always include 'Changed files:' section with file paths 6) Return ONLY the commit message, no explanations",
			},
			{
				"role":    "user",
				"content": "Generate a commit message for this diff:\n\n" + diff,
			},
		},
	}

	req, err := http.NewRequest("POST", "https://openrouter.ai/api/v1/chat/completions", nil)
	if err != nil {
		return types.NewError[llm_type.LlmResponseEntity](err)
	}

	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		keyFromKeyring, keyErr := config.GetLlmKey()
		if keyErr != nil || keyFromKeyring == "" {
			return types.NewFailure[llm_type.LlmResponseEntity]("API key not found. Set OPENROUTER_API_KEY environment variable or use 'gommit config set-key <your-api-key>' to store it securely")
		}
		apiKey = keyFromKeyring
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	reqBodyJson, err := json.Marshal(reqBody)
	if err != nil {
		return types.NewError[llm_type.LlmResponseEntity](err)
	}
	req.Body = io.NopCloser(bytes.NewBuffer(reqBodyJson))

	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return types.NewFailure[llm_type.LlmResponseEntity](fmt.Sprintf("Request failed for model %s: %v", model, err))
	}
	
	// Check HTTP status code
	if resp.StatusCode != http.StatusOK {
		return types.NewFailure[llm_type.LlmResponseEntity](fmt.Sprintf("API returned status %d for model %s", resp.StatusCode, model))
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return types.NewFailure[llm_type.LlmResponseEntity](fmt.Sprintf("Failed to read response body for model %s: %v", model, err))
	}

	var respBodyJson map[string]interface{}
	err = json.Unmarshal(respBody, &respBodyJson)
	if err != nil {
		return types.NewFailure[llm_type.LlmResponseEntity](fmt.Sprintf("Failed to parse JSON response for model %s: %v", model, err))
	}

	// Check for API error in response
	if errorObj, ok := respBodyJson["error"]; ok {
		return types.NewFailure[llm_type.LlmResponseEntity](fmt.Sprintf("API error for model %s: %v", model, errorObj))
	}

	// Extract message with safety checks
	var message string
	if choices, ok := respBodyJson["choices"].([]interface{}); ok && len(choices) > 0 {
		if choice, ok := choices[0].(map[string]interface{}); ok {
			if messageObj, ok := choice["message"].(map[string]interface{}); ok {
				if content, ok := messageObj["content"].(string); ok {
					message = content
				}
			}
		}
	}
	
	if message == "" {
		return types.NewFailure[llm_type.LlmResponseEntity](fmt.Sprintf("Failed to extract message from API response for model %s", model))
	}
	
	// Extract usage information
	var tokensUsed int
	if usage, ok := respBodyJson["usage"].(map[string]interface{}); ok {
		if totalTokens, ok := usage["total_tokens"].(float64); ok {
			tokensUsed = int(totalTokens)
		}
	}
	
	// Get context size from the selected model
	contextSize := 0
	if selectedModel, err := p.getModelInfo(model); err == nil {
		contextSize = selectedModel.ContextLength
	}
	
	response := llm_type.LlmResponseEntity{
		Message:     message,
		Model:       model,
		TokensUsed:  tokensUsed,
		ContextSize: contextSize,
	}
	return types.NewSuccess(response)
}

func GeneratePrMessage(diff string) *types.ResultEntity[llm_type.LlmResponseEntity] {
	message := "feat: implement PR message generation"
	response := llm_type.LlmResponseEntity{
		Message: message,
	}
	return types.NewSuccess(response)
}
