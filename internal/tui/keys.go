package tui

import "github.com/buker/revi/internal/tui/shared"

// KeyMap re-exports the shared KeyMap for backwards compatibility
type KeyMap = shared.KeyMap

// DefaultKeyMap returns the default keybindings
func DefaultKeyMap() KeyMap {
	return shared.DefaultKeyMap()
}

// Re-export help functions from shared package

// IssuesTableHelp returns help text for the issues table view
func IssuesTableHelp() string {
	return shared.IssuesTableHelp()
}

// IssueDetailHelp returns help text for the issue detail modal
func IssueDetailHelp(hasFix bool) string {
	return shared.IssueDetailHelp(hasFix)
}

// DiffPreviewHelp returns help text for the diff preview modal
func DiffPreviewHelp() string {
	return shared.DiffPreviewHelp()
}

// CommitConfirmHelp returns help text for the commit confirm view
func CommitConfirmHelp() string {
	return shared.CommitConfirmHelp()
}

// ProgressHelp returns help text for the progress view
func ProgressHelp() string {
	return shared.ProgressHelp()
}
