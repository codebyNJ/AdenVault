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

const FormatVersion = 2

var ErrVaultNotFound = errors.New("vault not found — run `adenV init` first")
var ErrVaultExists = errors.New("vault already exists")
var ErrEntryNotFound = errors.New("entry not found")

// EncryptedField holds one individually encrypted value.
// Nonce and Ciphertext are base64-encoded.
type EncryptedField struct {
	Nonce      string `json:"nonce"`
	Ciphertext string `json:"ciphertext"`
}

// Entry is one item in the personal vault: a label you recognise
// ("github", "netflix") with an encrypted username, password, optional
// URL, and optional notes.
type Entry struct {
	Username  *EncryptedField `json:"username,omitempty"`
	Password  *EncryptedField `json:"password,omitempty"`
	URL       *EncryptedField `json:"url,omitempty"`
	Notes     *EncryptedField `json:"notes,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

// EntryData is the fully-decrypted, in-memory view of an Entry.
// Never write this to disk.
type EntryData struct {
	Label     string
	Username  string
	Password  string
	URL       string
	Notes     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// EntryMeta is the plaintext metadata (no decryption needed).
type EntryMeta struct {
	Label     string
	HasUser   bool
	HasURL    bool
	HasNotes  bool
	UpdatedAt time.Time
	CreatedAt time.Time
}

type fileFormat struct {
	Version     int              `json:"version"`
	Project     string           `json:"project"`
	Environment string           `json:"environment"`
	KDF         string           `json:"kdf"`
	KDFParams   crypto.KDFParams `json:"kdf_params"`
	Salt        string           `json:"salt"`
	Entries     map[string]*Entry `json:"entries"`
}

// Config is written to config.json alongside the vault.
type Config struct {
	Project   string    `json:"project"`
	CreatedAt time.Time `json:"created_at"`
}

// Vault is an in-memory view of one vault file.
type Vault struct {
	path string
	key  []byte
	data fileFormat
}

// New creates a brand-new, empty vault in memory. Call Save to persist.
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
			Entries:     map[string]*Entry{},
		},
	}, nil
}

// LoadEncrypted reads the vault from disk without decrypting values.
// Used by list and the splash (no password required).
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
	if data.Entries == nil {
		data.Entries = map[string]*Entry{}
	}
	return &Vault{path: path, data: data}, nil
}

// Load reads the vault and derives the key from the password.
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

// VerifyPassword decrypts one field from an arbitrary entry to confirm
// the password is correct. Returns nil if the vault is empty.
func (v *Vault) VerifyPassword() error {
	for _, e := range v.data.Entries {
		f := e.Password
		if f == nil {
			f = e.Username
		}
		if f == nil {
			continue
		}
		if _, err := v.decryptField(f); err != nil {
			return err
		}
		return nil
	}
	return nil
}

// ──────────────────────────────────────────────────────
// Mutations (in-memory; call Save to persist)
// ──────────────────────────────────────────────────────

// Add stores (or replaces) an entry. All non-empty fields are
// individually encrypted before being written.
func (v *Vault) Add(d EntryData) error {
	now := time.Now().UTC()
	existing := v.data.Entries[d.Label]
	created := now
	if existing != nil {
		created = existing.CreatedAt
	}

	e := &Entry{CreatedAt: created, UpdatedAt: now}

	var err error
	if e.Username, err = v.encryptField(d.Username); err != nil {
		return err
	}
	if e.Password, err = v.encryptField(d.Password); err != nil {
		return err
	}
	if d.URL != "" {
		if e.URL, err = v.encryptField(d.URL); err != nil {
			return err
		}
	}
	if d.Notes != "" {
		if e.Notes, err = v.encryptField(d.Notes); err != nil {
			return err
		}
	}

	v.data.Entries[d.Label] = e
	return nil
}

// Get decrypts and returns one entry. Returns ErrEntryNotFound if the
// label is absent.
func (v *Vault) Get(label string) (EntryData, error) {
	e, ok := v.data.Entries[label]
	if !ok {
		return EntryData{}, ErrEntryNotFound
	}
	d := EntryData{
		Label:     label,
		CreatedAt: e.CreatedAt,
		UpdatedAt: e.UpdatedAt,
	}
	var err error
	if d.Username, err = v.decryptFieldStr(e.Username); err != nil {
		return EntryData{}, err
	}
	if d.Password, err = v.decryptFieldStr(e.Password); err != nil {
		return EntryData{}, err
	}
	if d.URL, err = v.decryptFieldStr(e.URL); err != nil {
		return EntryData{}, err
	}
	if d.Notes, err = v.decryptFieldStr(e.Notes); err != nil {
		return EntryData{}, err
	}
	return d, nil
}

// Delete removes an entry. Returns ErrEntryNotFound if the label is absent.
func (v *Vault) Delete(label string) error {
	if _, ok := v.data.Entries[label]; !ok {
		return ErrEntryNotFound
	}
	delete(v.data.Entries, label)
	return nil
}

// ──────────────────────────────────────────────────────
// Read-only views
// ──────────────────────────────────────────────────────

// Labels returns all entry labels in sorted order. No password needed.
func (v *Vault) Labels() []string {
	out := make([]string, 0, len(v.data.Entries))
	for k := range v.data.Entries {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// Metas returns plaintext metadata for all entries (no decryption).
func (v *Vault) Metas() []EntryMeta {
	out := make([]EntryMeta, 0, len(v.data.Entries))
	for label, e := range v.data.Entries {
		out = append(out, EntryMeta{
			Label:     label,
			HasUser:   e.Username != nil,
			HasURL:    e.URL != nil,
			HasNotes:  e.Notes != nil,
			UpdatedAt: e.UpdatedAt,
			CreatedAt: e.CreatedAt,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Label < out[j].Label })
	return out
}

// All decrypts every entry and returns them sorted by label.
func (v *Vault) All() ([]EntryData, error) {
	labels := v.Labels()
	out := make([]EntryData, 0, len(labels))
	for _, l := range labels {
		d, err := v.Get(l)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, nil
}

// Count returns the number of entries.
func (v *Vault) Count() int { return len(v.data.Entries) }

// Project returns the vault's project name.
func (v *Vault) Project() string { return v.data.Project }

// Environment returns the vault's environment name.
func (v *Vault) Environment() string { return v.data.Environment }

// ──────────────────────────────────────────────────────
// Persistence
// ──────────────────────────────────────────────────────

// Save writes the vault atomically (temp + fsync + rename).
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
	defer func() { _ = os.Remove(tmpPath) }()

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
	return os.Rename(tmpPath, v.path)
}

// SaveConfig writes config.json; idempotent.
func SaveConfig(p Project) error {
	if err := os.MkdirAll(p.Dir, 0o700); err != nil {
		return fmt.Errorf("create project dir: %w", err)
	}
	path := p.ConfigPath()
	if f, err := os.Open(path); err == nil {
		f.Close()
		return nil
	}
	cfg := Config{Project: p.Name, CreatedAt: time.Now().UTC()}
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o600)
}

// Exists reports whether the vault file is present.
func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func ReadRaw(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(f)
}

// ──────────────────────────────────────────────────────
// Internal crypto helpers
// ──────────────────────────────────────────────────────

func (v *Vault) encryptField(value string) (*EncryptedField, error) {
	nonce, ct, err := crypto.Encrypt(v.key, []byte(value))
	if err != nil {
		return nil, err
	}
	return &EncryptedField{
		Nonce:      base64.StdEncoding.EncodeToString(nonce),
		Ciphertext: base64.StdEncoding.EncodeToString(ct),
	}, nil
}

func (v *Vault) decryptField(f *EncryptedField) ([]byte, error) {
	nonce, err := base64.StdEncoding.DecodeString(f.Nonce)
	if err != nil {
		return nil, fmt.Errorf("decode nonce: %w", err)
	}
	ct, err := base64.StdEncoding.DecodeString(f.Ciphertext)
	if err != nil {
		return nil, fmt.Errorf("decode ciphertext: %w", err)
	}
	return crypto.Decrypt(v.key, nonce, ct)
}

func (v *Vault) decryptFieldStr(f *EncryptedField) (string, error) {
	if f == nil {
		return "", nil
	}
	b, err := v.decryptField(f)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
