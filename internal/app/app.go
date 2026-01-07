package app

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"lazybeads/internal/beads"
	"lazybeads/internal/models"
	"lazybeads/internal/ui"
)

const pollInterval = 2 * time.Second

// ViewMode represents the current view
type ViewMode int

const (
	ViewList ViewMode = iota
	ViewDetail
	ViewForm
	ViewHelp
	ViewConfirm
)

// FilterMode represents task filtering
type FilterMode int

const (
	FilterOpen FilterMode = iota
	FilterReady
	FilterClosed
	FilterAll
	filterModeCount // used for cycling
)

// taskItem wraps a Task for the list component
type taskItem struct {
	task models.Task
}

func (t taskItem) Title() string {
	return t.task.Title
}

func (t taskItem) Description() string {
	return t.task.ID
}

func (t taskItem) FilterValue() string {
	return t.task.Title + " " + t.task.ID
}

// taskDelegate is a custom delegate for rendering task items
type taskDelegate struct {
	height   int
	spacing  int
	listWidth int
}

func newTaskDelegate() taskDelegate {
	return taskDelegate{height: 1, spacing: 0}
}

func (d taskDelegate) Height() int                             { return d.height }
func (d taskDelegate) Spacing() int                            { return d.spacing }
func (d taskDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d taskDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	t, ok := item.(taskItem)
	if !ok {
		return
	}

	isSelected := index == m.Index()

	// Build the line content
	icon := t.task.StatusIcon()
	priority := t.task.PriorityString()
	title := t.task.Title

	// Calculate available width for the row
	width := m.Width()
	if width <= 0 {
		width = 80
	}

	if isSelected {
		// Selected: full-width subtle blue highlight, white text
		line := fmt.Sprintf(" %s %s %s", icon, priority, title)
		style := lipgloss.NewStyle().
			Foreground(lipgloss.Color("15")).
			Background(lipgloss.Color("#2a4a6d")).
			Bold(true).
			Width(width)
		fmt.Fprint(w, style.Render(line))
	} else {
		// Normal: colored priority
		iconStyle := ui.StatusStyle(t.task.Status)
		priorityStyle := ui.PriorityStyle(t.task.Priority)

		line := fmt.Sprintf(" %s %s %s",
			iconStyle.Render(icon),
			priorityStyle.Render(priority),
			title)
		fmt.Fprint(w, line)
	}
}

// Model is the main application state
type Model struct {
	client *beads.Client
	keys   ui.KeyMap
	help   help.Model

	// Data
	tasks    []models.Task
	selected *models.Task

	// UI state
	mode       ViewMode
	filterMode FilterMode
	width      int
	height     int
	err        error

	// Components
	list       list.Model
	detail     viewport.Model
	filterText textinput.Model

	// Form state
	formTitle    textinput.Model
	formDesc     textinput.Model
	formPriority int
	formType     string
	formFocus    int
	editing      bool
	editingID    string

	// Confirmation
	confirmMsg    string
	confirmAction func() tea.Cmd
}

// New creates a new application model
func New() Model {
	// Initialize help
	h := help.New()
	h.ShowAll = false

	// Initialize list - compact single-line items with subtle row highlight
	delegate := newTaskDelegate()

	l := list.New([]list.Item{}, delegate, 0, 0)
	l.Title = "Tasks"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(false)
	l.Styles.Title = ui.PanelTitleStyle
	l.SetStatusBarItemName("task", "tasks")

	// Initialize detail viewport
	vp := viewport.New(0, 0)

	// Initialize filter input
	filter := textinput.New()
	filter.Placeholder = "Search tasks..."
	filter.CharLimit = 100

	// Initialize form inputs
	formTitle := textinput.New()
	formTitle.Placeholder = "Task title"
	formTitle.CharLimit = 200

	formDesc := textinput.New()
	formDesc.Placeholder = "Description (optional)"
	formDesc.CharLimit = 1000

	return Model{
		client:       beads.NewClient(),
		keys:         ui.DefaultKeyMap(),
		help:         h,
		mode:         ViewList,
		filterMode:   FilterOpen,
		list:         l,
		detail:       vp,
		filterText:   filter,
		formTitle:    formTitle,
		formDesc:     formDesc,
		formPriority: 2,
		formType:     "task",
	}
}

// tasksLoadedMsg is sent when tasks are loaded
type tasksLoadedMsg struct {
	tasks []models.Task
	err   error
}

// taskCreatedMsg is sent when a task is created
type taskCreatedMsg struct {
	task *models.Task
	err  error
}

// taskUpdatedMsg is sent when a task is updated
type taskUpdatedMsg struct {
	err error
}

