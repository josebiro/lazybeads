package app

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"lazybeads/internal/models"
	"lazybeads/internal/ui"
)

// PanelModel represents a single panel showing a filtered list of tasks
type PanelModel struct {
	title    string
	tasks    []models.Task
	selected int
	focused  bool
	width    int
	height   int
	list     list.Model
}

// panelDelegate is a custom delegate for rendering task items in panels
type panelDelegate struct {
	listWidth int
}

func newPanelDelegate() panelDelegate {
	return panelDelegate{}
}

func (d panelDelegate) Height() int                             { return 1 }
func (d panelDelegate) Spacing() int                            { return 0 }
func (d panelDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d panelDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	t, ok := item.(taskItem)
	if !ok {
		return
	}

	isSelected := index == m.Index()

	icon := t.task.StatusIcon()
	priority := t.task.PriorityString()
	title := t.task.Title

	width := m.Width()
	if width <= 0 {
		width = 40
	}

	// Calculate available width for title (account for icon, priority, spaces)
	prefixWidth := lipgloss.Width(fmt.Sprintf(" %s %s ", icon, priority))
	maxTitleWidth := width - prefixWidth
	if maxTitleWidth < 5 {
		maxTitleWidth = 5
	}

	// Truncate title if too long
	if lipgloss.Width(title) > maxTitleWidth {
		// Truncate with ellipsis
		for lipgloss.Width(title+"...") > maxTitleWidth && len(title) > 0 {
			title = title[:len(title)-1]
		}
		title = title + "..."
	}

	if isSelected {
		line := fmt.Sprintf(" %s %s %s", icon, priority, title)
		style := lipgloss.NewStyle().
			Foreground(lipgloss.Color("15")).
			Background(lipgloss.Color("#2a4a6d")).
			Bold(true).
			Width(width)
		fmt.Fprint(w, style.Render(line))
	} else {
		iconStyle := ui.StatusStyle(t.task.Status)
		priorityStyle := ui.PriorityStyle(t.task.Priority)

		line := fmt.Sprintf(" %s %s %s",
			iconStyle.Render(icon),
			priorityStyle.Render(priority),
			title)
		// Ensure line doesn't exceed width
		style := lipgloss.NewStyle().Width(width).MaxWidth(width)
		fmt.Fprint(w, style.Render(line))
	}
}

// NewPanel creates a new panel with the given title
func NewPanel(title string) PanelModel {
	delegate := newPanelDelegate()
	l := list.New([]list.Item{}, delegate, 0, 0)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)
	l.SetShowTitle(false)
	l.SetShowPagination(false)

	return PanelModel{
		title:    title,
		tasks:    []models.Task{},
		selected: 0,
		focused:  false,
		list:     l,
	}
}

// SetTasks updates the panel's task list
func (p *PanelModel) SetTasks(tasks []models.Task) {
	p.tasks = tasks
	items := make([]list.Item, len(tasks))
	for i, t := range tasks {
		items[i] = taskItem{task: t}
	}
	p.list.SetItems(items)
}

// SetSize updates the panel dimensions
func (p *PanelModel) SetSize(width, height int) {
	p.width = width
	p.height = height
	// Content area: panel width minus borders and padding (│ + space on each side = 4)
	contentWidth := width - 4
	if contentWidth < 10 {
		contentWidth = 10
	}
	// Content height: panel height minus top and bottom borders (2 lines)
	contentHeight := height - 2
	if contentHeight < 1 {
		contentHeight = 1
	}
	p.list.SetSize(contentWidth, contentHeight)
}

// SetFocus sets whether this panel is focused
func (p *PanelModel) SetFocus(focused bool) {
	p.focused = focused
}

// IsFocused returns whether this panel is focused
func (p PanelModel) IsFocused() bool {
	return p.focused
}

// SelectedTask returns the currently selected task, if any
func (p PanelModel) SelectedTask() *models.Task {
	if len(p.tasks) == 0 {
		return nil
	}
	idx := p.list.Index()
	if idx >= 0 && idx < len(p.tasks) {
		return &p.tasks[idx]
	}
	return nil
}

// TaskCount returns the number of tasks in this panel
func (p PanelModel) TaskCount() int {
	return len(p.tasks)
}

// Update handles messages for the panel
func (p PanelModel) Update(msg tea.Msg) (PanelModel, tea.Cmd) {
	if !p.focused {
		return p, nil
	}

	// Don't pass key messages to list - we handle navigation in HandleKey
	// This prevents double-processing of j/k which causes cursor to skip items
	if _, isKey := msg.(tea.KeyMsg); isKey {
		return p, nil
	}

	var cmd tea.Cmd
	p.list, cmd = p.list.Update(msg)
	return p, cmd
}

