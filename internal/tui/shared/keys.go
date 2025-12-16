package shared

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines all keybindings for the TUI
type KeyMap struct {
	Up           key.Binding
	Down         key.Binding
	Enter        key.Binding
	Escape       key.Binding
	Quit         key.Binding
	Commit       key.Binding
	Apply        key.Binding
	Confirm      key.Binding
	Cancel       key.Binding
	Edit         key.Binding
	ScrollUp     key.Binding
	ScrollDown   key.Binding
	PageUp       key.Binding
	PageDown     key.Binding
	HalfPageUp   key.Binding
	HalfPageDown key.Binding
	Home         key.Binding
	End          key.Binding
}

// DefaultKeyMap returns the default keybindings
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("Enter", "details"),
		),
		Escape: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("Esc", "close"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Commit: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "commit"),
		),
		Apply: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "preview fix"),
		),
		Confirm: key.NewBinding(
			key.WithKeys("y"),
			key.WithHelp("y", "confirm"),
		),
		Cancel: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "cancel"),
		),
		Edit: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "edit"),
		),
		ScrollUp: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "scroll up"),
		),
		ScrollDown: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "scroll down"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("pgup", "ctrl+b"),
			key.WithHelp("PgUp", "page up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("pgdown", "ctrl+f"),
			key.WithHelp("PgDn", "page down"),
		),
		HalfPageUp: key.NewBinding(
			key.WithKeys("ctrl+u"),
			key.WithHelp("^u", "half page up"),
		),
		HalfPageDown: key.NewBinding(
			key.WithKeys("ctrl+d"),
			key.WithHelp("^d", "half page down"),
		),
		Home: key.NewBinding(
			key.WithKeys("home", "g"),
			key.WithHelp("Home/g", "top"),
		),
		End: key.NewBinding(
			key.WithKeys("end", "G"),
			key.WithHelp("End/G", "bottom"),
		),
	}
}

// IssuesTableHelp returns help text for the issues table view
func IssuesTableHelp() string {
	return " [↑/k] up  [↓/j] down  [Enter] details  [c] commit  [q] quit"
}

// IssueDetailHelp returns help text for the issue detail modal
func IssueDetailHelp(hasFix bool) string {
	if hasFix {
		return " [a] preview fix  [Esc] close"
	}
	return " [Esc] close"
}

// DiffPreviewHelp returns help text for the diff preview modal
func DiffPreviewHelp() string {
	return " [y] apply fix  [n/Esc] cancel"
}

// CommitConfirmHelp returns help text for the commit confirm view
func CommitConfirmHelp() string {
	return " [y] commit  [e] edit message  [n/Esc] cancel"
}

// ProgressHelp returns help text for the progress view
func ProgressHelp() string {
	return " [q] quit"
}
