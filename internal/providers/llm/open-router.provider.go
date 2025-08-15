package providers

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"

	config "github.com/PedroHercules/gommit/internal/modules/config/services"
	llm_type "github.com/PedroHercules/gommit/internal/providers/llm/types"
	"github.com/PedroHercules/gommit/internal/types"
	"github.com/joho/godotenv"
)

type OpenRouterProvider struct {
}

func NewOpenRouterProvider() *OpenRouterProvider {
	return &OpenRouterProvider{}
}

func (p *OpenRouterProvider) GenerateCommitMessage(diff string) *types.ResultEntity[llm_type.LlmResponseEntity] {
	godotenv.Load()

	reqBody := map[string]interface{}{
		"model": "openai/gpt-oss-20b:free",
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
