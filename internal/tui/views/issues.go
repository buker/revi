package views

import (
	"fmt"
	"strings"

	"github.com/buker/revi/internal/review"
	"github.com/buker/revi/internal/tui/shared"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// IssueItem represents an issue with its source mode for display
type IssueItem struct {
	Issue review.Issue
	Mode  review.Mode
	Fixed bool
}

// IssuesTableView displays a table of all issues
type IssuesTableView struct {
	width         int
	height        int
	issues        []IssueItem
	cursor        int
	commitMessage string
	blocked       bool
	blockReason   string
	keys          shared.KeyMap
}

// NewIssuesTableView creates a new issues table view
func NewIssuesTableView() *IssuesTableView {
	return &IssuesTableView{
		keys: shared.DefaultKeyMap(),
	}
}

// SetIssues sets the issues to display
func (v *IssuesTableView) SetIssues(results []*review.Result) {
	v.issues = nil
	for _, r := range results {
		if r == nil {
			continue
		}
		for _, issue := range r.Issues {
			v.issues = append(v.issues, IssueItem{
				Issue: issue,
				Mode:  r.Mode,
				Fixed: false,
			})
		}
	}
	v.cursor = 0
}

// SetCommitMessage sets the commit message to display
func (v *IssuesTableView) SetCommitMessage(msg string) {
	v.commitMessage = msg
}

// SetBlocked sets the blocked state and reason
func (v *IssuesTableView) SetBlocked(blocked bool, reason string) {
	v.blocked = blocked
	v.blockReason = reason
}

// MarkFixed marks an issue as fixed
func (v *IssuesTableView) MarkFixed(index int) {
	if index >= 0 && index < len(v.issues) {
		v.issues[index].Fixed = true
	}
}

// SetSize updates the view dimensions
func (v *IssuesTableView) SetSize(width, height int) {
	v.width = width
	v.height = height
}

// Cursor returns the current cursor position
func (v *IssuesTableView) Cursor() int {
	return v.cursor
}

// SelectedIssue returns the currently selected issue
func (v *IssuesTableView) SelectedIssue() *IssueItem {
	if v.cursor >= 0 && v.cursor < len(v.issues) {
		return &v.issues[v.cursor]
	}
	return nil
}

// IssueCount returns the total number of issues
func (v *IssuesTableView) IssueCount() int {
	return len(v.issues)
}

// Init initializes the view
func (v *IssuesTableView) Init() tea.Cmd {
	return nil
}

// Update handles key messages for navigation
func (v *IssuesTableView) Update(msg tea.Msg) (*IssuesTableView, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, v.keys.Up):
			if v.cursor > 0 {
				v.cursor--
			}
		case key.Matches(msg, v.keys.Down):
			if v.cursor < len(v.issues)-1 {
				v.cursor++
			}
		case key.Matches(msg, v.keys.Home):
			v.cursor = 0
		case key.Matches(msg, v.keys.End):
			v.cursor = len(v.issues) - 1
			if v.cursor < 0 {
				v.cursor = 0
			}
		}
	}
	return v, nil
}

// View renders the issues table
func (v *IssuesTableView) View() string {
	var b strings.Builder

	// Header with count and position
	title := fmt.Sprintf("revi - Issues (%d found)", len(v.issues))
	position := ""
	if len(v.issues) > 0 {
		position = fmt.Sprintf("[%d/%d]", v.cursor+1, len(v.issues))
	}

	// Calculate spacing for right-aligned position
	headerWidth := 54
	spacing := headerWidth - len(title) - len(position)
	if spacing < 1 {
		spacing = 1
	}

	b.WriteString(shared.TitleStyle.Render(title))
	b.WriteString(strings.Repeat(" ", spacing))
	b.WriteString(shared.HelpDescStyle.Render(position))
	b.WriteString("\n")
	b.WriteString(shared.RenderDivider(headerWidth))
	b.WriteString("\n")

	// Table header
	header := fmt.Sprintf(" %-4s │ %-11s │ %-14s │ %-32s │ %s", "SEV", "MODE", "LOCATION", "SUMMARY", "FIX")
	b.WriteString(shared.TableHeaderStyle.Render(header))
	b.WriteString("\n")
	b.WriteString(shared.RenderDivider(headerWidth + 30))
	b.WriteString("\n")

	// Table rows
	if len(v.issues) == 0 {
		b.WriteString(" No issues found\n")
	} else {
		for i, item := range v.issues {
			row := v.renderRow(i, item)
			b.WriteString(row)
			b.WriteString("\n")
		}
	}

	b.WriteString(shared.RenderDivider(headerWidth + 30))
	b.WriteString("\n")

	// Commit message preview (first line only) - only show when not blocked
	if v.commitMessage != "" && !v.blocked {
		firstLine := strings.Split(v.commitMessage, "\n")[0]
		b.WriteString("\n")
		b.WriteString(" Commit: ")
		b.WriteString(shared.HelpDescStyle.Render(truncate(firstLine, 60)))
		b.WriteString("\n")
		b.WriteString(shared.RenderDivider(headerWidth + 30))
		b.WriteString("\n")
	}

	// Show blocking info if blocked
	if v.blocked {
		b.WriteString("\n")
		b.WriteString(shared.HighSeverityStyle.Render(" ⚠ BLOCKED: " + v.blockReason))
		b.WriteString("\n")
		b.WriteString(shared.HelpDescStyle.Render(" Fix high-severity issues or use --no-block to override"))
		b.WriteString("\n")
		b.WriteString(shared.RenderDivider(headerWidth + 30))
		b.WriteString("\n")
	}

	// Help
	if v.blocked {
		b.WriteString(shared.HelpKeyStyle.Render(shared.IssuesTableHelpBlocked()))
	} else {
		b.WriteString(shared.HelpKeyStyle.Render(shared.IssuesTableHelp()))
	}

	return b.String()
}

// renderRow renders a single issue row
func (v *IssuesTableView) renderRow(index int, item IssueItem) string {
	isSelected := index == v.cursor

	// Selection marker
	marker := " "
	if isSelected {
		marker = shared.SelectionMarker.Render(shared.SelectionChar)
	}

	// Severity
	sevAbbrev := shared.SeverityAbbrev(item.Issue.Severity)
	sevStyle := shared.SeverityStyle(item.Issue.Severity)
	sev := sevStyle.Render(sevAbbrev)

	// Mode
	info := review.GetModeInfo(item.Mode)
	modeName := truncate(info.Name, 11)

	// Location
	location := truncate(item.Issue.Location, 14)
	if location == "" {
		location = "-"
	}

	// Summary (truncated description)
	summary := truncate(item.Issue.Description, 32)

	// Fix indicator
	var fixIndicator string
	if item.Fixed {
		fixIndicator = shared.StatusDoneStyle.Render("[FIXED]")
	} else if item.Issue.Fix != nil && item.Issue.Fix.Available {
		fixIndicator = shared.FixAvailableStyle.Render(shared.FixAvailableIndicator)
	} else {
		fixIndicator = shared.FixUnavailableStyle.Render(shared.FixUnavailableIndicator)
	}

	row := fmt.Sprintf("%s%-4s │ %-11s │ %-14s │ %-32s │ %s",
		marker,
		sev,
		modeName,
		location,
		summary,
		fixIndicator,
	)

	if isSelected {
		return shared.SelectedRowStyle.Render(row)
	}
	return row
}
