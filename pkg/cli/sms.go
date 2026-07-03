package cli

import (
	"fmt"

	"github.com/ravi-hq/cli/internal/api"
	"github.com/ravi-hq/cli/internal/output"
	"github.com/spf13/cobra"
)

var (
	smsSendTo   string
	smsSendBody string
)

// smsRootCmd is the top-level `sms` command for sending messages from the
// identity's phone channel. (Reading conversations lives under `inbox sms`.)
var smsRootCmd = &cobra.Command{
	Use:   "sms",
	Short: "Send SMS from the identity's phone number",
	Long:  "Send SMS messages from the identity's provisioned phone number.",
}

var smsSendCmd = &cobra.Command{
	Use:   "send",
	Short: "Send an SMS",
	Long: `Send an SMS from the identity's phone number.

Identity-scoped keys send from the bound identity's phone. Management keys
must target an identity with --identity <uuid>.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}

		msg, err := client.SendSMS(api.SmsSendRequest{
			ToNumber: smsSendTo,
			Body:     smsSendBody,
		})
		if err != nil {
			return err
		}

		if !humanOutput {
			return output.Current.Print(msg)
		}

		fmt.Printf("SMS sent from %s to %s\n", msg.FromNumber, msg.ToNumber)
		fmt.Printf("Body: %s\n", msg.Body)
		return nil
	},
}

func init() {
	smsSendCmd.Flags().StringVar(&smsSendTo, "to", "", "Recipient phone number in E.164 format (required)")
	smsSendCmd.Flags().StringVar(&smsSendBody, "body", "", "Message body (required)")
	smsSendCmd.MarkFlagRequired("to")
	smsSendCmd.MarkFlagRequired("body")

	smsRootCmd.AddCommand(smsSendCmd)
	rootCmd.AddCommand(smsRootCmd)
}
