package ui

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

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

// transferItem holds the parameters for a queued bulk transfer.
type transferItem struct {
	src, dst string
	upload   bool
	total    int64
}

// statResultMsg carries the result of an async Stat check before a transfer.
type statResultMsg struct {
	src, dst string
	upload   bool
	total    int64
	exists   bool
}

// ProgressMsg carries a transfer progress update.
type ProgressMsg struct {
	filename string
	written  int64
	total    int64
	done     bool
	err      error
}

// Model is the root bubbletea model for the dual-pane TUI.
type Model struct {
	left, right   PaneModel
	activePane    int
	mode          appMode
	lastErr       error
	width, height int
	leftBrowser   fs.Browser
	rightBrowser  fs.Browser
	progress      progressModel
	transferCh    <-chan int64
	transferErrCh <-chan error
	destPane      int
	confirmMsg       string
	pendingSrc       string
	pendingDst       string
	pendingUpload    bool
	pendingTotal     int64
	pendingTransfers []transferItem
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

	case ProgressMsg:
		m.progress.written = msg.written
		m.progress.total = msg.total
		if msg.done {
			m.transferCh = nil
			m.transferErrCh = nil
			if msg.err != nil {
				m.mode = modeError
				m.lastErr = msg.err
				m.pendingTransfers = nil
				if m.destPane == 0 {
					return m, loadEntriesCmd(m.leftBrowser, m.left.cwd, 0)
				}
				return m, loadEntriesCmd(m.rightBrowser, m.right.cwd, 1)
			}
			// Dequeue next bulk transfer if any.
			if len(m.pendingTransfers) > 0 {
				next := m.pendingTransfers[0]
				m.pendingTransfers = m.pendingTransfers[1:]
				var cmd tea.Cmd
				m, cmd = startTransferWith(m, next.src, next.dst, next.upload, next.total)
				return m, cmd
			}
			m.mode = modeBrowse
			// Reload the destination pane captured at transfer start.
			if m.destPane == 0 {
				return m, loadEntriesCmd(m.leftBrowser, m.left.cwd, 0)
			}
			return m, loadEntriesCmd(m.rightBrowser, m.right.cwd, 1)
		}
		// Reschedule a tick to drain the next value from the channel.
		if m.transferCh != nil {
			return m, progressTickCmd(m.transferCh, m.transferErrCh, msg.filename, msg.total, msg.written)
		}

	case statResultMsg:
		if msg.exists {
			m.mode = modeConfirm
			m.confirmMsg = fmt.Sprintf("Overwrite %s?", filepath.Base(msg.dst))
			m.pendingSrc = msg.src
			m.pendingDst = msg.dst
			m.pendingUpload = msg.upload
			m.pendingTotal = msg.total
			return m, nil
		}
		var cmd tea.Cmd
		m, cmd = startTransferWith(m, msg.src, msg.dst, msg.upload, msg.total)
		return m, cmd

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

	if m.mode == modeProgress {
		// Transfer in progress — block all keys; only InterruptMsg (Ctrl+C) can quit.
		return m, nil
	}

	if m.mode == modeConfirm {
		switch msg.Code {
		case tea.KeyEnter, 'y':
			var cmd tea.Cmd
			m, cmd = startTransferWith(m, m.pendingSrc, m.pendingDst, m.pendingUpload, m.pendingTotal)
			return m, cmd
		case tea.KeyEsc, 'n':
			m.mode = modeBrowse
		}
		return m, nil
	}

	switch msg.Code {
	case tea.KeySpace:
		if m.activePane == 0 {
			m.left.ToggleSelect()
		} else {
			m.right.ToggleSelect()
		}

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

	case 'a':
		if m.activePane == 0 {
			m.left.SelectAll()
		} else {
			m.right.SelectAll()
		}

	case 'A':
		if m.activePane == 0 {
			m.left.DeselectAll()
		} else {
			m.right.DeselectAll()
		}

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

	case 'c':
		if _, ok := m.rightBrowser.(fs.RemoteUploader); !ok {
			return m, func() tea.Msg {
				return errorMsg{err: fmt.Errorf("right browser does not support transfers")}
			}
		}
		upload := m.activePane == 0
		var allEntries []fs.FileEntry
		if m.activePane == 0 {
			allEntries = m.left.SelectedEntries()
		} else {
			allEntries = m.right.SelectedEntries()
		}
		// Filter out directories.
		var entries []fs.FileEntry
		for _, e := range allEntries {
			if !e.IsDir {
				entries = append(entries, e)
			}
		}
		if len(entries) == 0 {
			return m, nil
		}
		buildPaths := func(e fs.FileEntry) (src, dst string, dstBrowser fs.Browser) {
			if upload {
				src = filepath.Join(m.left.cwd, e.Name)
				dst = m.right.cwd + "/" + e.Name
				dstBrowser = m.rightBrowser
			} else {
				src = m.right.cwd + "/" + e.Name
				dst = filepath.Join(m.left.cwd, e.Name)
				dstBrowser = m.leftBrowser
			}
			return src, dst, dstBrowser
		}
		if len(entries) == 1 {
			// Single: use stat check to prompt before overwrite.
			src, dst, dstBrowser := buildPaths(entries[0])
			return m, statCmd(dstBrowser, src, dst, upload, entries[0].Size)
		}
		// Bulk: skip overwrite check, queue all transfers and start first.
		items := make([]transferItem, len(entries))
		for i, e := range entries {
			src, dst, _ := buildPaths(e)
			items[i] = transferItem{src: src, dst: dst, upload: upload, total: e.Size}
		}
		m.pendingTransfers = items[1:]
		var cmd tea.Cmd
		m, cmd = startTransferWith(m, items[0].src, items[0].dst, items[0].upload, items[0].total)
		return m, cmd

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

	if m.mode == modeProgress {
		inner := m.progress.Render(m.width - 6)
		lines := strings.Split(inner, "\n")
		boxW := m.width - 4
		if boxW < 20 {
			boxW = 20
		}
		var sb strings.Builder
		// Top border: ┌ + "─ Transferring " (15) + repeat + "─┐" (2) = boxW+4 total.
		sb.WriteString(fmt.Sprintf("┌─ Transferring %s─┐\n", strings.Repeat("─", boxW-14)))
		for _, l := range lines {
			sb.WriteString(fmt.Sprintf("│ %-*s │\n", boxW, truncateStr(l, boxW)))
		}
		sb.WriteString(fmt.Sprintf("└%s┘", strings.Repeat("─", boxW+2)))
		content = overlayCenter(content, sb.String())
	}

	if m.mode == modeConfirm {
		popup := fmt.Sprintf(
			"┌─ Confirm ───────────────────┐\n│ %-27s │\n│  [Enter/y] Yes  [Esc/n] No  │\n└─────────────────────────────┘",
			truncateStr(m.confirmMsg, 27),
		)
		content = overlayCenter(content, popup)
	}

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

// statCmd dispatches an async Stat check on dst, returning a statResultMsg.
func statCmd(b fs.Browser, src, dst string, upload bool, total int64) tea.Cmd {
	return func() tea.Msg {
		_, err := b.Stat(context.Background(), dst)
		return statResultMsg{src: src, dst: dst, upload: upload, total: total, exists: err == nil}
	}
}

// startTransferWith launches the transfer goroutine and returns the updated Model and first tick command.
func startTransferWith(m Model, src, dst string, upload bool, total int64) (Model, tea.Cmd) {
	rb := m.rightBrowser.(fs.RemoteUploader)
	ch := make(chan int64, 64)
	errCh := make(chan error, 1)
	filename := filepath.Base(src)

	go func() {
		ctx := context.Background()
		var err error
		if upload {
			err = rb.Upload(ctx, src, dst, ch)
		} else {
			err = rb.Download(ctx, src, dst, ch)
		}
		errCh <- err
	}()

	m.transferCh = ch
	m.transferErrCh = errCh
	m.mode = modeProgress
	m.destPane = 1 - m.activePane
	m.progress = progressModel{
		filename: filename,
		total:    total,
		startAt:  time.Now(),
	}
	return m, progressTickCmd(ch, errCh, filename, total, 0)
}

// progressTickCmd reads one value from ch and returns a ProgressMsg.
// When the channel is closed it drains errCh and returns a done ProgressMsg.
func progressTickCmd(ch <-chan int64, errCh <-chan error, filename string, total, prev int64) tea.Cmd {
	return func() tea.Msg {
		n, ok := <-ch
		if !ok {
			err := <-errCh
			return ProgressMsg{filename: filename, written: prev, total: total, done: true, err: err}
		}
		return ProgressMsg{filename: filename, written: prev + n, total: total, done: false}
	}
}
