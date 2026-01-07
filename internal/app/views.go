package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"lazybeads/internal/ui"
)

// View renders the application
func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	switch m.mode {
	case ViewHelp:
		return m.viewHelp()
	case ViewConfirm:
		return m.viewConfirm()
	case ViewForm:
		return m.viewForm()
	case ViewDetail:
		if m.width < 80 {
			// Narrow mode: full screen detail
			return m.viewDetailOverlay()
		}
		return m.viewMain()
	case ViewEditTitle, ViewEditStatus, ViewEditPriority, ViewEditType:
		return m.viewMainWithInlineBar()
	default:
		return m.viewMain()
	}
}

func (m Model) viewMain() string {
	var b strings.Builder

	// Title bar
	title := ui.TitleStyle.Render("lazybeads")
	focusInfo := m.focusPanelString()
	titleLine := lipgloss.JoinHorizontal(
		lipgloss.Left,
		title,
		strings.Repeat(" ", max(0, m.width-lipgloss.Width(title)-lipgloss.Width(focusInfo)-2)),
		ui.HelpDescStyle.Render(focusInfo),
	)
	b.WriteString(titleLine + "\n")

	// Content area
	contentHeight := m.height - 4

	// Stack visible panels vertically
	var panelViews []string
	if m.isInProgressVisible() {
		panelViews = append(panelViews, m.inProgressPanel.View())
	}
	panelViews = append(panelViews, m.openPanel.View())
	panelViews = append(panelViews, m.closedPanel.View())
	leftColumn := lipgloss.JoinVertical(lipgloss.Left, panelViews...)

	if m.width >= 80 {
		// Wide mode: panels on left, detail on right
		detailStyle := ui.PanelStyle
		if m.mode == ViewDetail {
			detailStyle = ui.FocusedPanelStyle
		}

		detailContent := ""
		if m.selected != nil {
			m.updateDetailContent()
			detailContent = m.detail.View()
		} else {
			detailContent = ui.HelpDescStyle.Render("Select a task to view details")
		}

		detailPanel := detailStyle.
			Width(m.width/2 - 2).
			Height(contentHeight).
			Render(detailContent)

		b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, leftColumn, detailPanel))
	} else {
		// Narrow mode: panels only
		b.WriteString(leftColumn)
	}

	b.WriteString("\n")

	// Error message if any
	if m.err != nil {
		b.WriteString(ui.ErrorStyle.Render("Error: " + m.err.Error()))
		b.WriteString("\n")
		m.err = nil
	}

	// Help bar
	helpText := m.renderHelpBar()
	b.WriteString(ui.HelpBarStyle.Render(helpText))

	return b.String()
}

func (m Model) viewDetailOverlay() string {
	var b strings.Builder

	title := ui.TitleStyle.Render("Task Details")
	b.WriteString(title + "\n\n")

	m.updateDetailContent()
	content := ui.OverlayStyle.
		Width(m.width - 4).
		Height(m.height - 6).
		Render(m.detail.View())
	b.WriteString(content)
	b.WriteString("\n")
	b.WriteString(ui.HelpBarStyle.Render("enter/esc: back  ?: help"))

	return b.String()
}

func (m Model) viewHelp() string {
	var b strings.Builder

	b.WriteString(ui.TitleStyle.Render("Keyboard Shortcuts") + "\n\n")

	helpContent := `
Navigation
  j/k, ↑/↓    Move up/down in focused panel
  g/G         Jump to top/bottom
  ^u/^d       Page up/down

Panels (h/l to cycle focus)
  In Progress Tasks with status "in_progress"
  Open        Tasks with status "open"
  Closed      Tasks with status "closed"

Actions
  enter       View task details
  a           Add new task
  x           Delete selected task
  R           Refresh list

Field Editing
  t           Edit title (modal)
  s           Edit status (modal)
  p           Edit priority (modal)
  y           Edit type (modal)
  d/e         Edit description ($EDITOR)

General
  ?           Toggle this help
  q           Quit
  esc         Back/cancel

Auto-refresh: polls every 2 seconds
`
	b.WriteString(ui.OverlayStyle.Render(helpContent))
	b.WriteString("\n")
	b.WriteString(ui.HelpBarStyle.Render("Press ? or esc to close"))

	return b.String()
}

