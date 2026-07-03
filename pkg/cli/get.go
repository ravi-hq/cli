package cli

import (
	"fmt"

	"github.com/ravi-hq/cli/internal/output"
	"github.com/spf13/cobra"
)

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get the identity's channels",
	Long:  "Get the identity's email address, phone number, or the account owner.",
}

var getPhoneCmd = &cobra.Command{
	Use:   "phone",
	Short: "Get the identity's phone number",
	Long:  "Get the phone number for the identity's phone channel.",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}

		phone, err := client.GetPhone()
		if err != nil {
			return err
		}

		if !humanOutput {
			return output.Current.Print(phone)
		}

		fmt.Printf("Phone number: %s\n", phone.PhoneNumber)
		return nil
	},
}

var getOwnerCmd = &cobra.Command{
	Use:   "owner",
	Short: "Get account owner's name",
	Long:  "Get the name of the account owner (the human who owns this Ravi account).",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}

		owner, err := client.GetOwner()
		if err != nil {
			return err
		}

		output.Current.Print(owner)
		return nil
	},
}

var getEmailCmd = &cobra.Command{
	Use:   "email",
	Short: "Get the identity's email address",
	Long:  "Get the email address for the identity's email channel.",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}

		email, err := client.GetEmail()
		if err != nil {
			return err
		}

		if !humanOutput {
			return output.Current.Print(email)
		}

		fmt.Printf("Email address: %s\n", email.Email)
		return nil
	},
}

func init() {
	getCmd.AddCommand(getOwnerCmd)
	getCmd.AddCommand(getPhoneCmd)
	getCmd.AddCommand(getEmailCmd)
	rootCmd.AddCommand(getCmd)
}
