# adenVault — release notes

---

## v1.0.0 — 2026-05-16

**The first public release of adenVault.**

adenVault is a local-first, password-locked secrets manager for developers. One binary, zero network calls, per-project encrypted vaults. Your API keys stay on your machine — nowhere else.

### what's included

**core vault**
- `adenV init` — creates a per-project vault at `~/.aden/<project>-<id>/vault.<env>.json`. Master password prompted twice; never written to disk
- `adenV set KEY VALUE` — encrypts each secret with AES-256-GCM (unique nonce per write) before it touches the filesystem
- `adenV get KEY` — decrypts and prints one value to stdout. Clean output — safe in `$(...)` subshells
- `adenV list` — lists key names + last-updated timestamps. No password required; key names are intentionally plaintext
- `adenV rm KEY` — removes a secret. Interactive confirmation in a TTY; auto-proceeds with `--password-stdin`
- `adenV export` — prints all secrets as `KEY=value` lines to stdout, suitable for `> .env` redirect
- `adenV run -- CMD` — decrypts all secrets, merges them with the current shell env (adenVault wins on conflict), execs the command. Exit code forwarded. No secrets written to disk during this flow

**security**
- AES-256-GCM encryption per secret, independent nonce per write
- Argon2id key derivation — `time=1, memory=64 MiB, threads=4` — brute-force hostile on 2026 hardware
- Wrong password and tampered ciphertext both return the same error (no oracle attack surface)
- Atomic vault writes (temp file + fsync + rename) — no partial writes on crash
- Vault permissions set to `0600` on creation
- Zero network calls at runtime — verifiable with `strings`

**multi-environment**
- `--env <name>` flag (default `dev`) creates isolated vaults per environment
- `dev`, `staging`, `prod` each get their own file and their own derived key — same password, different salt, different key

**project identity**
- Vault path is derived from the git remote URL (SHA-256 prefix), so renaming the working directory doesn't break it
- Falls back to directory name for non-git projects

**UX**
- Bubble Tea interactive password prompts (hidden input, confirmation on init)
- Lip Gloss ADENV wordmark with pink → violet → cyan gradient
- Gradient separator matching the wordmark palette
- Status splash on bare `adenV`: shows project, env, and secret count if a vault exists
- `aden list` view: boxed table with key names and timestamps
- `--quiet` flag for scripting — suppresses all decoration; `list` writes bare names to stdout
- `--password-stdin` for CI/non-TTY environments

**aliases** (all commands have short forms)

| command | aliases |
|---|---|
| `init` | `i`, `new`, `create` |
| `set` | `s`, `add`, `put`, `save` |
| `get` | `g`, `show`, `cat`, `read` |
| `list` | `ls`, `l`, `keys` |
| `rm` | `remove`, `delete`, `del`, `unset` |
| `export` | `e`, `dump`, `env` |
| `run` | `r`, `exec`, `x` |

**installers**
- `install.sh` — macOS / Linux one-liner via `curl | bash`
- `install.ps1` — Windows one-liner via `irm | iex`
- `make release` — cross-compiles `darwin-amd64`, `darwin-arm64`, `linux-amd64`, `linux-arm64`

### known limitations in v1.0.0

- no secret versioning or history
- no team sharing or sync across machines
- no OS keychain integration
- no automatic `.gitignore` management (reminder printed on `adenV export`)
- no `adenV import .env` bulk import (planned for v1.1.0)
- no `adenV rotate` (re-encrypt under a new master password, planned)
- `adenV run` on Windows uses `os/exec` — no `exec()`-style process replacement, the parent adenV process remains in the tree

### upgrading

Re-run the installer. The vault format is forwards-compatible — existing vaults created by v1.0.0 will be readable by future versions.

---

## upcoming — v1.1.0 (planned)

- `adenV import .env` — bulk import from an existing `.env` file
- `adenV copy KEY` — copy a secret to the clipboard without printing it
- `adenV rotate` — re-encrypt all secrets under a new master password
- shell completion (`adenV completion zsh|bash|fish|powershell`)
- `adenV diff --env staging --env prod` — compare key sets between two environments

---

*adenVault is built with [Bubble Tea](https://github.com/charmbracelet/bubbletea), [Lip Gloss](https://github.com/charmbracelet/lipgloss), [Cobra](https://github.com/spf13/cobra), and the Go standard library.*
*Source: [github.com/codebyNJ/AdenVault](https://github.com/codebyNJ/AdenVault)*
