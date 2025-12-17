package shared

import (
	"strings"
	"testing"
)

// =============================================================================
// Tests for IssuesTableHelp()
// =============================================================================

func TestIssuesTableHelp_ContainsCommitOption(t *testing.T) {
	help := IssuesTableHelp()

	if !strings.Contains(help, "[c] commit") {
		t.Error("IssuesTableHelp() should contain '[c] commit'")
	}
}

func TestIssuesTableHelp_ContainsNavigationKeys(t *testing.T) {
	help := IssuesTableHelp()

	expectedKeys := []string{"up", "down", "details", "quit"}
	for _, key := range expectedKeys {
		if !strings.Contains(help, key) {
			t.Errorf("IssuesTableHelp() should contain %q", key)
		}
	}
}

// =============================================================================
// Tests for IssuesTableHelpBlocked()
// =============================================================================

func TestIssuesTableHelpBlocked_DoesNotContainCommit(t *testing.T) {
	help := IssuesTableHelpBlocked()

	if strings.Contains(help, "commit") {
		t.Error("IssuesTableHelpBlocked() should NOT contain 'commit'")
	}
	if strings.Contains(help, "[c]") {
		t.Error("IssuesTableHelpBlocked() should NOT contain '[c]'")
	}
}

func TestIssuesTableHelpBlocked_ContainsNavigationKeys(t *testing.T) {
	help := IssuesTableHelpBlocked()

	expectedKeys := []string{"up", "down", "details", "quit"}
	for _, key := range expectedKeys {
		if !strings.Contains(help, key) {
			t.Errorf("IssuesTableHelpBlocked() should contain %q", key)
		}
	}
}

func TestIssuesTableHelpBlocked_DiffersFromNormalHelp(t *testing.T) {
	normalHelp := IssuesTableHelp()
	blockedHelp := IssuesTableHelpBlocked()

	if normalHelp == blockedHelp {
		t.Error("IssuesTableHelpBlocked() should differ from IssuesTableHelp()")
	}
}

// =============================================================================
// Tests for other help functions
// =============================================================================

func TestIssueDetailHelp_WithFix(t *testing.T) {
	help := IssueDetailHelp(true)

	if !strings.Contains(help, "preview fix") {
		t.Error("IssueDetailHelp(true) should contain 'preview fix'")
	}
	if !strings.Contains(help, "[a]") {
		t.Error("IssueDetailHelp(true) should contain '[a]' key")
	}
}

func TestIssueDetailHelp_WithoutFix(t *testing.T) {
	help := IssueDetailHelp(false)

	if strings.Contains(help, "preview fix") {
		t.Error("IssueDetailHelp(false) should NOT contain 'preview fix'")
	}
	if strings.Contains(help, "[a]") {
		t.Error("IssueDetailHelp(false) should NOT contain '[a]' key")
	}
	if !strings.Contains(help, "close") {
		t.Error("IssueDetailHelp(false) should contain 'close'")
	}
}

func TestProgressHelp_ContainsQuit(t *testing.T) {
	help := ProgressHelp()

	if !strings.Contains(help, "quit") {
		t.Error("ProgressHelp() should contain 'quit'")
	}
}

func TestCommitConfirmHelp_ContainsAllOptions(t *testing.T) {
	help := CommitConfirmHelp()

	expectedKeys := []string{"commit", "edit", "cancel"}
	for _, key := range expectedKeys {
		if !strings.Contains(help, key) {
			t.Errorf("CommitConfirmHelp() should contain %q", key)
		}
	}
}