// taskClosedMsg is sent when a task is closed
type taskClosedMsg struct {
	err error
}

// taskDeletedMsg is sent when a task is deleted
type taskDeletedMsg struct {
	err error
}

// tickMsg triggers periodic refresh
type tickMsg time.Time

// pollTick creates a command that ticks for polling
func pollTick() tea.Cmd {
	return tea.Tick(pollInterval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// loadTasks creates a command to load tasks
func (m Model) loadTasks() tea.Cmd {
	return func() tea.Msg {
		var tasks []models.Task
		var err error

		switch m.filterMode {
		case FilterReady:
			tasks, err = m.client.Ready()
		case FilterClosed:
			tasks, err = m.client.List("--status=closed")
		case FilterAll:
			tasks, err = m.client.List("--all")
		default:
			tasks, err = m.client.ListOpen()
		}

		return tasksLoadedMsg{tasks: tasks, err: err}
	}
}

// Init initializes the application
func (m Model) Init() tea.Cmd {
	return tea.Batch(m.loadTasks(), pollTick())
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateSizes()

	case tea.KeyMsg:
		// Global key handling - intercept before components
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "q":
			// Only quit from list view
			if m.mode == ViewList {
				return m, tea.Quit
			}
		case "esc":
			// Escape goes back to list, never quits
			if m.mode != ViewList {
				m.mode = ViewList
				return m, nil
			}
			// In list mode, do nothing
			return m, nil
		}

		prevMode := m.mode
		cmd := m.handleKeyPress(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		// If mode changed, don't pass key to child components
		if m.mode != prevMode {
			return m, tea.Batch(cmds...)
		}

	case tasksLoadedMsg:
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.tasks = msg.tasks
			m.updateList()
		}

	case taskCreatedMsg:
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.mode = ViewList
			cmds = append(cmds, m.loadTasks())
		}

	case taskUpdatedMsg:
		if msg.err != nil {
			m.err = msg.err
		}
		cmds = append(cmds, m.loadTasks())

	case taskClosedMsg:
		if msg.err != nil {
			m.err = msg.err
		}
		m.mode = ViewList
		cmds = append(cmds, m.loadTasks())

	case taskDeletedMsg:
		if msg.err != nil {
			m.err = msg.err
		}
		m.mode = ViewList
		cmds = append(cmds, m.loadTasks())

	case tickMsg:
		// Periodic refresh - reload tasks and schedule next tick
		cmds = append(cmds, m.loadTasks(), pollTick())
	}

	// Update child components
	switch m.mode {
	case ViewList:
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		cmds = append(cmds, cmd)
		// Always sync selected item with detail panel
		if item, ok := m.list.SelectedItem().(taskItem); ok {
			m.selected = &item.task
		}
	case ViewDetail:
		var cmd tea.Cmd
		m.detail, cmd = m.detail.Update(msg)
		cmds = append(cmds, cmd)
	case ViewForm:
		cmds = append(cmds, m.updateForm(msg))
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) handleKeyPress(msg tea.KeyMsg) tea.Cmd {
	switch m.mode {
	case ViewList:
		return m.handleListKeys(msg)
	case ViewDetail:
		return m.handleDetailKeys(msg)
	case ViewForm:
		return m.handleFormKeys(msg)
	case ViewHelp:
		return m.handleHelpKeys(msg)
	case ViewConfirm:
		return m.handleConfirmKeys(msg)
	}
	return nil
}

func (m *Model) handleListKeys(msg tea.KeyMsg) tea.Cmd {
	switch {
	case key.Matches(msg, m.keys.Select):
		if item, ok := m.list.SelectedItem().(taskItem); ok {
			m.selected = &item.task
			m.updateDetailContent()
			m.mode = ViewDetail
		}

	case key.Matches(msg, m.keys.Add):
		m.resetForm()
		m.editing = false
		m.mode = ViewForm
		m.formTitle.Focus()

	case key.Matches(msg, m.keys.Edit):
		if item, ok := m.list.SelectedItem().(taskItem); ok {
			m.editing = true
			m.editingID = item.task.ID
			m.formTitle.SetValue(item.task.Title)
			m.formDesc.SetValue(item.task.Description)
			m.formPriority = item.task.Priority
			m.formType = item.task.Type
			m.mode = ViewForm
			m.formTitle.Focus()
		}

	case key.Matches(msg, m.keys.Delete):
		if item, ok := m.list.SelectedItem().(taskItem); ok {
			m.confirmMsg = fmt.Sprintf("Delete task %s?", item.task.ID)
			taskID := item.task.ID
			m.confirmAction = func() tea.Cmd {
				return func() tea.Msg {
					err := m.client.Delete(taskID)
					return taskDeletedMsg{err: err}
				}
			}
			m.mode = ViewConfirm
		}

	case key.Matches(msg, m.keys.Close):
		if item, ok := m.list.SelectedItem().(taskItem); ok {
			m.confirmMsg = fmt.Sprintf("Close task %s?", item.task.ID)
			taskID := item.task.ID
			m.confirmAction = func() tea.Cmd {
				return func() tea.Msg {
					err := m.client.Close(taskID, "")
					return taskClosedMsg{err: err}
				}
			}
			m.mode = ViewConfirm
		}

	case key.Matches(msg, m.keys.PrevView):
		m.filterMode = (m.filterMode - 1 + filterModeCount) % filterModeCount
		return m.loadTasks()

	case key.Matches(msg, m.keys.NextView):
		m.filterMode = (m.filterMode + 1) % filterModeCount
		return m.loadTasks()

	case key.Matches(msg, m.keys.Refresh):
		return m.loadTasks()

	case key.Matches(msg, m.keys.Help):
		m.mode = ViewHelp
	}

	return nil
}

