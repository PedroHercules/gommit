package llm

import (
	"io"
	"net/http"
	"strings"
	"testing"
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
