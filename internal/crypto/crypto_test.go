package crypto

import (
	"bytes"
	"errors"
	"testing"
)

func deriveTestKey(t *testing.T, password string) ([]byte, []byte) {
	t.Helper()
	salt, err := RandomSalt()
	if err != nil {
		t.Fatalf("RandomSalt: %v", err)
	}
	// Use a faster KDF in tests so the suite runs in well under a second.
	p := KDFParams{Time: 1, Memory: 8 * 1024, Threads: 2, KeyLen: KeyLen}
	return DeriveKey([]byte(password), salt, p), salt
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	key, _ := deriveTestKey(t, "correct horse battery staple")
	plaintext := []byte("sk_live_supersecret")

	nonce, ct, err := Encrypt(key, plaintext)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if bytes.Contains(ct, plaintext) {
		t.Fatal("ciphertext contains plaintext — encryption is a no-op")
	}

	got, err := Decrypt(key, nonce, ct)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if !bytes.Equal(got, plaintext) {
		t.Fatalf("round-trip mismatch: got %q want %q", got, plaintext)
	}
}

func TestDecryptWrongKey(t *testing.T) {
	key1, _ := deriveTestKey(t, "password-one")
	key2, _ := deriveTestKey(t, "password-two")

	nonce, ct, err := Encrypt(key1, []byte("hello"))
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	if _, err := Decrypt(key2, nonce, ct); !errors.Is(err, ErrDecryptFailed) {
		t.Fatalf("expected ErrDecryptFailed, got %v", err)
	}
}

func TestDecryptTamperedCiphertext(t *testing.T) {
	key, _ := deriveTestKey(t, "any-password")

	nonce, ct, err := Encrypt(key, []byte("hello"))
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	ct[0] ^= 0xff // flip a bit

	if _, err := Decrypt(key, nonce, ct); !errors.Is(err, ErrDecryptFailed) {
		t.Fatalf("expected ErrDecryptFailed, got %v", err)
	}
}

func TestNonceIsUniquePerEncryption(t *testing.T) {
	key, _ := deriveTestKey(t, "any-password")

	n1, _, err := Encrypt(key, []byte("a"))
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	n2, _, err := Encrypt(key, []byte("a"))
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if bytes.Equal(n1, n2) {
		t.Fatal("two encryptions produced the same nonce — nonce is not random")
	}
}

func TestKeyWrongLengthRejected(t *testing.T) {
	short := make([]byte, 8)
	if _, _, err := Encrypt(short, []byte("x")); err == nil {
		t.Fatal("Encrypt accepted short key")
	}
	if _, err := Decrypt(short, make([]byte, NonceLen), []byte("x")); err == nil {
		t.Fatal("Decrypt accepted short key")
	}
}
