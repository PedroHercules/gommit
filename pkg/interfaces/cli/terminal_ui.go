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
	entries             []pickerEntry
	query               []rune
	cursor              int
	search              bool
	costNote            bool
	title               string
	message             string
	messageScroll       int
	messageViewportRows int
	messageLineCount    int
}

func promptChoice(title string, options []string) (string, error) {
	return promptChoiceWithMessage(title, "", options)
}

func promptChoiceWithMessage(title, message string, options []string) (string, error) {
	entries := make([]pickerEntry, len(options))
	for i, option := range options {
		entries[i] = pickerEntry{value: option, label: option}
	}
	selected, err := runInteractivePickerWithMessage(title, message, entries, false, false)
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
	selected, err := runInteractivePicker("Select a model", entries, true, true)
	if err != nil || selected < 0 {
		return "", err
	}
	return entries[selected].value, nil
}

func runInteractivePicker(title string, entries []pickerEntry, searchable, costNote bool) (int, error) {
	return runInteractivePickerWithMessage(title, "", entries, searchable, costNote)
}

func runInteractivePickerWithMessage(title, message string, entries []pickerEntry, searchable, costNote bool) (int, error) {
	screen, err := tcell.NewScreen()
	if err != nil {
		return -1, fmt.Errorf("could not start the terminal selector: %w", err)
	}
	if err := screen.Init(); err != nil {
		return -1, fmt.Errorf("could not start the terminal selector: %w", err)
	}
	defer screen.Fini()
	screen.EnablePaste()

	return runPickerOnScreenWithMessage(screen, title, message, entries, searchable, costNote)
}

type prPreviewState struct {
	lines        []string
	displayLines []string
	scroll       int
	viewport     int
	cursor       int
}

func reviewPRPreview(markdown string, defaultCreate bool) (string, error) {
	screen, err := tcell.NewScreen()
	if err != nil {
		return "", fmt.Errorf("could not open the PR preview: %w", err)
	}
	if err := screen.Init(); err != nil {
		return "", fmt.Errorf("could not open the PR preview: %w", err)
	}
	defer screen.Fini()
	screen.EnablePaste()
	return reviewPRPreviewOnScreen(screen, markdown, defaultCreate)
}

func reviewPRPreviewOnScreen(screen tcell.Screen, markdown string, defaultCreate bool) (string, error) {
	_, height := screen.Size()
	if height < 7 {
		return "", fmt.Errorf("terminal is too short to review the PR; resize it to at least 7 rows")
	}
	state := prPreviewState{lines: strings.Split(markdown, "\n")}
	if defaultCreate {
		state.cursor = 0
	} else {
		state.cursor = 1
	}
	state.render(screen)
	for {
		switch event := screen.PollEvent().(type) {
		case *tcell.EventResize:
			screen.Sync()
			state.render(screen)
		case *tcell.EventKey:
			if done, action := state.handleKey(event); done {
				return action, nil
			}
			state.render(screen)
		}
	}
}

func (s *prPreviewState) handleKey(event *tcell.EventKey) (bool, string) {
	switch event.Key() {
	case tcell.KeyEnter:
		if s.cursor == 0 {
			return true, createPRAction
		}
		return true, previewOnlyAction
	case tcell.KeyEscape, tcell.KeyCtrlC:
		return true, ""
	case tcell.KeyLeft, tcell.KeyRight, tcell.KeyTab:
		s.cursor = 1 - s.cursor
	case tcell.KeyPgUp:
		s.scrollBy(-max(1, s.viewport-1))
	case tcell.KeyPgDn:
		s.scrollBy(max(1, s.viewport-1))
	case tcell.KeyUp:
		s.scrollBy(-1)
	case tcell.KeyDown:
		s.scrollBy(1)
	case tcell.KeyHome:
		s.scroll = 0
	case tcell.KeyEnd:
		s.scroll = max(0, len(s.displayLines)-s.viewport)
	}
	return false, ""
}

func (s *prPreviewState) scrollBy(amount int) {
	s.scroll = min(max(0, len(s.displayLines)-s.viewport), max(0, s.scroll+amount))
}

