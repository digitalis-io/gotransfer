package ui

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"

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
	filter   string
	sortBy   SortField
	sortDesc bool
	isRemote bool
	sideIdx  int
	width    int
	height   int
	selected map[string]bool
}

// SetEntries replaces the current entry list and clamps the cursor into range.
func (p *PaneModel) SetEntries(entries []fs.FileEntry) {
	p.entries = entries
	p.clampCursor()
}

// visible returns the filtered and sorted view of entries.
func (p *PaneModel) visible() []fs.FileEntry {
	entries := p.entries
	if p.filter != "" {
		q := strings.ToLower(p.filter)
		var filtered []fs.FileEntry
		for _, e := range entries {
			if strings.Contains(strings.ToLower(e.Name), q) {
				filtered = append(filtered, e)
			}
		}
		entries = filtered
	}
	sorted := make([]fs.FileEntry, len(entries))
	copy(sorted, entries)
	sort.Slice(sorted, func(i, j int) bool {
		a, b := sorted[i], sorted[j]
		if a.IsDir != b.IsDir {
			return a.IsDir
		}
		switch p.sortBy {
		case SortSize:
			if a.Size != b.Size {
				if p.sortDesc {
					return a.Size > b.Size
				}
				return a.Size < b.Size
			}
		case SortTime:
			if !a.ModTime.Equal(b.ModTime) {
				if p.sortDesc {
					return a.ModTime.After(b.ModTime)
				}
				return a.ModTime.Before(b.ModTime)
			}
		default: // SortName
			an, bn := strings.ToLower(a.Name), strings.ToLower(b.Name)
			if an != bn {
				if p.sortDesc {
					return an > bn
				}
				return an < bn
			}
		}
		// tie-break by name to ensure stable strict ordering
		return strings.ToLower(a.Name) < strings.ToLower(b.Name)
	})
	return sorted
}

// MoveCursor moves the cursor by delta, clamped to [0, len(visible())-1].
func (p *PaneModel) MoveCursor(delta int) {
	p.cursor += delta
	p.clampCursor()
}

// SelectedEntry returns the entry at the current cursor position within the visible list.
// Returns false when the visible list is empty.
func (p *PaneModel) SelectedEntry() (fs.FileEntry, bool) {
	vis := p.visible()
	if len(vis) == 0 {
		return fs.FileEntry{}, false
	}
	if p.cursor >= len(vis) {
		return vis[len(vis)-1], true
	}
	return vis[p.cursor], true
}

// ToggleSelect toggles the selection state of the entry at the cursor, then advances the cursor.
func (p *PaneModel) ToggleSelect() {
	vis := p.visible()
	if len(vis) == 0 {
		return
	}
	e := vis[p.cursor]
	if p.selected[e.Name] {
		delete(p.selected, e.Name)
	} else {
		if p.selected == nil {
			p.selected = make(map[string]bool)
		}
		p.selected[e.Name] = true
	}
	p.MoveCursor(1)
}

// SelectAll marks every visible entry as selected.
func (p *PaneModel) SelectAll() {
	p.selected = make(map[string]bool)
	for _, e := range p.visible() {
		p.selected[e.Name] = true
	}
}

// DeselectAll clears all selections.
func (p *PaneModel) DeselectAll() {
	p.selected = nil
}

// SelectedEntries returns the selected entries in visible order.
// When nothing is explicitly selected it falls back to the single entry at the cursor.
func (p *PaneModel) SelectedEntries() []fs.FileEntry {
	if len(p.selected) == 0 {
		if e, ok := p.SelectedEntry(); ok {
			return []fs.FileEntry{e}
		}
		return nil
	}
	var out []fs.FileEntry
	for _, e := range p.visible() {
		if p.selected[e.Name] {
			out = append(out, e)
		}
	}
	return out
}

// SetFilter sets the active filter string and clamps the cursor into the new visible range.
func (p *PaneModel) SetFilter(q string) {
	p.filter = q
	p.clampCursor()
}

// CycleSort advances the sort field (Name→Size→Time→Name) and resets the cursor.
func (p *PaneModel) CycleSort() {
	p.sortBy = (p.sortBy + 1) % 3
	p.cursor = 0
}

// ToggleSortDir reverses the current sort direction.
func (p *PaneModel) ToggleSortDir() {
	p.sortDesc = !p.sortDesc
	p.cursor = 0
}

// EnterDir navigates into the selected directory and returns a load command.
// Returns nil if the selected entry is not a directory.
func (p *PaneModel) EnterDir(b fs.Browser) tea.Cmd {
	e, ok := p.SelectedEntry()
	if !ok || !e.IsDir {
		return nil
	}
	newPath := filepath.Join(p.cwd, e.Name)
	p.cwd = newPath
	p.cursor = 0
	p.filter = ""
	p.selected = nil
	return loadEntriesCmd(b, newPath, p.sideIdx)
}

// GoUp navigates to the parent directory and returns a load command.
// Returns nil if already at the root.
func (p *PaneModel) GoUp(b fs.Browser) tea.Cmd {
	parent := filepath.Dir(p.cwd)
	if parent == p.cwd {
		return nil
	}
	p.cwd = parent
	p.cursor = 0
	p.filter = ""
	p.selected = nil
	return loadEntriesCmd(b, parent, p.sideIdx)
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
	if p.width < 20 {
		msg := "<too narrow>"
		r := []rune(msg)
		if p.width < len(r) {
			msg = string(r[:p.width])
		}
		line := msg + strings.Repeat(" ", p.width-len(msg))
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

	vis := p.visible()

	if len(vis) == 0 && rowsH > 0 {
		empty := "(empty)"
		row := empty + strings.Repeat(" ", innerW-len(empty))
		row = ansi.Truncate(row, innerW, "")
		lines = append(lines, lipgloss.NewStyle().Foreground(colMuted).Render(row))
		for i := 1; i < rowsH; i++ {
			lines = append(lines, strings.Repeat(" ", innerW))
		}
		// footer
		footer := fmt.Sprintf("%d items", len(p.visible()))
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

	// Scroll window centred on cursor
	start := 0
	if rowsH > 0 && len(vis) > rowsH {
		start = p.cursor - rowsH/2
		if start < 0 {
			start = 0
		}
		if start+rowsH > len(vis) {
			start = len(vis) - rowsH
		}
	}

	for i := 0; i < rowsH; i++ {
		idx := start + i
		if idx >= len(vis) {
			lines = append(lines, strings.Repeat(" ", innerW))
			continue
		}

		e := vis[idx]

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
		if p.selected[e.Name] || idx == p.cursor {
			rowStyle = rowStyle.Background(colSel)
		}

		lines = append(lines, rowStyle.Render(row))
	}

	// Footer: item count (shows filtered/total when a filter is active)
	var footer string
	if p.filter != "" {
		footer = fmt.Sprintf("%d/%d items", len(vis), len(p.entries))
	} else {
		footer = fmt.Sprintf("%d items", len(vis))
	}
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
	vis := p.visible()
	if len(vis) == 0 {
		p.cursor = 0
		return
	}
	if p.cursor < 0 {
		p.cursor = 0
	}
	if p.cursor >= len(vis) {
		p.cursor = len(vis) - 1
	}
}
