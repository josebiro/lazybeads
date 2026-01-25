package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/josebiro/lazybeads/internal/models"
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
		if m.width < 80 || m.previousMode == ViewBoard {
			// Narrow mode OR coming from board: full screen detail overlay
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

	// Calculate available height:
	// - Title line + blank line = 2
	// - Content area = height - 4 (title, blank, content, help bar)
	// - Help bar = 1
	contentHeight := m.height - 4
	if contentHeight < 5 {
		contentHeight = 5
	}

	// Resize viewport for overlay mode
	m.detail.Width = m.width - 6  // Account for border padding
	m.detail.Height = contentHeight - 2  // Account for OverlayStyle border

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

	// Board view with 4 columns: Blocked, Ready, In Progress, Done
	// Each column has color-coded borders matching the bv tool style

	// Column border colors
	blockedColor := lipgloss.Color("1")    // Red for blocked
	readyColor := lipgloss.Color("2")      // Green for ready
	inProgressColor := lipgloss.Color("3") // Yellow for in progress
	doneColor := lipgloss.Color("6")       // Cyan for done

	// Collect tasks into 4 categories
	type boardTask struct {
		task     models.Task
		priority string
		id       string
		title    string
	}

	var blockedTasks, readyTasks, inProgressTasks, doneTasks []boardTask

	for _, t := range m.tasks {
		bt := boardTask{
			task:     t,
			priority: t.PriorityString(),
			id:       t.ID,
			title:    t.Title,
		}

		switch t.Status {
		case "open":
			// Open tasks: split into Blocked vs Ready
			if t.IsBlocked() {
				blockedTasks = append(blockedTasks, bt)
			} else {
				readyTasks = append(readyTasks, bt)
			}
		case "in_progress":
			inProgressTasks = append(inProgressTasks, bt)
		case "closed":
			doneTasks = append(doneTasks, bt)
		}
	}

	// Calculate column width (4 columns)
	colWidth := (m.width - 4) / 4 // -4 for minimal spacing between columns
	if colWidth < 20 {
		colWidth = 20
	}

	// Column height (screen height minus title and footer)
	colHeight := m.height - 4
	if colHeight < 8 {
		colHeight = 8
	}

	// Card height (each task card is 3 lines: top border, content, bottom border)
	cardHeight := 3
	// How many cards fit in the column (minus 2 for column borders)
	cardsPerColumn := (colHeight - 2) / cardHeight
	if cardsPerColumn < 1 {
		cardsPerColumn = 1
	}

	// Render a single task card with colored border
	renderCard := func(bt boardTask, borderColor lipgloss.Color, selected bool, cardWidth int) string {
		innerWidth := cardWidth - 4 // -4 for borders and padding

		// Build card content: "P# id title"
		priority := ui.PriorityStyle(bt.task.Priority).Render(bt.priority)
		idStyled := ui.HelpDescStyle.Render(bt.id)

		// Calculate available width for title
		prefixWidth := lipgloss.Width(bt.priority) + 1 + lipgloss.Width(bt.id) + 1
		titleWidth := innerWidth - prefixWidth
		if titleWidth < 5 {
			titleWidth = 5
		}

		title := bt.title
		if lipgloss.Width(title) > titleWidth {
			for lipgloss.Width(title+"…") > titleWidth && len(title) > 0 {
				title = title[:len(title)-1]
			}
			title = title + "…"
		}

		content := fmt.Sprintf("%s %s %s", priority, idStyled, title)

		// Pad content to fill inner width
		contentWidth := lipgloss.Width(content)
		if contentWidth < innerWidth {
			content = content + strings.Repeat(" ", innerWidth-contentWidth)
		}

		// Build card borders
		borderStyle := lipgloss.NewStyle().Foreground(borderColor)
		if selected {
			borderStyle = borderStyle.Bold(true)
		}

		topBorder := borderStyle.Render("╭" + strings.Repeat("─", cardWidth-2) + "╮")
		bottomBorder := borderStyle.Render("╰" + strings.Repeat("─", cardWidth-2) + "╯")

		// Content line with selection indicator
		var middleLine string
		if selected {
			// Highlight selected card
			highlightStyle := lipgloss.NewStyle().
				Background(lipgloss.Color("236")).
				Foreground(lipgloss.Color("15"))
			middleLine = borderStyle.Render("│") + " " + highlightStyle.Render(content) + " " + borderStyle.Render("│")
		} else {
			middleLine = borderStyle.Render("│") + " " + content + " " + borderStyle.Render("│")
		}

		return topBorder + "\n" + middleLine + "\n" + bottomBorder
	}

	// Render a column with tasks as cards
	renderColumn := func(tasks []boardTask, borderColor lipgloss.Color, focused bool, selectedRow int, header string) string {
		headerColor := borderColor
		if !focused {
			headerColor = ui.ColorMuted
		}

		// Header with count
		headerText := fmt.Sprintf(" %s (%d) ", header, len(tasks))
		headerStyle := lipgloss.NewStyle().
			Foreground(headerColor).
			Bold(focused)

		// Calculate scroll offset for cards
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

		// Build column content
		var cards []string

		// Scroll indicator at top
		if scrollOffset > 0 {
			cards = append(cards, ui.HelpDescStyle.Render(fmt.Sprintf(" ↑ %d more", scrollOffset)))
		}

		// Render visible cards
		endIdx := scrollOffset + cardsPerColumn
		if endIdx > len(tasks) {
			endIdx = len(tasks)
		}

		for i := scrollOffset; i < endIdx; i++ {
			isSelected := focused && i == selectedRow
			card := renderCard(tasks[i], borderColor, isSelected, colWidth-2)
			cards = append(cards, card)
		}

		// Scroll indicator at bottom
		if endIdx < len(tasks) {
			remaining := len(tasks) - endIdx
			cards = append(cards, ui.HelpDescStyle.Render(fmt.Sprintf(" ↓ %d more", remaining)))
		}

		// Empty column placeholder
		if len(tasks) == 0 {
			emptyStyle := lipgloss.NewStyle().Foreground(ui.ColorMuted).Italic(true)
			cards = append(cards, emptyStyle.Render(" (empty)"))
		}

		// Build column border
		borderStyle := lipgloss.NewStyle().Foreground(borderColor)
		if !focused {
			borderStyle = lipgloss.NewStyle().Foreground(ui.ColorBorder)
		}

		// Column header line embedded in top border
		headerWidth := lipgloss.Width(headerText)
		remainingWidth := colWidth - headerWidth - 4
		if remainingWidth < 0 {
			remainingWidth = 0
		}

		topBorder := borderStyle.Render("╭─") + headerStyle.Render(headerText) + borderStyle.Render(strings.Repeat("─", remainingWidth)+"─╮")

		// Join cards into column content
		columnContent := strings.Join(cards, "\n")

		// Wrap in column borders - calculate actual content height
		contentLines := strings.Split(columnContent, "\n")
		var borderedContent []string
		for _, line := range contentLines {
			lineWidth := lipgloss.Width(line)
			innerWidth := colWidth - 4
			if lineWidth < innerWidth {
				line = line + strings.Repeat(" ", innerWidth-lineWidth)
			}
			borderedContent = append(borderedContent, borderStyle.Render("│")+" "+line+" "+borderStyle.Render("│"))
		}

		// Pad to fill column height
		contentHeight := colHeight - 2 // -2 for top/bottom borders
		for len(borderedContent) < contentHeight {
			emptyLine := strings.Repeat(" ", colWidth-4)
			borderedContent = append(borderedContent, borderStyle.Render("│")+" "+emptyLine+" "+borderStyle.Render("│"))
		}
		// Truncate if too many
		if len(borderedContent) > contentHeight {
			borderedContent = borderedContent[:contentHeight]
		}

		bottomBorder := borderStyle.Render("╰" + strings.Repeat("─", colWidth-2) + "╯")

		return topBorder + "\n" + strings.Join(borderedContent, "\n") + "\n" + bottomBorder
	}

	// Render all 4 columns
	col0 := renderColumn(blockedTasks, blockedColor, m.boardColumn == 0, m.boardRow, "BLOCKED")
	col1 := renderColumn(readyTasks, readyColor, m.boardColumn == 1, m.boardRow, "READY")
	col2 := renderColumn(inProgressTasks, inProgressColor, m.boardColumn == 2, m.boardRow, "IN PROGRESS")
	col3 := renderColumn(doneTasks, doneColor, m.boardColumn == 3, m.boardRow, "DONE")

	boardContent := lipgloss.JoinHorizontal(lipgloss.Top, col0, col1, col2, col3)

	// Build title line
	titleLine := ui.TitleStyle.Render("BOARD [by: Status]") + "\n"

	// Write title and board content
	b.WriteString(titleLine)
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
