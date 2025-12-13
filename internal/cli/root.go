// Package cli implements the command-line interface for revi using cobra.
// It provides commands for the full review-and-commit workflow, standalone review,
// standalone commit message generation, and configuration management.
package cli

import (
	"context"
	"fmt"

	"github.com/buker/revi/internal/claude"
	"github.com/buker/revi/internal/config"
	"github.com/buker/revi/internal/git"
	"github.com/buker/revi/internal/review"
	"github.com/buker/revi/internal/tui"
	"github.com/spf13/cobra"
)

var (
	// Version is set at build time via -ldflags
	Version = "dev"

	rootCmd = &cobra.Command{
		Use:   "revi",
		Short: "AI-powered code review and commit message generator",
		Long: `revi is an AI-powered tool that automates code review and commit message generation.

When run without subcommands, it performs the full workflow:
1. Analyze staged changes
2. Run specialized code reviews (security, performance, style, etc.)
3. Generate and confirm commit message
4. Create the commit`,
		RunE: runFullWorkflow,
	}
)

func init() {
	cobra.OnInitialize(config.Init)

	// Global flags
	rootCmd.PersistentFlags().BoolP("review", "r", true, "Enable code review")
	rootCmd.PersistentFlags().BoolP("no-review", "R", false, "Disable code review")
	rootCmd.PersistentFlags().BoolP("block", "b", true, "Block commit if issues found")
	rootCmd.PersistentFlags().BoolP("no-block", "B", false, "Don't block commit on issues")
	rootCmd.PersistentFlags().BoolP("dry-run", "n", false, "Preview without committing")

	// Review mode flags
	rootCmd.PersistentFlags().Bool("security", false, "Enable security review")
	rootCmd.PersistentFlags().Bool("no-security", false, "Disable security review")
	rootCmd.PersistentFlags().Bool("performance", false, "Enable performance review")
	rootCmd.PersistentFlags().Bool("no-performance", false, "Disable performance review")
	rootCmd.PersistentFlags().Bool("style", false, "Enable style review")
	rootCmd.PersistentFlags().Bool("no-style", false, "Disable style review")
	rootCmd.PersistentFlags().Bool("errors", false, "Enable error handling review")
	rootCmd.PersistentFlags().Bool("no-errors", false, "Disable error handling review")
	rootCmd.PersistentFlags().Bool("testing", false, "Enable testing review")
	rootCmd.PersistentFlags().Bool("no-testing", false, "Disable testing review")
	rootCmd.PersistentFlags().Bool("docs", false, "Enable documentation review")
	rootCmd.PersistentFlags().Bool("no-docs", false, "Disable documentation review")
	rootCmd.PersistentFlags().BoolP("all", "a", false, "Run all review modes")

	// Bind flags to viper
	config.BindFlags(rootCmd)

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

	// Check if review is enabled
	reviewEnabled := config.IsReviewEnabled(cmd)
	blockOnIssues := config.IsBlockEnabled(cmd)
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	// Create and start TUI
	program := tui.NewProgram()

	// Run workflow with TUI
	err = program.RunWithCallbacks(
		ctx,
		// Detect modes
		func(ctx context.Context) ([]review.Mode, string, error) {
			if !reviewEnabled {
				return nil, "Review disabled", nil
			}

			// Check for explicit mode flags
			allModes, _ := cmd.Flags().GetBool("all")
			if allModes {
				return review.AllModes(), "All modes enabled", nil
			}

			// Use Claude for detection or heuristic fallback
			detector := review.NewClaudeDetector(claudeClient.DetectModes)
			modes, reasoning, err := detector.Detect(ctx, diff)
			if err != nil {
				// Fallback to heuristic
				heuristic := review.NewHeuristicDetector()
				return heuristic.Detect(ctx, diff)
			}

			// Filter by explicit flags
			modes = filterModesByFlags(cmd, modes)
			return modes, reasoning, nil
		},
		// Run review
		func(ctx context.Context, mode review.Mode) (*review.Result, error) {
			return claudeClient.RunReview(ctx, mode, diff)
		},
		// Generate commit message
		func(ctx context.Context) (string, error) {
			msg, err := claudeClient.GenerateCommitMessage(ctx, diff)
			if err != nil {
				return "", err
			}
			return msg.String(), nil
		},
		blockOnIssues,
	)

	if err != nil {
		return err
	}

	// Handle results
	if program.IsBlocked() {
		return fmt.Errorf("commit blocked due to review issues")
	}

	if !program.IsConfirmed() {
		fmt.Println("Commit cancelled.")
		return nil
	}

	if dryRun {
		fmt.Println("Dry run - commit not created.")
		return nil
	}

	// Create the commit
	commitMessage := program.GetCommitMessage()
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

func filterModesByFlags(cmd *cobra.Command, detected []review.Mode) []review.Mode {
	enabled := make(map[review.Mode]bool)
	disabled := make(map[review.Mode]bool)

	// Check enabled flags
	if sec, _ := cmd.Flags().GetBool("security"); sec {
		enabled[review.ModeSecurity] = true
	}
	if perf, _ := cmd.Flags().GetBool("performance"); perf {
		enabled[review.ModePerformance] = true
	}
	if style, _ := cmd.Flags().GetBool("style"); style {
		enabled[review.ModeStyle] = true
	}
	if errs, _ := cmd.Flags().GetBool("errors"); errs {
		enabled[review.ModeErrors] = true
	}
	if test, _ := cmd.Flags().GetBool("testing"); test {
		enabled[review.ModeTesting] = true
	}
	if docs, _ := cmd.Flags().GetBool("docs"); docs {
		enabled[review.ModeDocs] = true
	}

	// Check disabled flags
	if noSec, _ := cmd.Flags().GetBool("no-security"); noSec {
		disabled[review.ModeSecurity] = true
	}
	if noPerf, _ := cmd.Flags().GetBool("no-performance"); noPerf {
		disabled[review.ModePerformance] = true
	}
	if noStyle, _ := cmd.Flags().GetBool("no-style"); noStyle {
		disabled[review.ModeStyle] = true
	}
	if noErrs, _ := cmd.Flags().GetBool("no-errors"); noErrs {
		disabled[review.ModeErrors] = true
	}
	if noTest, _ := cmd.Flags().GetBool("no-testing"); noTest {
		disabled[review.ModeTesting] = true
	}
	if noDocs, _ := cmd.Flags().GetBool("no-docs"); noDocs {
		disabled[review.ModeDocs] = true
	}

	return review.FilterModes(detected, enabled, disabled)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("revi version %s\n", Version)
	},
}
