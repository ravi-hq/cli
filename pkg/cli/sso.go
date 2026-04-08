package cli

import (
	"github.com/ravi-hq/cli/internal/api"
	"github.com/ravi-hq/cli/internal/output"
	"github.com/spf13/cobra"
)

var ssoCmd = &cobra.Command{
	Use:   "sso",
	Short: "Single Sign-On operations",
}

var ssoTokenCmd = &cobra.Command{
	Use:   "token",
	Short: "Request a short-lived SSO token",
	Long:  "Request a short-lived SSO token (rvt_ prefix, 5-minute TTL). Requires an identity-scoped API key and an active subscription.",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := api.NewClient()
		if err != nil {
			return err
		}

		resp, err := client.RequestSSOToken()
		if err != nil {
			return err
		}

		output.Current.Print(resp)
		return nil
	},
}

func init() {
	ssoCmd.AddCommand(ssoTokenCmd)
	rootCmd.AddCommand(ssoCmd)
}
