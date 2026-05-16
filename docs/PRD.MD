# aden — product requirements document

**version:** 1.0
**status:** draft
**author:** aden project
**last updated:** 2026-05-16

---

## overview

`aden` is a local-first CLI secrets manager for developers. It stores encrypted secrets per project on the developer's own machine, and injects them into processes or exports them to `.env` files on demand.

The problem it solves: developers routinely store API keys, database URLs, and tokens in plaintext `.env` files that get accidentally committed, shared over Slack, or left in shell history. `aden` replaces that workflow with a single binary and a master password.

---

## problem statement

### the status quo

Most developers manage local secrets by:

1. storing them in plaintext `.env` files (committed by mistake more often than they should be)
2. hardcoding them in shell profiles (visible in `~/.zshrc`, shared across all projects)
3. using a SaaS secrets manager (Doppler, Infisical) which requires an account, internet access, and a team plan

None of these options is suitable for a solo developer working on a local project who wants simple, offline, encrypted secret storage.

### what developers actually need

- store a secret in one command
- retrieve it in one command
- load all secrets into a subprocess without touching `.env` files
- never worry about accidentally committing credentials
- no account, no internet, no team required

---

## goals

1. make storing and retrieving secrets as fast as using a `.env` file
2. ensure secrets are never written to disk in plaintext
3. work entirely offline — zero network calls
4. require no setup beyond installing the binary and running `aden init`
5. be simple enough to understand fully in 10 minutes

---

## non-goals

