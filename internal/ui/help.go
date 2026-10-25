package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// renderHelp returns a centred help overlay string for the given terminal dimensions.
func renderHelp(width, _ int) string {
	innerW := 38
	if width-6 < innerW {
		innerW = width - 6
		if innerW < 10 {
			innerW = 10
		}
	}

	// helpRow builds a single │ text │ line using display-column-aware padding.
	helpRow := func(s string) string {
		s = truncateStr(s, innerW)
		w := lipgloss.Width(s)
		pad := innerW - w
		if pad < 0 {
			pad = 0
		}
		return "│ " + s + strings.Repeat(" ", pad) + " │"
	}

	sections := []struct {
		heading string
		rows    [][2]string
	}{
		{"Navigation", [][2]string{
			{"Tab", "Switch pane"},
			{"j/↓", "Move down"},
			{"k/↑", "Move up"},
			{"Enter", "Enter directory"},
			{"-/Backspace", "Go up"},
		}},
		{"Selection & Transfer", [][2]string{
			{"Space", "Toggle select"},
			{"a", "Select all"},
			{"A", "Deselect all"},
			{"c", "Copy/transfer"},
			{"y", "Confirm"},
		}},
		{"Filter & Sort", [][2]string{
			{"/", "Filter"},
			{"s", "Cycle sort"},
			{"S", "Toggle sort direction"},
		}},
		{"File Operations", [][2]string{
			{"d", "New directory"},
			{"r", "Rename"},
			{"Delete", "Delete"},
			{"n", "Cancel/No"},
		}},
		{"Other", [][2]string{
			{"?", "Help"},
			{"Esc", "Cancel / quit"},
			{"q", "Quit"},
		}},
	}

	var lines []string
	lines = append(lines, fmt.Sprintf("┌─ Help %s─┐", strings.Repeat("─", innerW-6)))
	for _, sec := range sections {
		lines = append(lines, helpRow(sec.heading))
		for _, row := range sec.rows {
			label := fmt.Sprintf("  %-12s %s", row[0], row[1])
			lines = append(lines, helpRow(label))
		}
	}
	lines = append(lines, helpRow("Any key to close"))
	lines = append(lines, fmt.Sprintf("└%s┘", strings.Repeat("─", innerW+2)))

	return strings.Join(lines, "\n")
}
