# gotransfer Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a beautiful dual-pane SFTP TUI file manager invoked as `gotransfer user@host`.

**Architecture:** Bubble Tea v2 root model with two `PaneModel` values (local left, remote right). All rendering is hand-rolled with lipgloss — no Bubbles components. SSH/SFTP connection established before the TUI launches; the TUI starts directly in browse mode.

**Tech Stack:** `charm.land/bubbletea/v2`, `github.com/charmbracelet/lipgloss v1.1+`, `github.com/pkg/sftp`, `golang.org/x/crypto/ssh`

**Spec:** `docs/design/2026-10-25-gotransfer-design.md`

## Global Constraints

- Go 1.23+; module path `bitbucket.org/digitalisio/gotransfer`
- No Bubbles components — all rendering hand-rolled
- `make test-race` must pass after every task
- Signed + signed-off commits: `git commit -S -s`
- No plaintext credentials in source or tests; integration tests read env vars
- Integration test env vars: `TEST_SSH_HOST`, `TEST_SSH_USER`, `TEST_SSH_PASS` — set in shell, never in source
- Test servers: set via `TEST_SSH_HOST`/`TEST_SSH_USER`/`TEST_SSH_PASS` env vars
- go-specialist agent reviews all changes at the end of each phase before the next begins

## Review Focus

- Empty directory listing (no entries): both panes must render without panic
- Remote path that does not exist: `remote.List` must return a typed error, not hang
- Terminal narrower than 40 cols: pane layout must not produce negative widths
- Transfer of a file already existing at destination: conflict must surface as `modeConfirm`, not silent overwrite
- SFTP session drop mid-transfer: `transferCmd` must send an error `tea.Msg`, not deadlock

---

## File Map

```
cmd/gotransfer/main.go          CLI parsing, SSH dial, launch tea.Program
internal/fs/entry.go            FileEntry type + helpers (FormatSize, FormatMode)
internal/fs/browser.go          Browser interface
internal/fs/local.go            LocalBrowser: list, stat, delete, rename, mkdir
internal/fs/remote.go           RemoteBrowser: SFTP client, same ops + Upload/Download
internal/ui/styles.go           lipgloss colour tokens + named styles
internal/ui/keys.go             key constant strings
internal/ui/overlay.go          overlayCenter(base, popup string) string
internal/ui/pane.go             PaneModel: entries, cursor, cwd, filter, sort, Render()
internal/ui/model.go            Model: left/right PaneModel, active side, appMode, Update, View
```

---

## Phase 1 — Scaffold, connect, list

### Task 1: go.mod + dependencies

**Files:**
- Create: `go.mod`
- Create: `go.sum` (generated)
- Modify: `Makefile` (add `go mod tidy` to `tidy` target if absent)

- [ ] **Step 1:** Init module

```bash
cd /Users/sergio.rua/BitBucket/digitalis/gotransfer
go mod init bitbucket.org/digitalisio/gotransfer
```

- [ ] **Step 2:** Add dependencies

```bash
go get charm.land/bubbletea/v2
go get github.com/charmbracelet/lipgloss@v1.1.1
go get github.com/pkg/sftp
go get golang.org/x/crypto/ssh
go mod tidy
```

- [ ] **Step 3:** Verify `go.mod` has module path `bitbucket.org/digitalisio/gotransfer` and `go 1.23`

- [ ] **Step 4:** Commit

```bash
git add go.mod go.sum
git commit -S -s -m "chore: initialise go module with charm/sftp dependencies"
```

---

### Task 2: FileEntry type + Browser interface

**Files:**
- Create: `internal/fs/entry.go`
- Create: `internal/fs/browser.go`
- Create: `internal/fs/entry_test.go`

**Interfaces:**
- Produces:
  - `fs.FileEntry{Name string, Size int64, ModTime time.Time, Mode os.FileMode, IsDir bool, IsLink bool}`
  - `fs.FormatSize(n int64) string` — human-readable: `892 B`, `4.2 KB`, `1.3 MB`, `2.1 GB`
  - `fs.FormatMode(m os.FileMode) string` — 10-char Unix string e.g. `drwxr-xr-x`
  - `fs.Browser` interface: `List(ctx, path) ([]FileEntry, error)`, `Stat(ctx, path) (FileEntry, error)`, `Delete(ctx, path) error`, `Rename(ctx, src, dst) error`, `Mkdir(ctx, path) error`

- [ ] **Step 1:** Write failing tests

