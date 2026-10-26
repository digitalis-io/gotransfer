<p align="center">
  <img src="https://raw.githubusercontent.com/digitalis-io/digitalis-marketplace-assets/main/logos/digitalis-logo.png" width="300" alt="Digitalis.io">
</p>

# gotransfer

A dual-pane SFTP file manager for the terminal.

## Quick Start

Requires Go 1.23+.

```bash
git clone git@github.com:digitalis-io/gotransfer.git
cd gotransfer
make build
./gotransfer user@host                      # connect, uses SSH agent or prompts for password
./gotransfer user@host -p 2222             # custom port
./gotransfer user@host -i ~/.ssh/id_ed25519  # explicit key
```

First connection to a new host shows the server's SHA256 fingerprint and prompts `[y/N]`. Answering `y` appends the key to `~/.ssh/known_hosts`.

## Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-p` | int | 22 | SSH port |
| `-i` | path | — | Private key file (overrides agent and password prompt) |

## Keybindings

| Key | Action |
|-----|--------|
| Tab | Switch pane |
| j / ↓ | Move cursor down |
| k / ↑ | Move cursor up |
| Enter | Enter directory |
| - / Backspace | Go up one directory |
| Space | Toggle selection |
| a | Select all |
| A | Deselect all |
| c | Copy / transfer selected files |
| y | Confirm action |
| / | Filter entries by name (live) |
| s | Cycle sort field (name → size → time) |
| S | Toggle sort direction |
| d | Create new directory |
| r | Rename entry |
| Delete | Delete entry |
| ? | Show help overlay |
| q / Esc | Quit / cancel |

## Authentication

Authentication is attempted in this order:

1. Explicit private key (`-i` flag)
2. SSH agent (`$SSH_AUTH_SOCK`)
3. Password prompt

---

*This project is maintained by [Digitalis.io](https://digitalis.io). For support, visit [digitalis.io/contact](https://digitalis.io/contact).*
