package cli

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/PedroHercules/gommit/pkg/domain/repositories"
	"github.com/gdamore/tcell/v2"
)

type stagingRow struct {
	label  string
	file   repositories.FileStatus
	action string
}

func stagingRows(files []repositories.FileStatus) []stagingRow {
	rows := []stagingRow{{label: "[+] Stage all changes", action: "all"}, {label: "── Staged ──", action: "staged-header"}}
	for _, file := range files {
		if file.Index != ' ' && file.Index != '?' {
			name := displayPath(file.Path)
			if file.OriginalPath != "" {
				name = displayPath(file.OriginalPath) + " → " + name
			}
			if file.Worktree != ' ' {
				name += " (also has unstaged changes)"
			}
			rows = append(rows, stagingRow{label: fmt.Sprintf("[staged %c] %s", file.Index, name), file: file, action: "unstage"})
		}
	}
	rows = append(rows, stagingRow{label: "── Unstaged ──"})
	for _, file := range files {
		if file.Worktree != ' ' || file.Index == '?' {
			status := file.Worktree
			if file.Index == '?' {
				status = '?'
			}
			rows = append(rows, stagingRow{label: fmt.Sprintf("[unstaged %c] %s", status, displayPath(file.Path)), file: file, action: "stage"})
		}
	}
	return rows
}

func displayPath(path string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return '�'
		}
		return r
	}, path)
}

func pickStaging(repo repositories.StagingRepository) (bool, error) {
	screen, err := tcell.NewScreen()
	if err != nil {
		return false, fmt.Errorf("could not open file selector: %w", err)
	}
	if err := screen.Init(); err != nil {
		return false, fmt.Errorf("could not open file selector: %w", err)
	}
	defer screen.Fini()
	return pickStagingOnScreen(screen, repo)
}

func pickStagingOnScreen(screen tcell.Screen, repo repositories.StagingRepository) (bool, error) {
	cursor, scroll := 0, 0
	for {
		files, err := repo.ListChanges()
		if err != nil {
			return false, fmt.Errorf("could not list Git changes: %w", err)
		}
		rows := stagingRows(files)
		if cursor > len(rows) {
			cursor = len(rows)
		}
		width, height := screen.Size()
		if height < 7 || width < 20 {
			return false, fmt.Errorf("terminal is too small for file selection; resize to at least 20 columns and 7 rows")
		}
		viewport := height - 5
		if cursor < scroll {
			scroll = cursor
		}
		if cursor >= scroll+viewport {
			scroll = cursor - viewport + 1
		}
		screen.Clear()
		base := tcell.StyleDefault
		putLine(screen, 0, 0, "Select files · staged and unstaged", width, base.Foreground(tcell.ColorAqua).Bold(true))
		putLine(screen, 0, 1, "↑/↓ move · Space toggle · Enter continue · Esc cancel", width, base)
		putLine(screen, 0, 2, "Changes apply immediately; cancel keeps the current staging.", width, base.Foreground(tcell.ColorGray))
		for i := scroll; i < len(rows) && i < scroll+viewport; i++ {
			style, prefix := base, "  "
			if rows[i].action == "unstage" || rows[i].action == "staged-header" {
				style = base.Foreground(tcell.ColorGreen)
			}
			if i == cursor {
				style, prefix = base.Reverse(true), "> "
			}
			putLine(screen, 0, 3+i-scroll, prefix+rows[i].label, width, style)
		}
		if len(rows) > viewport {
			putLine(screen, 0, height-2, fmt.Sprintf("  %d–%d / %d · scroll with arrows", scroll+1, min(scroll+viewport, len(rows)), len(rows)), width, base.Foreground(tcell.ColorGray))
		}
		style, prefix := base, "  "
		if cursor == len(rows) {
			style, prefix = base.Reverse(true), "> "
		}
		putLine(screen, 0, height-1, prefix+"Continue", width, style)
		screen.Show()
		switch event := screen.PollEvent().(type) {
		case *tcell.EventResize:
			screen.Sync()
		case *tcell.EventKey:
			switch event.Key() {
			case tcell.KeyEsc, tcell.KeyCtrlC:
				return false, nil
			case tcell.KeyUp:
				if cursor > 0 {
					cursor--
				}
			case tcell.KeyDown:
				if cursor < len(rows) {
					cursor++
				}
			case tcell.KeyEnter:
				return true, nil
			case tcell.KeyRune:
				if event.Rune() != ' ' {
					break
				}
				if cursor == len(rows) {
					return true, nil
				}
				row := rows[cursor]
				switch row.action {
				case "all":
					err = repo.StageAll()
				case "unstage":
					err = repo.UnstageFile(row.file)
				case "stage":
					paths := []string{row.file.Path}
					if row.file.OriginalPath != "" {
						paths = append(paths, row.file.OriginalPath)
					}
					err = repo.StageFiles(paths)
				}
				if err != nil {
					return false, fmt.Errorf("could not update staging: %w", err)
				}
			}
		}
	}
}
