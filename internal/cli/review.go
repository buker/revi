package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/buker/revi/internal/claude"
	"github.com/buker/revi/internal/config"
	"github.com/buker/revi/internal/git"
	"github.com/buker/revi/internal/review"
	"github.com/spf13/cobra"
)

var reviewCmd = &cobra.Command{
	Use:   "review",
	Short: "Run code review only (no commit)",
	Long: `Run AI-powered code review on staged changes without creating a commit.

This command analyzes your staged git changes using specialized review agents
(security, performance, style, error handling, testing, documentation).`,
	RunE: runReview,
}

func runReview(cmd *cobra.Command, args []string) error {
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

	fmt.Println("revi - AI Code Review")
	fmt.Println(strings.Repeat("-", 40))

	// Detect review modes
	fmt.Println("\nAnalyzing diff...")

	var modes []review.Mode
	var reasoning string

	allModes, _ := cmd.Flags().GetBool("all")
	if allModes {
		modes = review.AllModes()
		reasoning = "All modes enabled"
	} else {
		detector := review.NewClaudeDetector(claudeClient.DetectModes)
		modes, reasoning, err = detector.Detect(ctx, diff)
		if err != nil {
			// Fallback to heuristic
			heuristic := review.NewHeuristicDetector()
			modes, reasoning, _ = heuristic.Detect(ctx, diff)
		}
		modes = filterModesByFlags(cmd, modes)
	}

	fmt.Printf("Detected: %s\n", reasoning)
	fmt.Printf("Running %d review(s)...\n\n", len(modes))

	// Run reviews
	runner := review.NewRunner(
		func(ctx context.Context, mode review.Mode, diff string) (*review.Result, error) {
			return claudeClient.RunReview(ctx, mode, diff)
		},
		func(mode review.Mode, status review.Status) {
			info := review.GetModeInfo(mode)
			fmt.Printf("%s: %s\n", info.Name, status)
		},
	)

	results := runner.Run(ctx, modes, diff)

	// Print results
	fmt.Println("\n" + strings.Repeat("=", 40))
	fmt.Println("REVIEW RESULTS")
	fmt.Println(strings.Repeat("=", 40))

	for _, r := range results {
		if r == nil {
			continue
		}
		printReviewResult(r)
	}

	// Print summary
	summary := review.Summarize(results)
	fmt.Println("\n" + strings.Repeat("-", 40))
	fmt.Println("SUMMARY")
	fmt.Println(strings.Repeat("-", 40))
	fmt.Printf("Total reviews:    %d\n", summary.TotalReviews)
	fmt.Printf("Issues found:     %d\n", summary.IssuesFound)
	if summary.IssuesFound > 0 {
		fmt.Printf("  High severity:  %d\n", summary.HighSeverity)
		fmt.Printf("  Medium:         %d\n", summary.MediumSeverity)
		fmt.Printf("  Low:            %d\n", summary.LowSeverity)
	}
	if summary.FailedReviews > 0 {
		fmt.Printf("Failed reviews:   %d\n", summary.FailedReviews)
	}

	// Check if should block
	blockOnIssues := config.IsBlockEnabled(cmd)
	if review.ShouldBlock(results, blockOnIssues) {
		return fmt.Errorf("high-severity issues found")
	}

	return nil
}

func printReviewResult(r *review.Result) {
	info := review.GetModeInfo(r.Mode)
	fmt.Printf("\n=== %s Review ===\n", info.Name)

	if r.Status == review.StatusFailed {
		fmt.Printf("Status: FAILED (%s)\n", r.Error)
		return
	}

	if len(r.Issues) == 0 {
		fmt.Println("Status: No issues found")
	} else {
		fmt.Printf("Status: %d issue(s) found\n", len(r.Issues))
	}

	if r.Summary != "" {
		fmt.Printf("\nSummary:\n  %s\n", r.Summary)
	}

	if len(r.Issues) > 0 {
		fmt.Println("\nIssues:")
		for _, issue := range r.Issues {
			loc := ""
			if issue.Location != "" {
				loc = fmt.Sprintf(" (%s)", issue.Location)
			}
			fmt.Printf("  - [%s] %s%s\n",
				strings.ToUpper(issue.Severity), issue.Description, loc)
		}
	}

	if len(r.Suggestions) > 0 {
		fmt.Println("\nSuggestions:")
		for _, s := range r.Suggestions {
			fmt.Printf("  - %s\n", s)
		}
	}
}
