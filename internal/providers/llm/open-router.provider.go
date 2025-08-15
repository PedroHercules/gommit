package providers

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"

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

	bestModel, err := p.getBestFreeModel()
	if err != nil {
		bestModel = "openai/gpt-oss-20b:free"
	}

	reqBody := map[string]interface{}{
		"model": bestModel,
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

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return types.NewError[llm_type.LlmResponseEntity](err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return types.NewError[llm_type.LlmResponseEntity](err)
	}

	var respBodyJson map[string]interface{}
	err = json.Unmarshal(respBody, &respBodyJson)
	if err != nil {
		return types.NewError[llm_type.LlmResponseEntity](err)
	}

	message := respBodyJson["choices"].([]interface{})[0].(map[string]interface{})["message"].(map[string]interface{})["content"].(string)
	response := llm_type.LlmResponseEntity{
		Message: message,
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
