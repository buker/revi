// Package cli implements the command-line interface for revi using cobra.
// It provides commands for commit message generation and standalone code review.
package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/buker/revi/internal/claude"
	"github.com/buker/revi/internal/config"
	"github.com/buker/revi/internal/git"
	"github.com/spf13/cobra"
)

var (
	// Version is set at build time via -ldflags
	Version = "dev"

	rootCmd = &cobra.Command{
		Use:   "revi",
		Short: "AI-powered commit message generator",
		Long: `revi generates AI-powered commit messages for staged changes.

Usage:
  revi           Generate commit message and commit
  revi review    Run AI code reviews on staged changes`,
		RunE: runFullWorkflow,
	}
)

func init() {
	cobra.OnInitialize(config.Init)

	// Root command flags
	rootCmd.Flags().BoolP("dry-run", "n", false, "Preview commit message without committing")
	rootCmd.Flags().StringP("message", "m", "", "Context explaining why this change was made")

	// Add subcommands
	rootCmd.AddCommand(reviewCmd)
	rootCmd.AddCommand(commitCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(versionCmd)
}

// Execute runs the root command and returns any error encountered.
// This is the main entry point for the CLI application.
func Execute() error {
	return rootCmd.Execute()
}

func runFullWorkflow(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	cfg := config.Get()

	// Check if Claude CLI is available
	claudeClient := claude.NewClient(cfg.Claude.Path, cfg.Claude.Timeout)
	if err := claudeClient.IsAvailable(); err != nil {
		return err
	}

	// Open git repository
	repo, err := git.OpenCurrent()
	if err != nil {
		return fmt.Errorf("failed to open git repository: %w", err)
	}

	// Check for staged changes
	hasStagedChanges, err := repo.HasStagedChanges()
	if err != nil {
		return fmt.Errorf("failed to check staged changes: %w", err)
	}
	if !hasStagedChanges {
		return fmt.Errorf("no staged changes found. Use 'git add' to stage files")
	}

	// Get staged diff
	diff, err := repo.GetStagedDiff()
	if err != nil {
		return fmt.Errorf("failed to get staged diff: %w", err)
	}

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	userContext, _ := cmd.Flags().GetString("message")

	fmt.Println("Generating commit message...")

	// Generate commit message
	msg, err := claudeClient.GenerateCommitMessage(ctx, diff, userContext)
	if err != nil {
		return fmt.Errorf("failed to generate commit message: %w", err)
	}
	commitMessage := msg.String()

	// Display commit message
	fmt.Println()
	fmt.Println(strings.Repeat("-", 40))
	fmt.Println("Commit message:")
	fmt.Println()
	fmt.Println("  " + strings.ReplaceAll(commitMessage, "\n", "\n  "))
	fmt.Println()
	fmt.Println(strings.Repeat("-", 40))

	// Ask for confirmation
	fmt.Print("\nProceed with commit? [y/N] ")
	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))

	if response != "y" && response != "yes" {
		fmt.Println("Commit cancelled.")
		return nil
	}

	if dryRun {
		fmt.Println("Dry run - commit not created.")
		return nil
	}

	// Create the commit
	hash, err := repo.Commit(commitMessage)
	if err != nil {
		return fmt.Errorf("failed to create commit: %w", err)
	}

	fmt.Printf("Created commit: %s\n", shortHash(hash))
	return nil
}

// shortHash returns a shortened version of a git hash (first 8 chars).
// Returns the full hash if it's shorter than 8 characters.
func shortHash(hash string) string {
	if len(hash) > 8 {
		return hash[:8]
	}
	return hash
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("revi version %s\n", Version)
	},
}
