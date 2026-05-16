package vault

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"

	"aden/internal/crypto"
)

// FormatVersion is the schema version written to new vaults.
const FormatVersion = 1

// ErrVaultNotFound is returned when the vault file does not exist.
var ErrVaultNotFound = errors.New("vault not found — run `adenV init` first")

// ErrVaultExists is returned when init would overwrite an existing
// vault and the caller hasn't confirmed.
var ErrVaultExists = errors.New("vault already exists")

// ErrKeyNotFound is returned by Get and Delete when the secret name
// is not present in the vault.
var ErrKeyNotFound = errors.New("key not found")

// Secret is the on-disk representation of a single secret. The value
// is always encrypted; only metadata is plaintext.
type Secret struct {
	Nonce      string    `json:"nonce"`      // base64
	Ciphertext string    `json:"ciphertext"` // base64
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// fileFormat is what we serialize to vault.<env>.json. Field order
// is stable for deterministic diffs.
type fileFormat struct {
	Version     int                `json:"version"`
	Project     string             `json:"project"`
	Environment string             `json:"environment"`
	KDF         string             `json:"kdf"`
	KDFParams   crypto.KDFParams   `json:"kdf_params"`
	Salt        string             `json:"salt"` // base64
	Secrets     map[string]*Secret `json:"secrets"`
}

// Config is what we serialize to config.json. It exists so that two
// repos that happen to share a vault-id prefix can be distinguished.
type Config struct {
	Project   string    `json:"project"`
	CreatedAt time.Time `json:"created_at"`
}

// Vault is an in-memory view of vault.<env>.json. All mutations happen
// in memory; callers must call Save to persist changes.
type Vault struct {
	path string
	key  []byte
	data fileFormat
}

// New creates a brand new vault in memory. It does NOT write to disk;
// callers must call Save to persist. Use this from `aden init`.
func New(path, project, environment string, password []byte) (*Vault, error) {
	salt, err := crypto.RandomSalt()
	if err != nil {
		return nil, err
	}
	params := crypto.DefaultKDFParams()
	key := crypto.DeriveKey(password, salt, params)

	return &Vault{
		path: path,
		key:  key,
		data: fileFormat{
			Version:     FormatVersion,
			Project:     project,
			Environment: environment,
			KDF:         "argon2id",
			KDFParams:   params,
			Salt:        base64.StdEncoding.EncodeToString(salt),
			Secrets:     map[string]*Secret{},
		},
	}, nil
}

// LoadEncrypted reads vault.<env>.json from disk but does NOT decrypt
// the key. Useful for `aden list` which doesn't need the master
// password.
func LoadEncrypted(path string) (*Vault, error) {
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrVaultNotFound
		}
		return nil, fmt.Errorf("open vault: %w", err)
	}
	defer f.Close()

	var data fileFormat
	if err := json.NewDecoder(f).Decode(&data); err != nil {
		return nil, fmt.Errorf("parse vault: %w", err)
	}
	if data.Secrets == nil {
		data.Secrets = map[string]*Secret{}
	}
	return &Vault{path: path, data: data}, nil
}

// Load reads vault.<env>.json from disk and derives the encryption
// key from the supplied password. It does not yet decrypt any secret
// — that happens lazily in Get / All / Export.
func Load(path string, password []byte) (*Vault, error) {
	v, err := LoadEncrypted(path)
	if err != nil {
		return nil, err
	}
	salt, err := base64.StdEncoding.DecodeString(v.data.Salt)
	if err != nil {
		return nil, fmt.Errorf("decode salt: %w", err)
	}
	v.key = crypto.DeriveKey(password, salt, v.data.KDFParams)
	return v, nil
}

// VerifyPassword tries to decrypt one secret to confirm the password
// is correct. If the vault has no secrets yet, it returns nil — there
// is nothing to verify against, and the caller can proceed.
func (v *Vault) VerifyPassword() error {
	for _, s := range v.data.Secrets {
		nonce, err := base64.StdEncoding.DecodeString(s.Nonce)
		if err != nil {
			return fmt.Errorf("decode nonce: %w", err)
		}
		ct, err := base64.StdEncoding.DecodeString(s.Ciphertext)
		if err != nil {
			return fmt.Errorf("decode ciphertext: %w", err)
		}
		if _, err := crypto.Decrypt(v.key, nonce, ct); err != nil {
			return err
		}
		return nil
	}
	return nil
}

// Set stores (or updates) a secret. The value is encrypted immediately
// but the change is not persisted until Save is called.
func (v *Vault) Set(name, value string) error {
	nonce, ct, err := crypto.Encrypt(v.key, []byte(value))
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	existing := v.data.Secrets[name]
	created := now
	if existing != nil {
		created = existing.CreatedAt
	}
	v.data.Secrets[name] = &Secret{
		Nonce:      base64.StdEncoding.EncodeToString(nonce),
		Ciphertext: base64.StdEncoding.EncodeToString(ct),
		CreatedAt:  created,
		UpdatedAt:  now,
	}
	return nil
}