```go
// internal/fs/entry_test.go
func TestFormatSize(t *testing.T) {
    cases := []struct{ n int64; want string }{
        {512, "512 B"}, {4300, "4.2 KB"}, {1363149, "1.3 MB"}, {2254857830, "2.1 GB"},
    }
    for _, c := range cases {
        assert.Equal(t, c.want, FormatSize(c.n))
    }
}

func TestFormatMode(t *testing.T) {
    assert.Equal(t, "drwxr-xr-x", FormatMode(os.ModeDir|0755))
    assert.Equal(t, "-rw-r--r--", FormatMode(0644))
}
```

- [ ] **Step 2:** Run — expect FAIL (package missing)

```bash
go test ./internal/fs/... 2>&1 | head -5
```

- [ ] **Step 3:** Implement `internal/fs/entry.go` with `FileEntry`, `FormatSize`, `FormatMode`

`FormatSize`: thresholds 1<<30 GB, 1<<20 MB, 1<<10 KB, else B; one decimal place for KB/MB/GB.
`FormatMode`: use `m.String()` from `os.FileMode` — it already returns the 10-char string.

- [ ] **Step 4:** Implement `internal/fs/browser.go` — `Browser` interface only, no implementation

- [ ] **Step 5:** Run tests — expect PASS

```bash
go test ./internal/fs/... -v -race
```

- [ ] **Step 6:** Commit

```bash
git add internal/fs/
git commit -S -s -m "feat: FileEntry type, FormatSize/FormatMode helpers, Browser interface"
```

---

### Task 3: LocalBrowser

**Files:**
- Create: `internal/fs/local.go`
- Create: `internal/fs/local_test.go`

**Interfaces:**
- Consumes: `fs.Browser`, `fs.FileEntry`
- Produces: `fs.NewLocalBrowser() Browser`

- [ ] **Step 1:** Write failing tests

```go
// internal/fs/local_test.go
func TestLocalBrowserList(t *testing.T) {
    tmp := t.TempDir()
    os.WriteFile(filepath.Join(tmp, "a.txt"), []byte("hi"), 0644)
    os.Mkdir(filepath.Join(tmp, "sub"), 0755)

    b := NewLocalBrowser()
    entries, err := b.List(context.Background(), tmp)
    require.NoError(t, err)
    names := make([]string, len(entries))
    for i, e := range entries { names[i] = e.Name }
    assert.Contains(t, names, "a.txt")
    assert.Contains(t, names, "sub")
}

func TestLocalBrowserListEmpty(t *testing.T) {
    tmp := t.TempDir()
    b := NewLocalBrowser()
    entries, err := b.List(context.Background(), tmp)
    require.NoError(t, err)
    assert.Empty(t, entries)
}

func TestLocalBrowserMkdirRenameDelete(t *testing.T) {
    tmp := t.TempDir()
    b := NewLocalBrowser()
    require.NoError(t, b.Mkdir(context.Background(), filepath.Join(tmp, "newdir")))
    require.NoError(t, b.Rename(context.Background(),
        filepath.Join(tmp, "newdir"), filepath.Join(tmp, "renamed")))
    require.NoError(t, b.Delete(context.Background(), filepath.Join(tmp, "renamed")))
    entries, _ := b.List(context.Background(), tmp)
    assert.Empty(t, entries)
}
```

- [ ] **Step 2:** Run — expect FAIL

```bash
go test ./internal/fs/... -run TestLocal -v 2>&1 | tail -5
```

- [ ] **Step 3:** Implement `internal/fs/local.go`

`List`: `os.ReadDir` → convert each `os.DirEntry` to `FileEntry`; dirs first, then files, both sorted by name. `Stat`: `os.Lstat`. `Delete`: `os.RemoveAll`. `Rename`: `os.Rename`. `Mkdir`: `os.MkdirAll`.

- [ ] **Step 4:** Run — expect PASS

```bash
go test ./internal/fs/... -v -race
```

- [ ] **Step 5:** Commit

```bash
git add internal/fs/local.go internal/fs/local_test.go
git commit -S -s -m "feat: LocalBrowser with list, stat, mkdir, rename, delete"
```

---

### Task 4: RemoteBrowser (SFTP)

**Files:**
- Create: `internal/fs/remote.go`
- Create: `internal/fs/remote_test.go`