func (s *prPreviewState) render(screen tcell.Screen) {
	width, height := screen.Size()
	screen.Clear()
	if width <= 0 || height <= 0 {
		screen.Show()
		return
	}
	style := tcell.StyleDefault
	header := style.Foreground(tcell.ColorAqua).Bold(true)
	muted := style.Foreground(tcell.ColorGray)
	s.displayLines = wrapText(strings.Join(s.lines, "\n"), max(1, width-2))
	s.viewport = max(1, height-4)
	s.scroll = min(max(0, len(s.displayLines)-s.viewport), max(0, s.scroll))
	visibleEnd := min(len(s.displayLines), s.scroll+s.viewport)
	putLine(screen, 0, 0, fmt.Sprintf("Pull request preview · lines %d-%d of %d", s.scroll+1, visibleEnd, len(s.displayLines)), width, header)
	for index, line := range s.displayLines[s.scroll:visibleEnd] {
		putLine(screen, 0, index+1, line, width, style)
	}
	selectedStyle := style.Foreground(tcell.ColorBlack).Background(tcell.ColorAqua).Bold(true)
	actionStyle := style.Foreground(tcell.ColorAqua)
	for index, action := range []string{createPRAction, previewOnlyAction} {
		actionLine := "  " + action
		lineStyle := actionStyle
		if index == s.cursor {
			actionLine = "> " + action
			lineStyle = selectedStyle
		}
		putLine(screen, 0, height-3+index, actionLine, width, lineStyle)
	}
	controls := "↑/↓ scroll · PgUp/PgDn page · Home/End jump · ←/→ actions · Enter select · Esc cancel"
	if width < 80 {
		controls = "↑↓ scroll · PgUp/Dn page · Home/End · ←→ actions · Enter select · Esc cancel"
	}
	if width < 52 {
		controls = "↑↓/PgUp/Dn scroll · ←→ actions · Enter select · Esc cancel"
	}
	putLine(screen, 0, height-1, controls, width, muted)
	screen.Show()
}

type reviewerPickerResult struct {
	Reviewers []string
	AddManual bool
	Cancelled bool
}

type reviewerPickerState struct {
	reviewers []string
	query     []rune
	cursor    int
	selected  map[string]string
	initial   map[string]string
	viewport  int
}

func pickPRReviewers(reviewers, selected []string) (reviewerPickerResult, error) {
	screen, err := tcell.NewScreen()
	if err != nil {
		return reviewerPickerResult{}, fmt.Errorf("could not open the reviewer selector: %w", err)
	}
	if err := screen.Init(); err != nil {
		return reviewerPickerResult{}, fmt.Errorf("could not open the reviewer selector: %w", err)
	}
	defer screen.Fini()
	screen.EnablePaste()
	return pickPRReviewersOnScreen(screen, reviewers, selected)
}

func pickPRReviewersOnScreen(screen tcell.Screen, reviewers, selected []string) (reviewerPickerResult, error) {
	_, height := screen.Size()
	if height < 7 {
		return reviewerPickerResult{}, fmt.Errorf("terminal is too short for reviewer selection; resize it to at least 7 rows")
	}
	state := reviewerPickerState{reviewers: uniqueReviewerLogins(reviewers), selected: reviewerMap(selected)}
	state.initial = reviewerMap(selected)
	state.render(screen)
	for {
		switch event := screen.PollEvent().(type) {
		case *tcell.EventResize:
			screen.Sync()
			state.render(screen)
		case *tcell.EventKey:
			result, done := state.handleKey(event)
			if done {
				return result, nil
			}
			state.render(screen)
		}
	}
}

func (s *reviewerPickerState) filteredReviewers() []string {
	query := strings.TrimSpace(string(s.query))
	filtered := make([]indexedReviewer, 0, len(s.reviewers))
	for _, login := range s.reviewers {
		score, matches := fuzzyScore(query, login)
		if matches {
			filtered = append(filtered, indexedReviewer{login: login, score: score})
		}
	}
	if query != "" {
		sort.SliceStable(filtered, func(i, j int) bool { return filtered[i].score < filtered[j].score })
	}
	logins := make([]string, len(filtered))
	for index, item := range filtered {
		logins[index] = item.login
	}
	return logins
}

type indexedReviewer struct {
	login string
	score int
}