// HandleKey handles key navigation within the panel
func (p *PanelModel) HandleKey(msg tea.KeyMsg, keys ui.KeyMap) bool {
	if !p.focused {
		return false
	}

	switch {
	case key.Matches(msg, keys.Up):
		p.list.CursorUp()
		return true
	case key.Matches(msg, keys.Down):
		p.list.CursorDown()
		return true
	case key.Matches(msg, keys.Top):
		p.list.Select(0)
		return true
	case key.Matches(msg, keys.Bottom):
		if len(p.tasks) > 0 {
			p.list.Select(len(p.tasks) - 1)
		}
		return true
	case key.Matches(msg, keys.PageUp):
		for i := 0; i < 10; i++ {
			p.list.CursorUp()
		}
		return true
	case key.Matches(msg, keys.PageDown):
		for i := 0; i < 10; i++ {
			p.list.CursorDown()
		}
		return true
	}
	return false
}

// View renders the panel with title embedded in the top border
func (p PanelModel) View() string {
	// Use the full allocated width/height
	width := p.width
	height := p.height
	if width < 10 {
		width = 10
	}
	if height < 3 {
		height = 3
	}

	// Choose colors based on focus
	var borderColor, titleColor lipgloss.Color
	if p.focused {
		borderColor = ui.ColorPrimary
		titleColor = ui.ColorPrimary
	} else {
		borderColor = ui.ColorBorder
		titleColor = ui.ColorMuted
	}

	borderStyle := lipgloss.NewStyle().Foreground(borderColor)
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(titleColor)

	// Build title with count
	titleText := fmt.Sprintf(" %s (%d) ", p.title, len(p.tasks))

	// Truncate title if too long (use lipgloss.Width for proper display width)
	maxTitleLen := width - 6 // Leave room for corners (╭─ and ─╮) and some border
	if lipgloss.Width(titleText) > maxTitleLen {
		// Truncate with ellipsis
		for lipgloss.Width(titleText) > maxTitleLen-3 && len(titleText) > 0 {
			titleText = titleText[:len(titleText)-1]
		}
		titleText = titleText + "..."
	}

	// Build top border: ╭─ Title ─────────╮
	// Use lipgloss.Width for proper character width calculation
	titleDisplayWidth := lipgloss.Width(titleText)
	remainingWidth := width - titleDisplayWidth - 4 // -4 for "╭─" and "─╮"
	if remainingWidth < 0 {
		remainingWidth = 0
	}
	topBorder := borderStyle.Render("╭─") +
		titleStyle.Render(titleText) +
		borderStyle.Render(strings.Repeat("─", remainingWidth)+"─╮")

	// Build content area
	contentWidth := width - 4 // -4 for side borders and padding (│ + space on each side)
	if contentWidth < 1 {
		contentWidth = 1
	}
	contentHeight := height - 2 // -2 for top/bottom borders
	if contentHeight < 1 {
		contentHeight = 1
	}

	// Render the task list or empty message
	var contentLines []string
	if len(p.tasks) == 0 {
		emptyStyle := lipgloss.NewStyle().Foreground(ui.ColorMuted).Italic(true)
		emptyMsg := emptyStyle.Render("(no tasks)")
		// Pad to content width
		padded := emptyMsg + strings.Repeat(" ", max(0, contentWidth-lipgloss.Width(emptyMsg)))
		contentLines = append(contentLines, padded)
	} else {
		// Get the list view and split into lines
		listView := p.list.View()
		contentLines = strings.Split(listView, "\n")
	}

	// Pad or truncate content lines to fit the full height
	var middleRows []string
	for i := 0; i < contentHeight; i++ {
		var line string
		if i < len(contentLines) {
			line = contentLines[i]
		} else {
			line = ""
		}
		// Ensure line fits within content width (with padding)
		lineWidth := lipgloss.Width(line)
		if lineWidth < contentWidth {
			line = line + strings.Repeat(" ", contentWidth-lineWidth)
		} else if lineWidth > contentWidth {
			// Truncate if too long
			line = lipgloss.NewStyle().Width(contentWidth).MaxWidth(contentWidth).Render(line)
		}
		// Add side borders with single space padding
		row := borderStyle.Render("│") + " " + line + " " + borderStyle.Render("│")
		middleRows = append(middleRows, row)
	}

	// Build bottom border: ╰───────────────────╯
	bottomBorder := borderStyle.Render("╰" + strings.Repeat("─", width-2) + "╯")

	// Combine all parts
	var result strings.Builder
	result.WriteString(topBorder + "\n")
	for _, row := range middleRows {
		result.WriteString(row + "\n")
	}
	result.WriteString(bottomBorder)

	return result.String()
}
