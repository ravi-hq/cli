package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ravi-hq/cli/internal/output"
	"github.com/ravi-hq/cli/internal/skill"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Set up Ravi CLI for Claude Code",
	Long:  "Write the Ravi CLI skill file to ~/.claude/skills/ so Claude Code sessions can use the ravi CLI.",
	RunE: func(cmd *cobra.Command, args []string) error {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("getting home directory: %w", err)
		}

		skillsDir := filepath.Join(homeDir, ".claude", "skills")
		if err := os.MkdirAll(skillsDir, 0700); err != nil {
			return fmt.Errorf("creating skills directory: %w", err)
		}

		skillPath := filepath.Join(skillsDir, "ravi-cli.md")
		if err := os.WriteFile(skillPath, []byte(skill.Content), 0644); err != nil {
			return fmt.Errorf("writing skill file: %w", err)
		}

		output.Current.PrintMessage(fmt.Sprintf("Skill file written to %s", skillPath))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
