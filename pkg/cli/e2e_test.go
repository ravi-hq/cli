package cli

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ravi-hq/cli/internal/config"
	"github.com/ravi-hq/cli/internal/crypto"
	"golang.org/x/crypto/nacl/box"
)

// saveTestAuth writes auth config to the temp home's ~/.ravi/auth.json.
func saveTestAuth(t *testing.T, tmpDir string, auth *config.AuthConfig) {
	t.Helper()

	raviDir := filepath.Join(tmpDir, ".ravi")
	if err := os.MkdirAll(raviDir, 0700); err != nil {
		t.Fatalf("Failed to create .ravi directory: %v", err)
	}

	data, err := json.MarshalIndent(auth, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal auth config: %v", err)
	}

	authPath := filepath.Join(raviDir, "auth.json")
	if err := os.WriteFile(authPath, data, 0600); err != nil {
		t.Fatalf("Failed to write auth file: %v", err)
	}
}

// deriveTestKeyPair generates a real KeyPair using crypto.DeriveKeyPair with
// random salt and a fixed PIN. Returns the keypair plus base64-encoded keys
// ready for storage in config.
func deriveTestKeyPair(t *testing.T) (kp *crypto.KeyPair, privB64, pubB64 string) {
	t.Helper()

	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		t.Fatalf("generating salt: %v", err)
	}

	kp, err := crypto.DeriveKeyPair("123456", salt)
	if err != nil {
		t.Fatalf("DeriveKeyPair: %v", err)
	}

	privB64 = base64.StdEncoding.EncodeToString(kp.PrivateKey[:])
	pubB64 = base64.StdEncoding.EncodeToString(kp.PublicKey[:])
	return kp, privB64, pubB64
}

// TestEnsureKeyPair_ValidConfig verifies that ensureKeyPair returns a valid
// KeyPair when the config contains properly base64-encoded keys.
func TestEnsureKeyPair_ValidConfig(t *testing.T) {
	tmpDir, cleanup := withTempHome(t)
	defer cleanup()

	originalKP, privB64, pubB64 := deriveTestKeyPair(t)

	saveTestAuth(t, tmpDir, &config.AuthConfig{
		PrivateKey: privB64,
		PublicKey:  pubB64,
	})

	kp, err := ensureKeyPair()
	if err != nil {
		t.Fatalf("ensureKeyPair() error = %v, want nil", err)
	}
	if kp == nil {
		t.Fatal("ensureKeyPair() returned nil KeyPair")
	}

	if kp.PrivateKey != originalKP.PrivateKey {
		t.Error("returned PrivateKey does not match the original")
	}
	if kp.PublicKey != originalKP.PublicKey {
		t.Error("returned PublicKey does not match the original")
	}
}

// TestEnsureKeyPair_MissingPrivateKey verifies that an error is returned when
// the config has a PublicKey but no PrivateKey.
func TestEnsureKeyPair_MissingPrivateKey(t *testing.T) {
	tmpDir, cleanup := withTempHome(t)
	defer cleanup()

	_, _, pubB64 := deriveTestKeyPair(t)

	saveTestAuth(t, tmpDir, &config.AuthConfig{
		PrivateKey: "",
		PublicKey:  pubB64,
	})

	_, err := ensureKeyPair()
	if err == nil {
		t.Fatal("ensureKeyPair() error = nil, want error for missing PrivateKey")
	}
	if !strings.Contains(err.Error(), "not authenticated") {
		t.Errorf("ensureKeyPair() error = %v, want error containing 'not authenticated'", err)
	}
}

// TestEnsureKeyPair_MissingPublicKey verifies that an error is returned when
// the config has a PrivateKey but no PublicKey.
func TestEnsureKeyPair_MissingPublicKey(t *testing.T) {
	tmpDir, cleanup := withTempHome(t)
	defer cleanup()

	_, privB64, _ := deriveTestKeyPair(t)

	saveTestAuth(t, tmpDir, &config.AuthConfig{
		PrivateKey: privB64,
		PublicKey:  "",
	})

	_, err := ensureKeyPair()
	if err == nil {
		t.Fatal("ensureKeyPair() error = nil, want error for missing PublicKey")
	}
	if !strings.Contains(err.Error(), "not authenticated") {
		t.Errorf("ensureKeyPair() error = %v, want error containing 'not authenticated'", err)
	}
}

// TestEnsureKeyPair_EmptyConfig verifies that a fresh config with no keys
// produces the "not authenticated" error.
func TestEnsureKeyPair_EmptyConfig(t *testing.T) {
	tmpDir, cleanup := withTempHome(t)
	defer cleanup()

	saveTestAuth(t, tmpDir, &config.AuthConfig{})

	_, err := ensureKeyPair()
	if err == nil {
		t.Fatal("ensureKeyPair() error = nil, want error for empty config")
	}
	if !strings.Contains(err.Error(), "not authenticated") {
		t.Errorf("ensureKeyPair() error = %v, want error containing 'not authenticated'", err)
	}
}

