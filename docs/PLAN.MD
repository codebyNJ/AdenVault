# aden — build plan

## goal

Ship a working local secrets manager CLI in Go within 4 hours. The binary must be installable, usable on a real project, and secure by default.

---

## constraints

- solo build, 4 hours total
- Go only — no external services, no databases
- must work on macOS and Linux
- zero network calls at runtime

---

## milestones

### milestone 0 — scaffold (0:00 – 0:30)

**goal:** the binary compiles, all commands are registered, and each prints a "not yet implemented" stub.

tasks:
- `go mod init github.com/<you>/aden`
- `go get github.com/spf13/cobra github.com/fatih/color golang.org/x/crypto golang.org/x/term`
- create `main.go` with `cobra.Execute()`
- create `cmd/root.go` with global `--env` flag (default `"dev"`)
- stub out `cmd/init.go`, `set.go`, `get.go`, `list.go`, `rm.go`, `export.go`, `run.go`
- verify: `go build -o aden . && ./aden --help` shows all subcommands

**exit criteria:** binary builds, all commands listed in `--help`, no panics.

---

### milestone 1 — crypto layer (0:30 – 1:15)

**goal:** a standalone, fully tested crypto package that can encrypt and decrypt arbitrary byte slices.

tasks:
- create `internal/crypto/kdf.go` — Argon2id key derivation with salt
- create `internal/crypto/crypto.go` — AES-256-GCM encrypt and decrypt functions
- write `internal/crypto/crypto_test.go` — round-trip test, wrong-password test, tampered-ciphertext test
- run `go test ./internal/crypto/...` — all green

key decisions to lock in here:
- salt length: 16 bytes
- nonce length: 12 bytes (GCM standard)
- Argon2id params: `time=1, memory=64MB, threads=4, keyLen=32`

**exit criteria:** `go test ./internal/crypto/...` passes. Encryption and decryption are independently testable without any vault or CLI code.

---

### milestone 2 — vault layer (1:15 – 1:45)

**goal:** a `Vault` struct that reads and writes `vault.json` on disk, using the crypto layer for individual secret encryption.

tasks:
- create `internal/vault/project.go` — resolve project name from `git remote` or folder name, hash to vault path
- create `internal/vault/vault.go`:
  - `Load(path, key) (*Vault, error)` — read and parse vault.json
  - `Save() error` — atomic write (temp file + rename)
  - `Set(name, value string)`
  - `Get(name string) (string, error)`
  - `Delete(name string) error`
  - `List() []string`
  - `All() map[string]string`
- write a basic test: init a vault, set a value, save, reload, get — verify round-trip

**exit criteria:** vault round-trip test passes. Vault file on disk contains no plaintext values.

---

### milestone 3 — wire commands (1:45 – 2:45)

**goal:** all 7 commands work end-to-end against a real vault file on disk.

order of implementation:

1. **`aden init`** — prompt for master password + confirm, create vault file, write config.json
2. **`aden set <KEY> <VALUE>`** — prompt password, load vault, encrypt + store, save
3. **`aden get <KEY>`** — prompt password, load vault, decrypt, print
4. **`aden list`** — load vault (no password needed — key names are plaintext), print names
5. **`aden rm <KEY>`** — prompt password, delete from vault, save
6. **`aden export`** — prompt password, decrypt all, write `KEY=value` lines to stdout
7. **`aden run -- <cmd>`** — prompt password, inject env vars, exec subprocess, forward exit code

password prompting lives in a shared helper in `cmd/root.go`:

```go
func promptPassword(prompt string) ([]byte, error) {
    fmt.Fprint(os.Stderr, prompt)
    return term.ReadPassword(int(os.Stdin.Fd()))
}
```

**exit criteria:** full manual smoke test — `init → set → get → list → export → run -- env | grep KEY → rm → get (should fail)`.

---

### milestone 4 — UX polish (2:45 – 3:30)

**goal:** the CLI feels intentional, not thrown together.

tasks:
- coloured output: green for success, red for errors, yellow for warnings (`fatih/color`)
- clear error messages with recovery hints:
  - "vault not found — run `aden init` first"
  - "wrong password or corrupted vault"
  - "key not found: DB_URL"
- `--vault-dir` global flag to override `~/.aden/` (useful for CI)
- `aden list` shows key count and last-updated timestamps
- `aden init` warns if vault already exists, prompts to confirm overwrite
- add `--quiet` flag to suppress status output (for scripting)

---

### milestone 5 — build + docs (3:30 – 4:00)

**goal:** installable binary and a README someone else could use.

tasks:
- write `Makefile` with `build`, `install`, and `test` targets
- `go build -ldflags="-s -w" -o aden .` (strip debug symbols)
- `make install` copies binary to `/usr/local/bin/aden`
- write `README.md`:
  - one-paragraph description
  - install instructions
  - quickstart (5 commands)
  - security model (one paragraph)
- final end-to-end smoke test on a real project

---

## stretch goals (if time allows)

these are not in scope for the 4-hour build but are next natural steps:

| feature | effort |
|---|---|
| `aden import .env` — bulk import from existing .env file | ~30 min |
| `aden copy <KEY>` — copy value to clipboard without printing | ~20 min |
| shell completion (`cobra completion zsh`) | ~10 min |
| `aden rotate` — re-encrypt all secrets with a new master password | ~45 min |
| `aden diff` — compare secrets between two environments | ~30 min |

---

## risk log

| risk | likelihood | mitigation |
|---|---|---|
| Argon2id too slow on older hardware | low | reduce memory param to 32MB if >1s on target machine |
| `x/term` not working in non-TTY (CI, piped input) | medium | add `--password-stdin` flag as fallback |
| vault file corruption on interrupted write | low | atomic write (temp + rename) prevents partial writes |
| project hash collision (two repos, same 8-char prefix) | very low | use full project name in config.json as sanity check |

---

## definition of done

- [ ] `aden init / set / get / list / rm / export / run` all work end-to-end
- [ ] incorrect password produces a clear error, never a panic
- [ ] vault file contains no plaintext secret values
- [ ] binary installs to `/usr/local/bin/aden` via `make install`
- [ ] README explains usage in under 5 minutes of reading
- [ ] `go test ./...` passes