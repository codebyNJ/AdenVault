// Package crypto provides stateless key-derivation and authenticated
// encryption primitives used by the aden vault.
//
// The package never touches the filesystem and never holds long-lived
// state. Callers are responsible for zeroing derived keys and salts.
package crypto

import (
	"crypto/rand"
	"fmt"

	"golang.org/x/crypto/argon2"
)

// KDF parameters. These are the defaults written into every new vault.
// They are tuned to resist brute-force on 2026-era hardware while
// finishing in well under 1.5s on a developer laptop.
const (
	KDFTime    uint32 = 1
	KDFMemory  uint32 = 64 * 1024 // 64 MiB
	KDFThreads uint8  = 4
	KeyLen     uint32 = 32
	SaltLen           = 16
	NonceLen          = 12
)

// KDFParams describes the parameters used to derive a key. They are
// persisted in vault.json so future versions of aden can read older
// vaults even if defaults change.
type KDFParams struct {
	Time    uint32 `json:"time"`
	Memory  uint32 `json:"memory"`
	Threads uint8  `json:"threads"`
	KeyLen  uint32 `json:"key_len"`
}

// DefaultKDFParams returns the parameters used when initialising a new
// vault.
func DefaultKDFParams() KDFParams {
	return KDFParams{
		Time:    KDFTime,
		Memory:  KDFMemory,
		Threads: KDFThreads,
		KeyLen:  KeyLen,
	}
}

// DeriveKey runs Argon2id over the password + salt with the supplied
// parameters and returns a key of the requested length.
func DeriveKey(password, salt []byte, p KDFParams) []byte {
	return argon2.IDKey(password, salt, p.Time, p.Memory, p.Threads, p.KeyLen)
}

// RandomSalt returns SaltLen cryptographically-random bytes.
func RandomSalt() ([]byte, error) {
	salt := make([]byte, SaltLen)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("generate salt: %w", err)
	}
	return salt, nil
}
