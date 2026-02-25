package crypto

import (
	"encoding/base64"
	"fmt"
	"os"
	"regexp"
	"strings"

	"golang.org/x/term"
)

// pinPattern matches exactly 6 ASCII digits.
var pinPattern = regexp.MustCompile(`^\d{6}$`)

// maxPINAttempts is the number of times the user may re-enter their PIN
// before the operation is aborted.
const maxPINAttempts = 3

// cachedKeyPair holds the in-memory keypair for the current process.
var cachedKeyPair *KeyPair

// GetOrPromptKeyPair returns the cached keypair or prompts the user for their
// PIN, derives the keypair, and verifies it against the server-stored verifier.
//
// saltB64 is the base64-encoded 16-byte salt from the server.
// verifierB64 is the base64-encoded SealedBox ciphertext of "ravi-e2e-verify".
func GetOrPromptKeyPair(saltB64, verifierB64 string) (*KeyPair, error) {
	if cachedKeyPair != nil {
		return cachedKeyPair, nil
	}

	salt, err := base64.StdEncoding.DecodeString(saltB64)
	if err != nil {
		return nil, fmt.Errorf("decoding salt: %w", err)
	}

	for attempt := 1; attempt <= maxPINAttempts; attempt++ {
		pin, err := PromptPIN("Enter your 6-digit encryption PIN: ")
		if err != nil {
			return nil, err
		}

		kp, err := DeriveKeyPair(pin, salt)
		if err != nil {
			return nil, fmt.Errorf("deriving keypair: %w", err)
		}

		if Verify(kp, verifierB64) {
			cachedKeyPair = kp
			return kp, nil
		}

		remaining := maxPINAttempts - attempt
		if remaining > 0 {
			fmt.Fprintf(os.Stderr, "Incorrect PIN. %d attempt(s) remaining.\n", remaining)
		}
	}

	return nil, fmt.Errorf("maximum PIN attempts exceeded")
}

// ClearCachedKeyPair discards the in-memory keypair (e.g. on logout).
func ClearCachedKeyPair() {
	cachedKeyPair = nil
}

// PromptPIN prompts the user for a 6-digit PIN with hidden input.
// The prompt string is written to stderr so it appears even when stdout is
// redirected.
func PromptPIN(prompt string) (string, error) {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return "", fmt.Errorf("PIN prompt requires an interactive terminal (stdin is not a TTY)")
	}

	fmt.Fprint(os.Stderr, prompt)

	raw, err := term.ReadPassword(int(os.Stdin.Fd()))
	// Print a newline after the hidden input so the next output starts on a
	// fresh line.
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", fmt.Errorf("reading PIN: %w", err)
	}

	pin := strings.TrimSpace(string(raw))
	if !pinPattern.MatchString(pin) {
		return "", fmt.Errorf("PIN must be exactly 6 digits")
	}

	return pin, nil
}
