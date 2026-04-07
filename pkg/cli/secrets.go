package cli

import (
	"fmt"

	"github.com/ravi-hq/cli/internal/api"
	"github.com/ravi-hq/cli/internal/output"
	"github.com/spf13/cobra"
)

var secretsCmd = &cobra.Command{
	Use:   "secrets",
	Short: "Manage key-value secrets",
}

var secretListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all stored secrets",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := api.NewClient()
		if err != nil {
			return err
		}

		entries, err := client.ListSecrets()
		if err != nil {
			return err
		}

		if !humanOutput {
			return output.Current.Print(entries)
		}

		if len(entries) == 0 {
			output.Current.PrintMessage("No secrets found")
			return nil
		}

		headers := []string{"UUID", "KEY", "VALUE", "CREATED"}
		rows := make([][]string, len(entries))
		for i, e := range entries {
			rows[i] = []string{
				truncate(e.UUID, 12),
				truncate(e.Key, 25),
				"[hidden]",
				e.CreatedDt,
			}
		}
		output.Current.PrintTable(headers, rows)
		return nil
	},
}

var secretGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Show a stored secret by key",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := api.NewClient()
		if err != nil {
			return err
		}

		entry, err := client.GetSecret(args[0])
		if err != nil {
			return err
		}
		if entry == nil {
			return fmt.Errorf("secret not found: %s", args[0])
		}

		if !humanOutput {
			return output.Current.Print(entry)
		}

		fmt.Printf("Key:     %s\n", entry.Key)
		fmt.Printf("Value:   %s\n", entry.Value)
		if entry.Notes != "" {
			fmt.Printf("Notes:   %s\n", entry.Notes)
		}
		fmt.Printf("UUID:    %s\n", entry.UUID)
		fmt.Printf("Created: %s\n", entry.CreatedDt)
		return nil
	},
}

var secretSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Create or update a secret",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := api.NewClient()
		if err != nil {
			return err
		}

		// Upsert: check if key already exists, update if so.
		existing, err := client.GetSecret(args[0])
		if err != nil {
			return err
		}

		var result *api.SecretEntry
		if existing != nil {
			result, err = client.UpdateSecret(existing.UUID, map[string]interface{}{
				"value": args[1],
			})
		} else {
			result, err = client.CreateSecret(api.SecretEntry{
				Key:   args[0],
				Value: args[1],
			})
		}
		if err != nil {
			return err
		}

		if !humanOutput {
			return output.Current.Print(result)
		}

		fmt.Printf("Secret stored: %s (UUID: %s)\n", result.Key, result.UUID)
		return nil
	},
}

var secretDeleteCmd = &cobra.Command{
	Use:   "delete <uuid>",
	Short: "Delete a stored secret by UUID",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := api.NewClient()
		if err != nil {
			return err
		}

		if err := client.DeleteSecret(args[0]); err != nil {
			return err
		}

		if !humanOutput {
			return output.Current.Print(map[string]string{"status": "deleted"})
		}

		fmt.Println("Secret deleted.")
		return nil
	},
}

func init() {
	secretsCmd.AddCommand(secretListCmd)
	secretsCmd.AddCommand(secretGetCmd)
	secretsCmd.AddCommand(secretSetCmd)
	secretsCmd.AddCommand(secretDeleteCmd)
	rootCmd.AddCommand(secretsCmd)
}
