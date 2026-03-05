package cli

import (
	"fmt"

	"github.com/ravi-hq/cli/internal/api"
	"github.com/ravi-hq/cli/internal/output"
	"github.com/spf13/cobra"
)

// Flag variables for contacts commands
var (
	ctEmail       string
	ctPhone       string
	ctDisplayName string
	ctNickname    string
	ctTrusted     bool
)

var contactsCmd = &cobra.Command{
	Use:   "contacts",
	Short: "Manage contacts",
}

var ctListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all contacts",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := api.NewClient()
		if err != nil {
			return err
		}

		entries, err := client.ListContacts()
		if err != nil {
			return err
		}

		if jsonOutput {
			return output.Current.Print(entries)
		}

		if len(entries) == 0 {
			output.Current.PrintMessage("No contacts found")
			return nil
		}

		headers := []string{"UUID", "EMAIL", "PHONE", "DISPLAY NAME", "TRUSTED", "SOURCE"}
		rows := make([][]string, len(entries))
		for i, e := range entries {
			rows[i] = []string{
				truncate(e.UUID, 12),
				truncate(e.Email, 30),
				truncate(e.PhoneNumber, 16),
				truncate(e.DisplayName, 25),
				fmt.Sprintf("%v", e.IsTrusted),
				e.Source,
			}
		}
		output.Current.PrintTable(headers, rows)
		return nil
	},
}

var ctSearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search contacts",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := api.NewClient()
		if err != nil {
			return err
		}

		entries, err := client.SearchContacts(args[0])
		if err != nil {
			return err
		}

		if jsonOutput {
			return output.Current.Print(entries)
		}

		if len(entries) == 0 {
			output.Current.PrintMessage("No contacts found")
			return nil
		}

		headers := []string{"UUID", "EMAIL", "PHONE", "DISPLAY NAME", "TRUSTED", "SOURCE"}
		rows := make([][]string, len(entries))
		for i, e := range entries {
			rows[i] = []string{
				truncate(e.UUID, 12),
				truncate(e.Email, 30),
				truncate(e.PhoneNumber, 16),
				truncate(e.DisplayName, 25),
				fmt.Sprintf("%v", e.IsTrusted),
				e.Source,
			}
		}
		output.Current.PrintTable(headers, rows)
		return nil
	},
}

var ctGetCmd = &cobra.Command{
	Use:   "get <uuid>",
	Short: "Show a contact",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := api.NewClient()
		if err != nil {
			return err
		}

		entry, err := client.GetContact(args[0])
		if err != nil {
			return err
		}

		if jsonOutput {
			return output.Current.Print(entry)
		}

		fmt.Printf("Email:        %s\n", entry.Email)
		fmt.Printf("Phone:        %s\n", entry.PhoneNumber)
		fmt.Printf("Display Name: %s\n", entry.DisplayName)
		if entry.Nickname != "" {
			fmt.Printf("Nickname:     %s\n", entry.Nickname)
		}
		fmt.Printf("Trusted:      %v\n", entry.IsTrusted)
		fmt.Printf("Source:       %s\n", entry.Source)
		fmt.Printf("UUID:         %s\n", entry.UUID)
		fmt.Printf("Created:      %s\n", entry.CreatedDt)
		return nil
	},
}

var ctCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new contact",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := api.NewClient()
		if err != nil {
			return err
		}

		entry := api.ContactEntry{
			Email:       ctEmail,
			PhoneNumber: ctPhone,
			DisplayName: ctDisplayName,
			Nickname:    ctNickname,
			IsTrusted:   ctTrusted,
		}

		result, err := client.CreateContact(entry)
		if err != nil {
			return err
		}

		if jsonOutput {
			return output.Current.Print(result)
		}

		fmt.Printf("Contact created (UUID: %s)\n", result.UUID)
		return nil
	},
}

var ctEditCmd = &cobra.Command{
	Use:   "update <uuid>",
	Short: "Update a contact",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := api.NewClient()
		if err != nil {
			return err
		}

		fields := map[string]interface{}{}
		if cmd.Flags().Changed("email") {
			fields["email"] = ctEmail
		}
		if cmd.Flags().Changed("phone") {
			fields["phone_number"] = ctPhone
		}
		if cmd.Flags().Changed("display-name") {
			fields["display_name"] = ctDisplayName
		}
		if cmd.Flags().Changed("nickname") {
			fields["nickname"] = ctNickname
		}
		if cmd.Flags().Changed("trusted") {
			fields["is_trusted"] = ctTrusted
		}

		if len(fields) == 0 {
			return fmt.Errorf("no fields specified to update")
		}

		result, err := client.UpdateContact(args[0], fields)
		if err != nil {
			return err
		}

		if jsonOutput {
			return output.Current.Print(result)
		}

		fmt.Printf("Contact updated (UUID: %s)\n", result.UUID)
		return nil
	},
}

var ctDeleteCmd = &cobra.Command{
	Use:   "delete <uuid>",
	Short: "Delete a contact",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := api.NewClient()
		if err != nil {
			return err
		}

		if err := client.DeleteContact(args[0]); err != nil {
			return err
		}

		if jsonOutput {
			return output.Current.Print(map[string]string{"status": "deleted"})
		}

		fmt.Println("Contact deleted.")
		return nil
	},
}

func init() {
	// Create flags
	ctCreateCmd.Flags().StringVar(&ctEmail, "email", "", "Contact email address")
	ctCreateCmd.Flags().StringVar(&ctPhone, "phone", "", "Contact phone number")
	ctCreateCmd.Flags().StringVar(&ctDisplayName, "display-name", "", "Contact display name")
	ctCreateCmd.Flags().StringVar(&ctNickname, "nickname", "", "Contact nickname")
	ctCreateCmd.Flags().BoolVar(&ctTrusted, "trusted", false, "Mark contact as trusted")

	// Edit flags
	ctEditCmd.Flags().StringVar(&ctEmail, "email", "", "New email address")
	ctEditCmd.Flags().StringVar(&ctPhone, "phone", "", "New phone number")
	ctEditCmd.Flags().StringVar(&ctDisplayName, "display-name", "", "New display name")
	ctEditCmd.Flags().StringVar(&ctNickname, "nickname", "", "New nickname")
	ctEditCmd.Flags().BoolVar(&ctTrusted, "trusted", false, "Set trusted status")

	// Wire up command tree
	contactsCmd.AddCommand(ctListCmd)
	contactsCmd.AddCommand(ctSearchCmd)
	contactsCmd.AddCommand(ctGetCmd)
	contactsCmd.AddCommand(ctCreateCmd)
	contactsCmd.AddCommand(ctEditCmd)
	contactsCmd.AddCommand(ctDeleteCmd)
	rootCmd.AddCommand(contactsCmd)
}
