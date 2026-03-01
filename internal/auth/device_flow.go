package auth

import (
	"bufio"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/ravi-hq/cli/internal/api"
	"github.com/ravi-hq/cli/internal/config"
	"github.com/ravi-hq/cli/internal/crypto"
	"github.com/ravi-hq/cli/internal/output"
)

const (
	// DefaultSpinnerCharSet is the Braille spinner pattern (index 14 in yacspin).
	DefaultSpinnerCharSet = 14
)

// DeviceFlow handles the device code authentication flow
type DeviceFlow struct {
	client  *api.Client
	spinner *spinner.Spinner
}

// NewDeviceFlow creates a new device flow handler
func NewDeviceFlow() (*DeviceFlow, error) {
	client, err := api.NewClientWithTokens("", "")
	if err != nil {
		return nil, err
	}

	s := spinner.New(spinner.CharSets[DefaultSpinnerCharSet], 100*time.Millisecond)
	s.Suffix = " Waiting for authorization..."

	return &DeviceFlow{
		client:  client,
		spinner: s,
	}, nil
}

// Run executes the device code flow
func (d *DeviceFlow) Run() error {
	// Request device code
	codeResp, err := d.client.RequestDeviceCode()
	if err != nil {
		return fmt.Errorf("failed to request device code: %w", err)
	}

	// Display instructions
	fmt.Println()
	fmt.Println("To authenticate, visit:")
	fmt.Printf("  %s\n", codeResp.VerificationURI)
	fmt.Println()
	fmt.Println("And enter the code:")
	fmt.Printf("  %s\n", codeResp.UserCode)
	fmt.Println()

	// Try to open browser
	if err := openBrowser(codeResp.VerificationURI + "?user_code=" + codeResp.UserCode); err != nil {
		// Not a fatal error, user can manually visit URL
		fmt.Println("(Could not open browser automatically)")
	}

	// Start polling with spinner
	d.spinner.Start()
	defer d.spinner.Stop()

	interval := time.Duration(codeResp.Interval) * time.Second
	deadline := time.Now().Add(time.Duration(codeResp.ExpiresIn) * time.Second)

	for time.Now().Before(deadline) {
		tokenResp, errCode, err := d.client.PollForToken(codeResp.DeviceCode)
		if err != nil {
			return fmt.Errorf("polling error: %w", err)
		}

		// Check error codes
		switch errCode {
		case "authorization_pending":
			// Still waiting, continue polling
			time.Sleep(interval)
			continue
		case "expired_token":
			return fmt.Errorf("device code expired. Please try again")
		case "":
			// Success! Build auth config with unbound tokens.
			d.spinner.Stop()

			auth := &config.AuthConfig{
				AccessToken:  tokenResp.Access,
				RefreshToken: tokenResp.Refresh,
				UserEmail:    tokenResp.User.Email,
			}

			output.Current.PrintMessage(fmt.Sprintf("Authenticated as %s", tokenResp.User.Email))

			// Recreate client with the new tokens (unbound, no identity).
			d.client, err = api.NewClientWithTokens(tokenResp.Access, tokenResp.Refresh)
			if err != nil {
				return fmt.Errorf("failed to reinitialize client: %w", err)
			}

			// Set up or unlock encryption.
			if err := d.setupEncryption(auth); err != nil {
				return fmt.Errorf("encryption setup failed: %w", err)
			}

			// Select an identity.
			if err := d.selectIdentity(); err != nil {
				return fmt.Errorf("identity selection failed: %w", err)
			}

			// Save auth.json (identity already saved to config.json above).
			if err := config.SaveAuth(auth); err != nil {
				return fmt.Errorf("failed to save auth: %w", err)
			}

			return nil
		default:
			return fmt.Errorf("authentication error: %s", errCode)
		}
	}

	return fmt.Errorf("authentication timed out")
}

// setupEncryption handles both first-time encryption setup and unlocking
// existing encryption. On first setup, it generates a recovery key.
func (d *DeviceFlow) setupEncryption(auth *config.AuthConfig) error {
	meta, err := d.client.GetEncryptionMeta()
	if err != nil {
		return fmt.Errorf("fetching encryption metadata: %w", err)
	}

	if meta.PublicKey == "" {
		// First-time setup: prompt for PIN, generate keys, register with server.
		return d.initialEncryptionSetup(auth)
	}

	// Existing encryption: prompt for PIN, verify, store keys.
	return d.unlockExistingEncryption(auth, meta)
}

