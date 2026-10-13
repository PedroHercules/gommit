package cli

import (
	"fmt"
	"reflect"
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

	selected, err := runPickerOnScreen(screen, "Select a model", entries, true, true)
	if err != nil {
		t.Fatalf("runPickerOnScreen() error = %v", err)
	}
	if selected != 1 {
		t.Fatalf("selected entry index = %d, want 1", selected)
	}
	if content := screenContent(screen); !strings.Contains(content, "Search  free_") || !strings.Contains(content, "Free models are recommended") || !strings.Contains(content, "Price: free") {
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

	selected, err := runPickerOnScreen(screen, "Select a model", entries, true, false)
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

	selected, err := runPickerOnScreen(screen, "Select provider", []pickerEntry{{value: "OpenRouter", label: "OpenRouter"}}, false, false)
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

	selected, err := runPickerOnScreen(screen, "Select a provider", entries, false, false)
	if err != nil {
		t.Fatalf("runPickerOnScreen() error = %v", err)
	}
	if selected != 1 {
		t.Fatalf("selected entry index = %d, want 1", selected)
	}
}

func TestCommitActionPickerShowsGeneratedMessageAboveOptions(t *testing.T) {
	screen := newTestPickerScreen(t, 64, 16)
	entries := []pickerEntry{
		{value: commitAndPushAction, label: commitAndPushAction},
		{value: commitOnlyAction, label: commitOnlyAction},
		{value: cancelCommitAction, label: cancelCommitAction},
	}
	screen.InjectKey(tcell.KeyDown, 0, 0)
	screen.InjectKey(tcell.KeyEnter, 0, 0)

	selected, err := runPickerOnScreenWithMessage(screen, "Choose what to do with the generated commit", "feat(auth): add token refresh support", entries, false, false)
	if err != nil {
		t.Fatalf("runPickerOnScreenWithMessage() error = %v", err)
	}
	if selected != 1 {
		t.Fatalf("selected entry index = %d, want 1", selected)
	}
	content := screenContent(screen)
	for _, expected := range []string{"Commit message preview:", "feat(auth): add token refresh support", commitAndPushAction, commitOnlyAction, cancelCommitAction} {
		if !strings.Contains(content, expected) {
			t.Errorf("picker screen missing %q: %s", expected, content)
		}
	}
}

func TestLongCommitMessageScrollsWithoutHidingActions(t *testing.T) {
	screen := newTestPickerScreen(t, 64, 12)
	entries := commitActionEntries()
	var messageLines []string
	for i := 0; i < 20; i++ {
		messageLines = append(messageLines, fmt.Sprintf("commit-line-%02d", i))
	}
	state := pickerState{
		entries: entries,
		title:   "Choose what to do with the generated commit",
		message: strings.Join(messageLines, "\n"),
	}
	state.render(screen)
	if content := screenContent(screen); !strings.Contains(content, "commit-line-00") || !strings.Contains(content, "PgUp/PgDn preview") {
		t.Fatalf("first message line and scroll instructions should be visible initially: %s", content)
	}

	for i := 0; i < 100; i++ {
		state.handleKey(tcell.NewEventKey(tcell.KeyPgDn, 0, tcell.ModNone))
	}
	state.render(screen)
	content := screenContent(screen)
	for _, expected := range []string{"commit-line-19", commitAndPushAction, commitOnlyAction, cancelCommitAction} {
		if !strings.Contains(content, expected) {
			t.Errorf("scrolled picker missing %q: %s", expected, content)
		}
	}
	if state.cursor != 0 {
		t.Fatalf("scrolling the message changed the selected action to %d", state.cursor)
	}

	for i := 0; i < 100; i++ {
		state.handleKey(tcell.NewEventKey(tcell.KeyPgUp, 0, tcell.ModNone))
	}
	state.render(screen)
	if content := screenContent(screen); !strings.Contains(content, "commit-line-00") {
		t.Fatalf("PageUp should return to the start of the complete message: %s", content)
	}
}

func TestCommitMessageWrappingPreservesNewlinesAndAllText(t *testing.T) {
	message := "feat: add token refresh\n\nA longer explanation for this change."
	wrapped := wrapText(message, 12)
	joined := strings.Join(wrapped, "\n")
	want := "feat: add \ntoken \nrefresh\n\nA longer \nexplanation \nfor this \nchange."
	if joined != want {
		t.Fatalf("wrapped message = %q, want %q", joined, want)
	}
}

func TestCommitMessageWrappingPreservesListIndentation(t *testing.T) {
	message := "feat: add support\n\n  - first item\n    - nested item"
	want := []string{"feat: add ", "support", "", "  - first ", "item", "    - ", "nested item"}
	if got := wrapText(message, 12); !reflect.DeepEqual(got, want) {
		t.Fatalf("wrapped message = %#v, want %#v", got, want)
	}
}

func TestCommitActionPickerRequiresEnoughTerminalHeight(t *testing.T) {
	screen := newTestPickerScreen(t, 80, 8)
	_, err := runPickerOnScreenWithMessage(screen, "Choose an action", "feat: add test", commitActionEntries(), false, false)
	if err == nil || !strings.Contains(err.Error(), "at least 9 rows") {
		t.Fatalf("expected actionable minimum-height error, got %v", err)
	}
}

func TestCommitMessagePreviewAdjustsAfterTerminalResize(t *testing.T) {
	screen := newTestPickerScreen(t, 64, 12)
	var messageLines []string
	for i := 0; i < 20; i++ {
		messageLines = append(messageLines, fmt.Sprintf("resize-line-%02d", i))
	}
	state := pickerState{entries: commitActionEntries(), title: "Choose an action", message: strings.Join(messageLines, "\n")}
	state.render(screen)
	for i := 0; i < 100; i++ {
		state.handleKey(tcell.NewEventKey(tcell.KeyPgDn, 0, tcell.ModNone))
	}
	screen.SetSize(64, 24)
	state.render(screen)
	if state.messageScroll+state.messageViewportRows != state.messageLineCount {
		t.Fatalf("after resize, message viewport %d-%d does not reach the final line of %d", state.messageScroll+1, state.messageScroll+state.messageViewportRows, state.messageLineCount)
	}
	content := screenContent(screen)
	for _, action := range []string{commitAndPushAction, commitOnlyAction, cancelCommitAction} {
		if !strings.Contains(content, action) {
			t.Errorf("resized picker hides %q: %s", action, content)
		}
	}
	state.handleKey(tcell.NewEventKey(tcell.KeyPgUp, 0, tcell.ModNone))
	state.handleKey(tcell.NewEventKey(tcell.KeyPgUp, 0, tcell.ModNone))
	state.render(screen)
	if content := screenContent(screen); !strings.Contains(content, "resize-line-00") {
		t.Fatalf("resized preview cannot scroll back to the first message line: %s", content)
	}
}

func commitActionEntries() []pickerEntry {
	return []pickerEntry{
		{value: commitAndPushAction, label: commitAndPushAction},
		{value: commitOnlyAction, label: commitOnlyAction},
		{value: cancelCommitAction, label: cancelCommitAction},
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
	for _, expected := range []string{"cannot cap charges", "Price: free", "Price: unknown", "prompt $0.001000, completion $0.002000"} {
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
		{name: "unknown", model: repositories.LLMModel{ID: "provider/unknown"}, want: "[PRICE?]"},
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
