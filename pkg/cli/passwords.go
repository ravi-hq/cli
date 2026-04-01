package cli

import (
	"encoding/base64"
	"fmt"

	"github.com/ravi-hq/cli/internal/api"
	"github.com/ravi-hq/cli/internal/crypto"
	"github.com/ravi-hq/cli/internal/output"
	"github.com/spf13/cobra"
)

// Flag variables for passwords commands
var (
	pwGenerate     bool
	pwLength       int
	pwNoSpecial    bool
	pwNoDigits     bool
	pwExcludeChars string
	pwUsername     string
	pwPassword     string
	pwNotes        string
	pwDomain       string
)

var vaultCmd = &cobra.Command{
	Use:   "vault",
	Short: "Manage credential vault",
}

var pwListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all stored passwords",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := api.NewClient()
		if err != nil {
			return err
		}

		entries, err := client.ListPasswords()
		if err != nil {
			return err
		}

		kp, err := ensureKeyPair()
		if err != nil {
			return err
		}

		for i := range entries {
			entries[i].Username = tryDecrypt(entries[i].Username, kp)
		}

		if jsonOutput {
			return output.Current.Print(entries)
		}

		if len(entries) == 0 {
			output.Current.PrintMessage("No passwords found")
			return nil
		}

		headers := []string{"UUID", "DOMAIN", "USERNAME", "CREATED"}
		rows := make([][]string, len(entries))
		for i, e := range entries {
			rows[i] = []string{
				truncate(e.UUID, 12),
				truncate(e.Domain, 25),
				truncate(e.Username, 30),
				e.CreatedDt,
			}
		}
		output.Current.PrintTable(headers, rows)
		return nil
	},
}

var pwGetCmd = &cobra.Command{
	Use:   "get <uuid>",
	Short: "Show a stored password",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := api.NewClient()
		if err != nil {
			return err
		}

		entry, err := client.GetPassword(args[0])
		if err != nil {
			return err
		}

		kp, err := ensureKeyPair()
		if err != nil {
			return err
		}

		entry.Username = tryDecrypt(entry.Username, kp)
		entry.Password = tryDecrypt(entry.Password, kp)
		entry.Notes = tryDecrypt(entry.Notes, kp)

		if jsonOutput {
			return output.Current.Print(entry)
		}

		fmt.Printf("Domain:   %s\n", entry.Domain)
		fmt.Printf("Username: %s\n", entry.Username)
		fmt.Printf("Password: %s\n", entry.Password)
		if entry.Notes != "" {
			fmt.Printf("Notes:    %s\n", entry.Notes)
		}
		fmt.Printf("UUID:     %s\n", entry.UUID)
		fmt.Printf("Created:  %s\n", entry.CreatedDt)
		return nil
	},
}

var pwCreateCmd = &cobra.Command{
	Use:   "create <domain>",
	Short: "Create a new password entry",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := api.NewClient()
		if err != nil {
			return err
		}

		kp, err := ensureKeyPair()
		if err != nil {
			return err
		}

		password := pwPassword
		if pwGenerate || password == "" {
			opts := api.PasswordGenOpts{
				Length:       pwLength,
				NoDigits:     pwNoDigits,
				NoSpecial:    pwNoSpecial,
				ExcludeChars: pwExcludeChars,
			}
			gen, err := client.GeneratePassword(opts)
			if err != nil {
				return fmt.Errorf("generating password: %w", err)
			}
			password = gen.Password
			if !pwGenerate {
				fmt.Printf("Generated password: %s\n", password)
			}
		}

		entry := api.PasswordEntry{Domain: args[0]}
		if kp != nil {
			pubKeyB64 := encodePublicKey(kp)
			entry.Username, err = crypto.Encrypt(pwUsername, pubKeyB64)
			if err != nil {
				return fmt.Errorf("encrypting username: %w", err)
			}
			entry.Password, err = crypto.Encrypt(password, pubKeyB64)
			if err != nil {
				return fmt.Errorf("encrypting password: %w", err)
			}
			entry.Notes, err = crypto.Encrypt(pwNotes, pubKeyB64)
			if err != nil {
				return fmt.Errorf("encrypting notes: %w", err)
			}
		} else {
			entry.Username = pwUsername
			entry.Password = password
			entry.Notes = pwNotes
		}

		result, err := client.CreatePassword(entry)
		if err != nil {
			return err
		}

		if jsonOutput {
			return output.Current.Print(result)
		}

		fmt.Printf("Password entry created for %s (UUID: %s)\n", result.Domain, result.UUID)
		return nil
	},
}

