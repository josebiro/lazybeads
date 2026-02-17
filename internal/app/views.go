package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/josebiro/bb/internal/models"
	"github.com/josebiro/bb/internal/ui"
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
		if m.width < 80 || m.previousMode == ViewBoard {
			// Narrow mode OR coming from board: full screen detail overlay
			return m.viewDetailOverlay()
		}
		return m.viewMain()
	case ViewEditTitle, ViewEditStatus, ViewEditPriority, ViewEditType, ViewFilter, ViewAddBlocker, ViewRemoveBlocker, ViewEditText:
		return m.viewMainWithModal()
	case ViewAddComment:
		return m.viewAddComment()
	case ViewBoard:
		return m.viewBoard()
	default:
		return m.viewMain()
	}
}

func (m Model) viewMain() string {
	var b strings.Builder

	// Content area
	contentHeight := m.height - 2

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
			Height(contentHeight - 2). // -2 for lipgloss border (top + bottom)
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

	// Status bar (shows key bindings by default, search results when filtering)
	statusText := m.renderStatusBar()
	b.WriteString(ui.HelpBarStyle.Render(statusText))

	return b.String()
}

func (m Model) viewDetailOverlay() string {
	var b strings.Builder

	// Calculate available height:
	// - Title line + blank line = 2
	// - Content area = height - 4 (title, blank, content, help bar)
	// - Help bar = 1
	contentHeight := m.height - 4
	if contentHeight < 5 {
		contentHeight = 5
	}

	// Resize viewport for overlay mode
	m.detail.Width = m.width - 6        // Account for border padding
	m.detail.Height = contentHeight - 2 // Account for OverlayStyle border

	title := ui.TitleStyle.Render("Task Details")
	b.WriteString(title + "\n\n")

	m.updateDetailContent()
	content := ui.OverlayStyle.
		Width(m.width - 4).
		Height(contentHeight).
		Render(m.detail.View())
	b.WriteString(content)
	b.WriteString("\n")
	b.WriteString(ui.HelpBarStyle.Render("enter/esc: back  ?: help"))

	return b.String()
}

func (m *Model) viewHelp() string {
	var b strings.Builder

	b.WriteString(ui.TitleStyle.Render("Keyboard Shortcuts") + "\n\n")

	helpContent := `Navigation
  j/k, ↑/↓    Move up/down in focused panel
  g/G         Jump to top/bottom
  ^u/^d       Page up/down

Panels (h/l to cycle focus)
  In Progress Tasks with status "in_progress"
  Open        Tasks with status "open"
  Closed      Tasks with status "closed"

Views
  b           Toggle board view (Kanban columns)

Filtering
  /           Start inline search in status bar
  (typing)    Filter updates live as you type
  enter       Confirm filter and return to navigation
  esc         Clear filter and return to navigation
  backspace   On empty input, exit search mode
  o           Toggle open filter (open + in_progress)
  O           Toggle closed filter (closed only)
  r           Toggle ready filter (no blockers)
  A           Clear all filters

Actions
  enter       View task details
  a           Add new task
  x           Delete selected task
  R           Refresh list
  S           Cycle sort mode (Default/Created/Priority/Updated)

Field Editing
  e           Edit title (modal)
  s           Edit status (modal)
  p           Edit priority (modal)
  t           Edit type (modal)
  y           Copy issue ID to clipboard
  d           Edit description (modal)
  n           Edit notes (modal)
  C           Add comment
  B           Add blocker (dependency)
  D           Remove blocker

General
  ?           Toggle this help
  q           Quit
  esc         Back/cancel

Auto-refresh: polls every 2 seconds
`
	// Add custom commands section if any are configured
	if len(m.customCommands) > 0 {
		helpContent += "\nCustom Commands\n"
		for _, cmd := range m.customCommands {
			helpContent += fmt.Sprintf("  %-10s  %s (%s)\n", cmd.Key, cmd.Description, cmd.Context)
		}
	}

	// Set content on the viewport
	m.helpViewport.SetContent(helpContent)

	// Render viewport inside overlay style
	viewportContent := ui.OverlayStyle.
		Width(m.width - 4).
		Height(m.helpViewport.Height).
		Render(m.helpViewport.View())
	b.WriteString(viewportContent)
	b.WriteString("\n")

	// Build status bar with scroll indicator
	scrollInfo := fmt.Sprintf("%d%%", int(m.helpViewport.ScrollPercent()*100))
	helpBar := fmt.Sprintf("j/k:scroll  ^u/^d:page  ?/esc:close  %s", scrollInfo)
	b.WriteString(ui.HelpBarStyle.Render(helpBar))

	return b.String()
}

