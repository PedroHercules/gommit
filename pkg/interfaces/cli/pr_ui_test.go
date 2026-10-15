package cli

import (
	"fmt"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
)

func TestPRPreviewViewerScrollsThroughEntireDescription(t *testing.T) {
	screen := newTestPickerScreen(t, 90, 12)
	var lines []string
	for i := 0; i < 30; i++ {
		lines = append(lines, fmt.Sprintf("preview-detail-%02d", i))
	}
	screen.InjectKey(tcell.KeyEnd, 0, 0)
	screen.InjectKey(tcell.KeyEnter, 0, 0)
	proceed, err := reviewPRPreviewOnScreen(screen, strings.Join(lines, "\n"))
	if err != nil || !proceed {
		t.Fatalf("reviewPRPreviewOnScreen() = %t, %v", proceed, err)
	}
	content := screenContent(screen)
	for _, expected := range []string{"Pull request preview · lines 21-30 of 30", "preview-detail-29", "Enter continue", "Esc cancel"} {
		if !strings.Contains(content, expected) {
			t.Errorf("review screen missing %q: %s", expected, content)
		}
	}
}

func TestPRPreviewViewerCanReturnToBeginningOrCancel(t *testing.T) {
	text := strings.Join([]string{"first detail", "second detail", "third detail", "fourth detail", "fifth detail", "sixth detail"}, "\n")
	screen := newTestPickerScreen(t, 48, 7)
	screen.InjectKey(tcell.KeyEnd, 0, 0)
	screen.InjectKey(tcell.KeyHome, 0, 0)
	screen.InjectKey(tcell.KeyEnter, 0, 0)
	proceed, err := reviewPRPreviewOnScreen(screen, text)
	if err != nil || !proceed {
		t.Fatalf("reviewPRPreviewOnScreen() = %t, %v", proceed, err)
	}
	if content := screenContent(screen); !strings.Contains(content, "Pull request preview · lines 1-5 of 6") || !strings.Contains(content, "first detail") {
		t.Fatalf("Home did not return to the beginning of the preview: %s", content)
	}

	cancelScreen := newTestPickerScreen(t, 48, 7)
	cancelScreen.InjectKey(tcell.KeyEscape, 0, 0)
	proceed, err = reviewPRPreviewOnScreen(cancelScreen, text)
	if err != nil || proceed {
		t.Fatalf("Escape result = proceed %t, error %v", proceed, err)
	}
}

func TestReviewerPickerFiltersSelectsMultipleAndConfirms(t *testing.T) {
	screen := newTestPickerScreen(t, 64, 14)
	state := reviewerPickerState{reviewers: []string{"alice", "bob", "carol"}, selected: map[string]string{}}
	for _, char := range "ali" {
		state.handleKey(tcell.NewEventKey(tcell.KeyRune, char, tcell.ModNone))
	}
	state.handleKey(tcell.NewEventKey(tcell.KeyRune, ' ', tcell.ModNone))
	for i := 0; i < 3; i++ {
		state.handleKey(tcell.NewEventKey(tcell.KeyBackspace2, 0, tcell.ModNone))
	}
	state.handleKey(tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone))
	state.handleKey(tcell.NewEventKey(tcell.KeyRune, ' ', tcell.ModNone))
	state.handleKey(tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone))
	state.handleKey(tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone))
	state.handleKey(tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone))
	result, done := state.handleKey(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone))
	if !done {
		t.Fatal("Done action did not close reviewer picker")
	}
	state.render(screen)
	if result.AddManual || result.Cancelled || strings.Join(result.Reviewers, ",") != "alice,bob" {
		t.Fatalf("reviewer selection = %+v", result)
	}
	if content := screenContent(screen); !strings.Contains(content, "[x] alice") || !strings.Contains(content, "[x] bob") {
		t.Fatalf("selected reviewers should remain marked in the picker: %s", content)
	}
}

func TestReviewerPickerOffersManualEntryAndEscapeKeepsAcceptedSelection(t *testing.T) {
	manualScreen := newTestPickerScreen(t, 64, 10)
	manualScreen.InjectKey(tcell.KeyEnter, 0, 0)
	result, err := pickPRReviewersOnScreen(manualScreen, nil, nil)
	if err != nil || !result.AddManual {
		t.Fatalf("manual action result = %+v, %v", result, err)
	}
	if got := parseReviewerLogins("alice, team/reviewers, alice,   "); strings.Join(got, ",") != "alice,team/reviewers" {
		t.Fatalf("parseReviewerLogins() = %#v", got)
	}

	screen := newTestPickerScreen(t, 64, 10)
	screen.InjectKey(tcell.KeyEscape, 0, 0)
	result, err = pickPRReviewersOnScreen(screen, []string{"alice"}, []string{"accepted-team/reviewers"})
	if err != nil || !result.Cancelled || strings.Join(result.Reviewers, ",") != "accepted-team/reviewers" {
		t.Fatalf("Escape result = %+v, %v", result, err)
	}
}
