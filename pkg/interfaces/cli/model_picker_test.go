package cli

import (
	"strings"
	"testing"

	"github.com/PedroHercules/gommit/pkg/domain/repositories"
)

func TestFilterModelsMatchesIDNameAndProvider(t *testing.T) {
	models := []repositories.LLMModel{
		{ID: "openai/gpt-model", Name: "GPT Model", Provider: "OpenAI"},
		{ID: "anthropic/sonnet", Name: "Claude Sonnet", Provider: "Anthropic"},
	}

	tests := []struct {
		name  string
		query string
		want  string
	}{
		{name: "matches ID without case sensitivity", query: "GPT-MODEL", want: "openai/gpt-model"},
		{name: "matches display name", query: "claude", want: "anthropic/sonnet"},
		{name: "matches provider", query: "ANTHROPIC", want: "anthropic/sonnet"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered := filterModels(models, tt.query)
			if len(filtered) != 1 || filtered[0].ID != tt.want {
				t.Fatalf("filterModels(%q) = %+v, want only %q", tt.query, filtered, tt.want)
			}
		})
	}

	if filtered := filterModels(models, "no such model"); len(filtered) != 0 {
		t.Fatalf("filterModels() returned results for absent query: %+v", filtered)
	}
	if filtered := filterModels(models, ""); len(filtered) != len(models) {
		t.Fatalf("empty query returned %d models, want %d", len(filtered), len(models))
	}
}

func TestRunModelPickerFiltersAndSelectsWithArrowKeys(t *testing.T) {
	models := []repositories.LLMModel{
		{ID: "provider/free-one", Name: "Free One", Provider: "Provider", Free: true, PricingAvailable: true},
		{ID: "provider/free-two", Name: "Free Two", Provider: "Provider", Free: true, PricingAvailable: true},
		{ID: "provider/paid", Name: "Paid Model", Provider: "Provider", PricingAvailable: true, PromptCostPer1K: 0.01, CompletionCostPer1K: 0.02},
	}
	var output strings.Builder
	selected, err := runModelPicker(models, strings.NewReader("free\x1b[B\r"), &output, 80, 24)
	if err != nil {
		t.Fatalf("runModelPicker() error = %v", err)
	}
	if selected != "provider/free-two" {
		t.Fatalf("selected model = %q, want provider/free-two", selected)
	}
	if !strings.Contains(output.String(), "Search: free") || !strings.Contains(output.String(), "Prefer free models") {
		t.Fatalf("picker output does not show search field and cost notice: %s", output.String())
	}
}

func TestRunModelPickerCancelsOnEscape(t *testing.T) {
	models := []repositories.LLMModel{{ID: "provider/model", Free: true, PricingAvailable: true}}
	var output strings.Builder
	selected, err := runModelPicker(models, strings.NewReader("\x1b"), &output, 80, 24)
	if err != nil {
		t.Fatalf("runModelPicker() error = %v", err)
	}
	if selected != "" {
		t.Fatalf("Escape selected %q, want cancellation", selected)
	}
}

func TestPrintModelsIncludesPricingAndCostNotice(t *testing.T) {
	models := []repositories.LLMModel{
		{ID: "provider/free", PricingAvailable: true, Free: true},
		{ID: "provider/unknown", PricingAvailable: false},
		{ID: "provider/paid", PricingAvailable: true, PromptCostPer1K: 0.001, CompletionCostPer1K: 0.002},
	}
	var output strings.Builder
	if err := printModels(&output, models); err != nil {
		t.Fatalf("printModels() error = %v", err)
	}
	for _, expected := range []string{"does not control or limit costs", "Price: free", "Price: unknown", "prompt $0.001000, completion $0.002000"} {
		if !strings.Contains(output.String(), expected) {
			t.Errorf("printModels() output missing %q", expected)
		}
	}
}
