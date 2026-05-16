```
 █████╗ ██████╗ ███████╗███╗   ██╗██╗   ██╗
██╔══██╗██╔══██╗██╔════╝████╗  ██║██║   ██║
███████║██║  ██║█████╗  ██╔██╗ ██║██║   ██║
██╔══██║██║  ██║██╔══╝  ██║╚██╗██║╚██╗ ██╔╝
██║  ██║██████╔╝███████╗██║ ╚████║ ╚████╔╝
╚═╝  ╚═╝╚═════╝ ╚══════╝╚═╝  ╚═══╝  ╚═══╝
```

> **adenVault** — your offline password manager. no cloud. no subscription. no breach.

```sh
adenV add github        # interactive form — username, password, url, notes
adenV copy github       # password silently in your clipboard
adenV get github --show # reveal the full entry
adenV list              # see everything in one glance
```

One binary. One master password. Your passwords live on your machine and nowhere else.

---

## what is adenVault?

adenVault is a **personal CLI password manager** — the terminal-native alternative to 1Password, Bitwarden, or LastPass for people who live in the command line.

You give each entry a label you'll remember (`github`, `gmail`, `aws-root`). adenVault stores a **username**, a **password**, an optional **URL**, and optional **notes** — all individually encrypted with AES-256-GCM before they ever touch your disk. Your master password is never stored; it's typed once per session to unlock the vault, then gone.

- **stores passwords, not just env vars** — first-class username + password + url + notes per entry
- **copy without revealing** — `adenV copy github` puts the password in your clipboard, never on screen
- **masked by default** — `adenV get` shows `•••••••••` until you pass `--show`
- **no internet, ever** — the binary makes zero network calls; there is nothing to breach remotely
- **one binary** — no daemon, no browser extension, no electron app, no background process
- **per-vault, per-environment** — separate vaults for personal / work / prod via `--env`

---

## why adenVault?

1Password costs $36/year. Bitwarden has a great free tier but still needs an account, an internet connection, and trust in their servers. KeePass needs a GUI. `pass` is Unix-only and needs GPG set up.

adenVault is for the person who wants **none of that overhead**:

- no account to create
- no server to trust
- no master password stored anywhere but your head
- no internet connection needed — works on a plane, in a Faraday cage, after the zombie apocalypse
- no GUI — just a terminal and one fast binary

If your passwords ever leak from adenVault, an attacker got into your machine and also knew your master password. That's the threat model, and it's the same one you accept with every offline password manager.

---

## how to install

### macOS / Linux — one-liner

```sh
curl -fsSL https://raw.githubusercontent.com/codebyNJ/AdenVault/main/install.sh | bash
```

Detects your OS and architecture, downloads the right binary from the latest GitHub release, installs to `/usr/local/bin/adenV` (or `~/.local/bin` if not writable). No Go needed.

Pin to a specific version:

```sh
ADENV_VERSION=v1.0.0 curl -fsSL https://raw.githubusercontent.com/codebyNJ/AdenVault/main/install.sh | bash
```

Want to inspect the script before running it?

```sh
curl -fsSL https://raw.githubusercontent.com/codebyNJ/AdenVault/main/install.sh -o install.sh
less install.sh && bash install.sh
```

### Windows — one-liner (PowerShell)

```powershell
irm https://raw.githubusercontent.com/codebyNJ/AdenVault/main/install.ps1 | iex
```

Downloads `adenV.exe` into `$env:USERPROFILE\.adenV\bin`, permanently adds it to your user `PATH`. Open a new terminal and type `adenV --version`.

### build from source (Go 1.24+)

```sh
git clone https://github.com/codebyNJ/AdenVault.git
cd AdenVault
sudo make install    # → /usr/local/bin/adenV
```

### verify

```sh
adenV --version
adenV              # shows the splash with your current vault status
```

---

## the commands

| command | aliases | what it does |
|---|---|---|
| `adenV init` | `i`, `new`, `create` | create a new vault, prompts for a master password |
| `adenV add <label>` | `a`, `new`, `set`, `save` | interactive form: username, password, url, notes |
| `adenV get <label>` | `g`, `show`, `view`, `open` | display an entry — password masked by default |
| `adenV copy <label>` | `cp`, `c`, `yank` | copy password to clipboard without displaying it |
| `adenV list` | `ls`, `l`, `all` | list all entry labels + field indicators, no password needed |
| `adenV rm <label>` | `remove`, `delete`, `del`, `d` | permanently delete an entry |
| `adenV export` | `e`, `dump` | print `LABEL=password` lines to stdout |
| `adenV run -- CMD` | `r`, `exec`, `x` | run a command with passwords injected as env vars |

