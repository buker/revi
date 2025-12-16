package cli

import (
	"github.com/spf13/cobra"
)

func init() {
	// Share the flags with the commit subcommand
	commitCmd.Flags().BoolP("dry-run", "n", false, "Preview commit message without committing")
	commitCmd.Flags().StringP("message", "m", "", "Context explaining why this change was made")
}

var commitCmd = &cobra.Command{
	Use:   "commit",
	Short: "Generate commit message and commit (alias for revi)",
	Long:  `Generate an AI-powered commit message for staged changes and create the commit.`,
	RunE:  runFullWorkflow, // Reuse root command logic
}