**Interfaces:**
- Consumes: `fs.Browser`, `fs.FileEntry`
- Produces:
  - `fs.NewRemoteBrowser(client *sftp.Client) Browser`
  - `fs.RemoteBrowser.Upload(ctx context.Context, localPath, remotePath string) error`
  - `fs.RemoteBrowser.Download(ctx context.Context, remotePath, localPath string) error`

- [ ] **Step 1:** Write failing integration tests (skipped when env vars absent)

```go
// internal/fs/remote_test.go
func sftpClient(t *testing.T) *sftp.Client {
    t.Helper()
    host := os.Getenv("TEST_SSH_HOST")
    user := os.Getenv("TEST_SSH_USER")
    pass := os.Getenv("TEST_SSH_PASS")
    if host == "" {
        t.Skip("TEST_SSH_HOST not set")
    }
    config := &ssh.ClientConfig{
        User:            user,
        Auth:            []ssh.AuthMethod{ssh.Password(pass)},
        HostKeyCallback: ssh.InsecureIgnoreHostKey(),
    }
    conn, err := ssh.Dial("tcp", host+":22", config)
    require.NoError(t, err)
    t.Cleanup(func() { conn.Close() })
    client, err := sftp.NewClient(conn)
    require.NoError(t, err)
    t.Cleanup(func() { client.Close() })
    return client
}

func TestRemoteBrowserList(t *testing.T) {
    b := NewRemoteBrowser(sftpClient(t))
    entries, err := b.List(context.Background(), "/tmp")
    require.NoError(t, err)
    assert.NotNil(t, entries)
}

func TestRemoteBrowserListNonexistent(t *testing.T) {
    b := NewRemoteBrowser(sftpClient(t))
    _, err := b.List(context.Background(), "/nonexistent-path-gotransfer")
    assert.Error(t, err)
}

func TestRemoteBrowserUploadDownload(t *testing.T) {
    rb := NewRemoteBrowser(sftpClient(t)).(RemoteUploader)
    local := filepath.Join(t.TempDir(), "test.txt")
    require.NoError(t, os.WriteFile(local, []byte("gotransfer test"), 0644))
    remote := "/tmp/gotransfer-test-upload.txt"
    require.NoError(t, rb.Upload(context.Background(), local, remote))
    dst := filepath.Join(t.TempDir(), "downloaded.txt")
    require.NoError(t, rb.Download(context.Background(), remote, dst))
    got, _ := os.ReadFile(dst)
    assert.Equal(t, "gotransfer test", string(got))
}
```

`RemoteUploader` interface (defined in remote.go): `Browser` + `Upload` + `Download`.

- [ ] **Step 2:** Run — expect SKIP locally without env vars

```bash
go test ./internal/fs/... -run TestRemote -v 2>&1 | tail -5
```

- [ ] **Step 3:** Implement `internal/fs/remote.go`

`List`: `sftpClient.ReadDir(path)` → convert `os.FileInfo` to `FileEntry`; same dir-first sort as local. `Stat`: `sftpClient.Lstat`. `Delete`: `sftpClient.Remove` (file) or `sftpClient.RemoveDirectory` (dir). `Rename`: `sftpClient.Rename`. `Mkdir`: `sftpClient.MkdirAll`. `Upload`: open local, `sftpClient.Create(remote)`, `io.Copy`. `Download`: `sftpClient.Open(remote)`, create local, `io.Copy`.

- [ ] **Step 4:** Run integration tests against test server (set env vars in shell first)

```bash
go test ./internal/fs/... -run TestRemote -v -race
```

Expected: PASS

- [ ] **Step 5:** Commit

```bash
git add internal/fs/remote.go internal/fs/remote_test.go
git commit -S -s -m "feat: RemoteBrowser with SFTP list, stat, mkdir, rename, delete, upload, download"
```

---

### Task 5: Phase 1 go-specialist review

- [ ] **Step 1:** Dispatch `go-specialist` agent

Prompt: "Review Phase 1 files in gotransfer: `internal/fs/entry.go`, `internal/fs/browser.go`, `internal/fs/local.go`, `internal/fs/remote.go` and their tests. Check idiomatic Go, error handling, interface design, test coverage. Report findings; do not apply fixes."

- [ ] **Step 2:** Apply any blocking findings before Phase 2

---

## Phase 2 — TUI scaffold + dual-pane browse

### Task 6: Styles, keys, overlay

**Files:**
- Create: `internal/ui/styles.go`
- Create: `internal/ui/keys.go`
- Create: `internal/ui/overlay.go`
- Create: `internal/ui/overlay_test.go`

