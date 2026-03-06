package cli

import (
	"github.com/ravi-hq/cli/internal/api"
	"github.com/ravi-hq/cli/internal/output"
	"github.com/spf13/cobra"
)

var messageUnreadOnly bool

var messageCmd = &cobra.Command{
	Use:   "message",
	Short: "Access individual messages",
	Long:  "Access individual SMS and email messages (not grouped by conversation/thread).",
}

// SMS message commands
var messageSMSCmd = &cobra.Command{
	Use:   "sms [message_id]",
	Short: "List or view SMS messages",
	Long: `List all SMS messages or view a specific message by ID.

Without arguments, lists all SMS messages (newest first).
With a message ID, shows the specific message details.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := api.NewClient()
		if err != nil {
			return err
		}

		// If message ID provided, fetch that specific message
		if len(args) > 0 {
			message, err := client.GetSMSMessage(args[0])
			if err != nil {
				return err
			}

			output.Current.Print(message)
			return nil
		}

		// Otherwise list all messages
		messages, err := client.ListSMSMessages(messageUnreadOnly)
		if err != nil {
			return err
		}

		output.Current.Print(messages)
		return nil
	},
}

// Email message commands
var messageEmailCmd = &cobra.Command{
	Use:   "email [message_id]",
	Short: "List or view email messages",
	Long: `List all email messages or view a specific message by ID.

Without arguments, lists all email messages (newest first).
With a message ID, shows the specific message details.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := api.NewClient()
		if err != nil {
			return err
		}

		// If message ID provided, fetch that specific message
		if len(args) > 0 {
			message, err := client.GetEmailMessage(args[0])
			if err != nil {
				return err
			}

			output.Current.Print(message)
			return nil
		}

		// Otherwise list all messages
		messages, err := client.ListEmailMessages(messageUnreadOnly)
		if err != nil {
			return err
		}

		output.Current.Print(messages)
		return nil
	},
}

func init() {
	messageSMSCmd.Flags().BoolVar(&messageUnreadOnly, "unread", false, "Show only unread messages")
	messageEmailCmd.Flags().BoolVar(&messageUnreadOnly, "unread", false, "Show only unread messages")

	messageCmd.AddCommand(messageSMSCmd)
	messageCmd.AddCommand(messageEmailCmd)
	rootCmd.AddCommand(messageCmd)
}
