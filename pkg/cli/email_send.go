package cli

import (
	"fmt"
	"strings"

	"github.com/ravi-hq/cli/internal/api"
	"github.com/ravi-hq/cli/internal/output"
	"github.com/spf13/cobra"
)

var emailSendCmd = &cobra.Command{
	Use:   "email",
	Short: "Send emails (compose, reply, reply-all, forward)",
	Long:  "Compose new emails, reply to existing ones, or forward them, with optional attachments.",
}

var composeCmd = &cobra.Command{
	Use:   "compose",
	Short: "Compose and send a new email",
	Long: `Compose and send a new email from your Ravi email address.

The --body flag accepts HTML content for formatting.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		to, _ := cmd.Flags().GetString("to")
		subject, _ := cmd.Flags().GetString("subject")
		body, _ := cmd.Flags().GetString("body")
		cc, _ := cmd.Flags().GetString("cc")
		bcc, _ := cmd.Flags().GetString("bcc")
		attachPaths, _ := cmd.Flags().GetStringSlice("attach")

		client, err := api.NewClient()
		if err != nil {
			return err
		}

		inboxID, err := client.GetInboxID()
		if err != nil {
			return fmt.Errorf("failed to get inbox: %w", err)
		}

		attachmentUUIDs, err := uploadAttachments(client, attachPaths)
		if err != nil {
			return err
		}

		req := api.ComposeRequest{
			ToEmail:         to,
			Subject:         subject,
			Content:         body,
			AttachmentUUIDs: attachmentUUIDs,
		}
		if cc != "" {
			req.CC = splitAndTrim(cc)
		}
		if bcc != "" {
			req.BCC = splitAndTrim(bcc)
		}

		result, err := client.ComposeEmail(inboxID, req)
		if err != nil {
			return err
		}

		return output.Current.Print(result)
	},
}

var replyCmd = &cobra.Command{
	Use:   "reply <message_id>",
	Short: "Reply to an email (sender only)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		body, _ := cmd.Flags().GetString("body")
		cc, _ := cmd.Flags().GetString("cc")
		bcc, _ := cmd.Flags().GetString("bcc")
		attachPaths, _ := cmd.Flags().GetStringSlice("attach")

		client, err := api.NewClient()
		if err != nil {
			return err
		}

		attachmentUUIDs, err := uploadAttachments(client, attachPaths)
		if err != nil {
			return err
		}

		req := api.ReplyRequest{
			Content:         body,
			AttachmentUUIDs: attachmentUUIDs,
		}
		if cc != "" {
			req.CC = splitAndTrim(cc)
		}
		if bcc != "" {
			req.BCC = splitAndTrim(bcc)
		}

		result, err := client.ReplyEmail(args[0], req)
		if err != nil {
			return err
		}

		return output.Current.Print(result)
	},
}

var replyAllCmd = &cobra.Command{
	Use:   "reply-all <message_id>",
	Short: "Reply to all recipients of an email",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		body, _ := cmd.Flags().GetString("body")
		cc, _ := cmd.Flags().GetString("cc")
		bcc, _ := cmd.Flags().GetString("bcc")
		attachPaths, _ := cmd.Flags().GetStringSlice("attach")

		client, err := api.NewClient()
		if err != nil {
			return err
		}

		attachmentUUIDs, err := uploadAttachments(client, attachPaths)
		if err != nil {
			return err
		}

		req := api.ReplyRequest{
			Content:         body,
			AttachmentUUIDs: attachmentUUIDs,
		}
		if cc != "" {
			req.CC = splitAndTrim(cc)
		}
		if bcc != "" {
			req.BCC = splitAndTrim(bcc)
		}

		result, err := client.ReplyAllEmail(args[0], req)
		if err != nil {
			return err
		}

		return output.Current.Print(result)
	},
}

var forwardCmd = &cobra.Command{
	Use:   "forward <message_id>",
	Short: "Forward an email",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		to, _ := cmd.Flags().GetString("to")
		body, _ := cmd.Flags().GetString("body")
		cc, _ := cmd.Flags().GetString("cc")
		bcc, _ := cmd.Flags().GetString("bcc")
		attachPaths, _ := cmd.Flags().GetStringSlice("attach")

		client, err := api.NewClient()
		if err != nil {
			return err
		}

		attachmentUUIDs, err := uploadAttachments(client, attachPaths)
		if err != nil {
			return err
		}

		req := api.ForwardRequest{
			ToEmail:         to,
			Content:         body,
			AttachmentUUIDs: attachmentUUIDs,
		}
		if cc != "" {
			req.CC = splitAndTrim(cc)
		}
		if bcc != "" {
			req.BCC = splitAndTrim(bcc)
		}

		result, err := client.ForwardEmail(args[0], req)
		if err != nil {
			return err
		}

		return output.Current.Print(result)
	},
}

// uploadAttachments uploads files and returns their UUIDs.
func uploadAttachments(client *api.Client, paths []string) ([]string, error) {
	if len(paths) == 0 {
		return nil, nil
	}

	uuids := make([]string, 0, len(paths))
	for _, path := range paths {
		uuid, err := client.UploadAttachment(path)
		if err != nil {
			return nil, fmt.Errorf("attachment %q: %w", path, err)
		}
		uuids = append(uuids, uuid)
	}
	return uuids, nil
}

// splitAndTrim splits a comma-separated string and trims whitespace.
func splitAndTrim(s string) []string {
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

func init() {
	// Compose flags
	composeCmd.Flags().String("to", "", "Recipient email address (required)")
	composeCmd.Flags().String("subject", "", "Email subject (required)")
	composeCmd.Flags().String("body", "", "Email body — HTML supported (required)")
	composeCmd.Flags().String("cc", "", "CC recipients (comma-separated)")
	composeCmd.Flags().String("bcc", "", "BCC recipients (comma-separated)")
	composeCmd.Flags().StringSlice("attach", nil, "File paths to attach")
	composeCmd.MarkFlagRequired("to")
	composeCmd.MarkFlagRequired("subject")
	composeCmd.MarkFlagRequired("body")

	// Reply flags (each command gets its own flag set, no shared state)
	for _, cmd := range []*cobra.Command{replyCmd, replyAllCmd} {
		cmd.Flags().String("body", "", "Email body — HTML supported (required)")
		cmd.Flags().String("cc", "", "CC recipients (comma-separated)")
		cmd.Flags().String("bcc", "", "BCC recipients (comma-separated)")
		cmd.Flags().StringSlice("attach", nil, "File paths to attach")
		cmd.MarkFlagRequired("body")
	}

	// Forward flags
	forwardCmd.Flags().String("to", "", "Recipient email address (required)")
	forwardCmd.Flags().String("body", "", "Email body — HTML supported (required)")
	forwardCmd.Flags().String("cc", "", "CC recipients (comma-separated)")
	forwardCmd.Flags().String("bcc", "", "BCC recipients (comma-separated)")
	forwardCmd.Flags().StringSlice("attach", nil, "File paths to attach")
	forwardCmd.MarkFlagRequired("to")
	forwardCmd.MarkFlagRequired("body")

	emailSendCmd.AddCommand(composeCmd)
	emailSendCmd.AddCommand(replyCmd)
	emailSendCmd.AddCommand(replyAllCmd)
	emailSendCmd.AddCommand(forwardCmd)
	rootCmd.AddCommand(emailSendCmd)
}
