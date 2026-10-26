# gotransfer — Design Document
_2026-10-25_

## Intent & Why

A beautiful, keyboard-driven dual-pane SFTP file manager for the terminal.
Fills the gap between clunky `scp` one-liners and heavy GUI clients like Cyberduck.
Built with the same Charm stack used in `cloudflare-dns-tui` so it integrates
naturally into the developer terminal workflow.

## Goals

- Dual-pane TUI: local filesystem left, remote SFTP right
- CLI invocation: `gotransfer user@host` (flags: `-p port`, `-i keyfile`)
- Full file operations: transfer, rename, delete, mkdir, permissions
- Multi-select with visual selection state
- Transfer progress overlay with speed/ETA
- Beautiful: proper borders, colour hierarchy, adaptive light/dark

## Non-Goals

- FTP, WebDAV, S3, GCS, Kubernetes, SCP (SFTP only for now)
- GUI / mouse-only interaction
- Built-in SSH tunnelling / jump hosts (future)
- File preview / diff

## Anti-Goals

- Anything that makes it feel like a Midnight Commander clone
- Dependency on Bubbles components — keep rendering hand-rolled (dns-tui precedent)
- Electron / web frontend

## Constraints

- Same Charm stack as dns-tui: `charm.land/bubbletea/v2`, `lipgloss v1.1+`
- Go 1.23+
- Single binary, zero runtime deps other than the terminal

---

## Architecture

### Directory layout

```
cmd/gotransfer/main.go          entry point + CLI arg parsing
internal/
  ui/
    model.go                    root Model + Update + View
    pane.go                     PaneModel (shared by local & remote)
    styles.go                   all lipgloss styles + colour tokens
    keys.go                     key binding constants
    overlay.go                  dialog/confirm/progress compositing
  fs/
    local.go                    local filesystem operations
    remote.go                   SFTP client wrapper (pkg/sftp)
    entry.go                    FileEntry type (shared)
```

### Dependencies

```
charm.land/bubbletea/v2
github.com/charmbracelet/lipgloss v1.1+
github.com/pkg/sftp
golang.org/x/crypto/ssh
```

---

## CLI Usage

```
gotransfer user@host
gotransfer user@host -p 2222
gotransfer user@host -i ~/.ssh/id_ed25519
```

Password auth: if no `-i` given and no key found via ssh-agent, prompt in terminal
before TUI launches (not inside the TUI). Unknown host: prompt in terminal, same pattern.

---

## App Modes (appMode enum)

| Mode | Description |
|------|-------------|
| `modeBrowse` | Main dual-pane browser (initial mode) |
| `modeRename` | Inline rename input on active entry |
| `modeFilter` | `/`-triggered filter input |
| `modeMkdir` | New directory name input |
| `modeConfirm` | Delete/overwrite confirmation dialog |
| `modeProgress` | Non-blocking transfer progress overlay |
| `modeError` | Error overlay (dismissible) |

---

## PaneModel

Both panes share the same struct; the active pane receives key events.

```go
type PaneModel struct {
    entries   []fs.FileEntry
    cursor    int
    selected  map[int]bool   // multi-select
    cwd       string
    sortBy    SortField      // name | size | mtime
    sortDesc  bool
    filter    string
    isRemote  bool
}
```

---

## FileEntry

```go
type FileEntry struct {
    Name    string
    Size    int64
    ModTime time.Time
    Mode    fs.FileMode
    IsDir   bool
    IsLink  bool
}
```

---

## Key Bindings

| Key | Action |
|-----|--------|
| `j` / `↓` | move cursor down |
| `k` / `↑` | move cursor up |
| `Enter` | enter directory |
| `-` / `Backspace` | go up one level |
| `Tab` | switch active pane |
| `Space` | toggle selection on entry |
| `a` | select all |
| `A` | deselect all |
| `F5` / `c` | copy / transfer to other pane |
| `F6` / `r` | rename |
| `F7` / `d` | mkdir |
| `F8` / `Delete` | delete (with confirm) |
| `/` | filter entries |
| `Esc` | clear filter / dismiss overlay / back |
| `s` | cycle sort field |
| `S` | toggle sort direction |
| `q` | quit |