// TestEnsureKeyPair_NoConfigFile verifies the behaviour when no config file
// exists on disk (Load returns an empty Config, not an error).
func TestEnsureKeyPair_NoConfigFile(t *testing.T) {
	_, cleanup := withTempHome(t)
	defer cleanup()

	// No auth file created -- config.LoadAuth returns empty AuthConfig.
	_, err := ensureKeyPair()
	if err == nil {
		t.Fatal("ensureKeyPair() error = nil, want error when no config file exists")
	}
	if !strings.Contains(err.Error(), "not authenticated") {
		t.Errorf("ensureKeyPair() error = %v, want error containing 'not authenticated'", err)
	}
}

// TestEnsureKeyPair_LoggedInButNoPIN verifies that a user who is logged in
// (has AccessToken) but hasn't set up encryption gets a message directing
// them to the dashboard, not telling them to log in again.
func TestEnsureKeyPair_LoggedInButNoPIN(t *testing.T) {
	tmpDir, cleanup := withTempHome(t)
	defer cleanup()

	saveTestAuth(t, tmpDir, &config.AuthConfig{
		AccessToken: "some-valid-token",
	})

	_, err := ensureKeyPair()
	if err == nil {
		t.Fatal("ensureKeyPair() error = nil, want error for missing encryption keys")
	}
	if !strings.Contains(err.Error(), "encryption not set up") {
		t.Errorf("ensureKeyPair() error = %v, want error containing 'encryption not set up'", err)
	}
	if strings.Contains(err.Error(), "not authenticated") {
		t.Errorf("ensureKeyPair() error = %v, should not say 'not authenticated' when user has AccessToken", err)
	}
}

// TestEnsureKeyPair_InvalidBase64PrivateKey verifies that a non-base64
// PrivateKey produces an error mentioning "decoding private key".
func TestEnsureKeyPair_InvalidBase64PrivateKey(t *testing.T) {
	tmpDir, cleanup := withTempHome(t)
	defer cleanup()

	_, _, pubB64 := deriveTestKeyPair(t)

	saveTestAuth(t, tmpDir, &config.AuthConfig{
		PrivateKey: "not-valid-base64!!!",
		PublicKey:  pubB64,
	})

	_, err := ensureKeyPair()
	if err == nil {
		t.Fatal("ensureKeyPair() error = nil, want error for invalid base64 PrivateKey")
	}
	if !strings.Contains(err.Error(), "decoding private key") {
		t.Errorf("ensureKeyPair() error = %v, want error containing 'decoding private key'", err)
	}
}

// TestEnsureKeyPair_InvalidBase64PublicKey verifies that a non-base64
// PublicKey produces an error mentioning "decoding public key".
func TestEnsureKeyPair_InvalidBase64PublicKey(t *testing.T) {
	tmpDir, cleanup := withTempHome(t)
	defer cleanup()

	_, privB64, _ := deriveTestKeyPair(t)

	saveTestAuth(t, tmpDir, &config.AuthConfig{
		PrivateKey: privB64,
		PublicKey:  "not-valid-base64!!!",
	})

	_, err := ensureKeyPair()
	if err == nil {
		t.Fatal("ensureKeyPair() error = nil, want error for invalid base64 PublicKey")
	}
	if !strings.Contains(err.Error(), "decoding public key") {
		t.Errorf("ensureKeyPair() error = %v, want error containing 'decoding public key'", err)
	}
}

// TestEnsureKeyPair_RoundTrip verifies the full flow: derive a keypair,
// store it in config, load it back with ensureKeyPair, then use the
// returned keypair to decrypt a ciphertext encrypted with the original
// public key.
func TestEnsureKeyPair_RoundTrip(t *testing.T) {
	tmpDir, cleanup := withTempHome(t)
	defer cleanup()

	originalKP, privB64, pubB64 := deriveTestKeyPair(t)

	// Encrypt a test message using the original public key.
	plaintext := "OTP: 847291"
	ciphertext, err := box.SealAnonymous(nil, []byte(plaintext), &originalKP.PublicKey, rand.Reader)
	if err != nil {
		t.Fatalf("SealAnonymous: %v", err)
	}

	// Persist the keys in config.
	saveTestAuth(t, tmpDir, &config.AuthConfig{
		PrivateKey: privB64,
		PublicKey:  pubB64,
	})

	// Load them back through ensureKeyPair.
	loadedKP, err := ensureKeyPair()
	if err != nil {
		t.Fatalf("ensureKeyPair() error = %v", err)
	}

	// Decrypt using the loaded keypair.
	got, err := crypto.Decrypt(ciphertext, loadedKP)
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}

	if string(got) != plaintext {
		t.Errorf("Decrypt() = %q, want %q", string(got), plaintext)
	}
}

// TestEnsureKeyPair_BothKeysInvalid verifies the error when both keys are
// invalid base64. The function should fail on the private key first.
func TestEnsureKeyPair_BothKeysInvalid(t *testing.T) {
	tmpDir, cleanup := withTempHome(t)
	defer cleanup()

	saveTestAuth(t, tmpDir, &config.AuthConfig{
		PrivateKey: "!!!bad-private!!!",
		PublicKey:  "!!!bad-public!!!",
	})

	_, err := ensureKeyPair()
	if err == nil {
		t.Fatal("ensureKeyPair() error = nil, want error for invalid base64 keys")
	}

	// Private key is decoded first, so the error should reference it.
	if !strings.Contains(err.Error(), "decoding private key") {
		t.Errorf("ensureKeyPair() error = %v, want error containing 'decoding private key'", err)
	}
}