// initialEncryptionSetup creates encryption keys from a user-chosen PIN and
// registers them with the server. Also generates and saves a recovery key.
func (d *DeviceFlow) initialEncryptionSetup(auth *config.AuthConfig) error {
	fmt.Println("\nSet up encryption for your vault.")
	pin, err := crypto.PromptPIN("Choose a 6-digit encryption PIN: ")
	if err != nil {
		return err
	}

	// Confirm PIN.
	confirm, err := crypto.PromptPIN("Confirm PIN: ")
	if err != nil {
		return err
	}
	if pin != confirm {
		return fmt.Errorf("PINs do not match")
	}

	// Generate random salt (16 bytes).
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return fmt.Errorf("generating salt: %w", err)
	}

	// Derive keypair from PIN + salt.
	kp, err := crypto.DeriveKeyPair(pin, salt)
	if err != nil {
		return fmt.Errorf("deriving keypair: %w", err)
	}

	// Create verifier (encrypted "ravi-e2e-verify" with public key).
	verifier, err := crypto.CreateVerifier(kp)
	if err != nil {
		return fmt.Errorf("creating verifier: %w", err)
	}

	saltB64 := base64.StdEncoding.EncodeToString(salt)
	pubKeyB64 := base64.StdEncoding.EncodeToString(kp.PublicKey[:])

	// Save recovery key BEFORE registering with server (can be retried if it fails).
	recoveryKey := base64.StdEncoding.EncodeToString(salt)
	if err := config.SaveRecoveryKey(recoveryKey); err != nil {
		return fmt.Errorf("saving recovery key: %w", err)
	}

	fmt.Fprintf(os.Stderr, "\nRecovery key saved to %s — back this up!\n", config.RecoveryKeyPath())

	// Register with server (point of no return).
	err = d.client.UpdateEncryptionMeta(map[string]string{
		"salt":       saltB64,
		"public_key": pubKeyB64,
		"verifier":   verifier,
	})
	if err != nil {
		return fmt.Errorf("registering encryption keys: %w", err)
	}

	// Store keys in auth config.
	auth.PINSalt = saltB64
	auth.PublicKey = pubKeyB64
	auth.PrivateKey = base64.StdEncoding.EncodeToString(kp.PrivateKey[:])

	output.Current.PrintMessage("Encryption set up successfully")
	return nil
}

// unlockExistingEncryption prompts for the PIN and verifies it against
// server-stored metadata.
func (d *DeviceFlow) unlockExistingEncryption(auth *config.AuthConfig, meta *api.EncryptionMeta) error {
	fmt.Println()
	kp, err := crypto.GetOrPromptKeyPair(meta.Salt, meta.Verifier)
	if err != nil {
		return err
	}

	// Verify that the locally-derived public key matches the server record.
	derivedPub := base64.StdEncoding.EncodeToString(kp.PublicKey[:])
	if derivedPub != meta.PublicKey {
		return fmt.Errorf("derived public key does not match server record — possible data corruption")
	}

	auth.PINSalt = meta.Salt
	auth.PublicKey = meta.PublicKey
	auth.PrivateKey = base64.StdEncoding.EncodeToString(kp.PrivateKey[:])

	output.Current.PrintMessage("Encryption unlocked")
	return nil
}

// selectIdentity lists identities and saves the selected one to the global config.
func (d *DeviceFlow) selectIdentity() error {
	identities, err := d.client.ListIdentities()
	if err != nil {
		return fmt.Errorf("listing identities: %w", err)
	}

	if len(identities) == 0 {
		return fmt.Errorf("no identities found — encryption setup may have failed, try `ravi auth login` again")
	}

	var selected api.Identity

	if len(identities) == 1 {
		selected = identities[0]
		output.Current.PrintMessage(fmt.Sprintf("Using identity: %s", identityLabel(selected)))
	} else {
		fmt.Println("\nSelect an identity:")
		for i, id := range identities {
			fmt.Printf("  %d) %s\n", i+1, identityLabel(id))
		}
		fmt.Print("> ")

		reader := bufio.NewReader(os.Stdin)
		line, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("reading input: %w", err)
		}
		trimmed := strings.TrimSpace(line)
		choice, err := strconv.Atoi(trimmed)
		if err != nil {
			return fmt.Errorf("invalid selection %q — enter a number between 1 and %d", trimmed, len(identities))
		}
		if choice < 1 || choice > len(identities) {
			return fmt.Errorf("selection %d out of range — enter a number between 1 and %d", choice, len(identities))
		}
		selected = identities[choice-1]
	}

	// Save identity to global config.
	if err := config.SaveGlobalConfig(&config.Config{
		IdentityUUID: selected.UUID,
		IdentityName: selected.Name,
	}); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	output.Current.PrintMessage(fmt.Sprintf("Identity set: %s", identityLabel(selected)))
	return nil
}

// identityLabel returns a human-readable label for an identity
// e.g. "Personal (user@ravi.app)" or just "Personal".
func identityLabel(id api.Identity) string {
	detail := id.Email
	if detail == "" && id.Phone != "" {
		detail = id.Phone
	}
	if detail != "" {
		return fmt.Sprintf("%s (%s)", id.Name, detail)
	}
	return id.Name
}

// openBrowser opens the default browser to the given URL
func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		return fmt.Errorf("unsupported platform")
	}
	return cmd.Start()
}
