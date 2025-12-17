// Package shared provides shared types, styles, and constants for the TUI package.
package shared

import "github.com/charmbracelet/lipgloss"

// Color definitions for the TUI
var (
	ColorHigh     = lipgloss.Color("#FF5555") // Red - high severity
	ColorMedium   = lipgloss.Color("#FFAA00") // Yellow/Orange - medium severity
	ColorLow      = lipgloss.Color("#888888") // Gray - low severity
	ColorGreen    = lipgloss.Color("#55FF55") // Green - additions/success
	ColorBorder   = lipgloss.Color("#444444") // Border color
	ColorDimmed   = lipgloss.Color("#666666") // Dimmed text
	ColorAccent   = lipgloss.Color("#7B68EE") // Accent color (medium slate blue)
	ColorSelected = lipgloss.Color("#333333") // Selected row background
)

// Style definitions for shared components
var (
	// Header styles
	HeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF"))

	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorAccent)

	// Table styles
	TableHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#FFFFFF")).
				BorderBottom(true).
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(ColorBorder)

	SelectedRowStyle = lipgloss.NewStyle().
				Background(ColorSelected)

	// Severity styles
	HighSeverityStyle = lipgloss.NewStyle().
				Foreground(ColorHigh).
				Bold(true)

	MediumSeverityStyle = lipgloss.NewStyle().
				Foreground(ColorMedium)

	LowSeverityStyle = lipgloss.NewStyle().
				Foreground(ColorLow)

	// Modal styles
	ModalBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorder).
			Padding(1, 2)

	ModalTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorAccent).
			MarginBottom(1)

	// Status indicator styles
	StatusPendingStyle = lipgloss.NewStyle().
				Foreground(ColorDimmed)

	StatusRunningStyle = lipgloss.NewStyle().
				Foreground(ColorMedium)

	StatusDoneStyle = lipgloss.NewStyle().
			Foreground(ColorGreen)

	StatusFailedStyle = lipgloss.NewStyle().
				Foreground(ColorHigh)

	// Diff styles
	DiffAddedStyle = lipgloss.NewStyle().
			Foreground(ColorGreen)

	DiffRemovedStyle = lipgloss.NewStyle().
				Foreground(ColorHigh)

	DiffContextStyle = lipgloss.NewStyle().
				Foreground(ColorDimmed)

	DiffHunkStyle = lipgloss.NewStyle().
			Foreground(ColorAccent)

	// Help/Footer styles
	HelpKeyStyle = lipgloss.NewStyle().
			Foreground(ColorAccent)

	HelpDescStyle = lipgloss.NewStyle().
			Foreground(ColorDimmed)

	// Divider
	DividerStyle = lipgloss.NewStyle().
			Foreground(ColorBorder)

	// Fix indicator styles
	FixAvailableStyle = lipgloss.NewStyle().
				Foreground(ColorGreen)

	FixUnavailableStyle = lipgloss.NewStyle().
				Foreground(ColorDimmed)

	// Selection marker
	SelectionMarker = lipgloss.NewStyle().
			Foreground(ColorAccent).
			Bold(true)
)

// Status indicators
const (
	StatusIndicatorPending = "○"
	StatusIndicatorRunning = "◐"
	StatusIndicatorDone    = "✓"
	StatusIndicatorFailed  = "✗"

	FixAvailableIndicator   = "✓"
	FixUnavailableIndicator = "✗"

	SelectionChar = "▶"
)

// RenderDivider creates a horizontal divider of the specified width
func RenderDivider(width int) string {
	return DividerStyle.Render(repeatChar('─', width))
}

// repeatChar returns a string with the character repeated n times
func repeatChar(char rune, n int) string {
	if n <= 0 {
		return ""
	}
	result := make([]rune, n)
	for i := range result {
		result[i] = char
	}
	return string(result)
}

// SeverityStyle returns the appropriate style for a severity level
func SeverityStyle(severity string) lipgloss.Style {
	switch severity {
	case "high":
		return HighSeverityStyle
	case "medium":
		return MediumSeverityStyle
	default:
		return LowSeverityStyle
	}
}

// SeverityAbbrev returns a 3-letter abbreviation for severity
func SeverityAbbrev(severity string) string {
	switch severity {
	case "high":
		return "HIG"
	case "medium":
		return "MED"
	default:
		return "LOW"
	}
}
