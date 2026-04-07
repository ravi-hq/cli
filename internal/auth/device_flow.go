package auth

import (
	"bufio"
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
	client, err := api.NewUnauthenticatedClient()
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
			// Success!
			d.spinner.Stop()

			output.Current.PrintMessage(fmt.Sprintf("Authenticated as %s", tokenResp.User.Email))

			// Handle signup (got identity key directly) vs login (need to select identity)
			if tokenResp.IdentityKey != "" {
				// Signup flow: got management_key + identity_key + identity
				return d.handleSignup(tokenResp)
			}

			// Login flow: got management_key + identities list
			return d.handleLogin(tokenResp)
		default:
			return fmt.Errorf("authentication error: %s", errCode)
		}
	}

	return fmt.Errorf("authentication timed out")
}

// handleSignup saves the config after a signup (management key + identity key + identity provided).
func (d *DeviceFlow) handleSignup(tokenResp *api.DeviceTokenResponse) error {
	cfg := &config.Config{
		ManagementKey: tokenResp.ManagementKey,
		IdentityKey:   tokenResp.IdentityKey,
		UserEmail:     tokenResp.User.Email,
	}

	if tokenResp.Identity != nil {
		cfg.IdentityUUID = tokenResp.Identity.UUID
		cfg.IdentityName = tokenResp.Identity.Name
		output.Current.PrintMessage(fmt.Sprintf("Identity set: %s", identityLabel(*tokenResp.Identity)))
	}

	if err := config.SaveGlobalConfig(cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	return nil
}

// handleLogin lets the user select an identity and creates an identity key for it.
func (d *DeviceFlow) handleLogin(tokenResp *api.DeviceTokenResponse) error {
	identities := tokenResp.Identities
	if len(identities) == 0 {
		// Save management key only, no identity to select.
		if err := config.SaveGlobalConfig(&config.Config{
			ManagementKey: tokenResp.ManagementKey,
			UserEmail:     tokenResp.User.Email,
		}); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}
		output.Current.PrintMessage("No identities found — create one with `ravi identity create`")
		return nil
	}

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
	selected := identities[choice-1]

	// Create identity key via management key.
	mgmtClient, err := api.NewUnauthenticatedClient()
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	// Temporarily use management key for this request by saving and loading config.
	tempCfg := &config.Config{
		ManagementKey: tokenResp.ManagementKey,
		UserEmail:     tokenResp.User.Email,
	}
	if err := config.SaveGlobalConfig(tempCfg); err != nil {
		return fmt.Errorf("saving temp config: %w", err)
	}
	mgmtClient, err = api.NewManagementClient()
	if err != nil {
		return fmt.Errorf("creating management client: %w", err)
	}

	keyResp, err := mgmtClient.CreateIdentityKey(selected.UUID, "cli")
	if err != nil {
		return fmt.Errorf("creating identity key: %w", err)
	}

	// Save final config with both keys.
	if err := config.SaveGlobalConfig(&config.Config{
		ManagementKey: tokenResp.ManagementKey,
		IdentityKey:   keyResp.Key,
		IdentityUUID:  selected.UUID,
		IdentityName:  selected.Name,
		UserEmail:     tokenResp.User.Email,
	}); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	output.Current.PrintMessage(fmt.Sprintf("Identity set: %s", identityLabel(selected)))
	return nil
}

// identityLabel returns a human-readable label for an identity
// e.g. "Personal (user@ravi.id)" or just "Personal".
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