### `adenV add` in detail

Run with just a label to get a beautiful interactive form:

```
adenV add github
```

```
  new entry: github

  username   ▏john@example.com
  password   ▏••••••••••••
  url        ▏https://github.com
  notes      ▏work account

  tab/↓ next  ·  shift+tab/↑ prev  ·  ctrl+s save  ·  esc cancel
```

Or skip the form with flags:

```sh
adenV add github --user john@example.com --pass mysecret --url https://github.com
adenV add github --user john@example.com --pass mysecret --note "2FA on phone"
```

Run `adenV add github` again on an existing entry to edit it — the form pre-fills with the current values.

### `adenV get` in detail

```
adenV get github           # password shown as ••••••••••
adenV get github --show    # password revealed in the card
```

```
  ╭────────────────────────────────╮
  │ github                         │
  │                                │
  │ username  john@example.com     │
  │ password  sk_live_...          │
  │ url       https://github.com   │
  │ updated   2026-05-16 15:49     │
  ╰────────────────────────────────╯
```

### `adenV copy` in detail

```sh
adenV copy github           # copies the password
adenV copy github --user    # copies the username instead
```

The value goes directly to your clipboard. Nothing is printed to the terminal — safe even with screen-sharing active.

### `adenV list` in detail

```
  LABEL      FIELDS   UPDATED
  ─────────────────────────────────
  github     uw·      2026-05-16 15:49
  gmail      u··      2026-05-16 10:00
  netflix    u·n      2026-05-15 09:00

  3 entries in dev vault
  icons: username  website  notes
```

The field indicators show at a glance what each entry contains: `u` username, `w` website/url, `n` notes. No password is needed to list.

### global flags

| flag | default | what it does |
|---|---|---|
| `--env <name>` | `dev` | use a different vault (`work`, `personal`, `prod`, anything). Each env has its own encryption key |
| `--vault-dir <path>` | `~/.aden` | store the vault somewhere else — useful for syncing via encrypted cloud storage |
| `--password-stdin` | off | read the master password from stdin; for CI or scripts |
| `--quiet` | off | suppress decoration; `list` prints bare labels to stdout |

---

## security

adenVault protects your passwords with the same primitives cloud password managers use — the difference is the vault file never leaves your machine.

| what | how |
|---|---|
| encryption | AES-256-GCM per field — username, password, url, and notes are encrypted individually |
| key derivation | Argon2id (`time=1, mem=64 MiB, threads=4`) — brute-force hostile |
| master password | never written to disk, ever — lives in memory for milliseconds |
| nonce | fresh random 12-byte nonce per encryption — no nonce reuse possible |
| wrong password | returns the same error as a tampered file — no oracle attack surface |
| file permissions | vault written as `0600` — only your user can read it |
| network | zero calls — the binary doesn't know what TCP is |

**what it protects against:** accidental exposure, stolen disks, reading your vault file without the master password, shoulder-surfing (masked by default).

**what it doesn't protect against:** a compromised machine where an attacker can read your process memory, a keylogger on your keyboard.

---

## the vault file

Vaults live at `~/.aden/<vault>-<id>/vault.<env>.json`. They're just JSON — you can read them, back them up, sync them with syncthing or an encrypted Dropbox. Without the master password the values are meaningless ciphertext.

```json
{
  "version": 2,
  "entries": {
    "github": {
      "username":  { "nonce": "...", "ciphertext": "..." },
      "password":  { "nonce": "...", "ciphertext": "..." },
      "url":       { "nonce": "...", "ciphertext": "..." },
      "created_at": "2026-05-16T15:49:00Z",
      "updated_at": "2026-05-16T15:49:00Z"
    }
  }
}
```

Entry labels are plaintext so `adenV list` works without a password. Every value is encrypted.

---

## one more thing

If you forget your master password, the vault is unrecoverable — by design. Put the master password in a place you actually trust: a physical notebook, a trusted password manager, a note printed and locked in a drawer. That's the tradeoff for "no cloud".

---

**source** — [github.com/codebyNJ/AdenVault](https://github.com/codebyNJ/AdenVault)
**releases** — [github.com/codebyNJ/AdenVault/releases](https://github.com/codebyNJ/AdenVault/releases)
**release notes** — `RELEASE_NOTES.md`
