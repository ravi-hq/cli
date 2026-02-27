package cli

import (
	"fmt"

	"github.com/ravi-hq/cli/internal/api"
	"github.com/ravi-hq/cli/internal/config"
	"github.com/ravi-hq/cli/internal/output"
	"github.com/spf13/cobra"
)

var identityNameFlag string

var identityCmd = &cobra.Command{
	Use:   "identity",
	Short: "Manage identities",
	Long:  "List, create, and switch identities. Each identity bundles an email, phone, and vault.",
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

		identity, err := client.CreateIdentity(identityNameFlag)
		if err != nil {
			return err
		}

		return output.Current.Print(identity)
	},
}

var identityUseCmd = &cobra.Command{
	Use:   "use <name-or-uuid>",
	Short: "Set the active identity for this machine",
	Long: `Set which identity is used for all ravi commands globally.
Use a project-level .ravi/config.json to override per-directory.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target := args[0]

		// Resolve identity by name or UUID.
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
			if id.Name == target || id.UUID == target {
				matched = &id
				break
			}
		}
		if matched == nil {
			return fmt.Errorf("identity %q not found", target)
		}

		if err := config.SaveGlobalConfig(&config.Config{
			IdentityUUID: matched.UUID,
			IdentityName: matched.Name,
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

	identityCmd.AddCommand(identityListCmd)
	identityCmd.AddCommand(identityCreateCmd)
	identityCmd.AddCommand(identityUseCmd)
	rootCmd.AddCommand(identityCmd)
}
