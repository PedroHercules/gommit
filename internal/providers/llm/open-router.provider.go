package providers

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"

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
		"model": "openai/gpt-3.5-turbo",
		"messages": []map[string]string{
			{
				"role":    "system",
				"content": "You are a commit message generator. Analyze the git diff and generate ONLY a commit message following conventional commits format. Rules: 1) Return ONLY the commit message, no explanations or extra text. 2) Format: type(scope): description\n\ndetailed changes. 3) Use types: feat, fix, docs, style, refactor, test, chore. 4) Keep title under 50 chars. 5) Add details about what was changed. 6) No markdown, quotes, or special characters. 7) Use present tense.",
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
		return types.NewFailure[llm_type.LlmResponseEntity]("OPENROUTER_API_KEY environment variable not set")
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
