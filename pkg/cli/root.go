package cli

import (
	"github.com/ravi-hq/cli/internal/api"
	"github.com/ravi-hq/cli/internal/output"
	"github.com/ravi-hq/cli/internal/version"
	"github.com/spf13/cobra"
)

var (
	humanOutput bool
	// identityFlag is the value of the global --identity flag. When set, it is
	// appended as ?identity=<uuid> to per-identity calls (contacts, passwords,
	// secrets, calls, events, messages). Empty leaves scoping to the active key.
	identityFlag string
)

// newClient builds an identity-scoped API client, applying the global
// --identity flag as the ?identity=<uuid> filter when set.
func newClient() (*api.Client, error) {
	client, err := api.NewClient()
	if err != nil {
		return nil, err
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

Setup: ravi auth login (one-time, requires human for Google OAuth)
After setup, agents self-service everything.

Identity: .ravi/config.json in CWD > ~/.ravi/config.json > unscoped

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

The key is an auth fence; the caller chooses the identity. Use --identity <uuid>
to target a specific identity for contacts, passwords, secrets, calls, events,
and messages. When omitted, the active identity key scopes the request.

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
	rootCmd.PersistentFlags().StringVar(&identityFlag, "identity", "", "Target identity UUID for per-identity calls (contacts, passwords, secrets, calls, events, messages)")

	// Add version command
	rootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			output.Current.PrintMessage(version.Info())
		},
	})
}