func (s *reviewerPickerState) handleKey(event *tcell.EventKey) (reviewerPickerResult, bool) {
	filtered := s.filteredReviewers()
	maxCursor := len(filtered) + 1
	if s.cursor > maxCursor {
		s.cursor = maxCursor
	}
	switch event.Key() {
	case tcell.KeyEscape, tcell.KeyCtrlC:
		return reviewerPickerResult{Reviewers: reviewerMapValues(s.initial), Cancelled: true}, true
	case tcell.KeyUp:
		if s.cursor > 0 {
			s.cursor--
		}
	case tcell.KeyDown:
		if s.cursor < maxCursor {
			s.cursor++
		}
	case tcell.KeyRune:
		if event.Rune() == ' ' {
			if s.cursor < len(filtered) {
				s.toggle(filtered[s.cursor])
			}
		} else if !unicode.IsControl(event.Rune()) {
			s.query = append(s.query, event.Rune())
			s.cursor = 0
		}
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if len(s.query) > 0 {
			s.query = s.query[:len(s.query)-1]
			s.cursor = 0
		}
	case tcell.KeyEnter:
		switch s.cursor {
		case len(filtered):
			return reviewerPickerResult{Reviewers: reviewerMapValues(s.selected), AddManual: true}, true
		case len(filtered) + 1:
			return reviewerPickerResult{Reviewers: reviewerMapValues(s.selected)}, true
		default:
			if len(filtered) > 0 {
				s.toggle(filtered[s.cursor])
			}
		}
	}
	return reviewerPickerResult{}, false
}

func (s *reviewerPickerState) toggle(login string) {
	key := strings.ToLower(login)
	if _, exists := s.selected[key]; exists {
		delete(s.selected, key)
	} else {
		s.selected[key] = login
	}
}

func (s *reviewerPickerState) render(screen tcell.Screen) {
	width, height := screen.Size()
	screen.Clear()
	if width <= 0 || height <= 0 {
		screen.Show()
		return
	}
	style := tcell.StyleDefault
	selectedStyle := style.Foreground(tcell.ColorBlack).Background(tcell.ColorWhite).Bold(true)
	header := style.Foreground(tcell.ColorAqua).Bold(true)
	muted := style.Foreground(tcell.ColorGray)
	filtered := s.filteredReviewers()
	putLine(screen, 0, 0, fmt.Sprintf("Select reviewers · %d selected", len(s.selected)), width, header)
	putLine(screen, 0, 1, "Search  "+string(s.query)+"_", width, style.Bold(true))
	putLine(screen, 0, 2, "↑/↓ move · type to filter · Space/Enter toggle · Esc skip", width, muted)
	listStart := 3
	actionStart := height - 3
	s.viewport = max(1, actionStart-listStart)
	start := 0
	if s.cursor < len(filtered) && s.cursor >= s.viewport {
		start = s.cursor - s.viewport + 1
	}
	for row := 0; row < s.viewport && start+row < len(filtered); row++ {
		index := start + row
		login := filtered[index]
		marker, lineStyle := "[ ] ", style
		if _, exists := s.selected[strings.ToLower(login)]; exists {
			marker = "[x] "
		}
		if index == s.cursor {
			lineStyle = selectedStyle
		}
		putLine(screen, 0, listStart+row, marker+login, width, lineStyle)
	}
	if len(filtered) == 0 {
		putLine(screen, 2, listStart, "No matching collaborators · use manual entry below", width-2, muted)
	}
	manualRow, doneRow := len(filtered), len(filtered)+1
	manualStyle, doneStyle := style, style
	if s.cursor == manualRow {
		manualStyle = selectedStyle
	}
	if s.cursor == doneRow {
		doneStyle = selectedStyle
	}
	putLine(screen, 0, actionStart, "+ Add username or team manually", width, manualStyle)
	putLine(screen, 0, actionStart+1, fmt.Sprintf("✓ Done · %d selected", len(s.selected)), width, doneStyle)
	putLine(screen, 0, height-1, "Enter choose · Esc keep current selection", width, muted)
	screen.Show()
}

func reviewerMap(logins []string) map[string]string {
	selected := make(map[string]string, len(logins))
	for _, login := range logins {
		login = strings.TrimSpace(login)
		if login != "" {
			selected[strings.ToLower(login)] = login
		}
	}
	return selected
}

func reviewerMapValues(values map[string]string) []string {
	result := make([]string, 0, len(values))
	for _, login := range values {
		result = append(result, login)
	}
	sort.Slice(result, func(i, j int) bool { return strings.ToLower(result[i]) < strings.ToLower(result[j]) })
	return result
}

func uniqueReviewerLogins(logins []string) []string {
	return reviewerMapValues(reviewerMap(logins))
}

func runPickerOnScreen(screen tcell.Screen, title string, entries []pickerEntry, searchable, costNote bool) (int, error) {
	return runPickerOnScreenWithMessage(screen, title, "", entries, searchable, costNote)
}