func (m *Model) handleDetailKeys(msg tea.KeyMsg) tea.Cmd {
	switch {
	case key.Matches(msg, m.keys.Cancel), key.Matches(msg, m.keys.Select):
		m.mode = ViewList
	case key.Matches(msg, m.keys.Help):
		m.mode = ViewHelp
	}
	return nil
}

func (m *Model) handleFormKeys(msg tea.KeyMsg) tea.Cmd {
	switch {
	case key.Matches(msg, m.keys.Cancel):
		m.mode = ViewList
		return nil

	case key.Matches(msg, m.keys.Submit):
		return m.submitForm()

	case msg.String() == "enter":
		// Enter submits from any field
		return m.submitForm()

	case key.Matches(msg, m.keys.Tab):
		m.formFocus = (m.formFocus + 1) % 4
		m.updateFormFocus()
	}

	return nil
}

func (m *Model) handleHelpKeys(msg tea.KeyMsg) tea.Cmd {
	switch {
	case key.Matches(msg, m.keys.Cancel), key.Matches(msg, m.keys.Help):
		m.mode = ViewList
	}
	return nil
}

func (m *Model) handleConfirmKeys(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "y", "Y":
		if m.confirmAction != nil {
			return m.confirmAction()
		}
		m.mode = ViewList
	case "n", "N", "esc":
		m.mode = ViewList
	}
	return nil
}

func (m *Model) updateForm(msg tea.Msg) tea.Cmd {
	var cmds []tea.Cmd

	switch m.formFocus {
	case 0:
		var cmd tea.Cmd
		m.formTitle, cmd = m.formTitle.Update(msg)
		cmds = append(cmds, cmd)
	case 1:
		var cmd tea.Cmd
		m.formDesc, cmd = m.formDesc.Update(msg)
		cmds = append(cmds, cmd)
	case 2:
		// Priority selection
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			switch keyMsg.String() {
			case "left", "h":
				if m.formPriority > 0 {
					m.formPriority--
				}
			case "right", "l":
				if m.formPriority < 4 {
					m.formPriority++
				}
			}
		}
	case 3:
		// Type selection
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			types := []string{"task", "bug", "feature", "epic", "chore"}
			idx := 0
			for i, t := range types {
				if t == m.formType {
					idx = i
					break
				}
			}
			switch keyMsg.String() {
			case "left", "h":
				idx = (idx - 1 + len(types)) % len(types)
			case "right", "l":
				idx = (idx + 1) % len(types)
			}
			m.formType = types[idx]
		}
	}

	return tea.Batch(cmds...)
}

func (m *Model) resetForm() {
	m.formTitle.SetValue("")
	m.formDesc.SetValue("")
	m.formPriority = 2
	m.formType = "task"
	m.formFocus = 0
	m.updateFormFocus()
}

func (m *Model) updateFormFocus() {
	m.formTitle.Blur()
	m.formDesc.Blur()
	switch m.formFocus {
	case 0:
		m.formTitle.Focus()
	case 1:
		m.formDesc.Focus()
	}
}

func (m *Model) submitForm() tea.Cmd {
	title := strings.TrimSpace(m.formTitle.Value())
	if title == "" {
		m.err = fmt.Errorf("title is required")
		return nil
	}

	if m.editing {
		return func() tea.Msg {
			err := m.client.Update(m.editingID, beads.UpdateOptions{
				Title:    title,
				Priority: &m.formPriority,
			})
			return taskUpdatedMsg{err: err}
		}
	}

	return func() tea.Msg {
		task, err := m.client.Create(beads.CreateOptions{
			Title:       title,
			Description: m.formDesc.Value(),
			Type:        m.formType,
			Priority:    m.formPriority,
		})
		return taskCreatedMsg{task: task, err: err}
	}
}

