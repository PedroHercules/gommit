package llm

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/PedroHercules/gommit/pkg/domain/entities"
)

func TestGrokAPIUsesResponsesEndpointAndNormalizesText(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost || req.URL.Path != "/v1/responses" {
			t.Fatalf("request = %s %s, want POST /v1/responses", req.Method, req.URL.Path)
		}
		if req.Header.Get("Authorization") != "Bearer xai-test-key" {
			t.Fatalf("authorization = %q", req.Header.Get("Authorization"))
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"model":"grok-4.7","status":"completed","output":[{"content":[{"type":"output_text","text":"feat: add Grok provider"}]}],"usage":{"total_tokens":21}}`)),
		}, nil
	})}
	repo := NewGrokAPIRepository("xai-test-key")
	repo.baseURL = "https://xai.test/v1"
	repo.httpClient = client

	response, err := repo.GenerateCommitMessage(&entities.GitDiff{Content: "diff --git a/a b/a", IsEmpty: false}, "grok-4.7")
	if err != nil {
		t.Fatalf("GenerateCommitMessage() error = %v", err)
	}
	if !response.Success || response.Message != "feat: add Grok provider" || response.Model != "grok-4.7" || response.TokensUsed != 21 {
		t.Fatalf("unexpected response: %+v", response)
	}
}

func TestGrokAPIListsLanguageModelsForConfiguredKey(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet || req.URL.Path != "/v1/language-models" {
			t.Fatalf("request = %s %s, want GET /v1/language-models", req.Method, req.URL.Path)
		}
		if req.Header.Get("Authorization") != "Bearer xai-test-key" {
			t.Fatalf("authorization = %q", req.Header.Get("Authorization"))
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"models":[{"id":"grok-4.7","name":"Grok 4.7","owned_by":"xai","context_length":500000,"prompt_text_token_price":200000000,"completion_text_token_price":600000000}]}`)),
		}, nil
	})}
	repo := NewGrokAPIRepository("xai-test-key")
	repo.baseURL = "https://xai.test/v1"
	repo.httpClient = client

	models, err := repo.GetAvailableModels()
	if err != nil {
		t.Fatalf("GetAvailableModels() error = %v", err)
	}
	if len(models) != 1 || models[0].ID != "grok-4.7" || models[0].ContextSize != 500000 {
		t.Fatalf("unexpected models: %+v", models)
	}
	if models[0].PromptCostPer1K != 20 || models[0].CompletionCostPer1K != 60 {
		t.Fatalf("unexpected per-1K prices: %+v", models[0])
	}
}

func TestExtractGrokCLITextFromJsonAndPlainText(t *testing.T) {
	for _, test := range []struct {
		name string
		data string
		want string
	}{
		{name: "json result", data: `{"result":"feat: add OAuth"}`, want: "feat: add OAuth"},
		{name: "nested content", data: `{"messages":[{"role":"assistant","content":"fix: parse output"}]}`, want: "fix: parse output"},
		{name: "plain output", data: "feat: use Grok CLI", want: "feat: use Grok CLI"},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := extractGrokCLIText([]byte(test.data))
			if err != nil {
				t.Fatalf("extractGrokCLIText() error = %v", err)
			}
			if got != test.want {
				t.Fatalf("extractGrokCLIText() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestParseGrokModelsDeduplicatesModelIDs(t *testing.T) {
	models := parseGrokModels([]byte("Available models\n  grok-4.7  Grok 4.7\n  grok-4.6  Grok 4.6\n  grok-4.7  Default\n"))
	if len(models) != 2 || models[0].ID != "grok-4.7" || models[1].ID != "grok-4.6" {
		t.Fatalf("parseGrokModels() = %+v", models)
	}
}
