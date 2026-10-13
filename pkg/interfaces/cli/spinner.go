package cli

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/term"
)

const (
	spinnerStartDelay = 140 * time.Millisecond
	spinnerInterval   = 100 * time.Millisecond
)

var spinnerFrames = []rune{'|', '/', '-', '\\'}

func withSpinner(label string, work func() error) error {
	interactive := term.IsTerminal(int(os.Stderr.Fd()))
	return runWithSpinner(os.Stderr, interactive, label, work)
}

func runWithSpinner(output io.Writer, interactive bool, label string, work func() error) error {
	return runWithSpinnerTiming(output, interactive, label, work, spinnerStartDelay, spinnerInterval)
}

func runWithSpinnerTiming(output io.Writer, interactive bool, label string, work func() error, startDelay, interval time.Duration) error {
	if !interactive {
		fmt.Fprintf(output, "› %s\n", label)
		err := work()
		writeSpinnerPlainResult(output, label, err)
		return err
	}

	result := make(chan error, 1)
	go func() {
		result <- work()
	}()

	delay := time.NewTimer(startDelay)
	defer delay.Stop()
	select {
	case err := <-result:
		writeSpinnerResult(output, label, err, 0)
		return err
	case <-delay.C:
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	previousWidth := 0
	frame := 0
	for {
		previousWidth = writeSpinnerLine(output, fmt.Sprintf("[%c] %s", spinnerFrames[frame], label), previousWidth)
		frame = (frame + 1) % len(spinnerFrames)
		select {
		case err := <-result:
			writeSpinnerResult(output, label, err, previousWidth)
			return err
		case <-ticker.C:
		}
	}
}

func writeSpinnerResult(output io.Writer, label string, err error, previousWidth int) {
	status := "✓ " + label + " complete."
	if err != nil {
		status = "✗ " + label + " failed."
	}
	writeSpinnerLine(output, status, previousWidth)
	fmt.Fprintln(output)
}

func writeSpinnerPlainResult(output io.Writer, label string, err error) {
	status := "✓ " + label + " complete."
	if err != nil {
		status = "✗ " + label + " failed."
	}
	fmt.Fprintln(output, status)
}

func writeSpinnerLine(output io.Writer, line string, previousWidth int) int {
	width := utf8.RuneCountInString(line)
	padding := max(0, previousWidth-width)
	fmt.Fprintf(output, "\r%s%s", line, strings.Repeat(" ", padding))
	return max(width, previousWidth)
}
