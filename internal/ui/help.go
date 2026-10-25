package ui

import (
	"fmt"
	"strings"
)

// renderHelp returns a centred help overlay string for the given terminal dimensions.
func renderHelp(_, _ int) string {
	const innerW = 38

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
			{"q", "Quit"},
		}},
	}

	var lines []string
	lines = append(lines, fmt.Sprintf("┌─ Help %s─┐", strings.Repeat("─", innerW-6)))
	for _, sec := range sections {
		lines = append(lines, fmt.Sprintf("│ %-*s │", innerW, truncateStr(sec.heading, innerW)))
		for _, row := range sec.rows {
			label := fmt.Sprintf("  %-12s %s", row[0], row[1])
			lines = append(lines, fmt.Sprintf("│ %-*s │", innerW, truncateStr(label, innerW)))
		}
	}
	lines = append(lines, fmt.Sprintf("│ %-*s │", innerW, truncateStr("Any key to close", innerW)))
	lines = append(lines, fmt.Sprintf("└%s┘", strings.Repeat("─", innerW+2)))

	return strings.Join(lines, "\n")
}
