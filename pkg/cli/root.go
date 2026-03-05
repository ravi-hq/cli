package cli

import (
	"github.com/ravi-hq/cli/internal/output"
	"github.com/ravi-hq/cli/internal/version"
	"github.com/spf13/cobra"
)

var (
	jsonOutput bool
)

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

All commands support --json for structured output.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		output.SetJSON(jsonOutput)
	},
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")

	// Add version command
	rootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			output.Current.PrintMessage(version.Info())
		},
	})
}
