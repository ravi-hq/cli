package cli

import (
	"fmt"

	"github.com/ravi-hq/cli/internal/api"
	"github.com/ravi-hq/cli/internal/output"
	"github.com/spf13/cobra"
)

var feedbackCmd = &cobra.Command{
	Use:   "feedback <message>",
	Short: "Send feedback to the Ravi team",
	Long:  "Send a feedback email to feedback@ravi.id from your Ravi email address.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		message := args[0]
		subject, _ := cmd.Flags().GetString("subject")

		client, err := api.NewClient()
		if err != nil {
			return err
		}

		inboxID, err := client.GetInboxID()
		if err != nil {
			return fmt.Errorf("failed to get inbox: %w", err)
		}

		req := api.ComposeRequest{
			ToEmail: "feedback@ravi.id",
			Subject: subject,
			Content: fmt.Sprintf("<p>%s</p>", message),
		}

		result, err := client.ComposeEmail(inboxID, req)
		if err != nil {
			return err
		}

		return output.Current.Print(result)
	},
}

func init() {
	feedbackCmd.Flags().String("subject", "Feedback", "Email subject")
	rootCmd.AddCommand(feedbackCmd)
}
