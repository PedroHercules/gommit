package cli

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"unicode"

	"github.com/PedroHercules/gommit/pkg/domain/repositories"
	"github.com/gdamore/tcell/v2"
)

type pickerEntry struct {
	value  string
	label  string
	detail string
	price  string
}

type pickerState struct {
	entries  []pickerEntry
	query    []rune
	cursor   int
	search   bool
	costNote bool
	title    string
}

func promptChoice(title string, options []string) (string, error) {
	entries := make([]pickerEntry, len(options))
	for i, option := range options {
		entries[i] = pickerEntry{value: option, label: option}
	}
	selected, err := runInteractivePicker(title, entries, false, false)
	if err != nil || selected < 0 {
		return "", err
	}
	return entries[selected].value, nil
}

func pickModel(models []repositories.LLMModel) (string, error) {
	if len(models) == 0 {
		return "", nil
	}
	entries := make([]pickerEntry, len(models))
	for i, model := range models {
		entries[i] = pickerEntry{
			value:  model.ID,
			label:  modelPickerLabel(model),
			detail: modelPickerDetails(model),
			price:  modelPickerPrice(model),
		}
	}
	selected, err := runInteractivePicker("Choose a model", entries, true, true)
	if err != nil || selected < 0 {
		return "", err
	}
	return entries[selected].value, nil
}

func runInteractivePicker(title string, entries []pickerEntry, searchable, costNote bool) (int, error) {
	screen, err := tcell.NewScreen()
	if err != nil {
		return -1, fmt.Errorf("could not initialize terminal: %w", err)
	}
	if err := screen.Init(); err != nil {
		return -1, fmt.Errorf("could not initialize terminal: %w", err)
	}
	defer screen.Fini()
	screen.EnablePaste()

	return runPickerOnScreen(screen, title, entries, searchable, costNote)
}

func runPickerOnScreen(screen tcell.Screen, title string, entries []pickerEntry, searchable, costNote bool) (int, error) {
	state := pickerState{
		entries:  entries,
		search:   searchable,
		costNote: costNote,
		title:    title,
	}
	state.render(screen)
	for {
		switch event := screen.PollEvent().(type) {
		case *tcell.EventResize:
			screen.Sync()
			state.render(screen)
		case *tcell.EventKey:
			done, selected := state.handleKey(event)
			if done {
				return selected, nil
			}
			state.render(screen)
		}
	}
}

func (s *pickerState) handleKey(event *tcell.EventKey) (bool, int) {
	filtered := s.filteredEntries()
	switch event.Key() {
	case tcell.KeyEscape, tcell.KeyCtrlC:
		return true, -1
	case tcell.KeyUp:
		if s.cursor > 0 {
			s.cursor--
		}
	case tcell.KeyDown:
		if s.cursor+1 < len(filtered) {
			s.cursor++
		}
	case tcell.KeyEnter:
		if len(filtered) > 0 {
			return true, filtered[s.cursor].index
		}
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if s.search && len(s.query) > 0 {
			s.query = s.query[:len(s.query)-1]
			s.cursor = 0
		}
	case tcell.KeyRune:
		if s.search && !unicode.IsControl(event.Rune()) {
			s.query = append(s.query, event.Rune())
			s.cursor = 0
		}
	}
	return false, -1
}

type indexedEntry struct {
	index int
	pickerEntry
	score int
}

func (s *pickerState) filteredEntries() []indexedEntry {
	query := strings.TrimSpace(string(s.query))
	results := make([]indexedEntry, 0, len(s.entries))
	for index, entry := range s.entries {
		score, matches := fuzzyScore(query, entry.label+" "+entry.detail+" "+entry.price+" "+entry.value)
		if matches {
			results = append(results, indexedEntry{index: index, pickerEntry: entry, score: score})
		}
	}
	if query != "" {
		sort.SliceStable(results, func(i, j int) bool { return results[i].score < results[j].score })
	}
	if s.cursor >= len(results) {
		s.cursor = max(0, len(results)-1)
	}
	return results
}

// fuzzyScore ranks exact and prefix matches first, then contiguous matches,
// followed by subsequence matches. Matching is Unicode-aware and case-folded.
func fuzzyScore(query, candidate string) (int, bool) {
	query = strings.ToLower(strings.TrimSpace(query))
	candidate = strings.ToLower(candidate)
	if query == "" {
		return 0, true
	}
	if candidate == query {
		return 0, true
	}
	if strings.HasPrefix(candidate, query) {
		return 1, true
	}
	if position := strings.Index(candidate, query); position >= 0 {
		return 10 + position, true
	}
	queryWords := strings.Fields(query)
	totalScore := 0
	for _, word := range queryWords {
		score, matches := fuzzySubsequenceScore(word, candidate)
		if !matches {
			return 0, false
		}
		totalScore += score
	}
	return 100 + totalScore, true
}

func fuzzySubsequenceScore(query, candidate string) (int, bool) {
	queryRunes := []rune(query)
	candidateRunes := []rune(candidate)
	queryIndex, first, previous := 0, -1, -1
	gaps := 0
	for index, char := range candidateRunes {
		if char != queryRunes[queryIndex] {
			continue
		}
		if first < 0 {
			first = index
		} else {
			gaps += index - previous - 1
		}
		previous = index
		queryIndex++
		if queryIndex == len(queryRunes) {
			return first*2 + gaps, true
		}
	}
	return 0, false
}

