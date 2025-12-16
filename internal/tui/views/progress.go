// Package views provides individual view components for the TUI.
package views

import (
	"fmt"
	"strings"
	"time"

	"github.com/buker/revi/internal/review"
	"github.com/buker/revi/internal/tui/shared"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ReviewStatus tracks the status and timing of a single review
type ReviewStatus struct {
	Mode      review.Mode
	Status    review.Status
	StartTime time.Time
	EndTime   time.Time
	Issues    int
}

// Duration returns the elapsed duration for this review
func (rs *ReviewStatus) Duration() time.Duration {
	if rs.Status == review.StatusPending {
		return 0
	}
	if rs.Status == review.StatusRunning {
		return time.Since(rs.StartTime)
	}
	return rs.EndTime.Sub(rs.StartTime)
}

// ProgressView displays the review progress table
type ProgressView struct {
	width    int
	height   int
	spinner  spinner.Model
	reviews  map[review.Mode]*ReviewStatus
	modes    []review.Mode
	complete int
	total    int
}

// NewProgressView creates a new progress view
func NewProgressView() *ProgressView {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(shared.ColorMedium)

	return &ProgressView{
		spinner: s,
		reviews: make(map[review.Mode]*ReviewStatus),
	}
}

// SetModes initializes the review modes to track
func (v *ProgressView) SetModes(modes []review.Mode) {
	v.modes = modes
	v.total = len(modes)
	v.reviews = make(map[review.Mode]*ReviewStatus)
	for _, mode := range modes {
		v.reviews[mode] = &ReviewStatus{
			Mode:   mode,
			Status: review.StatusPending,
		}
	}
}

// SetReviewStarted marks a review as started
func (v *ProgressView) SetReviewStarted(mode review.Mode) {
	if rs, ok := v.reviews[mode]; ok {
		rs.Status = review.StatusRunning
		rs.StartTime = time.Now()
	}
}

// SetReviewComplete marks a review as complete
func (v *ProgressView) SetReviewComplete(mode review.Mode, status review.Status, issues int) {
	if rs, ok := v.reviews[mode]; ok {
		rs.Status = status
		rs.EndTime = time.Now()
		rs.Issues = issues
		v.complete++
	}
}

// IsComplete returns true if all reviews are done
func (v *ProgressView) IsComplete() bool {
	return v.complete >= v.total
}

// SetSize updates the view dimensions
func (v *ProgressView) SetSize(width, height int) {
	v.width = width
	v.height = height
}

// Init initializes the view
func (v *ProgressView) Init() tea.Cmd {
	return v.spinner.Tick
}

// Update handles messages
func (v *ProgressView) Update(msg tea.Msg) (*ProgressView, tea.Cmd) {
	var cmd tea.Cmd
	v.spinner, cmd = v.spinner.Update(msg)
	return v, cmd
}

// View renders the progress table
func (v *ProgressView) View() string {
	var b strings.Builder

	// Header
	b.WriteString(shared.TitleStyle.Render("revi - AI Code Review"))
	b.WriteString("\n")
	b.WriteString(shared.RenderDivider(54))
	b.WriteString("\n")

	// Table header
	header := fmt.Sprintf(" %-14s │ %-11s │ %-8s │ %s", "MODE", "STATUS", "DURATION", "ISSUES")
	b.WriteString(shared.TableHeaderStyle.Render(header))
	b.WriteString("\n")
	b.WriteString(shared.RenderDivider(54))
	b.WriteString("\n")

	// Table rows
	for _, mode := range v.modes {
		rs := v.reviews[mode]
		if rs == nil {
			continue
		}

		info := review.GetModeInfo(mode)
		modeName := truncate(info.Name, 14)

		// Status with indicator
		var statusStr string
		var statusStyle lipgloss.Style
		switch rs.Status {
		case review.StatusPending:
			statusStr = shared.StatusIndicatorPending + " Pending"
			statusStyle = shared.StatusPendingStyle
		case review.StatusRunning:
			statusStr = v.spinner.View() + " Running"
			statusStyle = shared.StatusRunningStyle
		case review.StatusDone, review.StatusNoIssues, review.StatusIssues:
			statusStr = shared.StatusIndicatorDone + " Done"
			statusStyle = shared.StatusDoneStyle
		case review.StatusFailed:
			statusStr = shared.StatusIndicatorFailed + " Failed"
			statusStyle = shared.StatusFailedStyle
		default:
			statusStr = string(rs.Status)
			statusStyle = shared.StatusPendingStyle
		}

		// Duration
		var durationStr string
		if rs.Status == review.StatusPending {
			durationStr = "-"
		} else {
			d := rs.Duration()
			durationStr = fmt.Sprintf("%.1fs", d.Seconds())
		}

		// Issues count
		var issuesStr string
		switch rs.Status {
		case review.StatusPending, review.StatusRunning, review.StatusFailed:
			issuesStr = "-"
		default:
			issuesStr = fmt.Sprintf("%d", rs.Issues)
		}

		row := fmt.Sprintf(" %-14s │ %-11s │ %-8s │ %s",
			modeName,
			statusStyle.Render(padRight(statusStr, 11)),
			durationStr,
			issuesStr,
		)
		b.WriteString(row)
		b.WriteString("\n")
	}

	// Footer
	b.WriteString(shared.RenderDivider(54))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf(" Progress: %d/%d complete\n", v.complete, v.total))
	b.WriteString("\n")
	b.WriteString(shared.HelpKeyStyle.Render(shared.ProgressHelp()))

	return b.String()
}

// truncate truncates a string to max length
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}

// padRight pads a string to the given width
func padRight(s string, width int) string {
	// Account for ANSI codes by using visible length
	visible := visibleLen(s)
	if visible >= width {
		return s
	}
	return s + strings.Repeat(" ", width-visible)
}

// visibleLen returns the visible length of a string (excluding ANSI codes)
func visibleLen(s string) int {
	// Simple approximation - count non-escape characters
	// This is a basic implementation; for full ANSI handling, use a library
	count := 0
	inEscape := false
	for _, r := range s {
		if r == '\x1b' {
			inEscape = true
			continue
		}
		if inEscape {
			if r == 'm' {
				inEscape = false
			}
			continue
		}
		count++
	}
	return count
}