func runPickerOnScreenWithMessage(screen tcell.Screen, title, message string, entries []pickerEntry, searchable, costNote bool) (int, error) {
	if message != "" {
		_, height := screen.Size()
		if height < 9 {
			return -1, fmt.Errorf("terminal is too short to show the full commit message and actions; resize it to at least 9 rows")
		}
	}
	state := pickerState{
		entries:  entries,
		search:   searchable,
		costNote: costNote,
		title:    title,
		message:  message,
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
	case tcell.KeyPgUp:
		s.scrollMessage(-max(1, s.messageViewportRows-1))
	case tcell.KeyPgDn:
		s.scrollMessage(max(1, s.messageViewportRows-1))
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

func (s *pickerState) scrollMessage(amount int) {
	if s.message == "" || s.messageViewportRows == 0 {
		return
	}
	maxScroll := max(0, s.messageLineCount-s.messageViewportRows)
	s.messageScroll = min(maxScroll, max(0, s.messageScroll+amount))
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
	if s.message != "" && height < 9 {
		putLine(screen, 0, 0, "Terminal space is too small for the full preview.", width, headerStyle)
		putLine(screen, 0, 1, "Resize to at least 9 rows to continue.", width, mutedStyle)
		s.messageViewportRows = 0
		screen.Show()
		return
	}

	putLine(screen, 0, 0, s.title, width, headerStyle)
	row := 1
	if s.search {
		putLine(screen, 0, row, "Search  "+string(s.query)+"_", width, defaultStyle.Bold(true))
		row++
	}
	if s.costNote {
		putLine(screen, 0, row, "Provider controls pricing and billing; Gmit cannot cap charges.", width, mutedStyle)
		row++
		putLine(screen, 0, row, "Free models are recommended. Verify prices with the provider.", width, mutedStyle)
		row++
	}
	if s.message != "" {
		filtered := s.filteredEntries()
		messageLines := wrapText(s.message, width-2)
		s.messageLineCount = len(messageLines)
		optionRows := max(1, min(3, len(filtered)))
		availableMessageRows := max(1, height-row-4-optionRows)
		s.messageViewportRows = min(len(messageLines), availableMessageRows)
		maxScroll := max(0, len(messageLines)-s.messageViewportRows)
		s.messageScroll = min(maxScroll, max(0, s.messageScroll))
		if maxScroll > 0 {
			putLine(screen, 0, row, fmt.Sprintf("Commit message preview · lines %d-%d of %d:", s.messageScroll+1, s.messageScroll+s.messageViewportRows, len(messageLines)), width, headerStyle)
		} else {
			putLine(screen, 0, row, "Commit message preview:", width, headerStyle)
		}
		row++
		for _, line := range messageLines[s.messageScroll : s.messageScroll+s.messageViewportRows] {
			putLine(screen, 1, row, line, width-1, defaultStyle)
			row++
		}
	}
	filtered := s.filteredEntries()
	countText := fmt.Sprintf("%d choices", len(filtered))
	if s.search {
		countText = fmt.Sprintf("%d models · type to filter", len(filtered))
	}
	controls := countText + " · ↑/↓ move · Enter select · Esc cancel"
	if s.message != "" {
		controls = "↑/↓ choose · Enter confirm · Esc cancel"
		if len(wrapText(s.message, width-2)) > s.messageViewportRows {
			controls += " · PgUp/PgDn preview"
		}
	}
	putLine(screen, 0, row, controls, width, mutedStyle)
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
		putLine(screen, 2, listStart, "No matches · try a different search", width-2, mutedStyle)
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

func wrapText(text string, width int) []string {
	if width < 1 {
		width = 1
	}
	logicalLines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	lines := make([]string, 0, len(logicalLines))
	for _, logicalLine := range logicalLines {
		runes := []rune(logicalLine)
		if len(runes) == 0 {
			lines = append(lines, "")
			continue
		}
		for len(runes) > width {
			breakAt := width
			for i := width; i > 1; i-- {
				if runes[i-1] == ' ' || runes[i-1] == '\t' {
					breakAt = i
					break
				}
			}
			lines = append(lines, string(runes[:breakAt]))
			runes = runes[breakAt:]
		}
		lines = append(lines, string(runes))
	}
	return lines
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
		label = "[PRICE?] " + label
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
		return "Price unknown · check provider"
	case model.Free:
		return "Price: free"
	default:
		return fmt.Sprintf("Prompt $%.6f / completion $%.6f per 1K tokens", model.PromptCostPer1K, model.CompletionCostPer1K)
	}
}

func printModels(output io.Writer, models []repositories.LLMModel) error {
	if _, err := fmt.Fprintf(output, "\nMODELS  /  %d available\n\n", len(models)); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(output, "! Providers control pricing and billing; Gmit cannot cap charges. Prefer free models."); err != nil {
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
