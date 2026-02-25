package crypto

import (
	"crypto/rand"
	"crypto/sha512"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/curve25519"
	"golang.org/x/crypto/nacl/box"
)

// KeyPair holds a NaCl box keypair derived from a PIN.
type KeyPair struct {
	PublicKey  [32]byte
	PrivateKey [32]byte
}

// EncryptedPrefix is the prefix prepended to every E2E-encrypted field value.
const EncryptedPrefix = "e2e::"

// verifyPlaintext is the literal string encrypted inside the verifier.
const verifyPlaintext = "ravi-e2e-verify"

// Argon2id parameters -- must match libsodium's crypto_pwhash exactly.
// libsodium hardcodes parallelism=1 internally; the caller only controls
// opslimit (time) and memlimit (memory).
const (
	argon2Time    = 3
	argon2Memory  = 64 * 1024 // 64 MB expressed in KiB (the Go argon2 API uses KiB)
	argon2Threads = 1         // must be 1 to match libsodium's crypto_pwhash
	argon2KeyLen  = 32
)

// DeriveKeyPair derives a NaCl keypair from a PIN and salt using Argon2id.
//
// The derivation replicates libsodium's crypto_box_seed_keypair:
//  1. Argon2id(PIN, salt) -> 32-byte seed
//  2. SHA-512(seed) -> 64-byte hash
//  3. Take first 32 bytes, apply Curve25519 clamping
//  4. Scalar base multiplication -> public key
//
// The salt must be the raw 16-byte value (base64-decoded) stored on the server.
func DeriveKeyPair(pin string, salt []byte) (*KeyPair, error) {
	seed := argon2.IDKey([]byte(pin), salt, argon2Time, argon2Memory, argon2Threads, argon2KeyLen)

	// Replicate libsodium's crypto_box_seed_keypair:
	// 1. SHA-512 hash the seed
	hash := sha512.Sum512(seed)

	// 2. Take first 32 bytes and apply Curve25519 clamping
	var privateKey [32]byte
	copy(privateKey[:], hash[:32])
	privateKey[0] &= 248
	privateKey[31] &= 127
	privateKey[31] |= 64

	// 3. Derive public key via scalar base multiplication
	publicKey, err := curve25519.X25519(privateKey[:], curve25519.Basepoint)
	if err != nil {
		return nil, fmt.Errorf("deriving public key: %w", err)
	}

	var kp KeyPair
	copy(kp.PrivateKey[:], privateKey[:])
	copy(kp.PublicKey[:], publicKey)

	return &kp, nil
}

// Decrypt decrypts a NaCl SealedBox ciphertext using the keypair.
// The ciphertext must be the raw bytes (not base64-encoded, no prefix).
func Decrypt(ciphertext []byte, kp *KeyPair) ([]byte, error) {
	plaintext, ok := box.OpenAnonymous(nil, ciphertext, &kp.PublicKey, &kp.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("decryption failed: invalid ciphertext or wrong key")
	}
	return plaintext, nil
}

// DecryptField decrypts an "e2e::<base64>" string, returning the plaintext.
// If the value does not carry the encrypted prefix it is returned unchanged.
func DecryptField(value string, kp *KeyPair) (string, error) {
	if !IsEncrypted(value) {
		return value, nil
	}

	b64 := strings.TrimPrefix(value, EncryptedPrefix)
	ciphertext, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return "", fmt.Errorf("decoding base64 ciphertext: %w", err)
	}

	plaintext, err := Decrypt(ciphertext, kp)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

// IsEncrypted reports whether value carries the "e2e::" prefix.
func IsEncrypted(value string) bool {
	return strings.HasPrefix(value, EncryptedPrefix)
}

// Verify checks that a keypair can decrypt the server-stored verifier.
// verifierB64 is the base64-encoded SealedBox ciphertext of "ravi-e2e-verify".
func Verify(kp *KeyPair, verifierB64 string) bool {
	ciphertext, err := base64.StdEncoding.DecodeString(verifierB64)
	if err != nil {
		return false
	}

	plaintext, err := Decrypt(ciphertext, kp)
	if err != nil {
		return false
	}
	return string(plaintext) == verifyPlaintext
}

// CreateVerifier encrypts the literal "ravi-e2e-verify" with the public key
// and returns the base64-encoded ciphertext.
func CreateVerifier(kp *KeyPair) (string, error) {
	ciphertext, err := box.SealAnonymous(nil, []byte(verifyPlaintext), &kp.PublicKey, rand.Reader)
	if err != nil {
		return "", fmt.Errorf("creating verifier: %w", err)
	}
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Encrypt encrypts plaintext using NaCl SealedBox with the given public key.
// Returns an "e2e::<base64>" string. Empty plaintext returns empty string.
func Encrypt(plaintext string, publicKeyB64 string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	pubBytes, err := base64.StdEncoding.DecodeString(publicKeyB64)
	if err != nil {
		return "", fmt.Errorf("decoding public key: %w", err)
	}
	if len(pubBytes) != 32 {
		return "", fmt.Errorf("public key has invalid length %d, expected 32", len(pubBytes))
	}

	var pubKey [32]byte
	copy(pubKey[:], pubBytes)

	ciphertext, err := box.SealAnonymous(nil, []byte(plaintext), &pubKey, rand.Reader)
	if err != nil {
		return "", fmt.Errorf("encrypting: %w", err)
	}

	return EncryptedPrefix + base64.StdEncoding.EncodeToString(ciphertext), nil
}
