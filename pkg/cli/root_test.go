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
func newTestRootCmd() *cobra.Command {
	var humanOutput bool

	cmd := &cobra.Command{
		Use:   "ravi",
		Short: "Ravi CLI — identity, email, phone, and credentials for AI agents",
		Long: `Ravi CLI — identity, email, phone, and credentials for AI agents.

JSON output by default. Use --human for human-readable output.`,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			output.SetJSON(!humanOutput)
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.PersistentFlags().BoolVar(&humanOutput, "human", false, "Output in human-readable format")

	cmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Println(version.Info())
		},
	})

	return cmd
}

func TestRootCmd_Help(t *testing.T) {
	cmd := newTestRootCmd()

	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(stdout)
	cmd.SetArgs([]string{"--help"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Execute() with --help returned error: %v", err)
	}

	out := stdout.String()

	expectedStrings := []string{
		"ravi",
		"Ravi CLI",
		"--human",
		"human-readable",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(out, expected) {
			t.Errorf("Help output missing expected string %q\nGot:\n%s", expected, out)
		}
	}
}

func TestRootCmd_DefaultJSON(t *testing.T) {
	originalFormatter := output.Current
	defer func() { output.Current = originalFormatter }()

	output.SetJSON(false) // reset

	cmd := newTestRootCmd()
	testSubCmd := &cobra.Command{
		Use: "testcmd",
		Run: func(cmd *cobra.Command, args []string) {},
	}
	cmd.AddCommand(testSubCmd)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	// Without --human, should default to JSON
	cmd.SetArgs([]string{"testcmd"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	if _, ok := output.Current.(*output.JSONFormatter); !ok {
		t.Error("Default output should be JSONFormatter")
	}
}

func TestRootCmd_HumanFlag(t *testing.T) {
	originalFormatter := output.Current
	defer func() { output.Current = originalFormatter }()

	cmd := newTestRootCmd()
	testSubCmd := &cobra.Command{
		Use: "testcmd",
		Run: func(cmd *cobra.Command, args []string) {},
	}
	cmd.AddCommand(testSubCmd)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	// With --human, should use HumanFormatter
	cmd.SetArgs([]string{"--human", "testcmd"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() with --human returned error: %v", err)
	}

	if _, ok := output.Current.(*output.HumanFormatter); !ok {
		t.Error("With --human flag, formatter should be HumanFormatter")
	}
}

func TestRootCmd_Version(t *testing.T) {
	originalVersion := version.Version
	originalCommit := version.Commit
	originalBuildDate := version.BuildDate
	defer func() {
		version.Version = originalVersion
		version.Commit = originalCommit
		version.BuildDate = originalBuildDate
	}()

	version.Version = "1.0.0-test"
	version.Commit = "abc123test"
	version.BuildDate = "2024-06-15T12:00:00Z"

	cmd := newTestRootCmd()

	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(stdout)
	cmd.SetArgs([]string{"version"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Execute() version returned error: %v", err)
	}

	out := stdout.String()

	expectedStrings := []string{
		"ravi version",
		"1.0.0-test",
		"abc123test",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(out, expected) {
			t.Errorf("Version output missing expected string %q\nGot:\n%s", expected, out)
		}
	}
}