**Interfaces:**
- Produces (styles.go):
  - `colAccent`, `colDir`, `colExec`, `colLink`, `colSel`, `colBorder`, `colMuted`, `colProgress` — `lipgloss.AdaptiveColor` vars (exact hex values from spec)
  - `styleActiveBorder`, `styleInactiveBorder`, `styleStatusBar`, `styleHeader` — `lipgloss.Style`
- Produces (keys.go): string constants `KeyUp="up"`, `KeyDown="down"`, `KeyEnter="enter"`, `KeyTab="tab"`, `KeySpace=" "`, `KeyEsc="esc"`, `KeySlash="/"`, `KeyQ="q"`, `KeyC="c"`, `KeyR="r"`, `KeyD="d"`, `KeyA="a"`, `KeyDel="delete"`, `KeyBackspace="backspace"`, `KeyMinus="-"`, `KeyS="s"`, `KeyShiftS="S"`, `KeyShiftA="A"`
- Produces (overlay.go): `overlayCenter(base, popup string) string`

Colour tokens (from spec):
```
colAccent   Light:#1e66f5  Dark:#89b4fa
colDir      Light:#1e66f5  Dark:#89dceb
colExec     Light:#e64553  Dark:#f38ba8
colLink     Light:#8839ef  Dark:#cba6f7
colSel      Light:#df8e1d  Dark:#f9e2af
colBorder   Light:#bcc0cc  Dark:#313244
colMuted    Light:#6c7086  Dark:#6c7086
colProgress Light:#40a02b  Dark:#a6e3a1
```

- [ ] **Step 1:** Write failing test for `overlayCenter`

```go
// internal/ui/overlay_test.go
func TestOverlayCenterNoPanic(t *testing.T) {
    base := strings.Repeat("x", 80) + "\n" + strings.Repeat("x", 80)
    popup := "┌────┐\n│ hi │\n└────┘"
    result := overlayCenter(base, popup)
    assert.NotEmpty(t, result)
    assert.Contains(t, result, "hi")
}

func TestOverlayCenterNarrowTerminal(t *testing.T) {
    base := strings.Repeat("x", 20) + "\n" + strings.Repeat("x", 20)
    popup := "┌──────────────────────────────┐\n│ wider than base              │\n└──────────────────────────────┘"
    assert.NotPanics(t, func() { overlayCenter(base, popup) })
}
```

- [ ] **Step 2:** Run — expect FAIL

```bash
go test ./internal/ui/... -run TestOverlay -v 2>&1 | tail -5
```

- [ ] **Step 3:** Implement `internal/ui/overlay.go`

Port the `overlayCenter` + ANSI splice from `cloudflare-dns-tui`. Use `github.com/charmbracelet/x/ansi` for `StringWidth`, `Truncate`, `Cut`. Clamp popup position so it never starts at negative column/row.

- [ ] **Step 4:** Implement `internal/ui/styles.go` and `internal/ui/keys.go` (no tests needed)

- [ ] **Step 5:** Run all tests

```bash
go test ./... -v -race
```

- [ ] **Step 6:** Commit

```bash
git add internal/ui/
git commit -S -s -m "feat: TUI styles, key constants, overlay compositing"
```

---

### Task 7: PaneModel

**Files:**
- Create: `internal/ui/pane.go`
- Create: `internal/ui/pane_test.go`

**Interfaces:**
- Consumes: `fs.FileEntry`, `fs.FormatSize`, `fs.FormatMode`, all styles + keys
- Produces:
  - `PaneModel{entries []fs.FileEntry, cursor int, cwd string, filter string, sortBy SortField, sortDesc bool, isRemote bool, width int, height int}`
  - `SortField` int type: `SortName=0`, `SortSize=1`, `SortTime=2`
  - `(p *PaneModel) SetEntries(entries []fs.FileEntry)`
  - `(p *PaneModel) MoveCursor(delta int)`
  - `(p *PaneModel) SelectedEntry() (fs.FileEntry, bool)`
  - `(p *PaneModel) Render(active bool) string` — returns a string of exactly `p.height` lines, each `p.width` runes wide (ANSI-safe)

- [ ] **Step 1:** Write failing tests