func (m Model) viewConfirm() string {
	var b strings.Builder

	b.WriteString(ui.TitleStyle.Render("Confirm") + "\n\n")
	b.WriteString(ui.OverlayStyle.Render(m.confirmMsg + "\n\n(y)es / (n)o"))

	return b.String()
}

func (m Model) viewMainWithModal() string {
	// Render the modal centered on screen
	return m.modal.View(m.width, m.height)
}

func (m Model) viewAddComment() string {
	var b strings.Builder

	taskID := ""
	if m.selected != nil {
		taskID = m.selected.ID
	}

	b.WriteString(ui.TitleStyle.Render("Add Comment") + "\n")
	b.WriteString(ui.HelpDescStyle.Render("Issue: "+taskID) + "\n\n")

	// Comment input
	inputStyle := ui.FormInputFocusedStyle.Width(m.width - 10)
	b.WriteString(inputStyle.Render(m.commentInput.View()))
	b.WriteString("\n\n")

	// Help
	b.WriteString(ui.HelpBarStyle.Render("enter: submit  esc: cancel"))

	return b.String()
}

func (m Model) renderStatusBar() string {
	var parts []string

	// Show status message if present (flash notifications like "Copied!")
	if m.statusMsg != "" {
		parts = append(parts, ui.SuccessStyle.Render(m.statusMsg))
	}

	// When in search mode, show the search input
	if m.searchMode {
		// Search input with cursor
		searchPart := ui.HelpKeyStyle.Render("/: ") + m.searchInput.View()
		parts = append(parts, searchPart)

		// Live result counts
		inProgressCount := m.inProgressPanel.TaskCount()
		openCount := m.openPanel.TaskCount()
		closedCount := m.closedPanel.TaskCount()
		total := inProgressCount + openCount + closedCount

		resultsPart := ui.HelpDescStyle.Render(fmt.Sprintf("(%d results", total))
		if inProgressCount > 0 {
			resultsPart += ui.StatusStyle("in_progress").Render(fmt.Sprintf(": %d in progress", inProgressCount))
		}
		if openCount > 0 {
			resultsPart += ui.StatusStyle("open").Render(fmt.Sprintf(", %d open", openCount))
		}
		if closedCount > 0 {
			resultsPart += ui.HelpDescStyle.Render(fmt.Sprintf(", %d closed", closedCount))
		}
		resultsPart += ui.HelpDescStyle.Render(")")
		parts = append(parts, resultsPart)

		// Minimal key hints during search
		parts = append(parts, ui.HelpKeyStyle.Render("enter")+":"+ui.HelpDescStyle.Render("confirm"))
		parts = append(parts, ui.HelpKeyStyle.Render("esc")+":"+ui.HelpDescStyle.Render("clear"))
	} else if m.filterQuery != "" {
		// When filter is active (but not in search mode), show search results
		// Filter indicator
		filterPart := ui.HelpKeyStyle.Render("/") + ":" +
			ui.HelpDescStyle.Render(m.filterQuery)
		parts = append(parts, filterPart)

		// Search result counts
		inProgressCount := m.inProgressPanel.TaskCount()
		openCount := m.openPanel.TaskCount()
		closedCount := m.closedPanel.TaskCount()
		total := inProgressCount + openCount + closedCount

		resultsPart := ui.HelpDescStyle.Render(fmt.Sprintf("(%d results:", total))
		if inProgressCount > 0 {
			resultsPart += ui.StatusStyle("in_progress").Render(fmt.Sprintf(" %d in progress", inProgressCount))
		}
		if openCount > 0 {
			resultsPart += ui.StatusStyle("open").Render(fmt.Sprintf(" %d open", openCount))
		}
		if closedCount > 0 {
			resultsPart += ui.HelpDescStyle.Render(fmt.Sprintf(" %d closed", closedCount))
		}
		resultsPart += ui.HelpDescStyle.Render(")")
		parts = append(parts, resultsPart)

		// Minimal key bindings when filtering
		parts = append(parts, ui.HelpKeyStyle.Render("esc")+":"+ui.HelpDescStyle.Render("clear"))
	} else {
		// Default: show key bindings
		keys := []struct {
			key  string
			desc string
		}{
			{"enter", "detail"},
			{"c", "create"},
			{"e/s/p/t", "edit"},
			{"d", "description"},
			{"n", "notes"},
			{"x", "delete"},
			{"?", "help"},
			{"q", "quit"},
		}

		for _, k := range keys {
			part := ui.HelpKeyStyle.Render(k.key) + ":" + ui.HelpDescStyle.Render(k.desc)
			parts = append(parts, part)
		}

		// Show current filter mode if not All
		if m.filterMode != FilterAll {
			filterPart := ui.HelpDescStyle.Render("[") +
				ui.HelpKeyStyle.Render(m.filterMode.String()) +
				ui.HelpDescStyle.Render("]")
			parts = append(parts, filterPart)
		}

		// Show current sort mode if not default
		if m.sortMode != SortDefault {
			sortPart := ui.HelpDescStyle.Render("[") +
				ui.HelpKeyStyle.Render(m.sortMode.String()) +
				ui.HelpDescStyle.Render("]")
			parts = append(parts, sortPart)
		}
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

	titleWidth := m.detail.Width - 12 // 12 = label width
	if titleWidth < 20 {
		titleWidth = 20
	}
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top,
		ui.DetailLabelStyle.Render("Title:"),
		ui.DetailValueStyle.Width(titleWidth).Render(t.Title),
	))
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

	if t.Assignee != "" {
		b.WriteString(ui.DetailLabelStyle.Render("Assignee:"))
		b.WriteString(ui.DetailValueStyle.Render(t.Assignee))
		b.WriteString("\n")
	}

	if t.Owner != "" {
		b.WriteString(ui.DetailLabelStyle.Render("Owner:"))
		b.WriteString(ui.DetailValueStyle.Render(t.Owner))
		b.WriteString("\n")
	}

	if len(t.Labels) > 0 {
		b.WriteString(ui.DetailLabelStyle.Render("Labels:"))
		b.WriteString(ui.DetailValueStyle.Render(strings.Join(t.Labels, ", ")))
		b.WriteString("\n")
	}

	if t.DueDate != nil {
		b.WriteString(ui.DetailLabelStyle.Render("Due:"))
		b.WriteString(ui.DetailValueStyle.Render(t.DueDate.Format("2006-01-02")))
		b.WriteString("\n")
	}

	if t.DeferUntil != nil {
		b.WriteString(ui.DetailLabelStyle.Render("Deferred:"))
		b.WriteString(ui.DetailValueStyle.Render("until " + t.DeferUntil.Format("2006-01-02")))
		b.WriteString("\n")
	}

	if t.Description != "" {
		b.WriteString("\n")
		b.WriteString(ui.DetailLabelStyle.Render("Description:"))
		b.WriteString("\n")
		// Render markdown description
		descWidth := m.detail.Width - 2
		if descWidth < 20 {
			descWidth = 20
		}
		renderedDesc := ui.RenderMarkdown(t.Description, descWidth)
		b.WriteString(renderedDesc)
	}

	if t.Design != "" {
		b.WriteString("\n")
		b.WriteString(ui.DetailLabelStyle.Render("Design:"))
		b.WriteString("\n")
		descWidth := m.detail.Width - 2
		if descWidth < 20 {
			descWidth = 20
		}
		b.WriteString(ui.RenderMarkdown(t.Design, descWidth))
	}

	if t.Notes != "" {
		b.WriteString("\n")
		b.WriteString(ui.DetailLabelStyle.Render("Notes:"))
		b.WriteString("\n")
		descWidth := m.detail.Width - 2
		if descWidth < 20 {
			descWidth = 20
		}
		b.WriteString(ui.RenderMarkdown(t.Notes, descWidth))
	}

	if t.AcceptanceCriteria != "" {
		b.WriteString("\n")
		b.WriteString(ui.DetailLabelStyle.Render("Acceptance:"))
		b.WriteString("\n")
		descWidth := m.detail.Width - 2
		if descWidth < 20 {
			descWidth = 20
		}
		b.WriteString(ui.RenderMarkdown(t.AcceptanceCriteria, descWidth))
	}

	if t.CloseReason != "" {
		b.WriteString("\n")
		b.WriteString(ui.DetailLabelStyle.Render("Close Reason:"))
		b.WriteString("\n")
		// Render markdown close reason
		descWidth := m.detail.Width - 2
		if descWidth < 20 {
			descWidth = 20
		}
		renderedReason := ui.RenderMarkdown(t.CloseReason, descWidth)
		b.WriteString(renderedReason)
	}

	if len(t.BlockedBy) > 0 {
		b.WriteString("\n")
		b.WriteString(ui.DetailLabelStyle.Render("Blocked by:"))
		b.WriteString("\n")
		for _, id := range t.BlockedBy {
			if linked, ok := m.tasksMap[id]; ok {
				// Show priority, ID, title, and status
				priority := ui.PriorityStyle(linked.Priority).Render(linked.PriorityString())
				idStyled := ui.HelpDescStyle.Render(id)
				status := ui.StatusStyle(linked.Status).Render("[" + linked.Status + "]")
				b.WriteString(fmt.Sprintf("  %s %s %s %s\n", priority, idStyled, linked.Title, status))
			} else {
				// Fallback: just show ID if task not in memory
				b.WriteString("  - " + id + "\n")
			}
		}
	}

	if len(t.Blocks) > 0 {
		b.WriteString("\n")
		b.WriteString(ui.DetailLabelStyle.Render("Blocks:"))
		b.WriteString("\n")
		for _, id := range t.Blocks {
			if linked, ok := m.tasksMap[id]; ok {
				// Show priority, ID, title, and status
				priority := ui.PriorityStyle(linked.Priority).Render(linked.PriorityString())
				idStyled := ui.HelpDescStyle.Render(id)
				status := ui.StatusStyle(linked.Status).Render("[" + linked.Status + "]")
				b.WriteString(fmt.Sprintf("  %s %s %s %s\n", priority, idStyled, linked.Title, status))
			} else {
				// Fallback: just show ID if task not in memory
				b.WriteString("  - " + id + "\n")
			}
		}
	}

	// Timestamps section
	b.WriteString("\n")
	b.WriteString(ui.DetailLabelStyle.Render("Created:"))
	b.WriteString(ui.DetailValueStyle.Render(t.CreatedAt.Format("2006-01-02 15:04")))
	if t.CreatedBy != "" {
		b.WriteString(ui.HelpDescStyle.Render(" by " + t.CreatedBy))
	}
	b.WriteString("\n")

	b.WriteString(ui.DetailLabelStyle.Render("Updated:"))
	b.WriteString(ui.DetailValueStyle.Render(t.UpdatedAt.Format("2006-01-02 15:04")))
	b.WriteString("\n")

	if t.ClosedAt != nil {
		b.WriteString(ui.DetailLabelStyle.Render("Closed:"))
		b.WriteString(ui.DetailValueStyle.Render(t.ClosedAt.Format("2006-01-02 15:04")))
		b.WriteString("\n")
	}

	// Comments section
	if len(m.comments) > 0 {
		b.WriteString("\n")
		b.WriteString(ui.DetailLabelStyle.Render(fmt.Sprintf("Comments (%d):", len(m.comments))))
		b.WriteString("\n")
		for _, c := range m.comments {
			// Format: "author (date):"
			header := fmt.Sprintf("  %s (%s):",
				ui.HelpKeyStyle.Render(c.Author),
				ui.HelpDescStyle.Render(c.CreatedAt.Format("2006-01-02 15:04")))
			b.WriteString(header)
			b.WriteString("\n")
			// Indent comment text
			lines := strings.Split(c.Text, "\n")
			for _, line := range lines {
				b.WriteString("    " + line + "\n")
			}
		}
	}

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

