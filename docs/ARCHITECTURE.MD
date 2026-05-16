# aden — architecture

## overview

`aden` is a local-first CLI secrets manager. It encrypts secrets at rest using AES-256-GCM with an Argon2id-derived key, stores them in a per-project vault on disk, and injects them into subprocesses or exports them as `.env` files. There is no server, no sync, and no network dependency.

---

## system boundaries

```
┌─────────────────────────────────────────────┐
│                  user shell                  │
│                                              │
│   aden set DB_URL postgres://...             │
│   aden run -- npm start                      │
└────────────────┬────────────────────────────┘
                 │ CLI invocation
┌────────────────▼────────────────────────────┐
│               aden binary                    │
│                                              │
│  ┌──────────┐  ┌──────────┐  ┌───────────┐  │
│  │  cmd/    │  │ internal/│  │ internal/ │  │
│  │  cobra   │→ │  vault   │→ │  crypto   │  │
│  │  layer   │  │  layer   │  │  layer    │  │
│  └──────────┘  └────┬─────┘  └───────────┘  │
└───────────────────  │  ──────────────────────┘
                      │ read/write
┌─────────────────────▼───────────────────────┐
│           ~/.aden/<project-id>/              │
│                                              │
│   vault.json      encrypted secrets store   │
│   config.json     project metadata           │
└─────────────────────────────────────────────┘
```

---

## directory layout

```
aden/
├── main.go                  entry point, cobra execute
├── cmd/
│   ├── root.go              root command, global flags (--env, --vault-dir)
│   ├── init.go              aden init
│   ├── set.go               aden set <KEY> <VALUE>
│   ├── get.go               aden get <KEY>
│   ├── list.go              aden list
│   ├── rm.go                aden rm <KEY>
│   ├── export.go            aden export
│   └── run.go               aden run -- <command>
├── internal/
│   ├── vault/
│   │   ├── vault.go         load, save, and mutate vault.json
│   │   └── project.go       resolve project name and vault path
│   └── crypto/
│       ├── crypto.go        AES-256-GCM encrypt/decrypt
│       └── kdf.go           Argon2id key derivation
├── Makefile
└── README.md
```

---

## layers

### 1. cmd layer (`cmd/`)

Thin cobra command handlers. Each command:

- parses flags and positional args
- prompts for the master password via `x/term` (hidden input, never echoed)
- calls into the vault layer
- formats and prints output using `fatih/color`

No crypto or file I/O lives here. Commands are the only entry point for user interaction.

### 2. vault layer (`internal/vault/`)

Owns the on-disk data model. Responsible for:

- locating the vault directory (`~/.aden/<project-id>/`)
- reading and writing `vault.json` and `config.json` atomically (write-to-temp, rename)
- providing a `Vault` struct with methods: `Get`, `Set`, `Delete`, `List`, `All`
- calling into the crypto layer to encrypt/decrypt individual values

The vault layer never touches the master password directly — it receives a derived key from the caller.

### 3. crypto layer (`internal/crypto/`)

Stateless functions. No global state, no I/O.

**Key derivation (`kdf.go`):**

```
master_password + salt (16 bytes random) → argon2id → 32-byte key
```

Parameters: `time=1, memory=64MB, threads=4`. Salt is stored in `vault.json` alongside encrypted secrets.

**Encryption (`crypto.go`):**

```
plaintext + key → AES-256-GCM → nonce (12 bytes) + ciphertext
```

Each secret gets its own random nonce. The nonce is stored alongside the ciphertext. Authentication tag is appended by GCM automatically, so tampering is detected on decrypt.

---

## vault.json schema

```json
{
  "version": 1,
  "project": "myapp",
  "environment": "dev",
  "kdf": "argon2id",
  "kdf_params": {
    "time": 1,
    "memory": 65536,
    "threads": 4,
    "key_len": 32
  },
  "salt": "<base64-encoded 16 bytes>",
  "secrets": {
    "DB_URL": {
      "nonce": "<base64-encoded 12 bytes>",
      "ciphertext": "<base64-encoded N bytes>",
      "created_at": "2026-05-16T10:00:00Z",
      "updated_at": "2026-05-16T10:00:00Z"
    }
  }
}
```

Key names are stored in plaintext (only values are encrypted). This is intentional — listing keys must not require the master password.

---

## config.json schema

```json
{
  "project": "myapp",
  "created_at": "2026-05-16T10:00:00Z"
}
```

---

## project identification

`aden` resolves the vault path by:

1. reading the `git remote get-url origin` of the current directory
2. hashing it with SHA-256 (first 8 hex chars used as the directory name)
3. falling back to the directory name if not a git repo

Example: `github.com/user/myapp` → `~/.aden/myapp-a3f9b2c1/`

This means the vault follows the repo identity, not the local path — renaming the folder doesn't break it.

---

## environment support

The `--env` flag (default: `dev`) namespaces secrets within a project. Each environment gets its own `vault.json`:

```
~/.aden/myapp-a3f9b2c1/
  vault.dev.json
  vault.staging.json
  vault.prod.json
  config.json
```

---

## master password handling

- the master password is **never written to disk**
- prompted via `golang.org/x/term` (hidden, no echo)
- held in memory only for the duration of the command
- the salt (stored in vault.json) + password → key via Argon2id
- incorrect password causes GCM authentication failure, which returns an explicit `ErrDecryptFailed` error (no oracle attack surface)

---

## subprocess injection (`aden run`)

`aden run -- <cmd> [args]` works as follows:

1. decrypt all secrets for the current project + environment
2. build `os.Environ()` and append `KEY=value` pairs (aden secrets take precedence)
3. `exec.Command(cmd, args...).Run()` with the merged env
4. exit code of the subprocess is forwarded to the shell

Secrets are never written to disk during this flow.

---

## dependencies

| package | purpose |
|---|---|
| `github.com/spf13/cobra` | CLI command framework |
| `golang.org/x/crypto/argon2` | Argon2id key derivation |
| `crypto/aes` + `crypto/cipher` | AES-256-GCM (stdlib) |
| `golang.org/x/term` | hidden password prompt |
| `github.com/fatih/color` | coloured terminal output |
| `encoding/json` | vault serialization (stdlib) |
| `crypto/rand` | secure nonce + salt generation (stdlib) |

---

## security properties

| property | mechanism |
|---|---|
| confidentiality | AES-256-GCM per-secret encryption |
| integrity | GCM authentication tag detects tampering |
| password security | Argon2id with 64MB memory — brute-force resistant |
| no secret in logs | values never printed unless explicitly requested |
| no secret on disk (unencrypted) | only encrypted ciphertext is persisted |
| local only | zero network calls, zero telemetry |

---

## what aden is not

- not a team secrets manager (no sharing, no sync)
- not a replacement for Vault, Doppler, or AWS Secrets Manager in production
- not a keychain integration (intentionally simple — file on disk, you control it)