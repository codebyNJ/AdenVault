```
 █████╗ ██████╗ ███████╗███╗   ██╗██╗   ██╗
██╔══██╗██╔══██╗██╔════╝████╗  ██║██║   ██║
███████║██║  ██║█████╗  ██╔██╗ ██║██║   ██║
██╔══██║██║  ██║██╔══╝  ██║╚██╗██║╚██╗ ██╔╝
██║  ██║██████╔╝███████╗██║ ╚████║ ╚████╔╝
╚═╝  ╚═╝╚═════╝ ╚══════╝╚═╝  ╚═══╝  ╚═══╝
```

> **adenVault** — a vault that lives in your home directory, not in someone else's cloud.

```sh
adenV init
adenV set STRIPE_KEY sk_live_xxx
adenV run -- npm start
```

That's the whole product. No account, no dashboard, no monthly seat. Just one binary, one master password, and a sealed file your editor can't read.

---

## what is adenVault?

adenVault is a single-binary, password-locked safe for the API keys, database URLs, and access tokens that pile up on every developer's laptop.

- **per-project** — each repo gets its own vault, auto-discovered from the git remote so renaming the folder doesn't orphan your secrets
- **encrypted before it touches disk** — every value is sealed with AES-256-GCM under a fresh nonce; the key is derived from your master password via Argon2id (64 MiB, brute-force-hostile)
- **never on disk in plaintext** — only key names and a 16-byte random salt live in the clear; your password lives in process memory for milliseconds, then disappears
- **invisible to git** — the vault lives in `~/.aden/<project>/`, nowhere near your repo. There is nothing for `git add .` to grab
- **offline by design** — the binary makes zero network calls. No telemetry, no auto-updates, no "phone home". Verify with `strings` if you want
- **scriptable** — `adenV get KEY` writes only the value to stdout, so `$(adenV get DB_URL)` works in any shell

You invoke it as `adenV`. The project, the docs, the brand: **adenVault**.

---

## why you need it

If any of these sound familiar, adenVault is built for you:

- you've pushed a `.env` to a public repo and spent the rest of the day rotating keys
- your `~/.zshrc` has ten years of `export STRIPE_KEY=...` lines you're afraid to delete
- you've shared an API key over Slack DM because Doppler felt like overkill for a side project
- you've copy-pasted a token from 1Password into a terminal that quietly logged it to `~/.bash_history`
- you have three `.env.example`, `.env.local`, `.env.dev` files and no idea which one is real

adenVault replaces all of these habits with one binary. The right thing — encrypted secrets, per project, with no plaintext on disk — becomes the easy thing. `adenV set` and `adenV run --` are the only verbs you need.

**Anti-goals.** adenVault is not Doppler, not HashiCorp Vault, not AWS Secrets Manager. There is no team sync, no audit log, no rotation policy, no UI. It is a one-developer, one-laptop tool. If you outgrow it you can `adenV export > .env` and move on. No lock-in.

---

## how to install

You need **Go 1.24+** to build from source. Pre-built binaries can be cross-compiled with `make release`.

### macOS / Linux — one-liner

```sh
curl -fsSL https://raw.githubusercontent.com/codebyNJ/AdenVault/main/install.sh | bash
```

