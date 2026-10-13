package cli

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestRunWithSpinnerWritesStaticStatusWithoutTTY(t *testing.T) {
	var output bytes.Buffer
	err := runWithSpinner(&output, false, "Loading model catalog", func() error { return nil })
	if err != nil {
		t.Fatalf("runWithSpinner() error = %v", err)
	}
	text := output.String()
	if !strings.Contains(text, "› Loading model catalog\n") || !strings.Contains(text, "✓ Loading model catalog complete.\n") {
		t.Fatalf("static progress output = %q", text)
	}
	if strings.ContainsAny(text, "\r\\") || strings.Contains(text, "[|") {
		t.Fatalf("non-interactive output contains spinner control characters: %q", text)
	}
}

func TestRunWithSpinnerReportsFailure(t *testing.T) {
	var output bytes.Buffer
	wantErr := errors.New("provider unavailable")
	err := runWithSpinner(&output, false, "Drafting commit message", func() error { return wantErr })
	if !errors.Is(err, wantErr) {
		t.Fatalf("runWithSpinner() error = %v, want %v", err, wantErr)
	}
	if !strings.Contains(output.String(), "✗ Drafting commit message failed.") {
		t.Fatalf("failure status missing from output: %q", output.String())
	}
}

func TestRunWithSpinnerAnimatesOnlyLongInteractiveOperations(t *testing.T) {
	var output bytes.Buffer
	started := make(chan struct{})
	release := make(chan struct{})
	done := make(chan error, 1)
	const delay = 20 * time.Millisecond
	const interval = 10 * time.Millisecond
	go func() {
		done <- runWithSpinnerTiming(&output, true, "Building pull request preview", func() error {
			close(started)
			<-release
			return nil
		}, delay, interval)
	}()
	<-started
	time.Sleep(delay + interval + 5*time.Millisecond)
	close(release)
	if err := <-done; err != nil {
		t.Fatalf("runWithSpinner() error = %v", err)
	}
	text := output.String()
	if !strings.Contains(text, "[|") || !strings.Contains(text, "✓ Building pull request preview complete.") {
		t.Fatalf("interactive spinner output = %q", text)
	}
}

func TestRunWithSpinnerSkipsAnimationForFastInteractiveOperation(t *testing.T) {
	var output bytes.Buffer
	err := runWithSpinnerTiming(&output, true, "Loading model catalog", func() error { return nil }, time.Hour, spinnerInterval)
	if err != nil {
		t.Fatalf("runWithSpinner() error = %v", err)
	}
	if strings.Contains(output.String(), "[|") || !strings.Contains(output.String(), "✓ Loading model catalog complete.") {
		t.Fatalf("fast operation output = %q", output.String())
	}
}
