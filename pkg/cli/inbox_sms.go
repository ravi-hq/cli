package cli

import (
	"fmt"
	"strings"

	"github.com/ravi-hq/cli/internal/api"
	"github.com/ravi-hq/cli/internal/output"
	"github.com/spf13/cobra"
)

var smsUnread bool

var smsCmd = &cobra.Command{
	Use:   "sms [conversation_id]",
	Short: "List SMS conversations or view a specific conversation",
	Long: `List SMS conversations or view a specific conversation.

Without arguments, lists all SMS conversations.
With a conversation_id argument, shows the full conversation.

Conversation IDs are in the format: {phone_id}_{from_number}
Example: 1_+15551234567`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := api.NewClient()
		if err != nil {
			return err
		}

		// If conversation_id provided, show conversation detail
		if len(args) > 0 {
			return showSMSConversation(client, args[0])
		}

		// Otherwise, list conversations
		return listSMSConversations(client)
	},
}

func listSMSConversations(client *api.Client) error {
	conversations, err := client.ListSMSConversations(smsUnread)
	if err != nil {
		return err
	}

	kp, err := ensureKeyPair()
	if err != nil {
		return err
	}

	for i := range conversations {
		conversations[i].Preview = tryDecrypt(conversations[i].Preview, kp)
	}

	if jsonOutput {
		return output.Current.Print(conversations)
	}

	if len(conversations) == 0 {
		output.Current.PrintMessage("No SMS conversations found")
		return nil
	}

	headers := []string{"CONVERSATION ID", "FROM", "YOUR NUMBER", "PREVIEW", "MSGS", "UNREAD", "DATE"}
	rows := make([][]string, len(conversations))
	for i, c := range conversations {
		rows[i] = []string{
			truncate(c.ConversationID, 20),
			c.FromNumber,
			c.PhoneNumber,
			truncate(c.Preview, 25),
			fmt.Sprintf("%d", c.MessageCount),
			fmt.Sprintf("%d", c.UnreadCount),
			c.LatestMessageDt.Format("Jan 02 15:04"),
		}
	}
	output.Current.PrintTable(headers, rows)
	return nil
}

func showSMSConversation(client *api.Client, conversationID string) error {
	conversation, err := client.GetSMSConversation(conversationID)
	if err != nil {
		return err
	}

	kp, err := ensureKeyPair()
	if err != nil {
		return err
	}

	for i := range conversation.Messages {
		conversation.Messages[i].Body = tryDecrypt(conversation.Messages[i].Body, kp)
	}

	if jsonOutput {
		return output.Current.Print(conversation)
	}

	// Human-readable conversation display
	fmt.Printf("Conversation: %s\n", conversation.ConversationID)
	fmt.Printf("From: %s\n", conversation.FromNumber)
	fmt.Printf("Your Number: %s\n", conversation.Phone)
	fmt.Printf("Messages: %d\n", conversation.MessageCount)
	fmt.Println(strings.Repeat("-", 60))

	for _, msg := range conversation.Messages {
		direction := "->"
		sender := conversation.Phone
		if msg.Direction == "incoming" {
			direction = "<-"
			sender = conversation.FromNumber
		}
		readStatus := ""
		if !msg.IsRead {
			readStatus = " [UNREAD]"
		}

		fmt.Printf("\n%s %s%s\n", direction, sender, readStatus)
		fmt.Printf("  %s\n", msg.CreatedDt.Format("Jan 02, 2006 3:04 PM"))
		fmt.Println()
		fmt.Println(msg.Body)
		fmt.Println(strings.Repeat("-", 60))
	}

	return nil
}

func init() {
	smsCmd.Flags().BoolVar(&smsUnread, "unread", false, "Only show conversations with unread messages")
	inboxCmd.AddCommand(smsCmd)
}
