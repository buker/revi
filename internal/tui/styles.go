// Package tui provides the terminal user interface using Bubble Tea.
package tui

import (
	"github.com/buker/revi/internal/tui/shared"
	"github.com/charmbracelet/lipgloss"
)

// Re-export colors from shared package for backwards compatibility
var (
	ColorHigh     = shared.ColorHigh
	ColorMedium   = shared.ColorMedium
	ColorLow      = shared.ColorLow
	ColorGreen    = shared.ColorGreen
	ColorBorder   = shared.ColorBorder
	ColorDimmed   = shared.ColorDimmed
	ColorAccent   = shared.ColorAccent
	ColorSelected = shared.ColorSelected
)

// Re-export styles from shared package for backwards compatibility
var (
	HeaderStyle         = shared.HeaderStyle
	TitleStyle          = shared.TitleStyle
	TableHeaderStyle    = shared.TableHeaderStyle
	SelectedRowStyle    = shared.SelectedRowStyle
	HighSeverityStyle   = shared.HighSeverityStyle
	MediumSeverityStyle = shared.MediumSeverityStyle
	LowSeverityStyle    = shared.LowSeverityStyle
	ModalBoxStyle       = shared.ModalBoxStyle
	ModalTitleStyle     = shared.ModalTitleStyle
	StatusPendingStyle  = shared.StatusPendingStyle
	StatusRunningStyle  = shared.StatusRunningStyle
	StatusDoneStyle     = shared.StatusDoneStyle
	StatusFailedStyle   = shared.StatusFailedStyle
	DiffAddedStyle      = shared.DiffAddedStyle
	DiffRemovedStyle    = shared.DiffRemovedStyle
	DiffContextStyle    = shared.DiffContextStyle
	DiffHunkStyle       = shared.DiffHunkStyle
	HelpKeyStyle        = shared.HelpKeyStyle
	HelpDescStyle       = shared.HelpDescStyle
	DividerStyle        = shared.DividerStyle
	FixAvailableStyle   = shared.FixAvailableStyle
	FixUnavailableStyle = shared.FixUnavailableStyle
	SelectionMarker     = shared.SelectionMarker
)

// Re-export constants from shared package for backwards compatibility
const (
	StatusIndicatorPending  = shared.StatusIndicatorPending
	StatusIndicatorRunning  = shared.StatusIndicatorRunning
	StatusIndicatorDone     = shared.StatusIndicatorDone
	StatusIndicatorFailed   = shared.StatusIndicatorFailed
	FixAvailableIndicator   = shared.FixAvailableIndicator
	FixUnavailableIndicator = shared.FixUnavailableIndicator
	SelectionChar           = shared.SelectionChar
)

// Re-export functions from shared package for backwards compatibility

// RenderDivider creates a horizontal divider of the specified width
func RenderDivider(width int) string {
	return shared.RenderDivider(width)
}

// SeverityStyle returns the appropriate style for a severity level
func SeverityStyle(severity string) lipgloss.Style {
	return shared.SeverityStyle(severity)
}

// SeverityAbbrev returns a 3-letter abbreviation for severity
func SeverityAbbrev(severity string) string {
	return shared.SeverityAbbrev(severity)
}