func (m Model) viewConfirm() string {
	var b strings.Builder

	b.WriteString(ui.TitleStyle.Render("Confirm") + "\n\n")
	b.WriteString(ui.OverlayStyle.Render(m.confirmMsg + "\n\n(y)es / (n)o"))

	return b.String()
}

func (m Model) viewMainWithInlineBar() string {
	var b strings.Builder

	// Title bar
	title := ui.TitleStyle.Render("lazybeads")
	focusInfo := m.focusPanelString()
	titleLine := lipgloss.JoinHorizontal(
		lipgloss.Left,
		title,
		strings.Repeat(" ", max(0, m.width-lipgloss.Width(title)-lipgloss.Width(focusInfo)-2)),
		ui.HelpDescStyle.Render(focusInfo),
	)
	b.WriteString(titleLine + "\n")

	// Content area (same as viewMain but with one less line for the taller inline bar)
	contentHeight := m.height - 4

	// Stack visible panels vertically
	var panelViews []string
	if m.isInProgressVisible() {
		panelViews = append(panelViews, m.inProgressPanel.View())
	}
	panelViews = append(panelViews, m.openPanel.View())
	panelViews = append(panelViews, m.closedPanel.View())
	leftColumn := lipgloss.JoinVertical(lipgloss.Left, panelViews...)

	if m.width >= 80 {
		// Wide mode: panels on left, detail on right
		detailStyle := ui.PanelStyle

		detailContent := ""
		if m.selected != nil {
			m.updateDetailContent()
			detailContent = m.detail.View()
		} else {
			detailContent = ui.HelpDescStyle.Render("Select a task to view details")
		}

		detailPanel := detailStyle.
			Width(m.width/2 - 2).
			Height(contentHeight).
			Render(detailContent)

		b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, leftColumn, detailPanel))
	} else {
		// Narrow mode: panels only
		b.WriteString(leftColumn)
	}

	b.WriteString("\n")

	// Inline bar instead of help bar
	b.WriteString(m.inlineBar.View(m.width))

	return b.String()
}

func (m Model) focusPanelString() string {
	switch m.focusedPanel {
	case FocusInProgress:
		return "[in progress]"
	case FocusOpen:
		return "[open]"
	case FocusClosed:
		return "[closed]"
	default:
		return ""
	}
}

func (m Model) renderHelpBar() string {
	keys := []struct {
		key  string
		desc string
	}{
		{"j/k", "nav"},
		{"h/l", "panel"},
		{"enter", "detail"},
		{"t/s/p/y/d", "edit"},
		{"x", "delete"},
		{"?", "help"},
		{"q", "quit"},
	}

	var parts []string
	for _, k := range keys {
		part := ui.HelpKeyStyle.Render(k.key) + ":" + ui.HelpDescStyle.Render(k.desc)
		parts = append(parts, part)
	}

	return strings.Join(parts, "  ")
}

