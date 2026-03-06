package cli

import (
	"fmt"

	"github.com/ravi-hq/cli/internal/api"
	"github.com/ravi-hq/cli/internal/config"
	"github.com/ravi-hq/cli/internal/output"
	"github.com/spf13/cobra"
)

var identityNameFlag string
var identityEmailFlag string

var identityCmd = &cobra.Command{
	Use:   "identity",
	Short: "Manage identities",
	Long:  "List, create, and switch identities. Each identity bundles an email, phone, and credentials.",
}

var identityListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all identities",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := api.NewUnscopedClient()
		if err != nil {
			return err
		}

		identities, err := client.ListIdentities()
		if err != nil {
			return err
		}

		return output.Current.Print(identities)
	},
}

var identityCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new identity",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := api.NewUnscopedClient()
		if err != nil {
			return err
		}

		identity, err := client.CreateIdentity(identityNameFlag, identityEmailFlag)
		if err != nil {
			return err
		}

		return output.Current.Print(identity)
	},
}

var identityUseCmd = &cobra.Command{
	Use:   "use <uuid>",
	Short: "Set the active identity",
	Long: `Set which identity is used for all ravi commands.
Writes to .ravi/config.json in CWD if it exists, otherwise ~/.ravi/config.json.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target := args[0]

		// Resolve identity by UUID.
		client, err := api.NewUnscopedClient()
		if err != nil {
			return err
		}

		identities, err := client.ListIdentities()
		if err != nil {
			return err
		}

		var matched *api.Identity
		for _, id := range identities {
			if id.UUID == target {
				matched = &id
				break
			}
		}
		if matched == nil {
			return fmt.Errorf("identity %q not found", target)
		}

		// Bind identity to get identity-scoped tokens.
		bindResult, err := client.BindIdentity(matched.UUID)
		if err != nil {
			return fmt.Errorf("binding identity: %w", err)
		}

		// Save bound tokens + identity info to config (CWD-aware).
		if err := config.SaveConfig(&config.Config{
			IdentityUUID:      matched.UUID,
			IdentityName:      matched.Name,
			BoundAccessToken:  bindResult.Access,
			BoundRefreshToken: bindResult.Refresh,
		}); err != nil {
			return err
		}

		return output.Current.Print(map[string]string{
			"identity_name": matched.Name,
			"identity_uuid": matched.UUID,
			"status":        "active",
		})
	},
}

func init() {
	identityCreateCmd.Flags().StringVar(&identityNameFlag, "name", "", "Name for the new identity (required)")
	identityCreateCmd.MarkFlagRequired("name")
	identityCreateCmd.Flags().StringVar(&identityEmailFlag, "email", "", "Email address: local part (e.g. 'myagent'), full email (e.g. 'myagent@custom.com'), or omit for auto-generated")

	identityCmd.AddCommand(identityListCmd)
	identityCmd.AddCommand(identityCreateCmd)
	identityCmd.AddCommand(identityUseCmd)
	rootCmd.AddCommand(identityCmd)
}
