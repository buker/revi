package views

import (
	"fmt"
	"strings"

	"github.com/buker/revi/internal/review"
	"github.com/buker/revi/internal/tui/shared"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// IssueDetailModal displays the full details of a single issue
type IssueDetailModal struct {
	width    int
	height   int
	issue    *review.Issue
	mode     review.Mode
	viewport viewport.Model
	ready    bool
}

// NewIssueDetailModal creates a new issue detail modal
func NewIssueDetailModal() *IssueDetailModal {
	return &IssueDetailModal{}
}

// SetIssue sets the issue to display
func (v *IssueDetailModal) SetIssue(issue *review.Issue, mode review.Mode) {
	v.issue = issue
	v.mode = mode
	v.ready = false
}

// SetSize updates the modal dimensions
func (v *IssueDetailModal) SetSize(width, height int) {
	v.width = width
	v.height = height

	// Modal is 80% of screen, capped at reasonable max
	modalWidth := min(width*80/100, 70)
	modalHeight := min(height*80/100, 25)

	if !v.ready {
		v.viewport = viewport.New(modalWidth-4, modalHeight-8)
		v.ready = true
	} else {
		v.viewport.Width = modalWidth - 4
		v.viewport.Height = modalHeight - 8
	}

	// Update content
	if v.issue != nil {
		v.viewport.SetContent(v.renderContent())
	}
}

// HasFix returns true if the issue has a fix available
func (v *IssueDetailModal) HasFix() bool {
	return v.issue != nil && v.issue.Fix != nil && v.issue.Fix.Available
}

// Init initializes the modal
func (v *IssueDetailModal) Init() tea.Cmd {
	return nil
}

// Update handles messages for scrolling
func (v *IssueDetailModal) Update(msg tea.Msg) (*IssueDetailModal, tea.Cmd) {
	var cmd tea.Cmd
	if v.ready {
		v.viewport, cmd = v.viewport.Update(msg)
	}
	return v, cmd
}

// View renders the modal
func (v *IssueDetailModal) View() string {
	if v.issue == nil {
		return ""
	}

	modalWidth := min(v.width*80/100, 70)

	var b strings.Builder

	// Title
	info := review.GetModeInfo(v.mode)
	title := fmt.Sprintf("%s Issue", info.Name)
	b.WriteString(shared.ModalTitleStyle.Render(title))
	b.WriteString("\n")
	b.WriteString(shared.RenderDivider(modalWidth - 4))
	b.WriteString("\n")

	// Viewport with scrollable content
	if v.ready {
		b.WriteString(v.viewport.View())
	}

	b.WriteString("\n")
	b.WriteString(shared.RenderDivider(modalWidth - 4))
	b.WriteString("\n")

	// Help
	b.WriteString(shared.HelpKeyStyle.Render(shared.IssueDetailHelp(v.HasFix())))

	// Wrap in modal box
	content := b.String()
	modal := shared.ModalBoxStyle.
		Width(modalWidth).
		Render(content)

	// Center the modal
	return v.centerModal(modal)
}

// renderContent renders the scrollable content
func (v *IssueDetailModal) renderContent() string {
	var b strings.Builder

	// Location
	b.WriteString(shared.HeaderStyle.Render("Location: "))
	if v.issue.Location != "" {
		b.WriteString(v.issue.Location)
	} else {
		b.WriteString("-")
	}
	b.WriteString("\n")

	// Severity
	b.WriteString(shared.HeaderStyle.Render("Severity: "))
	sevStyle := shared.SeverityStyle(v.issue.Severity)
	b.WriteString(sevStyle.Render(strings.ToUpper(v.issue.Severity)))
	b.WriteString("\n\n")

	// Description
	b.WriteString(shared.HeaderStyle.Render("Description:"))
	b.WriteString("\n")
	b.WriteString(wordWrap(v.issue.Description, 60))
	b.WriteString("\n")

	// Fix information
	if v.issue.Fix != nil {
		b.WriteString("\n")
		if lineLen > 0 {
			result.WriteString(" ")
			lineLen++
		}
		result.WriteString(word)
		lineLen += wordLen
	}
				b.WriteString(shared.HeaderStyle.Render("Reason: "))
				b.WriteString(wordWrap(v.issue.Fix.Reason, 55))
			}
			if len(v.issue.Fix.Alternatives) > 0 {
				b.WriteString("\n")
				b.WriteString(shared.HeaderStyle.Render("Alternatives:"))
				for _, alt := range v.issue.Fix.Alternatives {
					b.WriteString("\n  • ")
					b.WriteString(alt)
				}
			}
		}
	}

	return b.String()
}

// centerModal centers the modal in the terminal
func (v *IssueDetailModal) centerModal(modal string) string {
	lines := strings.Split(modal, "\n")
	modalHeight := len(lines)
	modalWidth := 0
	for _, line := range lines {
		if len(line) > modalWidth {
			modalWidth = len(line)
		}
	}

	// Calculate vertical padding
	topPadding := (v.height - modalHeight) / 2
	if topPadding < 0 {
		topPadding = 0
	}

	// Calculate horizontal padding
	leftPadding := (v.width - modalWidth) / 2
	if leftPadding < 0 {
		leftPadding = 0
	}

	var b strings.Builder
	for i := 0; i < topPadding; i++ {
		b.WriteString("\n")
	}

	padStr := strings.Repeat(" ", leftPadding)
	for _, line := range lines {
		b.WriteString(padStr)
		b.WriteString(line)
		b.WriteString("\n")
	}

	return b.String()
}

// wordWrap wraps text to the specified width
func wordWrap(text string, width int) string {
	if width <= 0 {
		return text
	}

	var result strings.Builder
	words := strings.Fields(text)
	lineLen := 0

	for i, word := range words {
		wordLen := len(word)
		if lineLen+wordLen+1 > width && lineLen > 0 {
			result.WriteString("\n")
			lineLen = 0
		}
		if lineLen > 0 {
			result.WriteString(" ")
			lineLen++
		}
		result.WriteString(word)
		lineLen += wordLen

		// Check if this is the last word
		if i < len(words)-1 {
			continue
		}
	}

	return result.String()
}

// min returns the smaller of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// OverlayOnBackground renders the modal overlaid on a dimmed background
// Note: True overlay compositing in terminal is complex.
// This simplified version just shows the modal; proper overlay would require
// compositing the modal on top of the dimmed background.
func (v *IssueDetailModal) OverlayOnBackground(background string) string {
	_ = background // reserved for future overlay implementation
	return v.View()
}
