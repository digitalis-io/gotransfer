package ui

import (
	"fmt"
	"strings"

	"bitbucket.org/digitalisio/gotransfer/internal/fs"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// SortField identifies the column used to sort pane entries.
type SortField int

const (
	// SortName sorts entries alphabetically by name.
	SortName SortField = iota
	// SortSize sorts entries by file size.
	SortSize
	// SortTime sorts entries by modification time.
	SortTime
)

// PaneModel holds all state for one half of the dual-pane file browser.
type PaneModel struct {
	entries  []fs.FileEntry
	cursor   int
	cwd      string
	filter   string    //nolint:unused
	sortBy   SortField //nolint:unused
	sortDesc bool      //nolint:unused
	isRemote bool
	width    int
	height   int
}

// SetEntries replaces the current entry list and clamps the cursor into range.
func (p *PaneModel) SetEntries(entries []fs.FileEntry) {
	p.entries = entries
	p.clampCursor()
}

// MoveCursor moves the cursor by delta, clamped to [0, len(entries)-1].
func (p *PaneModel) MoveCursor(delta int) {
	p.cursor += delta
	p.clampCursor()
}

// SelectedEntry returns the entry at the current cursor position.
// Returns false when the entry list is empty.
func (p *PaneModel) SelectedEntry() (fs.FileEntry, bool) {
	if len(p.entries) == 0 {
		return fs.FileEntry{}, false
	}
	return p.entries[p.cursor], true
}

// Render returns the full pane as a styled string.
// active controls whether the active or inactive border style is used.
func (p *PaneModel) Render(active bool) string {
	if p.width < 4 || p.height < 4 {
		// Too small for a border box — return exact-dimension empty block
		line := strings.Repeat(" ", max(p.width, 0))
		lines := make([]string, max(p.height, 0))
		for i := range lines {
			lines[i] = line
		}
		return strings.Join(lines, "\n")
	}
	// border consumes 2 columns and 2 rows
	innerW := p.width - 2
	innerH := p.height - 2
	if innerW < 1 {
		innerW = 1
	}
	if innerH < 1 {
		innerH = 1
	}

	var lines []string

	// Header: label + cwd
	label := "LOCAL"
	if p.isRemote {
		label = "REMOTE"
	}
	header := label + "  " + p.cwd
	header = ansi.Truncate(header, innerW, "…")
	lines = append(lines, styleHeader.Width(innerW).Render(header))

	// Entry rows: innerH minus header row and footer row
	rowsH := innerH - 2
	if rowsH < 0 {
		rowsH = 0
	}

	// Scroll window centred on cursor
	start := 0
	if rowsH > 0 && len(p.entries) > rowsH {
		start = p.cursor - rowsH/2
		if start < 0 {
			start = 0
		}
		if start+rowsH > len(p.entries) {
			start = len(p.entries) - rowsH
		}
	}

	for i := 0; i < rowsH; i++ {
		idx := start + i
		if idx >= len(p.entries) {
			lines = append(lines, strings.Repeat(" ", innerW))
			continue
		}

		e := p.entries[idx]

		// Right-hand column: size (and mode for remote panes)
		var right string
		if e.IsDir {
			right = fmt.Sprintf("%7s", "<DIR>")
		} else {
			right = fmt.Sprintf("%7s", fs.FormatSize(e.Size))
		}
		if p.isRemote {
			right += " " + fs.FormatMode(e.Mode)
		}
		rightW := lipgloss.Width(right)

		// Name gets the rest, with 1 space separator
		nameW := innerW - rightW - 1
		if nameW < 0 {
			nameW = 0
		}

		// Build row text at exactly innerW visual columns
		var row string
		if nameW > 0 {
			name := ansi.Truncate(e.Name, nameW, "…")
			if w := lipgloss.Width(name); w < nameW {
				name += strings.Repeat(" ", nameW-w)
			}
			row = name + " " + right
		} else {
			row = ansi.Truncate(right, innerW, "")
		}
		// Hard-enforce innerW: truncate then pad
		row = ansi.Truncate(row, innerW, "")
		if w := lipgloss.Width(row); w < innerW {
			row += strings.Repeat(" ", innerW-w)
		}

		// Build entry style (foreground/background only; width is already exact)
		rowStyle := lipgloss.NewStyle()
		switch {
		case e.IsLink:
			rowStyle = rowStyle.Foreground(colLink)
		case e.IsDir:
			rowStyle = rowStyle.Foreground(colDir)
		case e.Mode&0o111 != 0:
			rowStyle = rowStyle.Foreground(colExec)
		}
		if idx == p.cursor {
			rowStyle = rowStyle.Background(colSel)
		}

		lines = append(lines, rowStyle.Render(row))
	}

	// Footer: item count
	footer := fmt.Sprintf("%d items", len(p.entries))
	footer = lipgloss.NewStyle().Foreground(colMuted).Width(innerW).Render(footer)
	lines = append(lines, footer)

	content := strings.Join(lines, "\n")

	var borderStyle lipgloss.Style
	if active {
		borderStyle = styleActiveBorder
	} else {
		borderStyle = styleInactiveBorder
	}

	return borderStyle.Width(innerW).Height(innerH).Render(content)
}

func (p *PaneModel) clampCursor() {
	if len(p.entries) == 0 {
		p.cursor = 0
		return
	}
	if p.cursor < 0 {
		p.cursor = 0
	}
	if p.cursor >= len(p.entries) {
		p.cursor = len(p.entries) - 1
	}
}
