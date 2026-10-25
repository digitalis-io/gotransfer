package ui

import (
	"context"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/lipgloss"

	"bitbucket.org/digitalisio/gotransfer/internal/fs"
)

// appMode is the current UI mode.
type appMode int

const (
	modeBrowse   appMode = iota
	modeConfirm          //nolint:deadcode,unused
	modeProgress         //nolint:deadcode,unused
	modeError            //nolint:deadcode,unused
	modeRename           //nolint:deadcode,unused
	modeFilter           //nolint:deadcode,unused
	modeMkdir            //nolint:deadcode,unused
)

// entriesLoadedMsg carries the result of a directory listing.
type entriesLoadedMsg struct {
	side    int
	path    string
	entries []fs.FileEntry
	err     error
}

// errorMsg carries an error to display in the TUI.
type errorMsg struct{ err error }

// Model is the root bubbletea model for the dual-pane TUI.
type Model struct {
	left, right   PaneModel
	activePane    int
	mode          appMode
	lastErr       error
	width, height int
	leftBrowser   fs.Browser
	rightBrowser  fs.Browser
}

// NewModel creates a Model wired to the given browsers and starting paths.
func NewModel(local, remote fs.Browser, localCwd, remoteCwd string) Model {
	return Model{
		left:         PaneModel{cwd: localCwd, sideIdx: 0},
		right:        PaneModel{cwd: remoteCwd, isRemote: true, sideIdx: 1},
		leftBrowser:  local,
		rightBrowser: remote,
	}
}

// Init starts entry loading for both panes.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		loadEntriesCmd(m.leftBrowser, m.left.cwd, 0),
		loadEntriesCmd(m.rightBrowser, m.right.cwd, 1),
	)
}

// Update handles all incoming messages.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		paneW := (m.width - 1) / 2
		paneH := m.height - 2
		m.left.width = paneW
		m.left.height = paneH
		m.right.width = paneW
		m.right.height = paneH

	case entriesLoadedMsg:
		if msg.err != nil {
			m.mode = modeError
			m.lastErr = msg.err
			return m, nil
		}
		if msg.side == 0 {
			if msg.path != m.left.cwd {
				return m, nil // stale response
			}
			m.left.SetEntries(msg.entries)
		} else {
			if msg.path != m.right.cwd {
				return m, nil // stale response
			}
			m.right.SetEntries(msg.entries)
		}

	case errorMsg:
		m.mode = modeError
		m.lastErr = msg.err

	case tea.KeyPressMsg:
		return m.handleBrowseKey(msg)

	case tea.InterruptMsg:
		return m, tea.Quit
	}
	return m, nil
}

func (m Model) handleBrowseKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.mode == modeError {
		if msg.Code == tea.KeyEsc {
			m.mode = modeBrowse
			m.lastErr = nil
		}
		return m, nil
	}

	if m.mode == modeFilter {
		switch {
		case msg.Code == tea.KeyEsc, msg.Code == tea.KeyEnter:
			m.mode = modeBrowse
		// future: handle rune input here in Task 18
		}
		return m, nil
	}

	switch msg.Code {
	case tea.KeyTab:
		if m.activePane == 0 {
			m.activePane = 1
		} else {
			m.activePane = 0
		}

	case tea.KeyUp, 'k':
		if m.activePane == 0 {
			m.left.MoveCursor(-1)
		} else {
			m.right.MoveCursor(-1)
		}

	case tea.KeyDown, 'j':
		if m.activePane == 0 {
			m.left.MoveCursor(1)
		} else {
			m.right.MoveCursor(1)
		}

	case tea.KeyEnter:
		if m.activePane == 0 {
			return m, m.left.EnterDir(m.leftBrowser)
		}
		return m, m.right.EnterDir(m.rightBrowser)

	case tea.KeyBackspace, '-':
		if m.activePane == 0 {
			return m, m.left.GoUp(m.leftBrowser)
		}
		return m, m.right.GoUp(m.rightBrowser)

	case '/':
		m.mode = modeFilter

	case 's':
		if m.activePane == 0 {
			m.left.CycleSort()
		} else {
			m.right.CycleSort()
		}

	case 'S':
		if m.activePane == 0 {
			m.left.ToggleSortDir()
		} else {
			m.right.ToggleSortDir()
		}

	case tea.KeyEsc, 'q':
		return m, tea.Quit
	}
	return m, nil
}

// View renders the full TUI.
func (m Model) View() tea.View {
	v := tea.NewView(m.render())
	v.AltScreen = true
	return v
}

func (m Model) render() string {
	panes := lipgloss.JoinHorizontal(
		lipgloss.Top,
		m.left.Render(m.activePane == 0),
		m.right.Render(m.activePane == 1),
	)
	footer := styleStatusBar.Render(
		"  [Tab] switch pane  [j/k] navigate  [q] quit",
	)
	var b strings.Builder
	b.WriteString(panes)
	b.WriteString("\n")
	b.WriteString(footer)
	content := b.String()

	if m.mode == modeError && m.lastErr != nil {
		popup := fmt.Sprintf(
			"┌─ Error ────────────────────┐\n│ %-28s │\n└────────────────────────────┘",
			truncateStr(m.lastErr.Error(), 28),
		)
		content = overlayCenter(content, popup)
	}

	return content
}

// truncateStr returns s truncated to at most n runes.
func truncateStr(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n-1]) + "…"
}

func loadEntriesCmd(b fs.Browser, path string, side int) tea.Cmd {
	return func() tea.Msg {
		entries, err := b.List(context.Background(), path)
		return entriesLoadedMsg{side: side, path: path, entries: entries, err: err}
	}
}
