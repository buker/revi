package views

import (
	"fmt"
	"strings"

	"github.com/buker/revi/internal/tui/shared"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

// CommitConfirmView displays the commit confirmation screen
type CommitConfirmView struct {
	width         int
	height        int
	commitMessage string
	issuesFound   int
	issuesFixed   int
	blocked       bool
	editing       bool
	textarea      textarea.Model
}

// NewCommitConfirmView creates a new commit confirm view
func NewCommitConfirmView() *CommitConfirmView {
	ta := textarea.New()
	ta.Placeholder = "Enter commit message..."
	ta.ShowLineNumbers = false
	ta.CharLimit = 0

	return &CommitConfirmView{
		textarea: ta,
	}
}

// SetCommitMessage sets the commit message to display
func (v *CommitConfirmView) SetCommitMessage(msg string) {
	v.commitMessage = msg
	v.textarea.SetValue(msg)
}

// SetReviewSummary sets the review summary information
func (v *CommitConfirmView) SetReviewSummary(issuesFound, issuesFixed int, blocked bool) {
	v.issuesFound = issuesFound
	v.issuesFixed = issuesFixed
	v.blocked = blocked
}

// SetSize updates the view dimensions
func (v *CommitConfirmView) SetSize(width, height int) {
	v.width = width
	v.height = height

	// Size the textarea for editing mode
	v.textarea.SetWidth(min(width-10, 60))
	v.textarea.SetHeight(8)
}

// GetCommitMessage returns the current commit message (may be edited)
func (v *CommitConfirmView) GetCommitMessage() string {
	if v.editing {
		return v.textarea.Value()
	}
	return v.commitMessage
}

// IsEditing returns true if in edit mode
func (v *CommitConfirmView) IsEditing() bool {
	return v.editing
}
// (remove the min function - it's already defined in detail.go)
// StartEditing enters edit mode
func (v *CommitConfirmView) StartEditing() tea.Cmd {
	v.editing = true
	v.textarea.SetValue(v.commitMessage)
	v.textarea.Focus()
	return textarea.Blink
}

// StopEditing exits edit mode and saves the message
func (v *CommitConfirmView) StopEditing() {
	v.editing = false
	v.commitMessage = v.textarea.Value()
	v.textarea.Blur()
}

// CancelEditing exits edit mode without saving
func (v *CommitConfirmView) CancelEditing() {
	v.editing = false
	v.textarea.SetValue(v.commitMessage)
	v.textarea.Blur()
}

// Init initializes the view
func (v *CommitConfirmView) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (v *CommitConfirmView) Update(msg tea.Msg) (*CommitConfirmView, tea.Cmd) {
	if v.editing {
		var cmd tea.Cmd
		v.textarea, cmd = v.textarea.Update(msg)
		return v, cmd
	}
	return v, nil
}

// View renders the commit confirm view
func (v *CommitConfirmView) View() string {
	var b strings.Builder

	// Header
	b.WriteString(shared.TitleStyle.Render("revi - Confirm Commit"))
	b.WriteString("\n")
	b.WriteString(shared.RenderDivider(54))
	b.WriteString("\n\n")

	// Commit message section
	b.WriteString(" ")
	b.WriteString(shared.HeaderStyle.Render("Commit Message:"))
	b.WriteString("\n")

	if v.editing {
		// Show textarea for editing
		b.WriteString(v.textarea.View())
		b.WriteString("\n\n")
		b.WriteString(shared.HelpKeyStyle.Render(" [Esc] cancel  [Ctrl+D] save"))
	} else {
		// Show message in a box
		b.WriteString(v.renderMessageBox())
	}

	b.WriteString("\n\n")

	// Review summary
	b.WriteString(" ")
	b.WriteString(shared.HeaderStyle.Render("Review Summary:"))
	b.WriteString("\n")
	b.WriteString(" ")
	b.WriteString(shared.RenderDivider(50))
	b.WriteString("\n")

	// Issues summary
	remaining := v.issuesFound - v.issuesFixed
	issuesSummary := fmt.Sprintf("  Issues: %d found", v.issuesFound)
	if v.issuesFixed > 0 {
		issuesSummary += fmt.Sprintf(" (%d fixed, %d remaining)", v.issuesFixed, remaining)
	}
	b.WriteString(issuesSummary)
	b.WriteString("\n")

	// Blocked status
	b.WriteString("  Blocked: ")
	if v.blocked {
		b.WriteString(shared.HighSeverityStyle.Render("Yes"))
	} else {
		b.WriteString(shared.StatusDoneStyle.Render("No"))
	}
	b.WriteString("\n\n")

	b.WriteString(shared.RenderDivider(54))
	b.WriteString("\n")

	// Help (only show if not editing)
	if !v.editing {
		b.WriteString(shared.HelpKeyStyle.Render(shared.CommitConfirmHelp()))
	}

	return b.String()
}

// renderMessageBox renders the commit message in a bordered box
func (v *CommitConfirmView) renderMessageBox() string {
	boxWidth := min(v.width-6, 55)
	if boxWidth < 20 {
		boxWidth = 55
	}

	lines := strings.Split(v.commitMessage, "\n")

	var b strings.Builder

	// Top border
	b.WriteString(" ┌")
	b.WriteString(strings.Repeat("─", boxWidth))
	b.WriteString("┐\n")

	// Message lines
	for _, line := range lines {
		// Truncate or pad line to fit
		displayLine := line
		if len(displayLine) > boxWidth-2 {
			displayLine = displayLine[:boxWidth-5] + "..."
		}
		padding := boxWidth - len(displayLine)
		if padding < 0 {
			padding = 0
		}

		b.WriteString(" │ ")
		b.WriteString(displayLine)
		b.WriteString(strings.Repeat(" ", padding-1))
		b.WriteString("│\n")
	}

	// Bottom border
	b.WriteString(" └")
	b.WriteString(strings.Repeat("─", boxWidth))
	b.WriteString("┘")

	return b.String()
}
