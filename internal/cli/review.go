package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/buker/revi/internal/claude"
	"github.com/buker/revi/internal/config"
	"github.com/buker/revi/internal/fix"
	"github.com/buker/revi/internal/git"
	"github.com/buker/revi/internal/review"
	"github.com/buker/revi/internal/tui"
	"github.com/spf13/cobra"
)

func init() {
	// Fix flag
	reviewCmd.Flags().BoolP("fix", "f", false, "Interactively fix detected issues")

	// Block flags
	reviewCmd.Flags().BoolP("block", "b", true, "Exit with error if high-severity issues found")
	reviewCmd.Flags().BoolP("no-block", "B", false, "Don't exit with error on issues")

	// TUI flag
	reviewCmd.Flags().Bool("no-tui", false, "Disable TUI (use plain text output)")

	// Review mode flags
	reviewCmd.Flags().Bool("security", false, "Enable security review")
	reviewCmd.Flags().Bool("no-security", false, "Disable security review")
	reviewCmd.Flags().Bool("performance", false, "Enable performance review")
	reviewCmd.Flags().Bool("no-performance", false, "Disable performance review")
	reviewCmd.Flags().Bool("style", false, "Enable style review")
	reviewCmd.Flags().Bool("no-style", false, "Disable style review")
	reviewCmd.Flags().Bool("errors", false, "Enable error handling review")
	reviewCmd.Flags().Bool("no-errors", false, "Disable error handling review")
	reviewCmd.Flags().Bool("testing", false, "Enable testing review")
	reviewCmd.Flags().Bool("no-testing", false, "Disable testing review")
	reviewCmd.Flags().Bool("docs", false, "Enable documentation review")
	reviewCmd.Flags().Bool("no-docs", false, "Disable documentation review")
	reviewCmd.Flags().BoolP("all", "a", false, "Run all review modes")
}

var reviewCmd = &cobra.Command{
	Use:   "review",
	Short: "Run code review only (no commit)",
	Long: `Run AI-powered code review on staged changes without creating a commit.

This command analyzes your staged git changes using specialized review agents
(security, performance, style, error handling, testing, documentation).

Use --fix to interactively apply suggested fixes after the review.`,
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

	noTUI, err := cmd.Flags().GetBool("no-tui")
	if err != nil {
		return fmt.Errorf("failed to get no-tui flag: %w", err)
	}
	if noTUI {
		return runReviewTextMode(cmd, ctx, claudeClient, repo, diff)
	}

	return runReviewTUI(cmd, ctx, claudeClient, repo, diff)
}

// runReviewTUI runs the review workflow with the interactive TUI
func runReviewTUI(cmd *cobra.Command, ctx context.Context, claudeClient *claude.Client, repo *git.Repository, diff string) error {
	allModes, _ := cmd.Flags().GetBool("all")
	blockOnIssues := isBlockEnabled(cmd)

	// Get repository root for fix applier
	repoRoot, err := repo.Root()
	if err != nil {
		return fmt.Errorf("failed to get repository root: %w", err)
	}
	// Define mode detection function
	detectFunc := func(ctx context.Context) ([]review.Mode, string, error) {
		if allModes {
			return review.AllModes(), "All modes enabled", nil
		}

		detector := review.NewClaudeDetector(claudeClient.DetectModes)
		modes, reasoning, err := detector.Detect(ctx, diff)
		if err != nil {
			// Fallback to heuristic
			heuristic := review.NewHeuristicDetector()
			modes, reasoning, err = heuristic.Detect(ctx, diff)
			if err != nil {
				return nil, "", fmt.Errorf("failed to detect review modes: %w", err)
			}
		}
		modes = filterModesByFlags(cmd, modes)
		return modes, reasoning, nil
	}
		detector := review.NewClaudeDetector(claudeClient.DetectModes)
		modes, reasoning, err := detector.Detect(ctx, diff)
		if err != nil {
			// Fallback to heuristic
			heuristic := review.NewHeuristicDetector()
			modes, reasoning, _ = heuristic.Detect(ctx, diff)
		}
		modes = filterModesByFlags(cmd, modes)
		return modes, reasoning, nil
	}

	// Define review function
	reviewFunc := func(ctx context.Context, mode review.Mode) (*review.Result, error) {
		return claudeClient.RunReview(ctx, mode, diff)
	}

	// Run the TUI workflow
	if err := program.RunReviewOnly(ctx, detectFunc, reviewFunc, blockOnIssues); err != nil {
		return err
	}

	// Check final state
	if program.IsBlocked() {
		return fmt.Errorf("high-severity issues found")
	}

	return nil
}

// runReviewTextMode runs the review workflow with plain text output (original behavior)
func runReviewTextMode(cmd *cobra.Command, ctx context.Context, claudeClient *claude.Client, repo *git.Repository, diff string) error {
	fmt.Println("revi - AI Code Review")
	fmt.Println(strings.Repeat("-", 40))

	// Detect review modes
	fmt.Println("\nAnalyzing diff...")

	var modes []review.Mode
	var reasoning string
	var err error

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

	// Run interactive fix phase if requested
	fixEnabled, _ := cmd.Flags().GetBool("fix")
	if fixEnabled && summary.IssuesFound > 0 {
		// Collect all issues from results
		var allIssues []review.Issue
		for _, r := range results {
			if r != nil && len(r.Issues) > 0 {
				allIssues = append(allIssues, r.Issues...)
			}
		}

		if len(allIssues) > 0 {
			// Get repository root for the applier
			repoRoot, err := repo.Root()
			if err != nil {
				return fmt.Errorf("failed to get repository root: %w", err)
			}

			applier := fix.NewApplier(repoRoot)
			fixer := fix.NewInteractiveFixer(os.Stdin, os.Stdout, applier.Apply)
			fixer.Run(allIssues)
		}
	}

	// Check if should block
	blockOnIssues := isBlockEnabled(cmd)
	if review.ShouldBlock(results, blockOnIssues) {
		return fmt.Errorf("high-severity issues found")
	}

	return nil
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

func isBlockEnabled(cmd *cobra.Command) bool {
	noBlock, _ := cmd.Flags().GetBool("no-block")
	if noBlock {
		return false
	}
	block, _ := cmd.Flags().GetBool("block")
	return block
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
