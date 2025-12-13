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
	"github.com/buker/revi/internal/review"
	"github.com/spf13/cobra"
)

var commitCmd = &cobra.Command{
	Use:   "commit",
	Short: "Generate commit message and commit",
	Long: `Generate an AI-powered commit message for staged changes and create the commit.

By default, this also runs code review unless --no-review is specified.`,
	RunE: runCommit,
}

func init() {
	// Commit-specific flags can be added here if needed
}

func runCommit(cmd *cobra.Command, args []string) error {
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

	fmt.Println("revi - AI Commit")
	fmt.Println(strings.Repeat("-", 40))

	// Run review if enabled
	reviewEnabled := config.IsReviewEnabled(cmd)
	blockOnIssues := config.IsBlockEnabled(cmd)
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if reviewEnabled {
		fmt.Println("\nRunning code review...")

		// Detect and run reviews (simplified - single mode for speed)
		heuristic := review.NewHeuristicDetector()
		modes, reasoning, err := heuristic.Detect(ctx, diff)
		if err != nil {
			// Fallback to all modes if detection fails
			modes = review.AllModes()
			reasoning = "Detection failed, running all modes"
		}
		modes = filterModesByFlags(cmd, modes)

		if len(modes) > 0 {
			fmt.Printf("Detected: %s\n", reasoning)
			runner := review.NewRunner(
				func(ctx context.Context, mode review.Mode, diff string) (*review.Result, error) {
					return claudeClient.RunReview(ctx, mode, diff)
				},
				nil,
			)

			results := runner.Run(ctx, modes, diff)

			// Check for blocking issues
			if review.ShouldBlock(results, blockOnIssues) {
				summary := review.Summarize(results)
				fmt.Printf("\nReview found %d high-severity issue(s).\n", summary.HighSeverity)
				fmt.Println("Use --no-block to override.")
				return fmt.Errorf("commit blocked due to review issues")
			}

			summary := review.Summarize(results)
			if summary.IssuesFound > 0 {
				fmt.Printf("Review found %d issue(s) (none blocking).\n", summary.IssuesFound)
			} else {
				fmt.Println("Review passed - no issues found.")
			}
		}
	}

	// Generate commit message
	fmt.Println("\nGenerating commit message...")
	msg, err := claudeClient.GenerateCommitMessage(ctx, diff)
	if err != nil {
		return fmt.Errorf("failed to generate commit message: %w", err)
	}

	commitMessage := msg.String()

	// Display commit message
	fmt.Println("\n" + strings.Repeat("-", 40))
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

	fmt.Printf("\nCreated commit: %s\n", shortHash(hash))
	return nil
}
