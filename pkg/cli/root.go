package cli

import (
	"fmt"

	"github.com/ravi-hq/cli/internal/api"
	"github.com/ravi-hq/cli/internal/output"
	"github.com/ravi-hq/cli/internal/version"
	"github.com/spf13/cobra"
)

var (
	humanOutput bool
	// identityFlag is the value of the global --identity flag. When set, it is
	// appended as ?identity=<uuid> to per-identity calls (get, inbox, contacts,
	// passwords, secrets, calls, events, messages). Empty leaves scoping to the
	// active key.
	identityFlag string
)

// newClient builds an API client for resource commands (get, inbox, contacts,
// passwords, secrets, calls, messages).
//
// Without --identity, the identity key is used (fenced to the bound identity
// in ~/.ravi/config.json). With --identity, the management key is used plus
// ?identity=<uuid> so a one-shot override can read that identity's inbox and
// channels. Identity keys cannot see another identity's mailbox; using them
// here returns a silent empty list or "no email address assigned".
func newClient() (*api.Client, error) {
	if identityFlag == "" {
		return api.NewClient()
	}
	client, err := api.NewManagementClient()
	if err != nil {
		return nil, err
	}
	if !client.IsAuthenticated() {
		return nil, fmt.Errorf("--identity requires a management key; an identity-scoped key cannot read another identity's inbox")
	}
	return client.WithIdentity(identityFlag), nil
}

// newManagementClient builds a management-key API client, applying the global
// --identity flag as the ?identity=<uuid> filter when set.
func newManagementClient() (*api.Client, error) {
	client, err := api.NewManagementClient()
	if err != nil {
		return nil, err
	}
	return client.WithIdentity(identityFlag), nil
}

// rootCmd is the base command
var rootCmd = &cobra.Command{
	Use:   "ravi",
	Short: "Ravi CLI — identity, email, phone, and credentials for AI agents",
	Long: `Ravi CLI — identity, email, phone, and credentials for AI agents.

The CLI holds one active identity per machine / config file
(~/.ravi/config.json, or .ravi/config.json in CWD). That file cannot run
multiple agents. ravi identity use replaces the single active identity; it
is not a multi-agent switcher.

Cursor: use the MCP Connect card (per-agent credentials), not ravi auth login.
Several agents on one host: call https://api.ravi.app with per-identity
ravi_id_ keys.

CLI login on this machine: ravi auth login (RFC 8628 device-code; human
approves in the browser at https://ravi.id/device).

Commands:
  auth       Authenticate (login/logout/status)
  identity   Manage identities (list/create/use)
  get        Retrieve resources (email/phone/owner)
  inbox      Read messages (sms/email)
  message    Individual message access
  email      Send emails (compose/reply/reply-all)
  passwords  Website passwords (create/get/list/update/delete/generate)
  secrets    Key-value secrets (list/get/set/delete)
  contacts   Manage contacts (list/search/get/create/update/delete)
  sms        Send SMS from the identity's phone number
  call       Place and manage phone calls

--identity scopes a single request so it does not leak another identity's
resources (for example get phone). It does not change the machine's active
identity and is not how you run multiple agents.

JSON output by default. Use --human for human-readable output.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		output.SetJSON(!humanOutput)
	},
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&humanOutput, "human", false, "Output in human-readable format")
	rootCmd.PersistentFlags().StringVar(&identityFlag, "identity", "", "Scope a single request to one identity's inbox, channels, and vault (does not change the machine's active identity)")

	// Add version command
	rootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			output.Current.PrintMessage(version.Info())
		},
	})
}