```go
// internal/ui/pane_test.go
func TestPaneMoveCursor(t *testing.T) {
    p := PaneModel{width: 40, height: 10}
    entries := []fs.FileEntry{{Name: "a"}, {Name: "b"}, {Name: "c"}}
    p.SetEntries(entries)
    p.MoveCursor(1)
    assert.Equal(t, 1, p.cursor)
    p.MoveCursor(10) // clamp to last
    assert.Equal(t, 2, p.cursor)
    p.MoveCursor(-10) // clamp to 0
    assert.Equal(t, 0, p.cursor)
}

func TestPaneRenderEmpty(t *testing.T) {
    p := PaneModel{width: 40, height: 5}
    p.SetEntries(nil)
    out := p.Render(true)
    assert.NotEmpty(t, out)
    assert.NotPanics(t, func() { p.Render(false) })
}

func TestPaneSelectedEntry(t *testing.T) {
    p := PaneModel{width: 40, height: 10}
    p.SetEntries([]fs.FileEntry{{Name: "x.txt"}})
    e, ok := p.SelectedEntry()
    assert.True(t, ok)
    assert.Equal(t, "x.txt", e.Name)
}

func TestPaneRenderNarrow(t *testing.T) {
    p := PaneModel{width: 10, height: 5}
    p.SetEntries([]fs.FileEntry{{Name: "very-long-file-name.txt", Size: 1234}})
    assert.NotPanics(t, func() { p.Render(true) })
}
```

- [ ] **Step 2:** Run — expect FAIL

```bash
go test ./internal/ui/... -run TestPane -v 2>&1 | tail -5
```

- [ ] **Step 3:** Implement `internal/ui/pane.go`

`Render`: header line with `cwd` (truncated); separator; entry rows (scroll window = `height - 3` centred on cursor); footer with count. Active pane: `styleActiveBorder` box; inactive: `styleInactiveBorder`. Entry row: name left-aligned (ANSI-truncated) + size right-aligned + mode for remote. Dirs → `colDir`, symlinks → `colLink`, executables → `colExec`.

- [ ] **Step 4:** Run — expect PASS

```bash
go test ./internal/ui/... -v -race
```

- [ ] **Step 5:** Commit

```bash
git add internal/ui/pane.go internal/ui/pane_test.go
git commit -S -s -m "feat: PaneModel with render, cursor, sort, entry selection"
```

---

### Task 8: Root model + main entry point

**Files:**
- Create: `internal/ui/model.go`
- Create: `internal/ui/model_test.go`
- Create: `cmd/gotransfer/main.go`

**Interfaces:**
- Consumes: `PaneModel`, `fs.Browser`, overlay, styles, keys
- Produces:
  - `appMode` int: `modeBrowse=0`, `modeConfirm=1`, `modeProgress=2`, `modeError=3`, `modeRename=4`, `modeFilter=5`, `modeMkdir=6`
  - `NewModel(local fs.Browser, remote fs.Browser, localCwd, remoteCwd string) Model`
  - `(m Model) Init() tea.Cmd`
  - `(m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd)`
  - `(m Model) View() tea.View`
  - `entriesLoadedMsg{side int, entries []fs.FileEntry, err error}`

- [ ] **Step 1:** Write smoke test

```go
// internal/ui/model_test.go
func TestModelInit(t *testing.T) {
    lb := fs.NewLocalBrowser()
    m := NewModel(lb, lb, t.TempDir(), t.TempDir())
    m.width, m.height = 120, 40
    out := m.View()
    assert.NotEmpty(t, out.String())
}
```

- [ ] **Step 2:** Run — expect FAIL

- [ ] **Step 3:** Implement `internal/ui/model.go`

`Init`: `tea.Batch(loadEntriesCmd(leftBrowser, leftCwd, 0), loadEntriesCmd(rightBrowser, rightCwd, 1))`.
`Update`: handle `tea.WindowSizeMsg` (recompute 50/50 pane widths), `entriesLoadedMsg`, `errorMsg`, `tea.KeyMsg` dispatched to `handleBrowseKey`. `handleBrowseKey`: `Tab` switches active pane; `j/↓` `k/↑` → `MoveCursor`; `q`/`Esc` → `tea.Quit`.
`View`: `lipgloss.JoinHorizontal` of both pane renders + key hint footer; overlay composited on top when `mode != modeBrowse`.
`loadEntriesCmd`: runs `b.List` in a goroutine, sends `entriesLoadedMsg`.

- [ ] **Step 4:** Implement `cmd/gotransfer/main.go`