func (m *Model) updateDetailContent() {
	if m.selected == nil {
		m.detail.SetContent("")
		return
	}

	t := m.selected
	var b strings.Builder

	b.WriteString(ui.DetailLabelStyle.Render("ID:"))
	b.WriteString(ui.DetailValueStyle.Render(t.ID))
	b.WriteString("\n")

	b.WriteString(ui.DetailLabelStyle.Render("Title:"))
	b.WriteString(ui.DetailValueStyle.Render(t.Title))
	b.WriteString("\n")

	b.WriteString(ui.DetailLabelStyle.Render("Status:"))
	b.WriteString(ui.StatusStyle(t.Status).Render(t.Status))
	b.WriteString("\n")

	b.WriteString(ui.DetailLabelStyle.Render("Priority:"))
	b.WriteString(ui.PriorityStyle(t.Priority).Render(t.PriorityString()))
	b.WriteString("\n")

	b.WriteString(ui.DetailLabelStyle.Render("Type:"))
	b.WriteString(ui.DetailValueStyle.Render(t.Type))
	b.WriteString("\n")

	if t.Description != "" {
		b.WriteString("\n")
		b.WriteString(ui.DetailLabelStyle.Render("Description:"))
		b.WriteString("\n")
		// Wrap description to fit panel width
		descWidth := m.detail.Width - 2
		if descWidth < 20 {
			descWidth = 20
		}
		wrappedDesc := lipgloss.NewStyle().Width(descWidth).Render(t.Description)
		b.WriteString(wrappedDesc)
		b.WriteString("\n")
	}

	if len(t.BlockedBy) > 0 {
		b.WriteString("\n")
		b.WriteString(ui.DetailLabelStyle.Render("Blocked by:"))
		b.WriteString("\n")
		for _, id := range t.BlockedBy {
			b.WriteString("  - " + id + "\n")
		}
	}

	if len(t.Blocks) > 0 {
		b.WriteString("\n")
		b.WriteString(ui.DetailLabelStyle.Render("Blocks:"))
		b.WriteString("\n")
		for _, id := range t.Blocks {
			b.WriteString("  - " + id + "\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(ui.DetailLabelStyle.Render("Created:"))
	b.WriteString(ui.DetailValueStyle.Render(t.CreatedAt.Format("2006-01-02 15:04")))

	m.detail.SetContent(b.String())
}

func (m Model) viewForm() string {
	var b strings.Builder

	if m.editing {
		b.WriteString(ui.TitleStyle.Render("Edit Task") + "\n\n")
	} else {
		b.WriteString(ui.TitleStyle.Render("New Task") + "\n\n")
	}

	// Title field
	titleLabel := ui.FormLabelStyle.Render("Title:")
	titleStyle := ui.FormInputStyle
	if m.formFocus == 0 {
		titleStyle = ui.FormInputFocusedStyle
	}
	titleInput := titleStyle.Width(m.width - 20).Render(m.formTitle.View())
	b.WriteString(titleLabel + "\n" + titleInput + "\n\n")

	// Description field
	descLabel := ui.FormLabelStyle.Render("Description:")
	descStyle := ui.FormInputStyle
	if m.formFocus == 1 {
		descStyle = ui.FormInputFocusedStyle
	}
	descInput := descStyle.Width(m.width - 20).Render(m.formDesc.View())
	b.WriteString(descLabel + "\n" + descInput + "\n\n")

	// Priority selector
	priLabel := ui.FormLabelStyle.Render("Priority:")
	priValue := ""
	for i := 0; i <= 4; i++ {
		style := ui.HelpDescStyle
		if i == m.formPriority {
			style = ui.PriorityStyle(i).Bold(true)
		}
		priValue += style.Render(fmt.Sprintf(" P%d ", i))
	}
	focusIndicator := ""
	if m.formFocus == 2 {
		focusIndicator = " <"
	}
	b.WriteString(priLabel + priValue + focusIndicator + "\n\n")

	// Type selector
	typeLabel := ui.FormLabelStyle.Render("Type:")
	types := []string{"task", "bug", "feature", "epic", "chore"}
	typeValue := ""
	for _, t := range types {
		style := ui.HelpDescStyle
		if t == m.formType {
			style = ui.HelpKeyStyle
		}
		typeValue += style.Render(fmt.Sprintf(" %s ", t))
	}
	focusIndicator = ""
	if m.formFocus == 3 {
		focusIndicator = " <"
	}
	b.WriteString(typeLabel + typeValue + focusIndicator + "\n\n")

	// Help
	b.WriteString("\n")
	b.WriteString(ui.HelpBarStyle.Render("tab/shift+tab: next/prev field  enter: submit  esc: cancel"))

	return b.String()
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
