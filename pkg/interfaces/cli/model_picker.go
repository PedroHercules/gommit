package cli

import (
	"fmt"
	"io"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/PedroHercules/gommit/pkg/domain/repositories"
)

const modelPickerEscapeDelay = 35 * time.Millisecond

type modelPickerInput struct {
	value byte
	err   error
}

type modelPickerKey int

const (
	modelPickerCharacter modelPickerKey = iota
	modelPickerEnter
	modelPickerBackspace
	modelPickerUp
	modelPickerDown
	modelPickerCancel
)

func filterModels(models []repositories.LLMModel, query string) []repositories.LLMModel {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return models
	}

	filtered := make([]repositories.LLMModel, 0)
	for _, model := range models {
		if strings.Contains(strings.ToLower(model.ID), query) ||
			strings.Contains(strings.ToLower(model.Name), query) ||
			strings.Contains(strings.ToLower(model.Provider), query) {
			filtered = append(filtered, model)
		}
	}
	return filtered
}

func runModelPicker(models []repositories.LLMModel, input io.Reader, output io.Writer, width, height int) (string, error) {
	inputEvents := make(chan modelPickerInput, 32)
	go func() {
		var value [1]byte
		for {
			_, err := io.ReadFull(input, value[:])
			if err != nil {
				inputEvents <- modelPickerInput{err: err}
				return
			}
			inputEvents <- modelPickerInput{value: value[0]}
		}
	}()

	query := ""
	selected := 0
	for {
		matches := filterModels(models, query)
		if selected >= len(matches) {
			selected = len(matches) - 1
		}
		if selected < 0 {
			selected = 0
		}
		if err := renderModelPicker(output, matches, query, selected, width, height); err != nil {
			return "", err
		}

		key, value, err := readModelPickerKey(inputEvents)
		if err != nil {
			return "", err
		}
		switch key {
		case modelPickerCharacter:
			query += string(value)
			selected = 0
		case modelPickerBackspace:
			if query != "" {
				_, size := utf8.DecodeLastRuneInString(query)
				query = query[:len(query)-size]
			}
			selected = 0
		case modelPickerUp:
			if selected > 0 {
				selected--
			}
		case modelPickerDown:
			if selected+1 < len(matches) {
				selected++
			}
		case modelPickerEnter:
			if len(matches) > 0 {
				return matches[selected].ID, nil
			}
		case modelPickerCancel:
			return "", nil
		}
	}
}

func readModelPickerKey(input <-chan modelPickerInput) (modelPickerKey, byte, error) {
	event, err := nextModelPickerInput(input)
	if err != nil {
		return 0, 0, err
	}
	switch event.value {
	case 3, 27:
		if event.value == 3 {
			return modelPickerCancel, 0, nil
		}
		select {
		case next := <-input:
			if next.err != nil {
				return modelPickerCancel, 0, nil
			}
			if next.value == '[' {
				select {
				case direction := <-input:
					switch direction.value {
					case 'A':
						return modelPickerUp, 0, nil
					case 'B':
						return modelPickerDown, 0, nil
					}
				case <-time.After(modelPickerEscapeDelay):
				}
			}
			return modelPickerCancel, 0, nil
		case <-time.After(modelPickerEscapeDelay):
			return modelPickerCancel, 0, nil
		}
	case '\r', '\n':
		return modelPickerEnter, 0, nil
	case 8, 127:
		return modelPickerBackspace, 0, nil
	default:
		if event.value >= 32 {
			return modelPickerCharacter, event.value, nil
		}
	}
	return 0, 0, nil
}

func nextModelPickerInput(input <-chan modelPickerInput) (modelPickerInput, error) {
	event := <-input
	if event.err != nil {
		return modelPickerInput{}, event.err
	}
	return event, nil
}

func renderModelPicker(output io.Writer, models []repositories.LLMModel, query string, selected, width, height int) error {
	if width <= 0 {
		width = 80
	}
	visibleRows := height - 8
	if visibleRows <= 0 || height <= 0 {
		visibleRows = 10
	}
	if visibleRows > 12 {
		visibleRows = 12
	}
	start := 0
	if selected >= visibleRows {
		start = selected - visibleRows + 1
	}
	end := start + visibleRows
	if end > len(models) {
		end = len(models)
	}

	if _, err := fmt.Fprint(output, "\x1b[2J\x1b[HOpenRouter models\n"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(output, "OpenRouter controls pricing and billing; Gommit cannot limit charges. Prefer free models."); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(output, "Search: %s\n", query); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(output, "Matches: %d | Type to filter | ↑/↓ navigate | Enter select | Esc cancel\n\n", len(models)); err != nil {
		return err
	}
	if len(models) == 0 {
		_, err := fmt.Fprintln(output, "  No matching models")
		return err
	}
	for index := start; index < end; index++ {
		model := models[index]
		marker := "  "
		if index == selected {
			marker = "> "
		}
		line := fmt.Sprintf("%s%s", marker, model.ID)
		if model.Name != "" {
			line += " | " + model.Name
		}
		if model.PricingAvailable && model.Free {
			line += " | Free"
		} else if !model.PricingAvailable {
			line += " | Price unknown"
		} else {
			line += fmt.Sprintf(" | $%.6f/$%.6f per 1K tokens", model.PromptCostPer1K, model.CompletionCostPer1K)
		}
		_, err := fmt.Fprintln(output, truncateModelPickerLine(line, width))
		if err != nil {
			return err
		}
	}
	return nil
}

func truncateModelPickerLine(line string, width int) string {
	if utf8.RuneCountInString(line) <= width {
		return line
	}
	if width <= 1 {
		return "…"
	}
	runes := []rune(line)
	return string(runes[:width-1]) + "…"
}

func printModels(output io.Writer, models []repositories.LLMModel) error {
	if _, err := fmt.Fprintf(output, "\nAvailable Models (%d):\n\n", len(models)); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(output, "OpenRouter model pricing can incur charges. Gommit does not control or limit costs; prefer free models."); err != nil {
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
		if !model.PricingAvailable {
			if _, err := fmt.Fprintln(output, "    Price: unknown"); err != nil {
				return err
			}
		} else if model.Free {
			if _, err := fmt.Fprintln(output, "    Price: free"); err != nil {
				return err
			}
		} else if _, err := fmt.Fprintf(output, "    Price per 1K tokens: prompt $%.6f, completion $%.6f\n", model.PromptCostPer1K, model.CompletionCostPer1K); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(output); err != nil {
			return err
		}
	}
	return nil
}