Parse `gotransfer user@host [-p port] [-i keyfile]`. Dial SSH before TUI. No `-i`: try ssh-agent (`ssh.PublicKeysCallback`), then prompt password with `golang.org/x/term`. Unknown host: print fingerprint + prompt y/n in terminal. On connect: `sftp.NewClient`, `NewRemoteBrowser`, launch `tea.NewProgram(model, tea.WithAltScreen())`.

- [ ] **Step 5:** Build and smoke-test

```bash
make build
./bin/gotransfer ${TEST_SSH_USER}@${TEST_SSH_HOST}
# dual pane visible; Tab switches; q quits
```

- [ ] **Step 6:** Run all tests

```bash
go test ./... -v -race
```

- [ ] **Step 7:** Commit

```bash
git add internal/ui/ cmd/
git commit -S -s -m "feat: root Bubble Tea model, main entry point, CLI arg parsing"
```

---

### Task 9: Phase 2 go-specialist review

- [ ] **Step 1:** Dispatch `go-specialist` agent

Prompt: "Review Phase 2 files in gotransfer: `internal/ui/styles.go`, `internal/ui/keys.go`, `internal/ui/overlay.go`, `internal/ui/pane.go`, `internal/ui/model.go`, `cmd/gotransfer/main.go` and tests. Check Bubble Tea v2 patterns, lipgloss usage, error handling, test coverage. Report findings; do not apply fixes."

- [ ] **Step 2:** Apply blocking findings before Phase 3

---

## Phase 3 — Navigation

### Task 10: Directory navigation + filter + sort

**Files:**
- Modify: `internal/ui/model.go`
- Modify: `internal/ui/pane.go`
- Modify: `internal/ui/pane_test.go`

**Interfaces:**
- Adds to PaneModel:
  - `(p *PaneModel) visible() []fs.FileEntry` — filtered + sorted view of entries
  - `(p *PaneModel) EnterDir(b fs.Browser) tea.Cmd`
  - `(p *PaneModel) GoUp(b fs.Browser) tea.Cmd`
  - `(p *PaneModel) SetFilter(q string)` — case-insensitive substring match on Name
  - `(p *PaneModel) CycleSort()` — SortName→SortSize→SortTime, cursor reset to 0
  - `(p *PaneModel) ToggleSortDir()`

- [ ] **Step 1:** Add failing tests

```go
func TestPaneFilter(t *testing.T) {
    p := PaneModel{width: 40, height: 10}
    p.SetEntries([]fs.FileEntry{{Name: "main.go"}, {Name: "README.md"}, {Name: "go.mod"}})
    p.SetFilter("go")
    for _, entry := range p.visible() {
        assert.NotEqual(t, "README.md", entry.Name)
    }
    assert.Len(t, p.visible(), 2)
}

func TestPaneSortBySize(t *testing.T) {
    p := PaneModel{width: 40, height: 10}
    p.SetEntries([]fs.FileEntry{{Name: "big", Size: 1000}, {Name: "small", Size: 1}})
    p.CycleSort() // SortSize ascending
    assert.Equal(t, "small", p.visible()[0].Name)
}
```

- [ ] **Step 2:** Run — expect FAIL

- [ ] **Step 3:** Implement `visible()` on PaneModel: applies filter then sort; `Render` and `MoveCursor` use it; cursor clamped to `len(visible())-1`

- [ ] **Step 4:** Implement `EnterDir`, `GoUp`, `SetFilter`, `CycleSort`, `ToggleSortDir`

- [ ] **Step 5:** Wire keys in `model.go handleBrowseKey`: `Enter`→`EnterDir`, `-`/`Backspace`→`GoUp`, `/`→`modeFilter`, `s`→`CycleSort`, `S`→`ToggleSortDir`

- [ ] **Step 6:** Run tests

```bash
go test ./... -v -race
```

- [ ] **Step 7:** Manual test on both panes against test server

- [ ] **Step 8:** Commit

```bash
git add internal/ui/
git commit -S -s -m "feat: directory navigation, filter, sort in dual panes"
```

---

### Task 11: Phase 3 go-specialist review

- [ ] **Step 1:** Dispatch `go-specialist` agent with navigation changes
- [ ] **Step 2:** Apply blocking findings

---

## Phase 4 — Transfer

### Task 12: Upload/download with progress overlay

**Files:**
- Modify: `internal/ui/model.go`
- Create: `internal/ui/progress.go`
- Create: `internal/ui/progress_test.go`
- Modify: `internal/fs/remote.go` (add progress reader to Upload/Download)

