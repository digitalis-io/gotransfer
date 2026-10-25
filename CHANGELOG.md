# Changelog

All notable changes to this project will be documented in this file.

## [Unreleased]

## [0.1.0] - 2026-10-25
### Added
- Dual-pane TUI file manager with local filesystem and SFTP remote panes
- SSH connection with known_hosts verification and interactive host key fingerprint prompt
- Directory navigation via Enter (descend), `-`/Backspace (ascend), Tab (switch panes)
- Filter mode (`/`) for name-based entry search within a pane
- Sort toggle (`s` cycle field, `S` reverse direction) within each pane
- Single-file transfer with progress overlay and conflict detection
- Multi-select with Space (toggle one), `a` (select all), `A` (deselect all)
- Bulk file transfer with per-file progress overlay
- Delete, rename, and mkdir operations with confirmation dialogs and inline input
- Help overlay (`?`) listing all key bindings
- Error display overlay for surfacing operation failures
- Narrow terminal guard (minimum width enforcement) and empty directory guard
- `FileEntry` type with `FormatSize` and `FormatMode` helper functions
- `LocalBrowser` implementing list, stat, mkdir, rename, and delete
- `RemoteBrowser` (SFTP) implementing the same `Browser` interface plus upload/download
