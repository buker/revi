package views

import (
	"strings"
	"testing"

	"github.com/buker/revi/internal/review"
)

// =============================================================================
// Tests for IssuesTableView.SetBlocked()
// =============================================================================

func TestIssuesTableView_SetBlocked_SetsState(t *testing.T) {
	view := NewIssuesTableView()

	view.SetBlocked(true, "2 high-severity issues found")

	if !view.blocked {
		t.Error("SetBlocked(true, ...) did not set blocked to true")
	}
	if view.blockReason != "2 high-severity issues found" {
		t.Errorf("SetBlocked() blockReason = %q, want %q", view.blockReason, "2 high-severity issues found")
	}
}

func TestIssuesTableView_SetBlocked_False(t *testing.T) {
	view := NewIssuesTableView()
	// First set to blocked
	view.SetBlocked(true, "some reason")
	// Then unblock
	view.SetBlocked(false, "")

	if view.blocked {
		t.Error("SetBlocked(false, ...) did not set blocked to false")
	}
}

// =============================================================================
// Tests for IssuesTableView.View() with blocked state
// =============================================================================

func TestIssuesTableView_View_ShowsBlockedMessage(t *testing.T) {
	view := NewIssuesTableView()
	view.SetSize(100, 50)
	view.SetBlocked(true, "1 high-severity issue found")

	output := view.View()

	if !strings.Contains(output, "BLOCKED") {
		t.Error("View() output should contain 'BLOCKED' when blocked")
	}
	if !strings.Contains(output, "1 high-severity issue found") {
		t.Error("View() output should contain the block reason")
	}
	if !strings.Contains(output, "--no-block") {
		t.Error("View() output should mention --no-block override option")
	}
}

func TestIssuesTableView_View_HidesCommitMessageWhenBlocked(t *testing.T) {
	view := NewIssuesTableView()
	view.SetSize(100, 50)
	view.SetCommitMessage("feat: add new feature")
	view.SetBlocked(true, "blocked reason")

	output := view.View()

	if strings.Contains(output, "feat: add new feature") {
		t.Error("View() should not show commit message when blocked")
	}
}

func TestIssuesTableView_View_ShowsCommitMessageWhenNotBlocked(t *testing.T) {
	view := NewIssuesTableView()
	view.SetSize(100, 50)
	view.SetCommitMessage("feat: add new feature")
	view.SetBlocked(false, "")

	output := view.View()

	if !strings.Contains(output, "feat: add new feature") {
		t.Error("View() should show commit message when not blocked")
	}
}

func TestIssuesTableView_View_ShowsBlockedHelpText(t *testing.T) {
	view := NewIssuesTableView()
	view.SetSize(100, 50)
	view.SetBlocked(true, "blocked")

	output := view.View()

	// Should not show [c] commit option when blocked
	if strings.Contains(output, "[c] commit") {
		t.Error("View() should not show [c] commit in help when blocked")
	}
	// Should show quit option
	if !strings.Contains(output, "[q] quit") {
		t.Error("View() should show [q] quit in help when blocked")
	}
}

func TestIssuesTableView_View_ShowsNormalHelpTextWhenNotBlocked(t *testing.T) {
	view := NewIssuesTableView()
	view.SetSize(100, 50)
	view.SetBlocked(false, "")

	output := view.View()

	// Should show [c] commit option when not blocked
	if !strings.Contains(output, "[c] commit") {
		t.Error("View() should show [c] commit in help when not blocked")
	}
}

// =============================================================================
// Tests for IssuesTableView with issues and blocked state
// =============================================================================

func TestIssuesTableView_View_ShowsIssuesWhenBlocked(t *testing.T) {
	view := NewIssuesTableView()
	view.SetSize(100, 50)

	results := []*review.Result{
		{
			Mode:   review.ModeSecurity,
			Status: review.StatusIssues,
			Issues: []review.Issue{
				{
					Severity:    "high",
					Description: "SQL injection vulnerability",
					Location:    "db/queries.go:42",
				},
			},
		},
	}
	view.SetIssues(results)
	view.SetBlocked(true, "1 high-severity issue found")

	output := view.View()

	// Should still show issues when blocked
	if !strings.Contains(output, "SQL injection") {
		t.Error("View() should display issues even when blocked")
	}
	// Should also show blocked message
	if !strings.Contains(output, "BLOCKED") {
		t.Error("View() should show blocked message along with issues")
	}
}
