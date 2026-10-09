package cli

import (
	"strings"
	"testing"

	"github.com/PedroHercules/gommit/pkg/domain/repositories"
	"github.com/gdamore/tcell/v2"
)

func TestFuzzyScorePrioritizesBetterMatches(t *testing.T) {
	tests := []struct {
		name      string
		query     string
		candidate string
		matches   bool
	}{
		{name: "case insensitive prefix", query: "GPT", candidate: "gpt-4o", matches: true},
		{name: "subsequence", query: "claude sonnet", candidate: "Anthropic Claude 3.7 Sonnet", matches: true},
		{name: "unicode", query: "mistrál", candidate: "Mistrál Small", matches: true},
		{name: "no match", query: "zzzzz", candidate: "gpt-4o", matches: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, matches := fuzzyScore(tt.query, tt.candidate)
			if matches != tt.matches {
				t.Fatalf("fuzzyScore(%q, %q) matches = %v, want %v", tt.query, tt.candidate, matches, tt.matches)
			}
		})
	}

	best, _ := fuzzyScore("claude", "claude 3.7 sonnet")
	worse, _ := fuzzyScore("claude", "anthropic claude 3.7 sonnet")
	if best >= worse {
		t.Fatalf("prefix score %d should rank above later match score %d", best, worse)
	}
}

func TestPickerFiltersAndSelectsWithArrowKeys(t *testing.T) {
	screen := newTestPickerScreen(t, 80, 24)
	models := []repositories.LLMModel{
		{ID: "provider/free-one", Name: "Free One", Provider: "Provider", Free: true, PricingAvailable: true},
		{ID: "provider/free-two", Name: "Free Two", Provider: "Provider", Free: true, PricingAvailable: true},
		{ID: "provider/paid", Name: "Paid Model", Provider: "Provider", PricingAvailable: true, PromptCostPer1K: 0.01, CompletionCostPer1K: 0.02},
	}
	entries := modelEntries(models)
	for _, char := range "free" {
		screen.InjectKey(tcell.KeyRune, char, 0)
	}
	screen.InjectKey(tcell.KeyDown, 0, 0)
	screen.InjectKey(tcell.KeyEnter, 0, 0)

	selected, err := runPickerOnScreen(screen, "Choose a model", entries, true, true)
	if err != nil {
		t.Fatalf("runPickerOnScreen() error = %v", err)
	}
	if selected != 1 {
		t.Fatalf("selected entry index = %d, want 1", selected)
	}
	if content := screenContent(screen); !strings.Contains(content, "Search: free") || !strings.Contains(content, "Free models are recommended") || !strings.Contains(content, "Price: free") {
		t.Fatalf("picker should keep search at top and show cost notice, got: %s", content)
	}
}

func TestPickerSupportsUnicodeSearchAndBackspace(t *testing.T) {
	screen := newTestPickerScreen(t, 60, 18)
	entries := []pickerEntry{
		{value: "provider/mistrál", label: "provider/mistrál — Mistrál"},
		{value: "provider/other", label: "provider/other — Other"},
	}
	for _, char := range "mistrál" {
		screen.InjectKey(tcell.KeyRune, char, 0)
	}
	screen.InjectKey(tcell.KeyBackspace2, 0, 0)
	screen.InjectKey(tcell.KeyEnter, 0, 0)

	selected, err := runPickerOnScreen(screen, "Choose a model", entries, true, false)
	if err != nil {
		t.Fatalf("runPickerOnScreen() error = %v", err)
	}
	if selected != 0 {
		t.Fatalf("selected entry index = %d, want 0", selected)
	}
}

func TestPickerCancelsOnEscape(t *testing.T) {
	screen := newTestPickerScreen(t, 80, 24)
	screen.InjectKey(tcell.KeyEscape, 0, 0)

	selected, err := runPickerOnScreen(screen, "Choose provider", []pickerEntry{{value: "OpenRouter", label: "OpenRouter"}}, false, false)
	if err != nil {
		t.Fatalf("runPickerOnScreen() error = %v", err)
	}
	if selected != -1 {
		t.Fatalf("Escape selected index %d, want -1", selected)
	}
}

func TestPickerMenuUsesArrowNavigation(t *testing.T) {
	screen := newTestPickerScreen(t, 80, 12)
	screen.InjectKey(tcell.KeyDown, 0, 0)
	screen.InjectKey(tcell.KeyEnter, 0, 0)
	entries := []pickerEntry{{value: "OpenRouter", label: "OpenRouter"}, {value: "Grok", label: "Grok"}}

	selected, err := runPickerOnScreen(screen, "Choose a provider", entries, false, false)
	if err != nil {
		t.Fatalf("runPickerOnScreen() error = %v", err)
	}
	if selected != 1 {
		t.Fatalf("selected entry index = %d, want 1", selected)
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

func TestModelPickerLabelsMakePricingStatusVisible(t *testing.T) {
	tests := []struct {
		name  string
		model repositories.LLMModel
		want  string
	}{
		{name: "free", model: repositories.LLMModel{ID: "provider/free", PricingAvailable: true, Free: true}, want: "[FREE]"},
		{name: "paid", model: repositories.LLMModel{ID: "provider/paid", PricingAvailable: true}, want: "[PAID]"},
		{name: "unknown", model: repositories.LLMModel{ID: "provider/unknown"}, want: "[PRICE UNKNOWN]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if label := modelPickerLabel(tt.model); !strings.HasPrefix(label, tt.want) {
				t.Fatalf("modelPickerLabel() = %q, want prefix %q", label, tt.want)
			}
		})
	}
}

func modelEntries(models []repositories.LLMModel) []pickerEntry {
	entries := make([]pickerEntry, len(models))
	for i, model := range models {
		entries[i] = pickerEntry{value: model.ID, label: modelPickerLabel(model), detail: modelPickerDetails(model), price: modelPickerPrice(model)}
	}
	return entries
}

func newTestPickerScreen(t *testing.T, width, height int) tcell.SimulationScreen {
	t.Helper()
	screen := tcell.NewSimulationScreen("")
	if err := screen.Init(); err != nil {
		t.Fatalf("screen.Init() error = %v", err)
	}
	screen.SetSize(width, height)
	t.Cleanup(screen.Fini)
	return screen
}

func screenContent(screen tcell.SimulationScreen) string {
	cells, width, height := screen.GetContents()
	var output strings.Builder
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			output.WriteString(string(cells[y*width+x].Runes))
		}
		output.WriteByte('\n')
	}
	return output.String()
}
