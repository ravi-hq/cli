package cli

import (
	"fmt"
	"strings"

	"github.com/ravi-hq/cli/internal/api"
	"github.com/ravi-hq/cli/internal/output"
	"github.com/spf13/cobra"
)

var emailUnread bool

var emailCmd = &cobra.Command{
	Use:   "email [thread_id]",
	Short: "List email threads or view a specific thread",
	Long: `List email threads or view a specific thread.

Without arguments, lists all email threads.
With a thread_id argument, shows the full thread conversation.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := api.NewClient()
		if err != nil {
			return err
		}

		// If thread_id provided, show thread detail
		if len(args) > 0 {
			return showEmailThread(client, args[0])
		}

		// Otherwise, list threads
		return listEmailThreads(client)
	},
}

func listEmailThreads(client *api.Client) error {
	threads, err := client.ListEmailThreads(emailUnread)
	if err != nil {
		return err
	}

	kp, err := ensureKeyPair()
	if err != nil {
		return err
	}

	for i := range threads {
		threads[i].Subject = tryDecrypt(threads[i].Subject, kp)
		threads[i].Preview = tryDecrypt(threads[i].Preview, kp)
	}

	if jsonOutput {
		return output.Current.Print(threads)
	}

	if len(threads) == 0 {
		output.Current.PrintMessage("No email threads found")
		return nil
	}

	headers := []string{"THREAD ID", "FROM", "SUBJECT", "MSGS", "UNREAD", "DATE"}
	rows := make([][]string, len(threads))
	for i, t := range threads {
		rows[i] = []string{
			truncate(t.ThreadID, 20),
			truncate(t.FromEmail, 25),
			truncate(t.Subject, 30),
			fmt.Sprintf("%d", t.MessageCount),
			fmt.Sprintf("%d", t.UnreadCount),
			t.LatestMessageDt.Format("Jan 02 15:04"),
		}
	}
	output.Current.PrintTable(headers, rows)
	return nil
}

func showEmailThread(client *api.Client, threadID string) error {
	thread, err := client.GetEmailThread(threadID)
	if err != nil {
		return err
	}

	kp, err := ensureKeyPair()
	if err != nil {
		return err
	}

	thread.Subject = tryDecrypt(thread.Subject, kp)
	for i := range thread.Messages {
		thread.Messages[i].Subject = tryDecrypt(thread.Messages[i].Subject, kp)
		thread.Messages[i].TextContent = tryDecrypt(thread.Messages[i].TextContent, kp)
		thread.Messages[i].HTMLContent = tryDecrypt(thread.Messages[i].HTMLContent, kp)
	}

	if jsonOutput {
		return output.Current.Print(thread)
	}

	// Human-readable thread display
	fmt.Printf("Thread: %s\n", thread.ThreadID)
	fmt.Printf("Subject: %s\n", thread.Subject)
	fmt.Printf("Messages: %d\n", thread.MessageCount)
	fmt.Println(strings.Repeat("-", 60))

	for _, msg := range thread.Messages {
		direction := "->"
		if msg.Direction == "incoming" {
			direction = "<-"
		}
		readStatus := ""
		if !msg.IsRead {
			readStatus = " [UNREAD]"
		}

		fmt.Printf("\n%s %s%s\n", direction, msg.FromEmail, readStatus)
		fmt.Printf("  To: %s\n", msg.ToEmail)
		if msg.CC != "" {
			fmt.Printf("  CC: %s\n", msg.CC)
		}
		fmt.Printf("  Date: %s\n", msg.CreatedDt.Format("Jan 02, 2006 3:04 PM"))
		fmt.Println()

		// Print text content (prefer over HTML)
		content := msg.TextContent
		if content == "" {
			content = "(HTML content only - view in browser)"
		}
		fmt.Println(content)
		fmt.Println(strings.Repeat("-", 60))
	}

	return nil
}

func init() {
	emailCmd.Flags().BoolVar(&emailUnread, "unread", false, "Only show threads with unread messages")
	inboxCmd.AddCommand(emailCmd)
}
