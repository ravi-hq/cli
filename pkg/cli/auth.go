package cli

import (
	"fmt"

	"github.com/ravi-hq/cli/internal/api"
	"github.com/ravi-hq/cli/internal/auth"
	"github.com/ravi-hq/cli/internal/config"
	"github.com/ravi-hq/cli/internal/output"
	"github.com/spf13/cobra"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage authentication",
}

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with Ravi",
	Long:  "Start the device code flow to authenticate with your Ravi account.",
	RunE: func(cmd *cobra.Command, args []string) error {
		flow, err := auth.NewDeviceFlow()
		if err != nil {
			return err
		}
		return flow.Run()
	},
}

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Clear stored credentials",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := config.Clear(); err != nil {
			return fmt.Errorf("failed to clear credentials: %w", err)
		}
		output.Current.PrintMessage("Logged out successfully")
		return nil
	},
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show authentication status",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := api.NewClient(nil)
		if err != nil {
			// Config exists but tokens are missing/corrupt — not authenticated
			output.Current.Print(map[string]interface{}{
				"authenticated": false,
			})
			return nil
		}

		if client.IsAuthenticated() {
			result := map[string]interface{}{
				"authenticated": true,
			}
			if email := client.GetUserEmail(); email != "" {
				result["email"] = email
			}
			if identity := client.GetIdentityName(); identity != "" {
				result["identity"] = identity
			}
			output.Current.Print(result)
		} else {
			// Tokens exist but are invalid — clear them
			config.Clear()
			output.Current.Print(map[string]interface{}{
				"authenticated": false,
				"message":       "Session expired. Run `ravi auth login` to re-authenticate.",
			})
		}
		return nil
	},
}

func init() {
	authCmd.AddCommand(loginCmd)
	authCmd.AddCommand(logoutCmd)
	authCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(authCmd)
}
