package auth

import (
	"bufio"
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
	client, err := api.NewClient(nil)
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
			// Success! Save tokens
			d.spinner.Stop()

			cfg := &config.Config{
				AccessToken:  tokenResp.Access,
				RefreshToken: tokenResp.Refresh,
				ExpiresAt:    time.Now().Add(api.TokenExpiryBuffer), // Assume ~5 min expiry
				UserEmail:    tokenResp.User.Email,
			}

			output.Current.PrintMessage(fmt.Sprintf("Authenticated as %s", tokenResp.User.Email))

			// Recreate client with the new tokens (in memory only)
			// so authenticated requests work before we persist.
			d.client, err = api.NewClient(cfg)
			if err != nil {
				return fmt.Errorf("failed to reinitialize client: %w", err)
			}

			// Select and bind an identity to this CLI session.
			if err := d.selectAndBindIdentity(cfg); err != nil {
				return fmt.Errorf("identity selection failed: %w", err)
			}

			// Prompt for PIN to unlock E2E decryption.
			// If the user exits here (Ctrl+C), nothing is saved to disk.
			if err := d.unlockEncryption(cfg); err != nil {
				return fmt.Errorf("encryption unlock failed: %w", err)
			}

			// Save only after auth + identity + PIN are all complete.
			if err := config.Save(cfg); err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}

			return nil
		default:
			return fmt.Errorf("authentication error: %s", errCode)
		}
	}

	return fmt.Errorf("authentication timed out")
}

// unlockEncryption fetches the user's encryption metadata, prompts for their
// PIN, verifies it, and persists the derived private key in the config file
// so subsequent commands can decrypt without re-prompting.
func (d *DeviceFlow) unlockEncryption(cfg *config.Config) error {
	meta, err := d.client.GetEncryptionMeta()
	if err != nil {
		return fmt.Errorf("fetching encryption metadata: %w", err)
	}

	if meta.PublicKey == "" {
		// User hasn't completed PIN setup on the dashboard yet.
		// This is OK — CLI will error on commands that need decryption.
		fmt.Println("\nEncryption not set up yet. Complete PIN setup on the dashboard to enable E2E decryption.")
		return nil
	}

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

	cfg.PINSalt = meta.Salt
	cfg.PublicKey = meta.PublicKey
	cfg.PrivateKey = base64.StdEncoding.EncodeToString(kp.PrivateKey[:])

	output.Current.PrintMessage("Encryption unlocked")
	return nil
}

// selectAndBindIdentity lists the user's identities and binds the chosen one
// to the JWT session. The identity is then locked into all future API calls.
func (d *DeviceFlow) selectAndBindIdentity(cfg *config.Config) error {
	identities, err := d.client.ListIdentities()
	if err != nil {
		return fmt.Errorf("listing identities: %w", err)
	}

	if len(identities) == 0 {
		return fmt.Errorf("no identities found — complete setup on the dashboard first")
	}

	var selected api.Identity

	if len(identities) == 1 {
		selected = identities[0]
		output.Current.PrintMessage(fmt.Sprintf("Using identity: %s", identityLabel(selected)))
	} else {
		fmt.Println("\nSelect an identity for this CLI session:")
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

	// Bind the identity to the JWT.
	bound, err := d.client.BindIdentity(selected.UUID)
	if err != nil {
		return fmt.Errorf("binding identity: %w", err)
	}
	if bound.Access == "" || bound.Refresh == "" {
		return fmt.Errorf("binding identity: server returned empty tokens")
	}

	cfg.AccessToken = bound.Access
	cfg.RefreshToken = bound.Refresh
	cfg.ExpiresAt = time.Now().Add(api.TokenExpiryBuffer)
	cfg.IdentityName = selected.Name

	// Recreate client with the bound tokens.
	d.client, err = api.NewClient(cfg)
	if err != nil {
		return fmt.Errorf("reinitializing client after bind: %w", err)
	}

	output.Current.PrintMessage(fmt.Sprintf("Bound to identity: %s", identityLabel(selected)))
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
