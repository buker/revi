package tui

import (
	"testing"

	"github.com/buker/revi/internal/review"
	tea "github.com/charmbracelet/bubbletea"
)

// =============================================================================
// Tests for blocked state transitions
// =============================================================================

func TestModel_BlockedState_TransitionsToIssuesTable(t *testing.T) {
	model := NewModel()

	// Send MsgAllReviewsComplete with blocked=true
	msg := MsgAllReviewsComplete{
		Results: []*review.Result{
			{
				Mode:   review.ModeSecurity,
				Status: review.StatusIssues,
				Issues: []review.Issue{
					{Severity: "high", Description: "critical issue"},
				},
			},
		},
		Blocked: true,
		Reason:  "1 high-severity issue found",
	}

	newModel, _ := model.Update(msg)
	m := newModel.(*Model)

	// Should transition to StateIssuesTable, not StateBlocking
	if m.state != StateIssuesTable {
		t.Errorf("state = %v, want StateIssuesTable when blocked", m.state)
	}

	// Should set blocked flag
	if !m.IsBlocked() {
		t.Error("IsBlocked() = false, want true")
	}
}

func TestModel_NotBlocked_TransitionsToIssuesTable(t *testing.T) {
	model := NewModel()

	// Send MsgAllReviewsComplete with blocked=false
	msg := MsgAllReviewsComplete{
		Results: []*review.Result{
			{
				Mode:   review.ModeSecurity,
				Status: review.StatusNoIssues,
				Issues: []review.Issue{},
			},
		},
		Blocked: false,
		Reason:  "",
	}

	newModel, _ := model.Update(msg)
	m := newModel.(*Model)

	if m.state != StateIssuesTable {
		t.Errorf("state = %v, want StateIssuesTable when not blocked", m.state)
	}

	if m.IsBlocked() {
		t.Error("IsBlocked() = true, want false")
	}
}

// =============================================================================
// Tests for commit key behavior when blocked
// =============================================================================

func TestModel_CommitKey_DisabledWhenBlocked(t *testing.T) {
	model := NewModel()
	model.state = StateIssuesTable
	model.blocked = true

	// Simulate pressing 'c' key
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}}
	newModel, _ := model.Update(keyMsg)
	m := newModel.(*Model)

	// Should stay in IssuesTable state, not transition to CommitConfirm
	if m.state != StateIssuesTable {
		t.Errorf("state = %v, want StateIssuesTable (commit should be blocked)", m.state)
	}
}

func TestModel_CommitKey_EnabledWhenNotBlocked(t *testing.T) {
	model := NewModel()
	model.state = StateIssuesTable
	model.blocked = false

	// Need to set some issues for the commit view
	model.results = []*review.Result{
		{
			Mode:   review.ModeSecurity,
			Status: review.StatusNoIssues,
			Issues: []review.Issue{},
		},
	}

	// Simulate pressing 'c' key
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}}
	newModel, _ := model.Update(keyMsg)
	m := newModel.(*Model)

	// Should transition to CommitConfirm state
	if m.state != StateCommitConfirm {
		t.Errorf("state = %v, want StateCommitConfirm when not blocked", m.state)
	}
}

// =============================================================================
// Tests for navigation keys when blocked
// =============================================================================

func TestModel_NavigationKeys_WorkWhenBlocked(t *testing.T) {
	model := NewModel()
	model.state = StateIssuesTable
	model.blocked = true

	// Set some issues to navigate
	results := []*review.Result{
		{
			Mode:   review.ModeSecurity,
			Status: review.StatusIssues,
			Issues: []review.Issue{
				{Severity: "high", Description: "issue 1"},
				{Severity: "medium", Description: "issue 2"},
			},
		},
	}
	model.issuesView.SetIssues(results)

	// Simulate pressing down key
	keyMsg := tea.KeyMsg{Type: tea.KeyDown}
	newModel, _ := model.Update(keyMsg)
	m := newModel.(*Model)

	// Should still be in IssuesTable and cursor should have moved
	if m.state != StateIssuesTable {
		t.Errorf("state = %v, want StateIssuesTable", m.state)
	}
	if m.issuesView.Cursor() != 1 {
		t.Errorf("Cursor() = %d, want 1 after pressing down", m.issuesView.Cursor())
	}
}

func TestModel_EnterKey_WorksWhenBlocked(t *testing.T) {
	model := NewModel()
	model.state = StateIssuesTable
	model.blocked = true

	// Set some issues
	results := []*review.Result{
		{
			Mode:   review.ModeSecurity,
			Status: review.StatusIssues,
			Issues: []review.Issue{
				{Severity: "high", Description: "issue 1"},
			},
		},
	}
	model.issuesView.SetIssues(results)

	// Simulate pressing Enter key
	keyMsg := tea.KeyMsg{Type: tea.KeyEnter}
	newModel, _ := model.Update(keyMsg)
	m := newModel.(*Model)

	// Should transition to issue detail view even when blocked
	if m.state != StateIssueDetail {
		t.Errorf("state = %v, want StateIssueDetail (should work even when blocked)", m.state)
	}
}

// =============================================================================
// Tests for quit behavior
// =============================================================================

func TestModel_QuitKey_WorksWhenBlocked(t *testing.T) {
	model := NewModel()
	model.state = StateIssuesTable
	model.blocked = true

	// Simulate pressing 'q' key
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	_, cmd := model.Update(keyMsg)

	// Should return quit command
	if cmd == nil {
		t.Error("pressing 'q' should return a command")
	}
}

// =============================================================================
// Tests for Model state access methods
// =============================================================================

func TestModel_IsBlocked_ThreadSafe(t *testing.T) {
	model := NewModel()

	// Set blocked via Update
	msg := MsgAllReviewsComplete{
		Results: []*review.Result{},
		Blocked: true,
		Reason:  "test",
	}
	newModel, _ := model.Update(msg)
	m := newModel.(*Model)

	// IsBlocked should be safe to call
	if !m.IsBlocked() {
		t.Error("IsBlocked() = false after setting blocked=true")
	}
}
