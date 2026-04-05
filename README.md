# psst

[![CI](https://github.com/ChengaDev/persist/actions/workflows/ci.yml/badge.svg)](https://github.com/ChengaDev/persist/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/ChengaDev/persist)](https://github.com/ChengaDev/persist/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

**psst** is your terminal clipboard, always within arm's reach. Save anything - tokens, connection strings, commands, snippets - and paste it anywhere in one command. No more hunting through notes or browser tabs. Sensitive values are encrypted and never leave your machine, so what's yours stays yours.

**[Why psst?](#features) · [Quick Start](#quick-start) · [Configuration](#configuration) · [Security](SECURITY.md) · [Contributing](CONTRIBUTING.md)**

---

**Plain entry — save once, paste anywhere:**
```sh
psst save deploy-cmd --value "kubectl apply -f prod.yaml"
psst get  deploy-cmd   # copies straight to clipboard
```

**Secure entry — encrypted, never touches shell history:**
```sh
psst save db-password --secure   # prompts with hidden input, encrypts with AES-256-GCM
psst get  db-password            # decrypts and copies straight to clipboard
```

---

## Features

- **Instant retrieval** — one command copies any saved value to the clipboard, no file hunting
- **AES-256-GCM encryption** — secure entries are encrypted at rest with a per-entry Argon2id-derived key
- **Zero shell-history leakage** — secure values are never accepted as CLI flags
- **Clipboard injection** — values go directly to the clipboard, never printed to stdout
- **Password session cache** — optional short-lived cache to avoid repeated password prompts
- **Auto clipboard wipe** — configurable timer to clear the clipboard after retrieval
- **Fuzzy key search** — substring filter on `list`, typo suggestions on `get`
- **No cloud, no network** — everything stays in `~/.persist/data.db` on your machine

---

## Installation

Download the binary for your platform from the [Releases](https://github.com/ChengaDev/persist/releases) page and move it to your PATH:

**macOS (Apple Silicon)**
```sh
curl -L https://github.com/ChengaDev/persist/releases/latest/download/psst-darwin-arm64 -o psst
chmod +x psst && sudo mv psst /usr/local/bin/
```

**macOS (Intel)**
```sh
curl -L https://github.com/ChengaDev/persist/releases/latest/download/psst-darwin-amd64 -o psst
chmod +x psst && sudo mv psst /usr/local/bin/
```

**Linux (amd64)**
```sh
curl -L https://github.com/ChengaDev/persist/releases/latest/download/psst-linux-amd64 -o psst
chmod +x psst && sudo mv psst /usr/local/bin/
```

**Windows** — download `psst-windows-amd64.exe` from the [Releases](https://github.com/ChengaDev/persist/releases) page and add it to your PATH.

### From source (Go 1.19+)

```sh
go install github.com/ChengaDev/persist@latest
```

---

## Quick start

```sh
# 1. Initialize (one-time) — set your master password
psst init
```

> **About the master password**
> psst never stores your master password anywhere. Instead, during `psst init` it encrypts
> a known canary value with a key derived from your password using Argon2id (64 MB memory,
> 3 iterations). On every subsequent unlock, psst re-derives the key and attempts to decrypt
> the canary — if it matches, your password is correct. There is no recovery mechanism:
> if you forget the master password, delete `~/.persist/data.db` and start over.
> Use `psst hint` to store a plain-text memory aid during init.

```sh
# 2. Save a plain entry (instant, no password needed to retrieve)
psst save deploy-cmd --value "kubectl apply -f prod.yaml"

# 3. Save a secure entry (prompts with hidden input, then encrypts)
psst save db-password --secure

# 4. Save a secure entry from clipboard (copy the value first)
psst save api-key --secure --from-clip

# 5. Retrieve — value copied to clipboard, never printed
psst get deploy-cmd
psst get db-password    # prompts for master password, then copies to clipboard

# 6. Browse your entries
psst list
psst list github        # filter by substring

# 7. Delete
psst delete deploy-cmd
```

---

## Commands

| Command | Description |
|---|---|
| `psst init` | Initialize with a master password |
| `psst save <key>` | Save a key-value pair |
| `psst get <key>` | Copy value to clipboard |
| `psst list [filter]` | List all keys (optional substring filter) |
| `psst delete <key>` | Remove an entry |
| `psst hint` | Show the password hint set during init |
| `psst lock` | Expire the current session immediately |
| `psst config show` | Print current configuration |
| `psst config set <key> <value>` | Update a setting |

### `psst save` flags

| Flag | Description |
|---|---|
| `--value <val>` | Inline value (plain entries only) |
| `--secure` | Encrypt the value |
| `--from-clip` | Read value from clipboard |
| `--force` | Overwrite if key already exists |

---

## Configuration

```sh
psst config show                          # view all settings

psst config set session_timeout 10        # cache password for 10 minutes
psst config set clipboard_clear_after 30  # wipe clipboard after 30 seconds
psst config set default_secure true       # always encrypt on psst save
```

| Key | Default | Description |
|---|---|---|
| `session_timeout` | `0` | Minutes to cache password after first unlock (0 = always prompt) |
| `clipboard_clear_after` | `0` | Seconds before clipboard is wiped after `psst get` (0 = never) |
| `default_secure` | `false` | Make `--secure` the default on every `psst save` |

---

## Security

For the full security model, known limitations, and how to report a vulnerability, see **[SECURITY.md](SECURITY.md)**.

---

## Development

```sh
make build        # compile
make test         # full test suite (includes slow Argon2id tests)
make test-short   # fast tests only
make install      # build + install to /usr/local/bin
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for the full guide.

---

## License

[MIT](LICENSE) © ChengaDev