func (m *Model) updateSizes() {
	// Reserve space for title and help
	contentHeight := m.height - 4
	if contentHeight < 0 {
		contentHeight = 0
	}

	// Wide mode: side-by-side panels
	if m.width >= 80 {
		listWidth := m.width/2 - 2
		m.list.SetSize(listWidth, contentHeight)
		m.detail.Width = m.width/2 - 4
		m.detail.Height = contentHeight - 2
	} else {
		// Narrow mode: full width list
		m.list.SetSize(m.width-2, contentHeight)
		m.detail.Width = m.width - 4
		m.detail.Height = contentHeight - 2
	}
}

func (m *Model) updateList() {
	items := make([]list.Item, len(m.tasks))
	for i, t := range m.tasks {
		items[i] = taskItem{task: t}
	}
	m.list.SetItems(items)

	// Update title with filter mode
	switch m.filterMode {
	case FilterReady:
		m.list.Title = fmt.Sprintf("Ready Tasks (%d)", len(m.tasks))
	case FilterClosed:
		m.list.Title = fmt.Sprintf("Closed Tasks (%d)", len(m.tasks))
	case FilterAll:
		m.list.Title = fmt.Sprintf("All Tasks (%d)", len(m.tasks))
	default:
		m.list.Title = fmt.Sprintf("Open Tasks (%d)", len(m.tasks))
	}
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
	default:
		return m.viewMain()
	}
}

func (m Model) viewMain() string {
	var b strings.Builder

	// Title bar
	title := ui.TitleStyle.Render("lazybeads")
	filterInfo := m.filterModeString()
	titleLine := lipgloss.JoinHorizontal(
		lipgloss.Left,
		title,
		strings.Repeat(" ", max(0, m.width-lipgloss.Width(title)-lipgloss.Width(filterInfo)-2)),
		ui.HelpDescStyle.Render(filterInfo),
	)
	b.WriteString(titleLine + "\n")

	// Content area
	contentHeight := m.height - 4

	if m.width >= 80 {
		// Wide mode: two panels - highlight focused panel
		listStyle := ui.PanelStyle
		detailStyle := ui.PanelStyle
		if m.mode == ViewList {
			listStyle = ui.FocusedPanelStyle
		} else if m.mode == ViewDetail {
			detailStyle = ui.FocusedPanelStyle
		}

		listPanel := listStyle.
			Width(m.width/2 - 2).
			Height(contentHeight).
			Render(m.list.View())

		detailContent := ""
		if m.selected != nil {
			m.updateDetailContent()
			detailContent = m.detail.View()
		} else {
			detailContent = ui.HelpDescStyle.Render("No tasks")
		}

		detailPanel := detailStyle.
			Width(m.width/2 - 2).
			Height(contentHeight).
			Render(detailContent)

		b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, listPanel, detailPanel))
	} else {
		// Narrow mode: list only
		listPanel := ui.FocusedPanelStyle.
			Width(m.width - 2).
			Height(contentHeight).
			Render(m.list.View())
		b.WriteString(listPanel)
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
	b.WriteString(ui.HelpBarStyle.Render("tab: next field  enter: submit  esc: cancel"))

	return b.String()
}

func (m Model) viewHelp() string {
	var b strings.Builder

	b.WriteString(ui.TitleStyle.Render("Keyboard Shortcuts") + "\n\n")

	helpContent := `
Navigation
  j/k, ↑/↓    Move up/down
  g/G         Jump to top/bottom
  ^u/^d       Page up/down

Views (h/l to cycle)
  open        Show open tasks
  ready       Show ready tasks (no blockers)
  closed      Show closed tasks
  all         Show all tasks

Actions
  enter       View task details
  a           Add new task
  e           Edit selected task
  c           Close selected task
  d           Delete selected task
  R           Refresh list
  /           Search/filter

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

func (m Model) filterModeString() string {
	switch m.filterMode {
	case FilterReady:
		return "[ready]"
	case FilterClosed:
		return "[closed]"
	case FilterAll:
		return "[all]"
	default:
		return "[open]"
	}
}

func (m Model) renderHelpBar() string {
	keys := []struct {
		key  string
		desc string
	}{
		{"j/k", "nav"},
		{"h/l", "view"},
		{"enter", "focus"},
		{"a", "add"},
		{"e", "edit"},
		{"c", "close"},
		{"d", "delete"},
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

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
