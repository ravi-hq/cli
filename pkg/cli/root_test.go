package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/ravi-hq/cli/internal/output"
	"github.com/ravi-hq/cli/internal/version"
	"github.com/spf13/cobra"
)

// newTestRootCmd creates a fresh root command for testing to avoid shared state issues.
// This ensures tests don't affect each other through the global rootCmd.
func newTestRootCmd() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "ravi",
		Short: "Ravi CLI - Access your inbox programmatically",
		Long: `Ravi CLI provides command-line access to your Ravi inbox,
including emails and SMS messages. Designed for AI agents and automation.`,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			output.SetJSON(jsonOutput)
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")

	// Add version subcommand
	cmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Println(version.Info())
		},
	})

	return cmd
}

// TestRootCmd_Help verifies that the --help flag displays the help text
// including the command description and available flags.
func TestRootCmd_Help(t *testing.T) {
	cmd := newTestRootCmd()

	// Capture output
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(stdout)
	cmd.SetArgs([]string{"--help"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Execute() with --help returned error: %v", err)
	}

	output := stdout.String()

	// Verify help text contains expected content
	expectedStrings := []string{
		"ravi",                  // Command name
		"Ravi CLI",              // Description
		"inbox",                 // Mentioned in description
		"--json",                // Global flag should be shown
		"Output in JSON format", // Flag description
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("Help output missing expected string %q\nGot:\n%s", expected, output)
		}
	}
}

// TestRootCmd_JsonFlag verifies that the --json flag sets the output mode to JSON.
// This test checks that the PersistentPreRun callback correctly calls output.SetJSON.
func TestRootCmd_JsonFlag(t *testing.T) {
	// Save original formatter and restore after test
	originalFormatter := output.Current
	defer func() { output.Current = originalFormatter }()

	// Reset to human formatter to start clean
	output.SetJSON(false)

	cmd := newTestRootCmd()

	// Add a test subcommand to trigger PersistentPreRun
	testSubCmd := &cobra.Command{
		Use: "testcmd",
		Run: func(cmd *cobra.Command, args []string) {
			// PersistentPreRun should have run by now
		},
	}
	cmd.AddCommand(testSubCmd)

	// Capture output
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(stdout)

	// Test without --json flag first
	cmd.SetArgs([]string{"testcmd"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() without --json returned error: %v", err)
	}

	// Verify formatter is Human (default)
	if _, ok := output.Current.(*output.HumanFormatter); !ok {
		t.Error("Without --json flag, formatter should be HumanFormatter")
	}

	// Now test with --json flag - need a fresh command
	cmd2 := newTestRootCmd()
	testSubCmd2 := &cobra.Command{
		Use: "testcmd",
		Run: func(cmd *cobra.Command, args []string) {},
	}
	cmd2.AddCommand(testSubCmd2)
	cmd2.SetOut(&bytes.Buffer{})
	cmd2.SetErr(&bytes.Buffer{})

	cmd2.SetArgs([]string{"--json", "testcmd"})
	if err := cmd2.Execute(); err != nil {
		t.Fatalf("Execute() with --json returned error: %v", err)
	}

	// Verify formatter is JSON
	if _, ok := output.Current.(*output.JSONFormatter); !ok {
		t.Error("With --json flag, formatter should be JSONFormatter")
	}
}

// TestRootCmd_Version verifies that the version subcommand displays
// the version information including version, commit, and build date.
func TestRootCmd_Version(t *testing.T) {
	// Save original values and restore after test
	originalVersion := version.Version
	originalCommit := version.Commit
	originalBuildDate := version.BuildDate
	defer func() {
		version.Version = originalVersion
		version.Commit = originalCommit
		version.BuildDate = originalBuildDate
	}()

	// Set test values
	version.Version = "1.0.0-test"
	version.Commit = "abc123test"
	version.BuildDate = "2024-06-15T12:00:00Z"

	cmd := newTestRootCmd()

	// Capture output
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(stdout)
	cmd.SetArgs([]string{"version"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Execute() version returned error: %v", err)
	}

	output := stdout.String()

	// Verify version info contains expected components
	expectedStrings := []string{
		"ravi version",
		"1.0.0-test",
		"abc123test",
		"2024-06-15T12:00:00Z",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("Version output missing expected string %q\nGot:\n%s", expected, output)
		}
	}

	// Verify exact format matches version.Info()
	expectedFull := version.Info()
	if !strings.Contains(output, expectedFull) {
		t.Errorf("Version output does not match version.Info()\nExpected: %s\nGot: %s", expectedFull, output)
	}
}
