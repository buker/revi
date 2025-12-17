package review

import (
	"context"
	"fmt"
	"sync"
)

// ReviewFunc defines the signature for a function that executes a single code review.
// It takes a context for cancellation, the review mode to run, and the code diff to analyze.
// Returns the review result or an error if the review failed.
type ReviewFunc func(ctx context.Context, mode Mode, diff string) (*Result, error)

// StatusCallback is invoked when a review mode's execution status changes.
// Used to update UI or logging as reviews transition between pending, running, and done states.
type StatusCallback func(mode Mode, status Status)

// Runner coordinates the parallel execution of multiple review modes.
// It manages goroutines for concurrent reviews and aggregates results.
type Runner struct {
	reviewFunc     ReviewFunc
	statusCallback StatusCallback
}

// NewRunner creates a new Runner with the given review function and optional status callback.
// The reviewFunc is called for each review mode, and statusCallback (if non-nil) receives
// status updates as reviews progress through their lifecycle.
func NewRunner(reviewFunc ReviewFunc, statusCallback StatusCallback) *Runner {
	return &Runner{
		reviewFunc:     reviewFunc,
		statusCallback: statusCallback,
	}
}

// Run executes all specified review modes in parallel using goroutines.
// It waits for all reviews to complete and returns results in the same order as modes.
// Each review's status is reported via the statusCallback if configured.
func (r *Runner) Run(ctx context.Context, modes []Mode, diff string) []*Result {
	results := make([]*Result, len(modes))
	var wg sync.WaitGroup

	for i, mode := range modes {
		wg.Add(1)
		go func(idx int, m Mode) {
			defer wg.Done()

			// Update status to running
			if r.statusCallback != nil {
				r.statusCallback(m, StatusRunning)
			}

			// Run the review
			result, err := r.reviewFunc(ctx, m, diff)
			if err != nil {
				result = &Result{
					Mode:   m,
					Status: StatusFailed,
					Error:  err.Error(),
				}
			}

			results[idx] = result

			// Update status to done/failed
			if r.statusCallback != nil {
				if result.Status == StatusFailed {
					r.statusCallback(m, StatusFailed)
				} else {
					r.statusCallback(m, StatusDone)
				}
			}
		}(i, mode)
	}

	wg.Wait()
	return results
}

// Summary aggregates statistics from a set of review results.
// It counts total reviews, issues by severity level, and failed reviews.
type Summary struct {
	TotalReviews   int // Total number of reviews executed
	IssuesFound    int // Total number of issues found across all reviews
	HighSeverity   int // Count of high-severity issues
	MediumSeverity int // Count of medium-severity issues
	LowSeverity    int // Count of low-severity issues
	FailedReviews  int // Number of reviews that failed to execute
}

// Summarize creates a Summary by aggregating statistics from the given review results.
// It iterates through all results, counting issues by severity and tracking failures.
func Summarize(results []*Result) Summary {
	var summary Summary
	summary.TotalReviews = len(results)

	for _, r := range results {
		if r == nil {
			continue
		}

		if r.Status == StatusFailed {
			summary.FailedReviews++
			continue
		}

		for _, issue := range r.Issues {
			summary.IssuesFound++
			switch issue.Severity {
			case "high":
				summary.HighSeverity++
			case "medium":
				summary.MediumSeverity++
			case "low":
				summary.LowSeverity++
			}
		}
	}

	return summary
}

// ShouldBlock determines if a commit should be blocked based on review results.
// Returns true if blockOnIssues is enabled and any result contains high-severity issues.
// This allows CI/CD pipelines to prevent commits that introduce critical problems.
func ShouldBlock(results []*Result, blockOnIssues bool) bool {
	if !blockOnIssues {
		return false
	}

	for _, r := range results {
		if r != nil && r.HasHighSeverityIssues() {
			return true
		}
	}

	return false
}

// GetBlockReason returns a human-readable reason explaining why a commit was blocked.
// It counts high-severity issues across all results and returns an appropriate message.
// Returns an empty string if there are no high-severity issues.
func GetBlockReason(results []*Result) string {
	var highIssues int
	for _, r := range results {
		if r != nil {
			for _, issue := range r.Issues {
				if issue.Severity == "high" {
					highIssues++
				}
			}
		}
	}

	if highIssues > 0 {
		if highIssues == 1 {
			return "1 high-severity issue found"
		}
		return fmt.Sprintf("%d high-severity issues found", highIssues)
	}

	return ""
}