The script detects your OS and architecture, downloads the right binary from the latest GitHub release, installs it to `/usr/local/bin/adenV` (or `~/.local/bin/adenV` if `/usr/local/bin` isn't writable), and prints a confirmation. No Go needed.

To pin a specific version:

```sh
ADENV_VERSION=v1.0.0 curl -fsSL https://raw.githubusercontent.com/codebyNJ/AdenVault/main/install.sh | bash
```

To change the install directory:

```sh
ADENV_INSTALL_DIR=~/.local/bin curl -fsSL https://raw.githubusercontent.com/codebyNJ/AdenVault/main/install.sh | bash
```

If you'd rather inspect the script before running it (good habit):

```sh
curl -fsSL https://raw.githubusercontent.com/codebyNJ/AdenVault/main/install.sh -o install.sh
less install.sh          # read it
bash install.sh          # run it
```

### macOS / Linux — from source (needs Go 1.24+)

```sh
git clone https://github.com/codebyNJ/AdenVault.git
cd AdenVault
sudo make install       # builds bin/adenV → /usr/local/bin/adenV
```

### Windows (PowerShell — one-liner)

```powershell
irm https://raw.githubusercontent.com/codebyNJ/AdenVault/main/install.ps1 | iex
```

Downloads and runs the install script — drops `adenV.exe` into `$env:USERPROFILE\.adenV\bin`, permanently adds it to your user `PATH`. Open a new terminal and you're done. No Go needed.

If you'd rather build from source:

```powershell
git clone https://github.com/codebyNJ/AdenVault.git
Set-Location AdenVault
go build -ldflags "-s -w" -o adenV.exe .
New-Item -ItemType Directory -Force "$env:USERPROFILE\.adenV\bin" | Out-Null
Move-Item .\adenV.exe "$env:USERPROFILE\.adenV\bin\adenV.exe"
# add $env:USERPROFILE\.adenV\bin to your PATH if not already there
```

### cross-compile release binaries

```sh
make release   # produces dist/adenV-{darwin,linux}-{amd64,arm64}
```

Upload the artefacts to a GitHub release and both install scripts will pick them up automatically.

### verify it worked

```sh
adenV --version
adenV --help
```

Bare `adenV` (no subcommand) shows a gradient splash with your current project's vault status — handy as a "where am I" command.

---

## the commands

Every command has short and long forms. Use whichever fits the flow — `adenV s API_KEY abc` and `adenV set API_KEY abc` do the same thing.

| primary | aliases | what it does |
|---|---|---|
| `adenV init` | `i`, `new`, `create` | seal a fresh vault for the current project, prompts for a master password twice |
| `adenV set KEY VALUE` | `s`, `add`, `put`, `save` | encrypt and store a secret |
| `adenV get KEY` | `g`, `show`, `cat`, `read` | decrypt and print one value to stdout (no decoration — safe in `$(...)`) |
| `adenV list` | `ls`, `l`, `keys` | list secret names + last-updated timestamps — **no password required** |
| `adenV rm KEY` | `remove`, `delete`, `del`, `unset` | delete a secret |
| `adenV export` | `e`, `dump`, `env` | print all secrets as `KEY=value` lines, designed to be `> .env` redirected |
| `adenV run -- CMD` | `r`, `exec`, `x` | run a command with secrets injected as env vars; exit code is forwarded |

### global flags

| flag | default | what it does |
|---|---|---|
| `--env <name>` | `dev` | which environment vault to use — `dev`, `staging`, `prod`, anything you want. Each env has its own encryption key |
| `--vault-dir <path>` | `~/.aden` | override the vault root (useful if you sync via encrypted Dropbox or 1Password vault attachments) |
| `--password-stdin` | off | read the master password from stdin instead of prompting. **The CI escape hatch** |
| `--quiet` | off | suppress decoration and status output. `list` prints bare names to stdout in this mode — perfect for scripting |

### a day in the life

```sh
# day 0: one-time setup
adenV init                                  # asks for a master password
adenV s DB_URL postgres://localhost/mydb    # store
adenV s STRIPE_KEY sk_live_xxxxxxxxxxxx     # store another

# day n: actually working
adenV l                                     # what's in here again?
adenV g STRIPE_KEY | pbcopy                 # copy to clipboard, no plaintext lingering
adenV x -- npm start                        # run your app with secrets in env

# a different environment
adenV --env prod s STRIPE_KEY sk_live_PROD_xxx
adenV --env prod x -- ./deploy.sh
```

### in CI

CI has no TTY, so you can't be prompted. Use `--password-stdin` and store your master password as a CI secret:

```yaml
# GitHub Actions
- name: deploy
  env:
    ADEN_MASTER: ${{ secrets.ADEN_MASTER }}
  run: |
    echo "$ADEN_MASTER" | adenV --password-stdin --env prod export > .env
    ./deploy.sh
```

---

## one more thing

The vault file is just JSON. You can read it, copy it, sync it, back it up. Without your master password, nobody — including you — can recover the values from it. So pick a good one, and put it in your password manager.

That's the whole tool.

---

**source** — [github.com/codebyNJ/AdenVault](https://github.com/codebyNJ/AdenVault)
**releases** — [github.com/codebyNJ/AdenVault/releases](https://github.com/codebyNJ/AdenVault/releases)
**release notes** — see `RELEASE_NOTES.md`
