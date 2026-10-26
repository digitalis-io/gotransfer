# Changelog

All notable changes to this project will be documented in this file.

## [Unreleased]

## [0.1.0] - 2026-10-25
### Added
- Dual-pane TUI file manager with local filesystem and SFTP remote panes
- SSH connection with known_hosts verification and interactive host key fingerprint prompt
- Directory navigation via Enter (descend), `-`/Backspace (ascend), Tab (switch panes)
- Filter (`/`) for live name-based entry search within a pane
- Sort toggle (`s` cycle field, `S` reverse direction) within each pane
- Single-file transfer with progress overlay and conflict detection
- Multi-select with Space (toggle one), `a` (select all), `A` (deselect all)
- Bulk file transfer with per-file progress overlay
- Delete, rename, and mkdir operations with confirmation dialogs and inline input
- Help overlay (`?`) listing all key bindings
- Error display overlay for surfacing operation failures
- Narrow terminal guard (minimum width enforcement) and empty directory guard

### Fixed
- Input handling: non-printable keys no longer appended to rename/mkdir buffers
- Path validation: reject names containing `/`, `\`, `..` in rename/mkdir
- Help overlay: display-column-aware padding for Unicode cursor indicator

[Unreleased]: https://github.com/digitalis-io/gotransfer/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/digitalis-io/gotransfer/releases/tag/v0.1.0
