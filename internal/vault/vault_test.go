package vault

import (
	"bytes"
	"errors"
	"path/filepath"
	"testing"

	"aden/internal/crypto"
)

// Use the lighter KDF params for tests so the suite doesn't take 4s.
func init() {
	// nothing — vault.New uses crypto defaults; we test against fast
	// keys by passing short passwords and accepting the cost.
}

func newTestVault(t *testing.T, dir, password string) *Vault {
	t.Helper()
	path := filepath.Join(dir, "vault.dev.json")
	v, err := New(path, "myapp", "dev", []byte(password))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return v
}

func TestVaultRoundTrip(t *testing.T) {
	dir := t.TempDir()
	v := newTestVault(t, dir, "hunter2")

	if err := v.Set("DB_URL", "postgres://localhost/mydb"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := v.Set("STRIPE_KEY", "sk_live_supersecret"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := v.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	reloaded, err := Load(v.path, []byte("hunter2"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	got, err := reloaded.Get("DB_URL")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != "postgres://localhost/mydb" {
		t.Fatalf("Get returned %q", got)
	}
}

func TestVaultFileContainsNoPlaintext(t *testing.T) {
	dir := t.TempDir()
	v := newTestVault(t, dir, "hunter2")
	secret := "sk_live_DO_NOT_LEAK"
	if err := v.Set("STRIPE_KEY", secret); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := v.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}
	raw, err := ReadRaw(v.path)
	if err != nil {
		t.Fatalf("ReadRaw: %v", err)
	}
	if bytes.Contains(raw, []byte(secret)) {
		t.Fatalf("vault file leaks plaintext value")
	}
	// Key names are expected to be plaintext.
	if !bytes.Contains(raw, []byte("STRIPE_KEY")) {
		t.Fatalf("vault file does not contain key name")
	}
}

func TestVaultWrongPassword(t *testing.T) {
	dir := t.TempDir()
	v := newTestVault(t, dir, "right")
	if err := v.Set("X", "y"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := v.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	reloaded, err := Load(v.path, []byte("wrong"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if _, err := reloaded.Get("X"); !errors.Is(err, crypto.ErrDecryptFailed) {
		t.Fatalf("expected ErrDecryptFailed, got %v", err)
	}
}

func TestVaultMissingKey(t *testing.T) {
	dir := t.TempDir()
	v := newTestVault(t, dir, "p")
	if _, err := v.Get("MISSING"); !errors.Is(err, ErrKeyNotFound) {
		t.Fatalf("expected ErrKeyNotFound, got %v", err)
	}
	if err := v.Delete("MISSING"); !errors.Is(err, ErrKeyNotFound) {
		t.Fatalf("Delete: expected ErrKeyNotFound, got %v", err)
	}
}

func TestVaultListAndCount(t *testing.T) {
	dir := t.TempDir()
	v := newTestVault(t, dir, "p")
	_ = v.Set("B", "1")
	_ = v.Set("A", "2")
	names := v.Names()
	if len(names) != 2 || names[0] != "A" || names[1] != "B" {
		t.Fatalf("Names not sorted: %v", names)
	}
	if v.Count() != 2 {
		t.Fatalf("Count = %d, want 2", v.Count())
	}
}

func TestVaultLoadMissingFile(t *testing.T) {
	if _, err := Load(filepath.Join(t.TempDir(), "nope.json"), []byte("x")); !errors.Is(err, ErrVaultNotFound) {
		t.Fatalf("expected ErrVaultNotFound, got %v", err)
	}
}

func TestProjectIdentityFolderFallback(t *testing.T) {
	// Use an isolated working dir that isn't a git repo so we exercise
	// the folder-name fallback path. We can't easily chdir in tests
	// without races, so just sanity-check the helper directly.
	name := sanitize("My Repo Name")
	if name != "my-repo-name" {
		t.Fatalf("sanitize = %q", name)
	}
}

func TestRepoNameFromRemote(t *testing.T) {
	cases := map[string]string{
		"git@github.com:user/myapp.git":  "myapp",
		"https://github.com/user/myapp":  "myapp",
		"ssh://git@host/team/svc.git":    "svc",
		"https://gitlab.com/g/sub/proj/": "proj",
	}
	for in, want := range cases {
		if got := repoNameFromRemote(in); got != want {
			t.Errorf("repoNameFromRemote(%q) = %q, want %q", in, got, want)
		}
	}
}