- team sharing or secret sync across machines (out of scope for v1)
- cloud backup of the vault (user's responsibility)
- replacing Vault, Doppler, or AWS Secrets Manager in production environments
- a GUI or web interface
- secret rotation policies or expiry

---

## target user

A solo developer or small team member who:

- works across multiple local projects
- manages API keys, database credentials, and tokens
- uses the terminal as their primary environment
- has been burned by committing secrets or exposing them in shell history
- does not want to pay for or depend on an external secrets service

---

## user stories

### core flows

**US-01 — initialise a vault**
As a developer starting a new project, I want to create an encrypted vault for my project so that I have a secure place to store secrets.

acceptance criteria:
- `aden init` prompts for a master password and confirmation
- a vault file is created at `~/.aden/<project-id>/vault.dev.json`
- re-running `aden init` on an existing vault warns before overwriting
- the vault file contains no plaintext data other than key names

---

**US-02 — store a secret**
As a developer, I want to store an API key securely so that it is never written to disk in plaintext.

acceptance criteria:
- `aden set STRIPE_KEY sk_live_...` prompts for the master password and saves the secret encrypted
- the value is AES-256-GCM encrypted before being written to disk
- the command confirms success: `✓ STRIPE_KEY saved`
- attempting to set a secret on an uninitialised vault returns a clear error

---

**US-03 — retrieve a secret**
As a developer, I want to retrieve a secret's value so that I can use it in a script or inspect it.

acceptance criteria:
- `aden get STRIPE_KEY` prints only the decrypted value to stdout (no labels, no colours)
- an incorrect password returns: `error: wrong password or corrupted vault`
- a missing key returns: `error: key not found: STRIPE_KEY`
- stdout contains only the value — safe to use in `$(aden get KEY)` subshell

---

**US-04 — list stored secrets**
As a developer, I want to see which secrets are stored in my vault without revealing their values.

acceptance criteria:
- `aden list` prints all key names, one per line, with the last-updated timestamp
- no password is required (key names are stored in plaintext)
- output includes a count: `3 secrets in dev vault`

---

**US-05 — delete a secret**
As a developer, I want to remove a secret I no longer need.

acceptance criteria:
- `aden rm STRIPE_KEY` prompts for the master password and removes the key
- the command confirms: `✓ STRIPE_KEY removed`
- subsequent `aden get STRIPE_KEY` returns key-not-found

---

**US-06 — export secrets as a .env file**
As a developer, I want to export all secrets as a `.env` file so that I can use them with tools that require one.

acceptance criteria:
- `aden export` prints `KEY=value` lines to stdout, one per secret
- output is safe to redirect: `aden export > .env`
- the generated `.env` file should be added to `.gitignore` — aden prints a reminder on first export
- decryption requires the master password

---

**US-07 — inject secrets into a subprocess**
As a developer, I want to run a command with my secrets available as environment variables without creating a `.env` file.

acceptance criteria:
- `aden run -- npm start` injects all secrets as env vars into the subprocess
- the subprocess inherits the current shell environment plus the decrypted secrets
- aden secrets take precedence over existing env vars with the same name
- the subprocess's exit code is forwarded to the shell
- no secrets are written to disk at any point during this flow

---

**US-08 — manage multiple environments**
As a developer, I want to maintain separate vaults for dev, staging, and prod so that I do not accidentally mix credentials.

acceptance criteria:
- `--env` flag selects the environment (default: `dev`)
- `aden --env prod set DB_URL postgres://...` stores to the prod vault
- `aden --env prod run -- ./deploy.sh` injects prod secrets
- environments are completely isolated — different encryption keys are used

---

### secondary flows

**US-09 — use in CI without a TTY**
As a developer, I want to use aden in a CI environment where there is no terminal for password input.

acceptance criteria:
- `echo $MASTER_PASSWORD | aden --password-stdin export` reads the password from stdin
- works in GitHub Actions, GitLab CI, and similar environments
- the `--password-stdin` flag is explicitly documented as a CI escape hatch, not the default flow

---

**US-10 — override vault location**
As a developer, I want to store the vault somewhere other than `~/.aden/` so that I can keep it in a backed-up location or a team shared drive.

acceptance criteria:
- `--vault-dir /path/to/dir` overrides the default vault directory
- the flag is available on all commands

---

## functional requirements

| ID | requirement | priority |
|---|---|---|
| FR-01 | secrets encrypted with AES-256-GCM before writing to disk | must have |
| FR-02 | master password derived via Argon2id (64MB memory) | must have |
| FR-03 | each secret has an independent nonce | must have |
| FR-04 | key names stored in plaintext; values always encrypted | must have |
| FR-05 | `aden run` forwards subprocess exit code | must have |
| FR-06 | `aden export` writes only to stdout | must have |
| FR-07 | vault file written atomically (temp + rename) | must have |
| FR-08 | `--env` flag for environment namespacing | must have |
| FR-09 | `--password-stdin` flag for non-TTY use | should have |
| FR-10 | `--vault-dir` flag to override default location | should have |
| FR-11 | `aden list` requires no password | should have |
| FR-12 | wrong password produces `ErrDecryptFailed`, not a panic | must have |

---

## non-functional requirements

| ID | requirement | target |
|---|---|---|
| NFR-01 | startup time | < 50ms on M-series Mac |
| NFR-02 | `aden set` end-to-end latency (including Argon2id) | < 1.5s |
| NFR-03 | binary size | < 10MB (stripped) |
| NFR-04 | platforms | macOS (arm64, amd64), Linux (amd64, arm64) |
| NFR-05 | no network calls at runtime | verified by test with network disabled |
| NFR-06 | zero third-party telemetry | hard requirement |

---

## security requirements

| ID | requirement |
|---|---|
| SEC-01 | master password never written to disk in any form |
| SEC-02 | Argon2id parameters must resist brute-force on 2026-era hardware |
| SEC-03 | GCM authentication tag must be verified before returning any plaintext |
| SEC-04 | nonce must be randomly generated per encryption operation |
| SEC-05 | salt must be randomly generated at `aden init` time |
| SEC-06 | `aden get` output must go to stdout only — never to logs or stderr |
| SEC-07 | incorrect password must not reveal whether the key exists |

---

## CLI interface specification

```
aden [--env <name>] [--vault-dir <path>] [--quiet] <command> [args]

commands:
  init                         create a new vault for this project
  set <KEY> <VALUE>            store an encrypted secret
  get <KEY>                    retrieve and print a secret
  list                         list all key names
  rm <KEY>                     delete a secret
  export                       print all secrets as KEY=value lines
  run -- <cmd> [args]          run a command with secrets in the environment

global flags:
  --env <name>                 environment name (default: dev)
  --vault-dir <path>           override default vault directory (~/.aden/)
  --password-stdin             read master password from stdin
  --quiet                      suppress status output

examples:
  aden init
  aden set DB_URL postgres://localhost/mydb
  aden get DB_URL
  aden list
  aden --env prod set STRIPE_KEY sk_live_...
  aden export > .env
  aden run -- npm start
  echo $PASS | aden --password-stdin export
```

---

## out of scope for v1

- secret versioning or history
- vault sharing or encryption for multiple recipients
- keychain / OS secure enclave integration
- web UI or REST API
- automatic `.gitignore` management
- audit logging

---

## success metrics

since `aden` is a developer tool, success is measured qualitatively at v1:

- a developer unfamiliar with the project can install and run `aden init && aden set && aden run` in under 5 minutes using only the README
- the vault file contains no recoverable plaintext values (verified by inspection)
- all 7 core commands work without panics on macOS and Linux
- `go test ./...` passes