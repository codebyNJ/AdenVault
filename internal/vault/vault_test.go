package vault

import (
	"bytes"
	"errors"
	"path/filepath"
	"testing"

	"aden/internal/crypto"
)

func newTestVault(t *testing.T, dir, password string) *Vault {
	t.Helper()
	path := filepath.Join(dir, "vault.dev.json")
	v, err := New(path, "myapp", "dev", []byte(password))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return v
}

func sampleEntry(label string) EntryData {
	return EntryData{
		Label:    label,
		Username: "john@example.com",
		Password: "hunter2",
		URL:      "https://example.com",
		Notes:    "backup codes: 111222",
	}
}

func TestVaultRoundTrip(t *testing.T) {
	dir := t.TempDir()
	v := newTestVault(t, dir, "masterpassword")

	if err := v.Add(sampleEntry("github")); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := v.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	reloaded, err := Load(v.path, []byte("masterpassword"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	got, err := reloaded.Get("github")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Username != "john@example.com" {
		t.Fatalf("Username = %q", got.Username)
	}
	if got.Password != "hunter2" {
		t.Fatalf("Password = %q", got.Password)
	}
	if got.URL != "https://example.com" {
		t.Fatalf("URL = %q", got.URL)
	}
}

func TestVaultFileContainsNoPlaintext(t *testing.T) {
	dir := t.TempDir()
	v := newTestVault(t, dir, "masterpassword")
	if err := v.Add(sampleEntry("github")); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := v.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}
	raw, err := ReadRaw(v.path)
	if err != nil {
		t.Fatalf("ReadRaw: %v", err)
	}
	for _, secret := range []string{"hunter2", "john@example.com", "https://example.com", "backup codes"} {
		if bytes.Contains(raw, []byte(secret)) {
			t.Fatalf("vault file leaks plaintext: %q", secret)
		}
	}
	// Label stays plaintext so list works without a password.
	if !bytes.Contains(raw, []byte("github")) {
		t.Fatal("vault file missing plaintext label")
	}
}

func TestVaultWrongPassword(t *testing.T) {
	dir := t.TempDir()
	v := newTestVault(t, dir, "right")
	_ = v.Add(sampleEntry("github"))
	_ = v.Save()

	bad, _ := Load(v.path, []byte("wrong"))
	if _, err := bad.Get("github"); !errors.Is(err, crypto.ErrDecryptFailed) {
		t.Fatalf("expected ErrDecryptFailed, got %v", err)
	}
}

func TestVaultEntryNotFound(t *testing.T) {
	dir := t.TempDir()
	v := newTestVault(t, dir, "p")
	if _, err := v.Get("missing"); !errors.Is(err, ErrEntryNotFound) {
		t.Fatalf("expected ErrEntryNotFound, got %v", err)
	}
	if err := v.Delete("missing"); !errors.Is(err, ErrEntryNotFound) {
		t.Fatalf("Delete: expected ErrEntryNotFound, got %v", err)
	}
}

func TestVaultCountAndLabels(t *testing.T) {
	dir := t.TempDir()
	v := newTestVault(t, dir, "p")
	_ = v.Add(EntryData{Label: "b", Password: "x"})
	_ = v.Add(EntryData{Label: "a", Password: "y"})
	if v.Count() != 2 {
		t.Fatalf("Count = %d, want 2", v.Count())
	}
	labels := v.Labels()
	if labels[0] != "a" || labels[1] != "b" {
		t.Fatalf("Labels not sorted: %v", labels)
	}
}

func TestVaultLoadMissing(t *testing.T) {
	if _, err := Load(filepath.Join(t.TempDir(), "nope.json"), []byte("x")); !errors.Is(err, ErrVaultNotFound) {
		t.Fatalf("expected ErrVaultNotFound, got %v", err)
	}
}
