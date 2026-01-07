package app

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
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

// PanelFocus represents which panel is focused
type PanelFocus int

const (
	FocusInProgress PanelFocus = iota
	FocusOpen
	FocusClosed
	panelCount
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

// Model is the main application state
type Model struct {
	client *beads.Client
	keys   ui.KeyMap
	help   help.Model

	// Data
	tasks    []models.Task
	selected *models.Task

	// UI state
	mode         ViewMode
	focusedPanel PanelFocus
	width        int
	height       int
	err          error

	// Panels (3 vertically stacked)
	inProgressPanel PanelModel
	openPanel       PanelModel
	closedPanel     PanelModel

	// Components
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

	// Initialize 3 panels
	inProgressPanel := NewPanel("In Progress")
	inProgressPanel.SetFocus(true) // Start with in progress focused
	openPanel := NewPanel("Open")
	closedPanel := NewPanel("Closed")

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
		client:          beads.NewClient(),
		keys:            ui.DefaultKeyMap(),
		help:            h,
		mode:            ViewList,
		focusedPanel:    FocusInProgress,
		inProgressPanel: inProgressPanel,
		openPanel:       openPanel,
		closedPanel:     closedPanel,
		detail:          vp,
		filterText:      filter,
		formTitle:       formTitle,
		formDesc:        formDesc,
		formPriority:    2,
		formType:        "task",
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

// loadTasks creates a command to load all tasks
func (m Model) loadTasks() tea.Cmd {
	return func() tea.Msg {
		// Load all tasks so we can distribute them to the 3 panels
		tasks, err := m.client.List("--all")
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
			m.distributeTasks()
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
		// Update the focused panel
		var cmd tea.Cmd
		switch m.focusedPanel {
		case FocusInProgress:
			m.inProgressPanel, cmd = m.inProgressPanel.Update(msg)
		case FocusOpen:
			m.openPanel, cmd = m.openPanel.Update(msg)
		case FocusClosed:
			m.closedPanel, cmd = m.closedPanel.Update(msg)
		}
		cmds = append(cmds, cmd)
		// Sync selected item with detail panel
		m.selected = m.getSelectedTask()
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
	// First, let the focused panel handle navigation keys
	switch m.focusedPanel {
	case FocusInProgress:
		if m.inProgressPanel.HandleKey(msg, m.keys) {
			m.selected = m.getSelectedTask()
			return nil
		}
	case FocusOpen:
		if m.openPanel.HandleKey(msg, m.keys) {
			m.selected = m.getSelectedTask()
			return nil
		}
	case FocusClosed:
		if m.closedPanel.HandleKey(msg, m.keys) {
			m.selected = m.getSelectedTask()
			return nil
		}
	}

	switch {
	case key.Matches(msg, m.keys.Select):
		if task := m.getSelectedTask(); task != nil {
			m.selected = task
			m.updateDetailContent()
			m.mode = ViewDetail
		}

	case key.Matches(msg, m.keys.Add):
		m.resetForm()
		m.editing = false
		m.mode = ViewForm
		m.formTitle.Focus()

	case key.Matches(msg, m.keys.Edit):
		if task := m.getSelectedTask(); task != nil {
			m.editing = true
			m.editingID = task.ID
			m.formTitle.SetValue(task.Title)
			m.formDesc.SetValue(task.Description)
			m.formPriority = task.Priority
			m.formType = task.Type
			m.mode = ViewForm
			m.formTitle.Focus()
		}

	case key.Matches(msg, m.keys.Delete):
		if task := m.getSelectedTask(); task != nil {
			m.confirmMsg = fmt.Sprintf("Delete task %s?", task.ID)
			taskID := task.ID
			m.confirmAction = func() tea.Cmd {
				return func() tea.Msg {
					err := m.client.Delete(taskID)
					return taskDeletedMsg{err: err}
				}
			}
			m.mode = ViewConfirm
		}

	case key.Matches(msg, m.keys.Close):
		if task := m.getSelectedTask(); task != nil {
			m.confirmMsg = fmt.Sprintf("Close task %s?", task.ID)
			taskID := task.ID
			m.confirmAction = func() tea.Cmd {
				return func() tea.Msg {
					err := m.client.Close(taskID, "")
					return taskClosedMsg{err: err}
				}
			}
			m.mode = ViewConfirm
		}

	case key.Matches(msg, m.keys.PrevView):
		m.cyclePanelFocus(-1)

	case key.Matches(msg, m.keys.NextView):
		m.cyclePanelFocus(1)

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
	// Reserve space for title bar (1 line) and help bar (1 line) + margins
	contentHeight := m.height - 3
	if contentHeight < 0 {
		contentHeight = 0
	}

	// Calculate panel heights - distribute evenly with remainder going to first panels
	panelHeight := contentHeight / 3
	remainder := contentHeight % 3
	if panelHeight < 4 {
		panelHeight = 4
	}

	// Give extra height to first panels to fill the space
	inProgressHeight := panelHeight
	openHeight := panelHeight
	closedHeight := panelHeight
	if remainder >= 1 {
		inProgressHeight++
	}
	if remainder >= 2 {
		openHeight++
	}

	// Wide mode: panels on left, detail on right
	if m.width >= 80 {
		panelWidth := m.width/2 - 1
		m.inProgressPanel.SetSize(panelWidth, inProgressHeight)
		m.openPanel.SetSize(panelWidth, openHeight)
		m.closedPanel.SetSize(panelWidth, closedHeight)
		m.detail.Width = m.width/2 - 4
		m.detail.Height = contentHeight - 2
	} else {
		// Narrow mode: full width panels stacked
		panelWidth := m.width - 2
		m.inProgressPanel.SetSize(panelWidth, inProgressHeight)
		m.openPanel.SetSize(panelWidth, openHeight)
		m.closedPanel.SetSize(panelWidth, closedHeight)
		m.detail.Width = m.width - 4
		m.detail.Height = contentHeight - 2
	}
}

func (m *Model) distributeTasks() {
	var inProgress, open, closed []models.Task
	for _, t := range m.tasks {
		switch t.Status {
		case "in_progress":
			inProgress = append(inProgress, t)
		case "open":
			open = append(open, t)
		case "closed":
			closed = append(closed, t)
		}
	}
	m.inProgressPanel.SetTasks(inProgress)
	m.openPanel.SetTasks(open)
	m.closedPanel.SetTasks(closed)
}

func (m *Model) getSelectedTask() *models.Task {
	switch m.focusedPanel {
	case FocusInProgress:
		return m.inProgressPanel.SelectedTask()
	case FocusOpen:
		return m.openPanel.SelectedTask()
	case FocusClosed:
		return m.closedPanel.SelectedTask()
	}
	return nil
}

func (m *Model) cyclePanelFocus(direction int) {
	// Clear focus from current panel
	switch m.focusedPanel {
	case FocusInProgress:
		m.inProgressPanel.SetFocus(false)
	case FocusOpen:
		m.openPanel.SetFocus(false)
	case FocusClosed:
		m.closedPanel.SetFocus(false)
	}

	// Cycle to next panel
	m.focusedPanel = PanelFocus((int(m.focusedPanel) + direction + int(panelCount)) % int(panelCount))

	// Set focus on new panel
	switch m.focusedPanel {
	case FocusInProgress:
		m.inProgressPanel.SetFocus(true)
	case FocusOpen:
		m.openPanel.SetFocus(true)
	case FocusClosed:
		m.closedPanel.SetFocus(true)
	}

	// Update selected task for detail panel
	m.selected = m.getSelectedTask()
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

	// Stack 3 panels vertically
	leftColumn := lipgloss.JoinVertical(lipgloss.Left,
		m.inProgressPanel.View(),
		m.openPanel.View(),
		m.closedPanel.View(),
	)

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
  e           Edit selected task
  c           Close selected task
  d           Delete selected task
  R           Refresh list

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
