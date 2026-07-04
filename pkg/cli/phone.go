package cli

import (
	"fmt"

	"github.com/ravi-hq/cli/internal/config"
	"github.com/ravi-hq/cli/internal/output"
	"github.com/spf13/cobra"
)

var phoneCountryCode string

var phoneCmd = &cobra.Command{
	Use:   "phone",
	Short: "Manage the identity's phone number",
	Long:  "Provision a phone number for an identity. Use 'ravi get phone' to view an existing number.",
}

var phoneCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Provision a phone number for the current identity",
	Long: `Provision a new phone number and link it to the identity.

Targets the bound identity by default; use the global --identity <uuid> flag to
target another identity. Requires an active paid plan. Fails if the identity
already has a phone number.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		identityUUID, err := resolveIdentityUUID()
		if err != nil {
			return err
		}

		client, err := newManagementClient()
		if err != nil {
			return err
		}

		identity, err := client.ProvisionPhone(identityUUID, phoneCountryCode)
		if err != nil {
			return err
		}

		if !humanOutput {
			return output.Current.Print(identity)
		}

		fmt.Printf("Provisioned phone %s for identity %s\n", identity.Phone, identity.Name)
		return nil
	},
}

// resolveIdentityUUID picks the target identity for provisioning: the global
// --identity flag when set, otherwise the bound identity from config.
func resolveIdentityUUID() (string, error) {
	if identityFlag != "" {
		return identityFlag, nil
	}
	cfg, err := config.LoadConfig()
	if err != nil {
		return "", err
	}
	if cfg.IdentityUUID == "" {
		return "", fmt.Errorf("no bound identity; log in or pass --identity <uuid>")
	}
	return cfg.IdentityUUID, nil
}

func init() {
	phoneCreateCmd.Flags().StringVar(&phoneCountryCode, "country-code", "US", "Country code for the number (e.g. US, CA)")
	phoneCmd.AddCommand(phoneCreateCmd)
	rootCmd.AddCommand(phoneCmd)
}