var pwEditCmd = &cobra.Command{
	Use:   "edit <uuid>",
	Short: "Edit a stored password entry",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := api.NewClient()
		if err != nil {
			return err
		}

		kp, err := ensureKeyPair()
		if err != nil {
			return err
		}

		fields := map[string]interface{}{}
		if cmd.Flags().Changed("domain") {
			fields["domain"] = pwDomain
		}
		if cmd.Flags().Changed("username") {
			if kp != nil {
				enc, err := crypto.Encrypt(pwUsername, encodePublicKey(kp))
				if err != nil {
					return fmt.Errorf("encrypting username: %w", err)
				}
				fields["username"] = enc
			} else {
				fields["username"] = pwUsername
			}
		}
		if cmd.Flags().Changed("password") {
			if kp != nil {
				enc, err := crypto.Encrypt(pwPassword, encodePublicKey(kp))
				if err != nil {
					return fmt.Errorf("encrypting password: %w", err)
				}
				fields["password"] = enc
			} else {
				fields["password"] = pwPassword
			}
		}
		if cmd.Flags().Changed("notes") {
			if kp != nil {
				enc, err := crypto.Encrypt(pwNotes, encodePublicKey(kp))
				if err != nil {
					return fmt.Errorf("encrypting notes: %w", err)
				}
				fields["notes"] = enc
			} else {
				fields["notes"] = pwNotes
			}
		}

		if len(fields) == 0 {
			return fmt.Errorf("no fields specified to update")
		}

		result, err := client.UpdatePassword(args[0], fields)
		if err != nil {
			return err
		}

		if jsonOutput {
			return output.Current.Print(result)
		}

		fmt.Printf("Password entry updated for %s\n", result.Domain)
		return nil
	},
}

var pwDeleteCmd = &cobra.Command{
	Use:   "delete <uuid>",
	Short: "Delete a stored password entry",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := api.NewClient()
		if err != nil {
			return err
		}

		if err := client.DeletePassword(args[0]); err != nil {
			return err
		}

		if jsonOutput {
			return output.Current.Print(map[string]string{"status": "deleted"})
		}

		fmt.Println("Password entry deleted.")
		return nil
	},
}

var pwGenerateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate a random password",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := api.NewClient()
		if err != nil {
			return err
		}

		opts := api.PasswordGenOpts{
			Length:       pwLength,
			NoDigits:     pwNoDigits,
			NoSpecial:    pwNoSpecial,
			ExcludeChars: pwExcludeChars,
		}

		gen, err := client.GeneratePassword(opts)
		if err != nil {
			return err
		}

		if jsonOutput {
			return output.Current.Print(gen)
		}

		fmt.Println(gen.Password)
		return nil
	},
}

// encodePublicKey converts a KeyPair's public key to base64.
func encodePublicKey(kp *crypto.KeyPair) string {
	return base64.StdEncoding.EncodeToString(kp.PublicKey[:])
}

func init() {
	// Create flags
	pwCreateCmd.Flags().StringVar(&pwPassword, "password", "", "Password (if empty, auto-generates)")
	pwCreateCmd.Flags().BoolVar(&pwGenerate, "generate", false, "Auto-generate password")
	pwCreateCmd.Flags().IntVar(&pwLength, "length", 16, "Generated password length")
	pwCreateCmd.Flags().BoolVar(&pwNoSpecial, "no-special", false, "Exclude special characters")
	pwCreateCmd.Flags().BoolVar(&pwNoDigits, "no-digits", false, "Exclude digits")
	pwCreateCmd.Flags().StringVar(&pwExcludeChars, "exclude-chars", "", "Exclude specific characters")
	pwCreateCmd.Flags().StringVar(&pwUsername, "username", "", "Username (defaults to identity email)")
	pwCreateCmd.Flags().StringVar(&pwNotes, "notes", "", "Optional notes")

	// Edit flags
	pwEditCmd.Flags().StringVar(&pwDomain, "domain", "", "New domain")
	pwEditCmd.Flags().StringVar(&pwUsername, "username", "", "New username")
	pwEditCmd.Flags().StringVar(&pwPassword, "password", "", "New password")
	pwEditCmd.Flags().StringVar(&pwNotes, "notes", "", "New notes")

	// Generate flags
	pwGenerateCmd.Flags().IntVar(&pwLength, "length", 16, "Password length")
	pwGenerateCmd.Flags().BoolVar(&pwNoSpecial, "no-special", false, "Exclude special characters")
	pwGenerateCmd.Flags().BoolVar(&pwNoDigits, "no-digits", false, "Exclude digits")
	pwGenerateCmd.Flags().StringVar(&pwExcludeChars, "exclude-chars", "", "Exclude specific characters")

	// Wire up command tree
	vaultCmd.AddCommand(pwListCmd)
	vaultCmd.AddCommand(pwGetCmd)
	vaultCmd.AddCommand(pwCreateCmd)
	vaultCmd.AddCommand(pwEditCmd)
	vaultCmd.AddCommand(pwDeleteCmd)
	vaultCmd.AddCommand(pwGenerateCmd)
	rootCmd.AddCommand(vaultCmd)
}
