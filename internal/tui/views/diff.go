package views

import (
	"fmt"
	"strings"

	"github.com/buker/revi/internal/review"
	"github.com/buker/revi/internal/tui/shared"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// DiffPreviewModal displays a diff preview for a fix
type DiffPreviewModal struct {
	width    int
	height   int
	fix      *review.Fix
	viewport viewport.Model
	ready    bool
}

// NewDiffPreviewModal creates a new diff preview modal
func NewDiffPreviewModal() *DiffPreviewModal {
	return &DiffPreviewModal{}
}

// SetFix sets the fix to preview
func (v *DiffPreviewModal) SetFix(fix *review.Fix) {
	v.fix = fix
	v.ready = false
}

// SetSize updates the modal dimensions
func (v *DiffPreviewModal) SetSize(width, height int) {
	v.width = width
	v.height = height

	// Modal is 80% of screen, capped at reasonable max
	modalWidth := min(width*80/100, 80)
	modalHeight := min(height*80/100, 30)

	if !v.ready {
		v.viewport = viewport.New(modalWidth-4, modalHeight-6)
		v.ready = true
	} else {
		v.viewport.Width = modalWidth - 4
		v.viewport.Height = modalHeight - 6
	}

	// Update content
	if v.fix != nil {
		v.viewport.SetContent(v.renderDiff())
	}
}

// Init initializes the modal
func (v *DiffPreviewModal) Init() tea.Cmd {
	return nil
}

// Update handles messages for scrolling
func (v *DiffPreviewModal) Update(msg tea.Msg) (*DiffPreviewModal, tea.Cmd) {
	var cmd tea.Cmd
	if v.ready {
		v.viewport, cmd = v.viewport.Update(msg)
	}
	return v, cmd
}

// View renders the modal
func (v *DiffPreviewModal) View() string {
	if v.fix == nil {
		return ""
	}

	modalWidth := min(v.width*80/100, 80)

	var b strings.Builder

	// Title
	title := fmt.Sprintf("Fix Preview: %s", v.fix.FilePath)
	b.WriteString(shared.ModalTitleStyle.Render(title))
	b.WriteString("\n")
	b.WriteString(shared.RenderDivider(modalWidth - 4))
	b.WriteString("\n")

	// Viewport with diff content
	if v.ready {
		b.WriteString(v.viewport.View())
	}

	b.WriteString("\n")
	b.WriteString(shared.RenderDivider(modalWidth - 4))
	b.WriteString("\n")

	// Help
	b.WriteString(shared.HelpKeyStyle.Render(shared.DiffPreviewHelp()))

	// Wrap in modal box
	content := b.String()
	modal := shared.ModalBoxStyle.
		Width(modalWidth).
		Render(content)

	// Center the modal
	return v.centerModal(modal)
}

// renderDiff renders the diff with syntax highlighting
func (v *DiffPreviewModal) renderDiff() string {
	if v.fix == nil || v.fix.Code == "" {
		return "No diff available"
	}

	var b strings.Builder

	// Show hunk header
	hunkHeader := fmt.Sprintf("@@ -%d,%d +%d,? @@",
		v.fix.StartLine,
		v.fix.EndLine-v.fix.StartLine+1,
		v.fix.StartLine,
	)
	b.WriteString(shared.DiffHunkStyle.Render(hunkHeader))
	b.WriteString("\n\n")

	// Show the replacement code with + prefix
	lines := strings.Split(v.fix.Code, "\n")
	for _, line := range lines {
		styledLine := shared.DiffAddedStyle.Render("+ " + line)
		b.WriteString(styledLine)
		b.WriteString("\n")
	}

	// Note about replacement
	b.WriteString("\n")
	b.WriteString(shared.HelpDescStyle.Render(
		fmt.Sprintf("This will replace lines %d-%d in %s",
			v.fix.StartLine, v.fix.EndLine, v.fix.FilePath)))

	return b.String()
}

// centerModal centers the modal in the terminal
func (v *DiffPreviewModal) centerModal(modal string) string {
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

// GetFix returns the current fix
func (v *DiffPreviewModal) GetFix() *review.Fix {
	return v.fix
}
