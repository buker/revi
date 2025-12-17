package fix

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/buker/revi/internal/review"
)

// Stats tracks the results of the interactive fix session.
type Stats struct {
	// Applied is the count of fixes that were successfully applied to files
	Applied int
	// Skipped is the count of fixes that were skipped by user choice or application failure
	Skipped int
	// Unfixable is the count of issues that cannot be automatically fixed
	Unfixable int
}

// ApplyFunc is a function that applies a fix to a file.
// It should return an error if the fix cannot be applied.
type ApplyFunc func(*review.Fix) error

// InteractiveFixer drives the interactive fix approval loop.
// It presents each issue to the user, shows the suggested fix if available,
// and prompts for approval before applying changes. Users can approve (y),
// skip (n), or skip all remaining issues (s).
type InteractiveFixer struct {
	reader  *bufio.Reader
	writer  io.Writer
	applyFn ApplyFunc
}

// NewInteractiveFixer creates a new InteractiveFixer.
func NewInteractiveFixer(reader io.Reader, writer io.Writer, applyFn ApplyFunc) *InteractiveFixer {
	return &InteractiveFixer{
		reader:  bufio.NewReader(reader),
		writer:  writer,
		applyFn: applyFn,
	}
}

// Run processes all issues and prompts for user approval on each fix.
func (f *InteractiveFixer) Run(issues []review.Issue) Stats {
	var stats Stats

	if len(issues) == 0 {
		return stats
	}

	// Write errors are intentionally ignored - if output fails, continue processing
	_, _ = fmt.Fprintln(f.writer, strings.Repeat("-", 40))
	_, _ = fmt.Fprintln(f.writer, "FIX ISSUES")
	_, _ = fmt.Fprintln(f.writer, strings.Repeat("-", 40))

	skipAll := false

	for i, issue := range issues {
		if skipAll {
			stats.Skipped++
			continue
		}

		// Write errors are intentionally ignored - if output fails, continue processing
		_, _ = fmt.Fprintf(f.writer, "\nIssue %d/%d: [%s] %s",
			i+1, len(issues), strings.ToUpper(issue.Severity), issue.Description)
		if issue.Location != "" {
			_, _ = fmt.Fprintf(f.writer, " (%s)", issue.Location)
		}
		_, _ = fmt.Fprintln(f.writer)

		// Check if fix is available
		if issue.Fix == nil || !issue.Fix.Available {
			f.handleUnfixable(issue.Fix)
			stats.Unfixable++
			continue
		}

		// Show the fix
		f.showFix(issue.Fix)

		// Prompt for approval
		response := f.prompt()

		switch response {
		case "y", "yes", "":
			if err := f.applyFn(issue.Fix); err != nil {
				// Write errors are intentionally ignored - if output fails, continue processing
				_, _ = fmt.Fprintf(f.writer, "  ✗ Failed: %v\n", err)
				stats.Skipped++
			} else {
				_, _ = fmt.Fprintln(f.writer, "  ✓ Applied")
				stats.Applied++
			}
		case "n", "no":
			_, _ = fmt.Fprintln(f.writer, "  - Skipped")
			stats.Skipped++
		case "s", "skip":
			_, _ = fmt.Fprintln(f.writer, "  - Skipping remaining issues")
			skipAll = true
			stats.Skipped++
		default:
			_, _ = fmt.Fprintln(f.writer, "  - Skipped (invalid input)")
			stats.Skipped++
		}
	}

	// Print summary - write errors are intentionally ignored
	_, _ = fmt.Fprintln(f.writer)
	_, _ = fmt.Fprintf(f.writer, "Applied %d fix(es), skipped %d", stats.Applied, stats.Skipped)
	if stats.Unfixable > 0 {
		_, _ = fmt.Fprintf(f.writer, ", %d unfixable", stats.Unfixable)
	}
	_, _ = fmt.Fprintln(f.writer)

	return stats
}

func (f *InteractiveFixer) showFix(fix *review.Fix) {
	// Show the suggested code change
	// Write errors are intentionally ignored - if output fails, continue processing
	if fix.Code != "" {
		_, _ = fmt.Fprintf(f.writer, "  After:  %s\n", strings.TrimSpace(fix.Code))
	}
	if fix.Explanation != "" {
		_, _ = fmt.Fprintf(f.writer, "  Why:    %s\n", fix.Explanation)
	}
}

func (f *InteractiveFixer) handleUnfixable(fix *review.Fix) {
	// Write errors are intentionally ignored - if output fails, continue processing
	_, _ = fmt.Fprintln(f.writer, "  ⚠ Cannot auto-fix")
	if fix != nil {
		if fix.Reason != "" {
			_, _ = fmt.Fprintf(f.writer, "  Reason: %s\n", fix.Reason)
		}
		if len(fix.Alternatives) > 0 {
			_, _ = fmt.Fprintln(f.writer, "  Alternatives:")
			for _, alt := range fix.Alternatives {
				_, _ = fmt.Fprintf(f.writer, "    - %s\n", alt)
			}
		}
	}
	_, _ = fmt.Fprint(f.writer, "\nPress Enter to continue...")
	// Read error is intentionally ignored - if read fails, we simply continue
	// without waiting for user input, which is acceptable for this use case
	_, _ = f.reader.ReadString('\n')
}

func (f *InteractiveFixer) prompt() string {
	// Write error is intentionally ignored - if output fails, continue to read input
	_, _ = fmt.Fprint(f.writer, "\nApply this fix? [y]es / [n]o / [s]kip remaining: ")
	input, err := f.reader.ReadString('\n')
	if err != nil {
		return "n" // Treat read errors as skip to avoid unintended changes
	}
	return strings.ToLower(strings.TrimSpace(input))
}
