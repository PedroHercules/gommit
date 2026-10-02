package llm

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/PedroHercules/gommit/pkg/domain/entities"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestGetAvailableModelsUsesOpenRouterCatalog(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/models" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Fatalf("expected API key header, got %q", r.Header.Get("Authorization"))
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body: io.NopCloser(strings.NewReader(`{"data":[
			{"id":"z-provider/paid-model","name":"Paid Model","context_length":4096,"pricing":{"prompt":"0.000002","completion":"0.000004"}},
			{"id":"a-provider/free-model:free","name":"Free Model","context_length":8192,"pricing":{"prompt":"0","completion":"0"}},
			{"id":"provider/unknown-price","pricing":{"prompt":"0.01"}},
			{"name":"Missing ID","pricing":{"prompt":"0","completion":"0"}}
		]}`)),
		}, nil
	})}

	repo := NewOpenRouterRepository("test-key")
	repo.baseURL = "https://openrouter.test/api/v1"
	repo.httpClient = client

	models, err := repo.GetAvailableModels()
	if err != nil {
		t.Fatalf("GetAvailableModels() error = %v", err)
	}
	if len(models) != 3 {
		t.Fatalf("expected 3 models with IDs, got %d", len(models))
	}
	if models[0].ID != "a-provider/free-model:free" {
		t.Fatalf("models should be sorted by ID; first ID = %q", models[0].ID)
	}
	if !models[0].PricingAvailable || !models[0].Free {
		t.Fatalf("expected free model with known zero pricing, got %+v", models[0])
	}
	if models[0].ContextSize != 8192 || models[0].Provider != "a-provider" {
		t.Fatalf("catalog metadata not mapped correctly: %+v", models[0])
	}
	if models[1].PricingAvailable || models[1].Free {
		t.Fatalf("missing price data must remain unknown, got %+v", models[1])
	}
	if !models[2].PricingAvailable || models[2].Free {
		t.Fatalf("expected paid model with known pricing, got %+v", models[2])
	}
	if models[2].PromptCostPer1K != 0.002 || models[2].CompletionCostPer1K != 0.004 {
		t.Fatalf("expected per-1K prices 0.002 and 0.004, got %.6f and %.6f", models[2].PromptCostPer1K, models[2].CompletionCostPer1K)
	}

	valid, err := repo.ValidateModel("z-provider/paid-model")
	if err != nil || !valid {
		t.Fatalf("ValidateModel() = %v, %v; want true, nil", valid, err)
	}
	valid, err = repo.ValidateModel("missing/model")
	if err != nil || valid {
		t.Fatalf("ValidateModel() = %v, %v; want false, nil for an absent ID", valid, err)
	}
}

func TestGetAvailableModelsReportsHTTPError(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(_ *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusServiceUnavailable,
			Body:       io.NopCloser(strings.NewReader("catalog unavailable")),
		}, nil
	})}
	repo := NewOpenRouterRepository("")
	repo.baseURL = "https://openrouter.test/api/v1"
	repo.httpClient = client

	if _, err := repo.GetAvailableModels(); err == nil {
		t.Fatal("expected an error for unsuccessful catalog response")
	}
}

func TestGetBestModelSelectsOnlyFreeModels(t *testing.T) {
	t.Run("falls back to a free catalog model", func(t *testing.T) {
		repo := repositoryWithCatalog(`{"data":[
			{"id":"paid/provider-model","pricing":{"prompt":"0.01","completion":"0.02"}},
			{"id":"free/provider-model","pricing":{"prompt":"0","completion":"0"}}
		]}`)

		model, err := repo.GetBestModel()
		if err != nil {
			t.Fatalf("GetBestModel() error = %v", err)
		}
		if model.ID != "free/provider-model" {
			t.Fatalf("GetBestModel() chose %q, want free model", model.ID)
		}
	})

	t.Run("requires explicit choice if catalog has no free model", func(t *testing.T) {
		repo := repositoryWithCatalog(`{"data":[
			{"id":"paid/provider-model","pricing":{"prompt":"0.01","completion":"0.02"}}
		]}`)

		if _, err := repo.GetBestModel(); err == nil {
			t.Fatal("expected error when no free model is available")
		}
	})
}

func TestGenerateCommitMessageNormalizesCompletionFormats(t *testing.T) {
	fence := strings.Repeat("`", 3)
	tests := []struct {
		name      string
		body      string
		wantText  string
		wantModel string
	}{
		{
			name:      "string with markdown fence and label",
			body:      strings.ReplaceAll(`{"model":"provider/model","choices":[{"finish_reason":"stop","message":{"content":"FENCEtext\nCommit message: feat: add feature.\nFENCE"}}],"usage":{"total_tokens":9}}`, "FENCE", fence),
			wantText:  "feat: add feature",
			wantModel: "provider/model",
		},
		{
			name:      "text content blocks",
			body:      `{"choices":[{"finish_reason":"stop","message":{"content":[{"type":"text","text":"feat: add feature"},{"type":"text","text":"- add detail"}]}}]}`,
			wantText:  "feat: add feature\n- add detail",
			wantModel: "provider/requested-model",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, _ := repositoryWithCompletion(http.StatusOK, tt.body)
			response, err := repo.GenerateCommitMessage(&entities.GitDiff{Content: "diff --git a/file b/file", IsEmpty: false}, "provider/requested-model")
			if err != nil {
				t.Fatalf("GenerateCommitMessage() error = %v", err)
			}
			if !response.Success || response.Message != tt.wantText {
				t.Fatalf("GenerateCommitMessage() = %+v, want successful message %q", response, tt.wantText)
			}
			if response.Model != tt.wantModel {
				t.Fatalf("response model = %q, want %q", response.Model, tt.wantModel)
			}
		})
	}
}

