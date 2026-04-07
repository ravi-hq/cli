package cli

import (
	"fmt"

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
		if err := config.ClearAll(); err != nil {
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
		cfg, err := config.LoadConfig()
		if err != nil {
			output.Current.Print(map[string]interface{}{
				"authenticated": false,
				"error":         err.Error(),
			})
			return nil
		}

		isAuthenticated := cfg.ManagementKey != "" || cfg.IdentityKey != ""

		if isAuthenticated {
			result := map[string]interface{}{
				"authenticated": true,
			}
			if cfg.UserEmail != "" {
				result["email"] = cfg.UserEmail
			}
			if cfg.IdentityUUID != "" {
				result["identity"] = cfg.IdentityName
				result["identity_uuid"] = cfg.IdentityUUID
			}
			output.Current.Print(result)
		} else {
			output.Current.Print(map[string]interface{}{
				"authenticated": false,
				"message":       "Not authenticated. Run `ravi auth login` to authenticate.",
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