// Get returns the decrypted value for name, or ErrKeyNotFound /
// crypto.ErrDecryptFailed.
func (v *Vault) Get(name string) (string, error) {
	s, ok := v.data.Secrets[name]
	if !ok {
		return "", ErrKeyNotFound
	}
	nonce, err := base64.StdEncoding.DecodeString(s.Nonce)
	if err != nil {
		return "", fmt.Errorf("decode nonce: %w", err)
	}
	ct, err := base64.StdEncoding.DecodeString(s.Ciphertext)
	if err != nil {
		return "", fmt.Errorf("decode ciphertext: %w", err)
	}
	pt, err := crypto.Decrypt(v.key, nonce, ct)
	if err != nil {
		return "", err
	}
	return string(pt), nil
}

// Delete removes a secret. Returns ErrKeyNotFound if name is absent.
func (v *Vault) Delete(name string) error {
	if _, ok := v.data.Secrets[name]; !ok {
		return ErrKeyNotFound
	}
	delete(v.data.Secrets, name)
	return nil
}

// Names returns the secret names in sorted order. No password needed.
func (v *Vault) Names() []string {
	out := make([]string, 0, len(v.data.Secrets))
	for k := range v.data.Secrets {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// Entry is a public, plaintext view of a secret's metadata. The Value
// is empty unless populated by All.
type Entry struct {
	Name      string
	Value     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Metadata returns the plaintext metadata for all secrets, in sorted
// order. No decryption is performed.
func (v *Vault) Metadata() []Entry {
	out := make([]Entry, 0, len(v.data.Secrets))
	for name, s := range v.data.Secrets {
		out = append(out, Entry{
			Name:      name,
			CreatedAt: s.CreatedAt,
			UpdatedAt: s.UpdatedAt,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// All decrypts every secret and returns them as a slice. The returned
// slice is sorted by name. If decryption of any secret fails, the
// first error is returned.
func (v *Vault) All() ([]Entry, error) {
	entries := v.Metadata()
	for i := range entries {
		s := v.data.Secrets[entries[i].Name]
		nonce, err := base64.StdEncoding.DecodeString(s.Nonce)
		if err != nil {
			return nil, fmt.Errorf("decode nonce for %q: %w", entries[i].Name, err)
		}
		ct, err := base64.StdEncoding.DecodeString(s.Ciphertext)
		if err != nil {
			return nil, fmt.Errorf("decode ciphertext for %q: %w", entries[i].Name, err)
		}
		pt, err := crypto.Decrypt(v.key, nonce, ct)
		if err != nil {
			return nil, err
		}
		entries[i].Value = string(pt)
	}
	return entries, nil
}

// Project returns the project name stored in the vault.
func (v *Vault) Project() string { return v.data.Project }

// Environment returns the environment name stored in the vault.
func (v *Vault) Environment() string { return v.data.Environment }

// Count returns the number of secrets in the vault.
func (v *Vault) Count() int { return len(v.data.Secrets) }

// Save writes the vault to disk atomically (write to temp file, fsync,
// rename). Parent directories are created with 0700 permissions.
func (v *Vault) Save() error {
	dir := filepath.Dir(v.path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create vault dir: %w", err)
	}

	tmp, err := os.CreateTemp(dir, ".vault-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	tmpPath := tmp.Name()
	// Clean up the temp file on any failure path.
	defer func() {
		_ = os.Remove(tmpPath)
	}()

	enc := json.NewEncoder(tmp)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v.data); err != nil {
		tmp.Close()
		return fmt.Errorf("encode vault: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("sync vault: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close vault: %w", err)
	}
	if err := os.Chmod(tmpPath, 0o600); err != nil {
		return fmt.Errorf("chmod vault: %w", err)
	}
	if err := os.Rename(tmpPath, v.path); err != nil {
		return fmt.Errorf("rename vault: %w", err)
	}
	return nil
}

// SaveConfig writes config.json next to the vault. It is idempotent
// — repeated calls don't bump CreatedAt.
func SaveConfig(p Project) error {
	if err := os.MkdirAll(p.Dir, 0o700); err != nil {
		return fmt.Errorf("create project dir: %w", err)
	}
	path := p.ConfigPath()
	if f, err := os.Open(path); err == nil {
		// Already exists — leave it alone.
		_ = f.Close()
		return nil
	}
	cfg := Config{Project: p.Name, CreatedAt: time.Now().UTC()}
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o600)
}

// Exists reports whether the vault file at path is present.
func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// CopyData is a defensive helper used in tests to verify that no
// plaintext value leaks into the on-disk representation.
func ReadRaw(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(f)
}