func (s *pickerState) render(screen tcell.Screen) {
	width, height := screen.Size()
	screen.Clear()
	if width <= 0 || height <= 0 {
		screen.Show()
		return
	}
	defaultStyle := tcell.StyleDefault
	selectedStyle := tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorWhite).Bold(true)
	headerStyle := tcell.StyleDefault.Foreground(tcell.ColorAqua).Bold(true)
	mutedStyle := tcell.StyleDefault.Foreground(tcell.ColorGray)

	putLine(screen, 0, 0, s.title, width, headerStyle)
	row := 1
	if s.search {
		putLine(screen, 0, row, "Search: "+string(s.query), width, defaultStyle.Bold(true))
		row++
	}
	if s.costNote {
		putLine(screen, 0, row, "Provider controls pricing and billing; Gommit cannot cap charges.", width, mutedStyle)
		row++
		putLine(screen, 0, row, "Free models are recommended. Verify prices with the provider.", width, mutedStyle)
		row++
	}
	filtered := s.filteredEntries()
	countText := fmt.Sprintf("%d options", len(filtered))
	if s.search {
		countText = fmt.Sprintf("%d models | Type to filter", len(filtered))
	}
	putLine(screen, 0, row, countText+" | ↑/↓ move | Enter select | Esc cancel", width, mutedStyle)
	row++

	detailsHeight := 0
	if s.search && len(filtered) > 0 && height >= 13 {
		detailsHeight = 4
	}
	listStart := row + 1
	listEnd := height - 1 - detailsHeight
	visibleRows := max(0, listEnd-listStart)
	start := 0
	if s.cursor >= visibleRows && visibleRows > 0 {
		start = s.cursor - visibleRows + 1
	}
	for screenRow := 0; screenRow < visibleRows && start+screenRow < len(filtered); screenRow++ {
		entryIndex := start + screenRow
		marker := "  "
		style := defaultStyle
		if entryIndex == s.cursor {
			marker = "> "
			style = selectedStyle
		}
		putLine(screen, 0, listStart+screenRow, marker+filtered[entryIndex].label, width, style)
	}
	if len(filtered) == 0 && visibleRows > 0 {
		putLine(screen, 2, listStart, "No matching options", width-2, mutedStyle)
	}

	if detailsHeight > 0 {
		selected := filtered[s.cursor]
		detailStart := height - detailsHeight
		putLine(screen, 0, detailStart, selected.value, width, headerStyle)
		putLine(screen, 0, detailStart+1, selected.detail, width, mutedStyle)
		putLine(screen, 0, detailStart+2, selected.price, width, mutedStyle)
	}
	screen.Show()
}

func putLine(screen tcell.Screen, x, y int, text string, width int, style tcell.Style) {
	if width <= 0 {
		return
	}
	screen.PutStrStyled(x, y, text, style)
}

func modelPickerLabel(model repositories.LLMModel) string {
	label := model.ID
	if model.Name != "" && model.Name != model.ID {
		label += " — " + model.Name
	}
	switch {
	case !model.PricingAvailable:
		label = "[PRICE UNKNOWN] " + label
	case model.Free:
		label = "[FREE] " + label
	default:
		label = "[PAID] " + label
	}
	return label
}

func modelPickerDetails(model repositories.LLMModel) string {
	parts := make([]string, 0, 3)
	if model.Provider != "" {
		parts = append(parts, "Provider: "+model.Provider)
	}
	if model.ContextSize > 0 {
		parts = append(parts, fmt.Sprintf("Context: %d tokens", model.ContextSize))
	}
	return strings.Join(parts, " | ")
}

func modelPickerPrice(model repositories.LLMModel) string {
	switch {
	case !model.PricingAvailable:
		return "Price unknown"
	case model.Free:
		return "Price: free"
	default:
		return fmt.Sprintf("Prompt $%.6f / completion $%.6f per 1K tokens", model.PromptCostPer1K, model.CompletionCostPer1K)
	}
}

func printModels(output io.Writer, models []repositories.LLMModel) error {
	if _, err := fmt.Fprintf(output, "\nAvailable Models (%d):\n\n", len(models)); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(output, "Model pricing can incur charges. The provider controls billing; Gommit does not control or limit costs. Prefer free models."); err != nil {
		return err
	}
	for _, model := range models {
		if _, err := fmt.Fprintf(output, "  - %s\n", model.ID); err != nil {
			return err
		}
		if model.Name != "" {
			if _, err := fmt.Fprintf(output, "    Name: %s\n", model.Name); err != nil {
				return err
			}
		}
		if model.Provider != "" {
			if _, err := fmt.Fprintf(output, "    Provider: %s\n", model.Provider); err != nil {
				return err
			}
		}
		if model.ContextSize > 0 {
			if _, err := fmt.Fprintf(output, "    Context: %d tokens\n", model.ContextSize); err != nil {
				return err
			}
		}
		switch {
		case !model.PricingAvailable:
			if _, err := fmt.Fprintln(output, "    Price: unknown"); err != nil {
				return err
			}
		case model.Free:
			if _, err := fmt.Fprintln(output, "    Price: free"); err != nil {
				return err
			}
		default:
			if _, err := fmt.Fprintf(output, "    Price per 1K tokens: prompt $%.6f, completion $%.6f\n", model.PromptCostPer1K, model.CompletionCostPer1K); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintln(output); err != nil {
			return err
		}
	}
	return nil
}
