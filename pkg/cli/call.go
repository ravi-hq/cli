package cli

import (
	"fmt"
	"strings"

	"github.com/ravi-hq/cli/internal/api"
	"github.com/ravi-hq/cli/internal/output"
	"github.com/spf13/cobra"
)

var callTo string

var callCmd = &cobra.Command{
	Use:   "call",
	Short: "Place and manage phone calls",
	Long: `Place outbound calls from the identity's phone number and manage them.

Running "call --to <e164>" places a call. Subcommands list calls, fetch a
transcript, and hang up an active call.

Identity-scoped keys call from the bound identity's phone. Management keys
must target an identity with --identity <uuid>.`,
	// Placing a call directly on `call --to` is a convenience; when a
	// subcommand is given, cobra dispatches to it instead.
	RunE: func(cmd *cobra.Command, args []string) error {
		if callTo == "" {
			return cmd.Help()
		}

		client, err := newClient()
		if err != nil {
			return err
		}

		call, err := client.StartCall(api.CallStartRequest{ToNumber: callTo})
		if err != nil {
			return err
		}

		if !humanOutput {
			return output.Current.Print(call)
		}

		fmt.Printf("Call started from %s to %s (id: %d, status: %s)\n",
			call.FromNumber, call.ToNumber, call.ID, call.Status)
		return nil
	},
}

var callListCmd = &cobra.Command{
	Use:   "list",
	Short: "List calls",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}

		calls, err := client.ListCalls()
		if err != nil {
			return err
		}

		if !humanOutput {
			return output.Current.Print(calls)
		}

		if len(calls) == 0 {
			output.Current.PrintMessage("No calls found")
			return nil
		}

		headers := []string{"ID", "FROM", "TO", "DIRECTION", "STATUS", "CREATED"}
		rows := make([][]string, len(calls))
		for i, c := range calls {
			rows[i] = []string{
				fmt.Sprintf("%d", c.ID),
				c.FromNumber,
				c.ToNumber,
				c.Direction,
				c.Status,
				c.CreatedDt.Format("Jan 02 15:04"),
			}
		}
		output.Current.PrintTable(headers, rows)
		return nil
	},
}

var callTranscriptCmd = &cobra.Command{
	Use:   "transcript <call_id>",
	Short: "Show a call transcript",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}

		segments, err := client.GetCallTranscript(args[0])
		if err != nil {
			return err
		}

		if !humanOutput {
			return output.Current.Print(segments)
		}

		if len(segments) == 0 {
			output.Current.PrintMessage("No transcript segments found")
			return nil
		}

		for _, seg := range segments {
			fmt.Printf("[%s] %s\n", seg.Speaker, seg.Text)
		}
		fmt.Println(strings.Repeat("-", 60))
		return nil
	},
}

var callHangupCmd = &cobra.Command{
	Use:   "hangup <call_id>",
	Short: "Hang up an active call",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}

		call, err := client.HangupCall(args[0])
		if err != nil {
			return err
		}

		if !humanOutput {
			return output.Current.Print(call)
		}

		fmt.Printf("Call %d hung up (status: %s)\n", call.ID, call.Status)
		return nil
	},
}

func init() {
	callCmd.Flags().StringVar(&callTo, "to", "", "Recipient phone number in E.164 format")

	callCmd.AddCommand(callListCmd)
	callCmd.AddCommand(callTranscriptCmd)
	callCmd.AddCommand(callHangupCmd)
	rootCmd.AddCommand(callCmd)
}