---

## Visual Layout

```
┌─────────────────────────────────────────────────────────────────────┐
│  gotransfer  v0.1.0            [user@host:/remote/path]             │
├──────────────────────────┬──────────────────────────────────────────┤
│ LOCAL  ~/projects/foo    │ REMOTE  /home/user/deploy                │
│ ──────────────────────── │ ──────────────────────────────────────── │
│ ▸ src/                   │   config/                                │
│   main.go        4.2 KB  │   logs/                                  │
│ ► Makefile       1.1 KB  │ ► deploy.sh          2.3 KB  rwxr-xr-x  │
│   README.md        892 B │   docker-compose.yml   980 B  rw-r--r--  │
│   ...                    │   ...                                    │
│                          │                                          │
│ ──────────────────────── │ ──────────────────────────────────────── │
│ 12 items, 2 selected     │ 8 items, 0 selected                      │
├──────────────────────────┴──────────────────────────────────────────┤
│ Tab:switch  Space:select  c:copy  r:rename  d:mkdir  Del:delete  q:quit │
└─────────────────────────────────────────────────────────────────────┘
```

Active pane gets a bright border. Inactive pane uses dim border.
Directories use a distinct colour (blue). Executables (magenta). Symlinks (cyan).
Selected entries get a highlight background row.

Transfer progress appears as a centred overlay:

```
┌──────────────────────────────┐
│  Transferring 3 files        │
│  deploy.sh → /home/user/     │
│  ████████████░░░░░░  62%     │
│  1.2 MB/s  ETA 4s            │
│                  [Esc: abort]│
└──────────────────────────────┘
```

---

## Transfer Logic

- Runs in a goroutine; sends `tea.Msg` updates via `tea.Cmd`
- Multi-file: queue of entries, progress tracks current + overall
- Conflict resolution: prompt overwrite/skip/rename per file
- Direction: always from active pane to inactive pane

---

## Colour Tokens (styles.go)

```go
var (
    colBase    = lipgloss.AdaptiveColor{Light: "#1a1a2e", Dark: "#cdd6f4"}
    colMuted   = lipgloss.AdaptiveColor{Light: "#6c7086", Dark: "#6c7086"}
    colAccent  = lipgloss.AdaptiveColor{Light: "#1e66f5", Dark: "#89b4fa"}   // active pane border, cursor
    colDir     = lipgloss.AdaptiveColor{Light: "#1e66f5", Dark: "#89dceb"}   // directories
    colExec    = lipgloss.AdaptiveColor{Light: "#e64553", Dark: "#f38ba8"}   // executables
    colLink    = lipgloss.AdaptiveColor{Light: "#8839ef", Dark: "#cba6f7"}   // symlinks
    colSel     = lipgloss.AdaptiveColor{Light: "#df8e1d", Dark: "#f9e2af"}   // selected highlight
    colBorder  = lipgloss.AdaptiveColor{Light: "#bcc0cc", Dark: "#313244"}   // inactive border
    colProgress= lipgloss.AdaptiveColor{Light: "#40a02b", Dark: "#a6e3a1"}   // progress bar fill
)
```

---

## Builder Calls

- Overlay compositing: same `overlayCenter` ANSI-splice approach from dns-tui.
- SFTP reads: async via `tea.Cmd`, spinner shown while loading.
- Pane scrolling: virtual window over `entries` slice, `cursor` clamped to viewport.
- File size formatting: human-readable (KB/MB/GB).
- SSH known-hosts: honour `~/.ssh/known_hosts`; prompt on unknown host.

---

## Phased Delivery

| Phase | Scope |
|-------|-------|
| 1 | CLI arg parsing, SSH/SFTP connect, dual-pane listing (local + remote) |
| 2 | Navigation: enter/up dir, sort, filter, active pane switching |
| 3 | Transfer (upload/download), progress overlay |
| 4 | Multi-select, bulk transfer |
| 5 | Rename, mkdir, delete (with confirm), permissions display |
| 6 | Polish: keybinding help overlay, error handling, edge cases |
