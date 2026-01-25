package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/josebiro/lazybeads/internal/ui"
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
	case ViewEditTitle, ViewEditStatus, ViewEditPriority, ViewEditType, ViewFilter, ViewAddBlocker, ViewRemoveBlocker:
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
  d           Edit description ($EDITOR)
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
			{"j/k", "nav"},
			{"h/l", "panel"},
			{"/", "search"},
			{"o/O/r", "filter"},
			{"S", "sort"},
			{"b", "board"},
			{"enter", "detail"},
			{"e/s/p/t/d", "edit"},
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

	if t.Assignee != "" {
		b.WriteString(ui.DetailLabelStyle.Render("Assignee:"))
		b.WriteString(ui.DetailValueStyle.Render(t.Assignee))
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

	// Board view uses full width - no detail panel to keep layout simple
	// Press Enter to see full task details
	boardWidth := m.width

	// Get tasks for each column
	var openTasks, inProgressTasks, closedTasks []string
	for _, t := range m.tasks {
		// Format: priority + ID + title (truncated)
		priority := ui.PriorityStyle(t.Priority).Render(t.PriorityString())
		id := ui.HelpDescStyle.Render(t.ID)
		title := t.Title
		maxTitleLen := 25 // More space for titles since no detail panel
		if len(title) > maxTitleLen {
			title = title[:maxTitleLen-3] + "..."
		}
		line := fmt.Sprintf("%s %s %s", priority, id, title)

		switch t.Status {
		case "open":
			openTasks = append(openTasks, line)
		case "in_progress":
			inProgressTasks = append(inProgressTasks, line)
		case "closed":
			closedTasks = append(closedTasks, line)
		}
	}

	// Calculate column width (3 columns with spacing)
	colWidth := (boardWidth - 8) / 3
	if colWidth < 15 {
		colWidth = 15
	}

	// Column height (screen height minus header and footer)
	colHeight := m.height - 6
	if colHeight < 5 {
		colHeight = 5
	}

	// Build column headers with counts
	openHeader := fmt.Sprintf("Open (%d)", len(openTasks))
	inProgressHeader := fmt.Sprintf("In Prog (%d)", len(inProgressTasks))
	closedHeader := fmt.Sprintf("Closed (%d)", len(closedTasks))

	// Style for focused vs unfocused columns
	focusedHeaderStyle := ui.TitleStyle.Copy().Width(colWidth).Align(lipgloss.Center)
	unfocusedHeaderStyle := ui.HelpDescStyle.Copy().Width(colWidth).Align(lipgloss.Center)

	// Render headers
	var headers []string
	if m.boardColumn == 0 {
		headers = append(headers, focusedHeaderStyle.Render(openHeader))
	} else {
		headers = append(headers, unfocusedHeaderStyle.Render(openHeader))
	}
	if m.boardColumn == 1 {
		headers = append(headers, focusedHeaderStyle.Render(inProgressHeader))
	} else {
		headers = append(headers, unfocusedHeaderStyle.Render(inProgressHeader))
	}
	if m.boardColumn == 2 {
		headers = append(headers, focusedHeaderStyle.Render(closedHeader))
	} else {
		headers = append(headers, unfocusedHeaderStyle.Render(closedHeader))
	}

	// Render column contents with scrolling support
	// We manually draw borders to ensure exact height control (lipgloss Height doesn't clip)
	renderColumn := func(tasks []string, focused bool, selectedRow int) string {
		// Content area height (subtract 2 for top/bottom border)
		contentHeight := colHeight - 2
		if contentHeight < 1 {
			contentHeight = 1
		}

		// Determine if we need scroll indicators
		needsScrollIndicators := len(tasks) > contentHeight

		// Content rows = available rows minus space for scroll indicators
		contentRows := contentHeight
		if needsScrollIndicators {
			contentRows = contentHeight - 2 // Reserve 2 rows for ↑/↓ indicators
			if contentRows < 1 {
				contentRows = 1
			}
		}

		// Calculate scroll offset
		scrollOffset := 0
		if len(tasks) > contentRows {
			if focused {
				// For focused column, center the selected row
				scrollOffset = selectedRow - contentRows/2
			} else {
				// For unfocused columns, just show from top
				scrollOffset = 0
			}
			if scrollOffset < 0 {
				scrollOffset = 0
			}
			maxOffset := len(tasks) - contentRows
			if scrollOffset > maxOffset {
				scrollOffset = maxOffset
			}
		}

		var lines []string

		// Show scroll indicator at top if scrolled
		if needsScrollIndicators && scrollOffset > 0 {
			lines = append(lines, ui.HelpDescStyle.Render(fmt.Sprintf("  ↑ %d more", scrollOffset)))
		} else if needsScrollIndicators {
			lines = append(lines, "") // Placeholder for alignment
		}

		// Calculate visible range
		endIdx := scrollOffset + contentRows
		if endIdx > len(tasks) {
			endIdx = len(tasks)
		}

		// Render visible tasks
		for i := scrollOffset; i < endIdx; i++ {
			task := tasks[i]
			if focused && i == selectedRow {
				lines = append(lines, ui.SelectedTaskStyle.Render("> "+task))
			} else {
				lines = append(lines, "  "+task)
			}
		}

		// Show scroll indicator at bottom if more items
		if needsScrollIndicators && endIdx < len(tasks) {
			remaining := len(tasks) - endIdx
			lines = append(lines, ui.HelpDescStyle.Render(fmt.Sprintf("  ↓ %d more", remaining)))
		} else if needsScrollIndicators {
			lines = append(lines, "") // Placeholder for alignment
		}

		// Handle empty column
		if len(tasks) == 0 {
			lines = []string{ui.HelpDescStyle.Render("  (empty)")}
		}

		// Pad to exact content height
		for len(lines) < contentHeight {
			lines = append(lines, "")
		}
		// Truncate if somehow we have too many
		if len(lines) > contentHeight {
			lines = lines[:contentHeight]
		}

		// Build box manually with exact dimensions
		borderColor := lipgloss.Color("240")
		if focused {
			borderColor = lipgloss.Color("63") // Blue for focused
		}

		// Top border
		topBorder := lipgloss.NewStyle().Foreground(borderColor).Render("┌" + strings.Repeat("─", colWidth-2) + "┐")

		// Content lines with side borders
		var contentLines []string
		for _, line := range lines {
			// Pad/truncate line to fit column width (minus borders and padding)
			innerWidth := colWidth - 4 // 2 for borders, 2 for padding
			lineWidth := lipgloss.Width(line)
			if lineWidth > innerWidth {
				// Truncate - need to be careful with ANSI codes
				line = line[:innerWidth]
			}
			padding := innerWidth - lipgloss.Width(line)
			if padding < 0 {
				padding = 0
			}
			paddedLine := line + strings.Repeat(" ", padding)
			contentLines = append(contentLines,
				lipgloss.NewStyle().Foreground(borderColor).Render("│")+" "+paddedLine+" "+lipgloss.NewStyle().Foreground(borderColor).Render("│"))
		}

		// Bottom border
		bottomBorder := lipgloss.NewStyle().Foreground(borderColor).Render("└" + strings.Repeat("─", colWidth-2) + "┘")

		return topBorder + "\n" + strings.Join(contentLines, "\n") + "\n" + bottomBorder
	}

	col0 := renderColumn(openTasks, m.boardColumn == 0, m.boardRow)
	col1 := renderColumn(inProgressTasks, m.boardColumn == 1, m.boardRow)
	col2 := renderColumn(closedTasks, m.boardColumn == 2, m.boardRow)

	boardContent := lipgloss.JoinHorizontal(lipgloss.Top, col0, col1, col2)

	// Build title line
	titleLine := ui.TitleStyle.Render("Board View") + "\n\n"

	// Build header row
	headerRow := lipgloss.JoinHorizontal(lipgloss.Top, headers...) + "\n"

	// Write title, headers, and board content
	b.WriteString(titleLine)
	b.WriteString(headerRow)
	b.WriteString(boardContent)
	b.WriteString("\n")

	// Status bar
	b.WriteString(ui.HelpBarStyle.Render("h/l:column  j/k:select  enter:detail  b:list view  ?:help  q:quit"))

	return b.String()
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