func TestGeneratePRDescriptionUsesSharedContentNormalization(t *testing.T) {
	fence := strings.Repeat("`", 3)
	body := strings.ReplaceAll(`{"choices":[{"finish_reason":"stop","message":{"content":[{"type":"text","text":"FENCEmarkdown\nPR description:\n## Summary\nUpdated the parser.\nFENCE"}]}}]}`, "FENCE", fence)
	repo, _ := repositoryWithCompletion(http.StatusOK, body)
	response, err := repo.GeneratePRDescription(&entities.GitDiff{Content: "diff --git a/file b/file", IsEmpty: false}, "provider/model")
	if err != nil {
		t.Fatalf("GeneratePRDescription() error = %v", err)
	}
	if !response.Success || response.Message != "## Summary\nUpdated the parser." {
		t.Fatalf("GeneratePRDescription() = %+v, want normalized markdown description", response)
	}
}

func TestGenerateCommitMessageReturnsActionableErrorsWithoutRetry(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantParts  []string
	}{
		{
			name:       "provider error envelope",
			statusCode: http.StatusBadGateway,
			body:       `{"error":{"code":502,"message":"Provider returned error","metadata":{"provider_name":"ExampleProvider","raw":"model is temporarily rate-limited upstream"}}}`,
			wantParts:  []string{"HTTP 502", "ExampleProvider", "Provider returned error", "temporarily rate-limited upstream"},
		},
		{
			name:       "provider error in successful http response",
			statusCode: http.StatusOK,
			body:       `{"error":{"code":"provider_error","message":"Provider returned error: no gmit commit","metadata":{"provider_name":"ExampleProvider"}}}`,
			wantParts:  []string{"provider_error", "ExampleProvider", "no gmit commit"},
		},
		{
			name:       "no choices",
			statusCode: http.StatusOK,
			body:       `{"choices":[]}`,
			wantParts:  []string{"no completion choices"},
		},
		{
			name:       "empty text",
			statusCode: http.StatusOK,
			body:       `{"choices":[{"message":{"content":"  "}}]}`,
			wantParts:  []string{"empty completion"},
		},
		{
			name:       "truncated response",
			statusCode: http.StatusOK,
			body:       `{"choices":[{"finish_reason":"length","message":{"content":"feat: incomplete"}}]}`,
			wantParts:  []string{"truncated", "token limit"},
		},
		{
			name:       "unsupported content type",
			statusCode: http.StatusOK,
			body:       `{"choices":[{"message":{"content":42}}]}`,
			wantParts:  []string{"unsupported completion content format"},
		},
		{
			name:       "non-json http error",
			statusCode: http.StatusServiceUnavailable,
			body:       "provider temporarily unavailable",
			wantParts:  []string{"HTTP 503", "provider temporarily unavailable"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, requestCount := repositoryWithCompletion(tt.statusCode, tt.body)
			response, err := repo.GenerateCommitMessage(&entities.GitDiff{Content: "diff --git a/file b/file", IsEmpty: false}, "provider/model")
			if err != nil {
				t.Fatalf("GenerateCommitMessage() error = %v", err)
			}
			if response.Success || response.Error == "" {
				t.Fatalf("expected a failed response with details, got %+v", response)
			}
			for _, part := range tt.wantParts {
				if !strings.Contains(response.Error, part) {
					t.Errorf("error %q does not include %q", response.Error, part)
				}
			}
			if *requestCount != 1 {
				t.Fatalf("request count = %d, want exactly one attempt", *requestCount)
			}
		})
	}
}

func repositoryWithCatalog(catalog string) *OpenRouterRepository {
	repo := NewOpenRouterRepository("")
	repo.baseURL = "https://openrouter.test/api/v1"
	repo.httpClient = &http.Client{Transport: roundTripFunc(func(_ *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(catalog)),
		}, nil
	})}
	return repo
}

func repositoryWithCompletion(statusCode int, body string) (*OpenRouterRepository, *int) {
	repo := NewOpenRouterRepository("test-key")
	repo.baseURL = "https://openrouter.test/api/v1"
	requestCount := 0
	repo.httpClient = &http.Client{Transport: roundTripFunc(func(_ *http.Request) (*http.Response, error) {
		requestCount++
		return &http.Response{
			StatusCode: statusCode,
			Body:       io.NopCloser(strings.NewReader(body)),
		}, nil
	})}
	return repo, &requestCount
}
