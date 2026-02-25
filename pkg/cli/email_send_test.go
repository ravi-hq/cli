package cli

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestEmailSendCommandRegistered(t *testing.T) {
	// Verify "email" command is registered on root
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "email" {
			found = true

			// Verify subcommands
			subNames := make(map[string]bool)
			for _, sub := range cmd.Commands() {
				subNames[sub.Name()] = true
			}

			for _, expected := range []string{"compose", "reply", "reply-all"} {
				if !subNames[expected] {
					t.Errorf("email command missing subcommand %q", expected)
				}
			}
			break
		}
	}
	if !found {
		t.Error("rootCmd does not have 'email' subcommand")
	}
}

func TestComposeRequiredFlags(t *testing.T) {
	// Verify required flags exist on compose command
	requiredFlags := []string{"to", "subject", "body"}
	for _, name := range requiredFlags {
		flag := composeCmd.Flags().Lookup(name)
		if flag == nil {
			t.Errorf("compose command missing flag %q", name)
			continue
		}
	}
}

func TestReplyRequiredFlags(t *testing.T) {
	requiredFlags := []string{"subject", "body"}
	for _, cmd := range []*cobra.Command{replyCmd, replyAllCmd} {
		for _, name := range requiredFlags {
			flag := cmd.Flags().Lookup(name)
			if flag == nil {
				t.Errorf("%s command missing flag %q", cmd.Name(), name)
			}
		}
	}
}

func TestSplitAndTrim(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"a@b.com, c@d.com", []string{"a@b.com", "c@d.com"}},
		{"a@b.com", []string{"a@b.com"}},
		{"", []string{}},
		{"  a@b.com  ,  c@d.com  ", []string{"a@b.com", "c@d.com"}},
	}

	for _, tc := range tests {
		result := splitAndTrim(tc.input)
		if len(result) != len(tc.expected) {
			t.Errorf("splitAndTrim(%q) = %v (len %d), want %v (len %d)",
				tc.input, result, len(result), tc.expected, len(tc.expected))
			continue
		}
		for i, v := range result {
			if v != tc.expected[i] {
				t.Errorf("splitAndTrim(%q)[%d] = %q, want %q", tc.input, i, v, tc.expected[i])
			}
		}
	}
}