**Interfaces:**
- Adds to remote.go: `Upload(ctx, localPath, remotePath string, progress chan<- int64) error` and `Download` same — wraps reader with a `countingReader` that sends bytes-written to progress channel; caller closes channel
- Produces:
  - `progressModel{filename string, written int64, total int64, speed float64, done bool}`
  - `(p progressModel) Render(width int) string`
  - `ProgressMsg{filename string, written, total int64, done bool, err error}`
  - `transferCmd(rb RemoteUploader, src, dst string, upload bool, total int64) tea.Cmd` in model.go

- [ ] **Step 1:** Write failing test for `progressModel.Render`

```go
// internal/ui/progress_test.go
func TestProgressRender(t *testing.T) {
    p := progressModel{filename: "deploy.sh", written: 620, total: 1000}
    out := p.Render(40)
    assert.Contains(t, out, "deploy.sh")
    assert.Contains(t, out, "62%")
}

func TestProgressRenderComplete(t *testing.T) {
    p := progressModel{filename: "f", written: 100, total: 100, done: true}
    out := p.Render(40)
    assert.Contains(t, out, "100%")
}
```

- [ ] **Step 2:** Run — expect FAIL

- [ ] **Step 3:** Implement `internal/ui/progress.go`

Bar: `████████░░░░ 62%` — filled block runes proportional to `written/total * barWidth`. Show speed (bytes/s) and ETA.

- [ ] **Step 4:** Update `Upload`/`Download` in remote.go with `countingReader` that sends bytes on progress channel after each read chunk

- [ ] **Step 5:** Implement `transferCmd` in model.go

`c` key → `SelectedEntries()` on active pane, direction determines upload vs download. Set `mode=modeProgress`. Each progress tick → `ProgressMsg`. On done/error → `mode=modeBrowse` + reload dest pane.

Conflict: before transfer, `Stat(dst)`; if exists → `mode=modeConfirm` with `confirmOverwrite` action.

- [ ] **Step 6:** Run tests

```bash
go test ./... -v -race
```

- [ ] **Step 7:** Manual integration test: transfer a file, observe progress overlay

- [ ] **Step 8:** Commit

```bash
git add internal/ui/ internal/fs/remote.go
git commit -S -s -m "feat: file transfer with progress overlay and conflict detection"
```

---

### Task 13: Phase 4 go-specialist review

- [ ] **Step 1:** Dispatch `go-specialist` agent with transfer changes
- [ ] **Step 2:** Apply blocking findings

---

## Phase 5 — Multi-select + bulk transfer

### Task 14: Multi-select

**Files:**
- Modify: `internal/ui/pane.go`
- Modify: `internal/ui/pane_test.go`
- Modify: `internal/ui/model.go`

**Interfaces:**
- Adds to PaneModel:
  - `selected map[int]bool` (keyed by index into `visible()`)
  - `(p *PaneModel) ToggleSelect()` — toggle cursor entry, cursor moves down
  - `(p *PaneModel) SelectAll()` / `DeselectAll()`
  - `(p *PaneModel) SelectedEntries() []fs.FileEntry` — returns selected entries; falls back to `[]SelectedEntry()` if none

- [ ] **Step 1:** Add failing tests

```go
func TestPaneMultiSelect(t *testing.T) {
    p := PaneModel{width: 40, height: 10}
    p.SetEntries([]fs.FileEntry{{Name: "a"}, {Name: "b"}, {Name: "c"}})
    p.ToggleSelect() // select a, cursor→1
    p.ToggleSelect() // select b, cursor→2
    assert.Len(t, p.SelectedEntries(), 2)
    p.DeselectAll()
    assert.Len(t, p.SelectedEntries(), 1) // fallback to cursor
}
```

- [ ] **Step 2:** Run — expect FAIL

- [ ] **Step 3:** Implement multi-select on PaneModel; `Render` marks selected rows with `colSel` background

- [ ] **Step 4:** Wire `Space`→`ToggleSelect`, `a`→`SelectAll`, `A`→`DeselectAll` in model.go

- [ ] **Step 5:** Update `transferCmd` to use `SelectedEntries()` for bulk; progress shows `n of total files`

- [ ] **Step 6:** Run tests + manual bulk transfer test

- [ ] **Step 7:** Commit

```bash
git add internal/ui/
git commit -S -s -m "feat: multi-select with Space/a/A and bulk transfer"
```

---

### Task 15: Phase 5 go-specialist review

