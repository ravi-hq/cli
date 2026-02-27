package cli

import (
	"encoding/base64"
	"fmt"
	"os"

	"github.com/ravi-hq/cli/internal/config"
	"github.com/ravi-hq/cli/internal/crypto"
)

// ensureKeyPair loads the persisted decryption keypair from auth.json.
// The private key is stored during login (after PIN verification) so that
// subsequent commands never need to re-prompt for the PIN.
func ensureKeyPair() (*crypto.KeyPair, error) {
	cfg, err := config.LoadAuth()
	if err != nil {
		return nil, fmt.Errorf("loading auth: %w", err)
	}

	if cfg.PrivateKey == "" || cfg.PublicKey == "" {
		if cfg.AccessToken != "" {
			return nil, fmt.Errorf("encryption not set up — complete PIN setup on the dashboard first")
		}
		return nil, fmt.Errorf("not authenticated — run `ravi auth login` first")
	}

	privBytes, err := base64.StdEncoding.DecodeString(cfg.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("decoding private key: %w", err)
	}
	if len(privBytes) != 32 {
		return nil, fmt.Errorf("private key has invalid length %d, expected 32", len(privBytes))
	}

	pubBytes, err := base64.StdEncoding.DecodeString(cfg.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("decoding public key: %w", err)
	}
	if len(pubBytes) != 32 {
		return nil, fmt.Errorf("public key has invalid length %d, expected 32", len(pubBytes))
	}

	var kp crypto.KeyPair
	copy(kp.PrivateKey[:], privBytes)
	copy(kp.PublicKey[:], pubBytes)

	return &kp, nil
}

// tryDecrypt attempts to decrypt an E2E-encrypted field. If the value is not
// encrypted it is returned as-is. On decryption failure a warning is printed
// to stderr and the original (encrypted) value is returned so the caller
// always has something to display.
func tryDecrypt(value string, kp *crypto.KeyPair) string {
	result, err := crypto.DecryptField(value, kp)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not decrypt field: %v\n", err)
		return value
	}
	return result
}