func (m Model) viewBoard() string {
	var b strings.Builder

	// Board view with 5 columns: Blocked, Open, Ready, In Progress, Done
	const totalColumns = 5
	const minColWidth = 30

	// Column border colors
	columnColors := [totalColumns]lipgloss.Color{
		lipgloss.Color("1"), // Red - Blocked
		lipgloss.Color("7"), // White - Open
		lipgloss.Color("2"), // Green - Ready
		lipgloss.Color("3"), // Yellow - In Progress
		lipgloss.Color("6"), // Cyan - Done
	}
	columnHeaders := [totalColumns]string{"BLOCKED", "OPEN", "READY", "IN PROGRESS", "DONE"}

	// Get tasks categorized into 5 columns
	columns := m.getBoardColumns()

	// Wrap tasks into boardTask structs
	type boardTask struct {
		task     models.Task
		priority string
		id       string
		title    string
	}
	var boardColumns [totalColumns][]boardTask
	for col := 0; col < totalColumns; col++ {
		for _, t := range columns[col] {
			boardColumns[col] = append(boardColumns[col], boardTask{
				task:     t,
				priority: t.PriorityString(),
				id:       t.ID,
				title:    t.Title,
			})
		}
	}

	// Responsive layout: how many columns fit?
	visibleCols := m.width / minColWidth
	if visibleCols > totalColumns {
		visibleCols = totalColumns
	}
	if visibleCols < 1 {
		visibleCols = 1
	}

	// Column width fills available space
	colWidth := m.width / visibleCols
	if colWidth < minColWidth {
		colWidth = minColWidth
	}

	// Ensure boardColumnOffset keeps focused column visible
	offset := m.boardColumnOffset
	if m.boardColumn < offset {
		offset = m.boardColumn
	}
	if m.boardColumn >= offset+visibleCols {
		offset = m.boardColumn - visibleCols + 1
	}
	if offset < 0 {
		offset = 0
	}
	if offset > totalColumns-visibleCols {
		offset = totalColumns - visibleCols
	}

	// Column height (screen height minus title and footer)
	colHeight := m.height - 4
	if colHeight < 8 {
		colHeight = 8
	}

	// Card height: 3 content lines + 1 divider = 4 lines per card
	cardHeight := 4
	cardsPerColumn := (colHeight - 2) / cardHeight
	if cardsPerColumn < 1 {
		cardsPerColumn = 1
	}

	// Helper to pad or truncate a string to exact visible width
	padToWidth := func(s string, width int) string {
		w := lipgloss.Width(s)
		if w < width {
			return s + strings.Repeat(" ", width-w)
		}
		return s
	}

	// Helper to truncate a string to fit within a visible width
	truncateToWidth := func(s string, width int) string {
		if lipgloss.Width(s) <= width {
			return s
		}
		for lipgloss.Width(s+"…") > width && len(s) > 0 {
			s = s[:len(s)-1]
		}
		return s + "…"
	}

	// Render a single task card (3 lines, no borders)
	// Returns 3 lines of content, each padded to innerWidth
	renderCard := func(bt boardTask, selected bool, innerWidth int) string {
		// Line 1: Priority + ID
		priority := ui.PriorityStyle(bt.task.Priority).Render(bt.priority)
		idStyled := ui.HelpDescStyle.Render(bt.id)
		line1 := priority + " " + idStyled

		// Line 2: Title (full width)
		title := truncateToWidth(bt.title, innerWidth)
		line2 := title

		// Line 3: Type + assignee
		typeStyled := ui.HelpDescStyle.Render(bt.task.Type)
		line3 := typeStyled
		if bt.task.Assignee != "" {
			assigneeStyled := lipgloss.NewStyle().Foreground(ui.ColorAccent).Render("@" + bt.task.Assignee)
			line3 = typeStyled + "  " + assigneeStyled
		}

		if selected {
			highlightStyle := lipgloss.NewStyle().
				Background(lipgloss.Color("236")).
				Foreground(lipgloss.Color("15"))
			line1 = highlightStyle.Render(padToWidth("▸"+priority+" "+bt.id, innerWidth))
			line2 = highlightStyle.Render(padToWidth("▸"+truncateToWidth(bt.title, innerWidth-1), innerWidth))
			// Line 3 for selected: re-render plain text with highlight
			meta := "▸" + bt.task.Type
			if bt.task.Assignee != "" {
				meta += "  @" + bt.task.Assignee
			}
			line3 = highlightStyle.Render(padToWidth(meta, innerWidth))
		} else {
			line1 = padToWidth(line1, innerWidth)
			line2 = padToWidth(line2, innerWidth)
			line3 = padToWidth(line3, innerWidth)
		}

		return line1 + "\n" + line2 + "\n" + line3
	}

	// Render a column
	renderColumn := func(tasks []boardTask, borderColor lipgloss.Color, focused bool, selectedRow int, header string, thisColWidth int) string {
		innerWidth := thisColWidth - 4 // -4 for column borders + padding

		headerColor := borderColor
		if !focused {
			headerColor = ui.ColorMuted
		}

		headerText := fmt.Sprintf(" %s (%d) ", header, len(tasks))
		headerStyle := lipgloss.NewStyle().
			Foreground(headerColor).
			Bold(focused)

		scrollOffset := 0
		if len(tasks) > cardsPerColumn {
			if focused {
				scrollOffset = selectedRow - cardsPerColumn/2
			}
			if scrollOffset < 0 {
				scrollOffset = 0
			}
			maxOffset := len(tasks) - cardsPerColumn
			if scrollOffset > maxOffset {
				scrollOffset = maxOffset
			}
		}

		// Build content lines (not yet wrapped in column borders)
		var contentLines []string

		if scrollOffset > 0 {
			contentLines = append(contentLines, ui.HelpDescStyle.Render(fmt.Sprintf(" ↑ %d more", scrollOffset)))
		}

		endIdx := scrollOffset + cardsPerColumn
		if endIdx > len(tasks) {
			endIdx = len(tasks)
		}

		dividerStyle := lipgloss.NewStyle().Foreground(ui.ColorBorder)
		divider := dividerStyle.Render(strings.Repeat("╌", innerWidth))

		for i := scrollOffset; i < endIdx; i++ {
			// Add divider between cards (not before first)
			if i > scrollOffset {
				contentLines = append(contentLines, divider)
			}
			isSelected := focused && i == selectedRow
			card := renderCard(tasks[i], isSelected, innerWidth)
			cardLines := strings.Split(card, "\n")
			contentLines = append(contentLines, cardLines...)
		}

		if endIdx < len(tasks) {
			remaining := len(tasks) - endIdx
			contentLines = append(contentLines, ui.HelpDescStyle.Render(fmt.Sprintf(" ↓ %d more", remaining)))
		}

		if len(tasks) == 0 {
			emptyStyle := lipgloss.NewStyle().Foreground(ui.ColorMuted).Italic(true)
			contentLines = append(contentLines, emptyStyle.Render(" (empty)"))
		}

		// Column border style
		borderStyle := lipgloss.NewStyle().Foreground(borderColor)
		if !focused {
			borderStyle = lipgloss.NewStyle().Foreground(ui.ColorBorder)
		}

		// Top border with embedded header
		headerWidth := lipgloss.Width(headerText)
		remainingWidth := thisColWidth - headerWidth - 4
		if remainingWidth < 0 {
			remainingWidth = 0
		}
		topBorder := borderStyle.Render("╭─") + headerStyle.Render(headerText) + borderStyle.Render(strings.Repeat("─", remainingWidth)+"─╮")

		// Wrap content lines in column borders
		var borderedContent []string
		for _, line := range contentLines {
			lineWidth := lipgloss.Width(line)
			if lineWidth < innerWidth {
				line = line + strings.Repeat(" ", innerWidth-lineWidth)
			}
			borderedContent = append(borderedContent, borderStyle.Render("│")+" "+line+" "+borderStyle.Render("│"))
		}

		// Pad to fill column height
		contentHeight := colHeight - 2
		for len(borderedContent) < contentHeight {
			emptyLine := strings.Repeat(" ", innerWidth)
			borderedContent = append(borderedContent, borderStyle.Render("│")+" "+emptyLine+" "+borderStyle.Render("│"))
		}
		if len(borderedContent) > contentHeight {
			borderedContent = borderedContent[:contentHeight]
		}

		bottomBorder := borderStyle.Render("╰" + strings.Repeat("─", thisColWidth-2) + "╯")

		return topBorder + "\n" + strings.Join(borderedContent, "\n") + "\n" + bottomBorder
	}

	// Render visible columns
	var colViews []string
	for i := offset; i < offset+visibleCols && i < totalColumns; i++ {
		col := renderColumn(
			boardColumns[i],
			columnColors[i],
			m.boardColumn == i,
			m.boardRow,
			columnHeaders[i],
			colWidth,
		)
		colViews = append(colViews, col)
	}

	boardContent := lipgloss.JoinHorizontal(lipgloss.Top, colViews...)

	// Build title line with scroll position indicator
	titleText := "BOARD"
	if visibleCols < totalColumns {
		titleText = fmt.Sprintf("BOARD [%d-%d of %d]", offset+1, offset+visibleCols, totalColumns)
	}

	// Add scroll arrows to title
	var titleParts []string
	if offset > 0 {
		titleParts = append(titleParts, ui.HelpDescStyle.Render("◀ "))
	}
	titleParts = append(titleParts, ui.TitleStyle.Render(titleText))
	if offset+visibleCols < totalColumns {
		titleParts = append(titleParts, ui.HelpDescStyle.Render(" ▶"))
	}
	titleLine := strings.Join(titleParts, "") + "\n"

	b.WriteString(titleLine)
	b.WriteString(boardContent)
	b.WriteString("\n")

	b.WriteString(ui.HelpBarStyle.Render("h/l:column  j/k:select  enter:detail  b:list view  ?:help  q:quit"))

	return b.String()
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
