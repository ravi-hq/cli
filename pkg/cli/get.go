package cli

import (
	"github.com/ravi-hq/cli/internal/api"
	"github.com/ravi-hq/cli/internal/output"
	"github.com/spf13/cobra"
)

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get assigned resources",
	Long:  "Get your assigned Ravi phone number or email address.",
}

var getPhoneCmd = &cobra.Command{
	Use:   "phone",
	Short: "Get your assigned phone number",
	Long:  "Get the Ravi phone number assigned to your account.",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := api.NewClient(nil)
		if err != nil {
			return err
		}

		phone, err := client.GetPhone()
		if err != nil {
			return err
		}

		output.Current.Print(phone)
		return nil
	},
}

var getOwnerCmd = &cobra.Command{
	Use:   "owner",
	Short: "Get account owner's name",
	Long:  "Get the name of the account owner (the human who owns this Ravi account).",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := api.NewClient(nil)
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
	Short: "Get your assigned email address",
	Long:  "Get the Ravi email address assigned to your account.",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := api.NewClient(nil)
		if err != nil {
			return err
		}

		email, err := client.GetEmail()
		if err != nil {
			return err
		}

		output.Current.Print(email)
		return nil
	},
}

func init() {
	getCmd.AddCommand(getOwnerCmd)
	getCmd.AddCommand(getPhoneCmd)
	getCmd.AddCommand(getEmailCmd)
	rootCmd.AddCommand(getCmd)
}
