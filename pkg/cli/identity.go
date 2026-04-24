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
var identityProvisionPhoneFlag bool
var identityProvisionPhoneCountryFlag string

var identityCmd = &cobra.Command{
	Use:   "identity",
	Short: "Manage identities",
	Long:  "List, create, and switch identities. Each identity bundles an email, phone, and credentials.",
}

var identityListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all identities",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := api.NewManagementClient()
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
		client, err := api.NewManagementClient()
		if err != nil {
			return err
		}

		identity, err := client.CreateIdentity(identityNameFlag, identityEmailFlag, identityProvisionPhoneFlag)
		if err != nil {
			return err
		}

		return output.Current.Print(identity)
	},
}

var identityProvisionPhoneCmd = &cobra.Command{
	Use:   "provision-phone <uuid>",
	Short: "Provision a phone number for an existing identity",
	Long: `Provision a phone number and link it to an existing identity that
was created without one. Requires a paid plan. Fails with an error if the
identity already has a phone number.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		uuid := args[0]

		client, err := api.NewManagementClient()
		if err != nil {
			return err
		}

		identity, err := client.ProvisionPhoneForIdentity(uuid, identityProvisionPhoneCountryFlag)
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
		client, err := api.NewManagementClient()
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

		// Create identity key for the selected identity.
		keyResp, err := client.CreateIdentityKey(matched.UUID, "cli")
		if err != nil {
			return fmt.Errorf("creating identity key: %w", err)
		}

		// Load existing config to preserve management key and user email.
		existingCfg, err := config.LoadConfig()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		// Save identity key + identity info to config (CWD-aware).
		if err := config.SaveConfig(&config.Config{
			ManagementKey: existingCfg.ManagementKey,
			IdentityKey:   keyResp.Key,
			IdentityUUID:  matched.UUID,
			IdentityName:  matched.Name,
			UserEmail:     existingCfg.UserEmail,
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
	identityCreateCmd.Flags().StringVar(&identityNameFlag, "name", "", "Name for the new identity (omit for auto-generated human name)")
	identityCreateCmd.Flags().StringVar(&identityEmailFlag, "email", "", "Email address: local part (e.g. 'myagent'), full email (e.g. 'myagent@custom.com'), or omit for auto-generated")
	identityCreateCmd.Flags().BoolVar(&identityProvisionPhoneFlag, "provision-phone", false, "Also provision a phone number for the new identity (requires a paid plan)")

	identityProvisionPhoneCmd.Flags().StringVar(&identityProvisionPhoneCountryFlag, "country-code", "", "ISO country code for the phone number (defaults to US)")

	identityCmd.AddCommand(identityListCmd)
	identityCmd.AddCommand(identityCreateCmd)
	identityCmd.AddCommand(identityProvisionPhoneCmd)
	identityCmd.AddCommand(identityUseCmd)
	rootCmd.AddCommand(identityCmd)
}
