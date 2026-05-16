package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
)

// ErrDecryptFailed is returned whenever GCM authentication fails. It
// covers both wrong-password and tampered-ciphertext cases — by design
// the two are indistinguishable to avoid an oracle attack surface.
var ErrDecryptFailed = errors.New("wrong password or corrupted vault")

// Encrypt seals plaintext with AES-256-GCM. The returned nonce is
// randomly generated per call; callers must store it alongside the
// ciphertext so Decrypt can reverse the operation.
func Encrypt(key, plaintext []byte) (nonce, ciphertext []byte, err error) {
	if len(key) != int(KeyLen) {
		return nil, nil, fmt.Errorf("encrypt: key must be %d bytes, got %d", KeyLen, len(key))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, fmt.Errorf("encrypt: new cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, fmt.Errorf("encrypt: new gcm: %w", err)
	}

	nonce = make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, nil, fmt.Errorf("encrypt: generate nonce: %w", err)
	}

	ciphertext = gcm.Seal(nil, nonce, plaintext, nil)
	return nonce, ciphertext, nil
}

// Decrypt opens a GCM-sealed ciphertext. Any authentication failure
// (wrong key, tampered ciphertext, wrong nonce) is mapped to
// ErrDecryptFailed so callers cannot tell the cases apart.
func Decrypt(key, nonce, ciphertext []byte) ([]byte, error) {
	if len(key) != int(KeyLen) {
		return nil, fmt.Errorf("decrypt: key must be %d bytes, got %d", KeyLen, len(key))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("decrypt: new cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("decrypt: new gcm: %w", err)
	}

	if len(nonce) != gcm.NonceSize() {
		return nil, ErrDecryptFailed
	}

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, ErrDecryptFailed
	}
	return plaintext, nil
}