- [ ] **Step 1:** Dispatch `go-specialist` agent
- [ ] **Step 2:** Apply blocking findings

---

## Phase 6 — File operations

### Task 16: Delete, rename, mkdir

**Files:**
- Modify: `internal/ui/model.go`
- Modify: `internal/ui/model_test.go`

**Interfaces:**
- Adds: `confirmAction` type — `confirmDelete=0`, `confirmOverwrite=1`
- Adds: `inputBuf string` field on Model (rename + mkdir)
- `pending confirmAction` field on Model

- [ ] **Step 1:** Add failing test

```go
func TestModelDeleteKeyEnterConfirm(t *testing.T) {
    lb := fs.NewLocalBrowser()
    tmp := t.TempDir()
    os.WriteFile(filepath.Join(tmp, "del.txt"), []byte{}, 0644)
    m := NewModel(lb, lb, tmp, tmp)
    m.width, m.height = 120, 40
    entries, _ := lb.List(context.Background(), tmp)
    m.left.SetEntries(entries)
    m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyDelete})
    assert.Equal(t, modeConfirm, m2.(Model).mode)
}
```

- [ ] **Step 2:** Run — expect FAIL

- [ ] **Step 3:** Implement in model.go:

`Delete` key → `mode=modeConfirm`, `pending=confirmDelete`. Confirm overlay: `"Delete <name>? [y/N]"`. `y` → `activeBrowser.Delete` cmd + reload pane + `mode=modeBrowse`. `n`/`Esc` → `mode=modeBrowse`.

`r` key → `mode=modeRename`, `inputBuf=current entry name`. `Enter` → `activeBrowser.Rename` + reload. `Esc` → cancel.

`d` key → `mode=modeMkdir`, `inputBuf=""`. `Enter` → `activeBrowser.Mkdir` + reload. `Esc` → cancel.

Inline input: `tea.KeyMsg` in `modeRename`/`modeMkdir`: printable runes append to `inputBuf`; `Backspace` removes last rune.

- [ ] **Step 4:** Run tests

```bash
go test ./... -v -race
```

- [ ] **Step 5:** Manual test: delete, rename, mkdir on local and remote

- [ ] **Step 6:** Commit

```bash
git add internal/ui/
git commit -S -s -m "feat: delete, rename, mkdir with confirm dialog and inline input"
```

---

### Task 17: Phase 6 go-specialist review

- [ ] **Step 1:** Dispatch `go-specialist` agent
- [ ] **Step 2:** Apply blocking findings

---

## Phase 7 — Polish

### Task 18: Help overlay + error display + edge cases

**Files:**
- Modify: `internal/ui/model.go`
- Create: `internal/ui/help.go`

**Interfaces:**
- Adds: `modeHelp` appMode value
- `renderHelp(width, height int) string` — centred overlay with full keybinding table from spec

- [ ] **Step 1:** Implement `internal/ui/help.go` — `renderHelp` centred overlay, full keybinding table

- [ ] **Step 2:** Wire `?` → `mode=modeHelp`; any key dismisses. Add `?:help` to key hint footer.

- [ ] **Step 3:** Error overlay: `modeError` renders `err.Error()` centred; `Esc`/`Enter` dismiss

- [ ] **Step 4:** Edge case guards:
  - `pane.width < 20` → render `"<too narrow>"` instead of normal layout
  - Empty `visible()` → render `"(empty)"` row
  - `tea.WindowSizeMsg` while `mode=modeProgress` → re-render progress at new width

- [ ] **Step 5:** Run all tests

```bash
make test-race
make lint
```

- [ ] **Step 6:** Commit

```bash
git add internal/ui/
git commit -S -s -m "feat: help overlay, error display, narrow terminal and empty dir guards"
```

---

### Task 19: Final review + release

- [ ] **Step 1:** Dispatch `go-specialist` agent over full codebase
- [ ] **Step 2:** Dispatch `go-quality` agent — lint, anti-patterns, idiomatic Go
- [ ] **Step 3:** Apply all blocking findings; re-run:

```bash
make test-race
make lint
make build
```

- [ ] **Step 4:** Manual smoke test with both test servers (key-based + password-based)

- [ ] **Step 5:** Update `CHANGELOG.md` with v0.1.0 entry (Keep a Changelog format)

- [ ] **Step 6:** Commit

```bash
git add CHANGELOG.md
git commit -S -s -m "chore: v0.1.0 release notes"
```
