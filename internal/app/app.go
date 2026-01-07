package app

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
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
	ViewEditTitle
	ViewEditStatus
	ViewEditPriority
	ViewEditType
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

	// Inline bar state (replaces modal)
	inlineBar ui.InlineBar
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
	formTitle.Prompt = ""
	formTitle.Placeholder = "Enter a brief, descriptive title for this task"
	formTitle.CharLimit = 200

	formDesc := textinput.New()
	formDesc.Prompt = ""
	formDesc.Placeholder = "Add details, context, or acceptance criteria (optional)"
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

// editorFinishedMsg is sent when external editor completes
type editorFinishedMsg struct {
	content string
	err     error
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

	case editorFinishedMsg:
		if msg.err != nil {
			m.err = msg.err
		} else if m.selected != nil {
			// Update description
			return m, func() tea.Msg {
				err := m.client.Update(m.selected.ID, beads.UpdateOptions{
					Description: msg.content,
				})
				return taskUpdatedMsg{err: err}
			}
		}
		m.mode = ViewList

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
	case ViewEditTitle:
		// Update text input in inline bar
		var cmd tea.Cmd
		m.inlineBar.Input, cmd = m.inlineBar.Input.Update(msg)
		cmds = append(cmds, cmd)
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
	case ViewEditTitle:
		return m.handleTitleBarKeys(msg)
	case ViewEditStatus:
		return m.handleSelectBarKeys(msg)
	case ViewEditPriority:
		return m.handleSelectBarKeys(msg)
	case ViewEditType:
		return m.handleSelectBarKeys(msg)
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

	case key.Matches(msg, m.keys.PrevView):
		m.cyclePanelFocus(-1)

	case key.Matches(msg, m.keys.NextView):
		m.cyclePanelFocus(1)

	case key.Matches(msg, m.keys.Refresh):
		return m.loadTasks()

	case key.Matches(msg, m.keys.Help):
		m.mode = ViewHelp

	case key.Matches(msg, m.keys.EditTitle):
		if task := m.getSelectedTask(); task != nil {
			m.inlineBar = ui.NewInlineBarInput("Title", task.ID, task.Title, m.width)
			m.mode = ViewEditTitle
		}

	case key.Matches(msg, m.keys.EditStatus):
		if task := m.getSelectedTask(); task != nil {
			options := []ui.InlineBarOption{
				{Label: "open", Value: "open", Shortcut: "o"},
				{Label: "in_progress", Value: "in_progress", Shortcut: "i"},
				{Label: "closed", Value: "closed", Shortcut: "c"},
			}
			m.inlineBar = ui.NewInlineBarSelect("Status", task.ID, options, task.Status)
			m.mode = ViewEditStatus
		}

	case key.Matches(msg, m.keys.EditPriority):
		if task := m.getSelectedTask(); task != nil {
			options := []ui.InlineBarOption{
				{Label: "P0", Value: "0", Shortcut: "0"},
				{Label: "P1", Value: "1", Shortcut: "1"},
				{Label: "P2", Value: "2", Shortcut: "2"},
				{Label: "P3", Value: "3", Shortcut: "3"},
				{Label: "P4", Value: "4", Shortcut: "4"},
			}
			m.inlineBar = ui.NewInlineBarSelect("Priority", task.ID, options, fmt.Sprintf("%d", task.Priority))
			m.mode = ViewEditPriority
		}

	case key.Matches(msg, m.keys.EditType):
		if task := m.getSelectedTask(); task != nil {
			options := []ui.InlineBarOption{
				{Label: "task", Value: "task", Shortcut: "t"},
				{Label: "bug", Value: "bug", Shortcut: "b"},
				{Label: "feature", Value: "feature", Shortcut: "f"},
				{Label: "epic", Value: "epic", Shortcut: "e"},
				{Label: "chore", Value: "chore", Shortcut: "r"},
			}
			m.inlineBar = ui.NewInlineBarSelect("Type", task.ID, options, task.Type)
			m.mode = ViewEditType
		}

	case key.Matches(msg, m.keys.EditDescription):
		if task := m.getSelectedTask(); task != nil {
			return m.editDescriptionInEditor(task)
		}
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

	case key.Matches(msg, m.keys.ShiftTab):
		m.formFocus = (m.formFocus - 1 + 4) % 4
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

func (m *Model) handleTitleBarKeys(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "enter":
		if m.selected != nil {
			newTitle := strings.TrimSpace(m.inlineBar.InputValue())
			if newTitle != "" {
				taskID := m.selected.ID
				m.mode = ViewList
				return func() tea.Msg {
					err := m.client.Update(taskID, beads.UpdateOptions{
						Title: newTitle,
					})
					return taskUpdatedMsg{err: err}
				}
			}
		}
		m.mode = ViewList
	case "esc":
		m.mode = ViewList
	}
	return nil
}

func (m *Model) handleSelectBarKeys(msg tea.KeyMsg) tea.Cmd {
	key := msg.String()

	// Check for shortcut keys first
	if m.inlineBar.SelectByShortcut(key) {
		// Shortcut matched, apply immediately
		if m.selected != nil {
			value := m.inlineBar.SelectedValue()
			taskID := m.selected.ID
			m.mode = ViewList
			return m.applyInlineBarSelection(taskID, value)
		}
	}

	switch key {
	case "h", "left":
		m.inlineBar.MoveLeft()
	case "l", "right":
		m.inlineBar.MoveRight()
	case "enter":
		if m.selected != nil {
			value := m.inlineBar.SelectedValue()
			taskID := m.selected.ID
			m.mode = ViewList
			return m.applyInlineBarSelection(taskID, value)
		}
		m.mode = ViewList
	case "esc":
		m.mode = ViewList
	}
	return nil
}

func (m *Model) applyInlineBarSelection(taskID, value string) tea.Cmd {
	// Determine what field to update based on inline bar title
	switch m.inlineBar.Title {
	case "Status":
		return func() tea.Msg {
			err := m.client.Update(taskID, beads.UpdateOptions{
				Status: value,
			})
			return taskUpdatedMsg{err: err}
		}
	case "Priority":
		priority := 2
		fmt.Sscanf(value, "%d", &priority)
		return func() tea.Msg {
			err := m.client.Update(taskID, beads.UpdateOptions{
				Priority: &priority,
			})
			return taskUpdatedMsg{err: err}
		}
	case "Type":
		return func() tea.Msg {
			err := m.client.Update(taskID, beads.UpdateOptions{
				Type: value,
			})
			return taskUpdatedMsg{err: err}
		}
	}
	return nil
}

func (m *Model) editDescriptionInEditor(task *models.Task) tea.Cmd {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "nano"
	}

	// Create temp file with .md extension for syntax highlighting
	tmpfile, err := os.CreateTemp("", "lazybeads-*.md")
	if err != nil {
		m.err = fmt.Errorf("failed to create temp file: %w", err)
		return nil
	}

	// Write current description to temp file
	if _, err := tmpfile.WriteString(task.Description); err != nil {
		tmpfile.Close()
		os.Remove(tmpfile.Name())
		m.err = fmt.Errorf("failed to write to temp file: %w", err)
		return nil
	}
	tmpfile.Close()

	tmpPath := tmpfile.Name()
	c := exec.Command(editor, tmpPath)

	return tea.ExecProcess(c, func(err error) tea.Msg {
		defer os.Remove(tmpPath)
		if err != nil {
			return editorFinishedMsg{err: err}
		}
		content, readErr := os.ReadFile(tmpPath)
		if readErr != nil {
			return editorFinishedMsg{err: readErr}
		}
		return editorFinishedMsg{content: string(content)}
	})
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

	// Determine how many panels are visible
	visiblePanels := m.getVisiblePanels()
	numPanels := len(visiblePanels)
	if numPanels == 0 {
		numPanels = 1 // Shouldn't happen, but avoid division by zero
	}

	// Calculate panel heights - distribute evenly with remainder going to first panels
	panelHeight := contentHeight / numPanels
	remainder := contentHeight % numPanels
	if panelHeight < 4 {
		panelHeight = 4
	}

	// Wide mode: panels on left, detail on right
	var panelWidth int
	if m.width >= 80 {
		panelWidth = m.width/2 - 1
		m.detail.Width = m.width/2 - 4
		m.detail.Height = contentHeight - 2
	} else {
		// Narrow mode: full width panels stacked
		panelWidth = m.width - 2
		m.detail.Width = m.width - 4
		m.detail.Height = contentHeight - 2
	}

	// Distribute heights to visible panels
	panelIndex := 0
	for _, panel := range visiblePanels {
		h := panelHeight
		if panelIndex < remainder {
			h++
		}
		switch panel {
		case FocusInProgress:
			m.inProgressPanel.SetSize(panelWidth, h)
		case FocusOpen:
			m.openPanel.SetSize(panelWidth, h)
		case FocusClosed:
			m.closedPanel.SetSize(panelWidth, h)
		}
		panelIndex++
	}

	// Set size 0 for hidden panels (In Progress when empty)
	if !m.isInProgressVisible() {
		m.inProgressPanel.SetSize(panelWidth, 0)
	}

	// Update form input widths for placeholder text display
	formWidth := m.width - 24 // Account for padding and borders
	if formWidth < 20 {
		formWidth = 20
	}
	m.formTitle.Width = formWidth
	m.formDesc.Width = formWidth
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

	// Sort closed tasks by ClosedAt (most recently closed first)
	sort.Slice(closed, func(i, j int) bool {
		// Tasks with ClosedAt come before those without
		if closed[i].ClosedAt == nil && closed[j].ClosedAt == nil {
			return false
		}
		if closed[i].ClosedAt == nil {
			return false
		}
		if closed[j].ClosedAt == nil {
			return true
		}
		// Most recently closed first (descending order)
		return closed[i].ClosedAt.After(*closed[j].ClosedAt)
	})

	m.inProgressPanel.SetTasks(inProgress)
	m.openPanel.SetTasks(open)
	m.closedPanel.SetTasks(closed)

	// If In Progress panel disappears while focused, move focus to Open panel
	if m.focusedPanel == FocusInProgress && len(inProgress) == 0 {
		m.inProgressPanel.SetFocus(false)
		m.focusedPanel = FocusOpen
		m.openPanel.SetFocus(true)
		m.selected = m.getSelectedTask()
	}

	// Recalculate sizes since panel visibility may have changed
	m.updateSizes()
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

// isInProgressVisible returns true if the In Progress panel should be shown
func (m *Model) isInProgressVisible() bool {
	return m.inProgressPanel.TaskCount() > 0
}

// getVisiblePanels returns the list of currently visible panel focus values
func (m *Model) getVisiblePanels() []PanelFocus {
	var panels []PanelFocus
	if m.isInProgressVisible() {
		panels = append(panels, FocusInProgress)
	}
	panels = append(panels, FocusOpen)
	panels = append(panels, FocusClosed)
	return panels
}

func (m *Model) cyclePanelFocus(direction int) {
	visiblePanels := m.getVisiblePanels()
	if len(visiblePanels) == 0 {
		return
	}

	// Clear focus from current panel
	switch m.focusedPanel {
	case FocusInProgress:
		m.inProgressPanel.SetFocus(false)
	case FocusOpen:
		m.openPanel.SetFocus(false)
	case FocusClosed:
		m.closedPanel.SetFocus(false)
	}

	// Find current panel index in visible panels
	currentIdx := -1
	for i, p := range visiblePanels {
		if p == m.focusedPanel {
			currentIdx = i
			break
		}
	}

	// If current panel is not visible (e.g., In Progress disappeared), start from first visible
	if currentIdx == -1 {
		currentIdx = 0
	}

	// Cycle to next visible panel
	newIdx := (currentIdx + direction + len(visiblePanels)) % len(visiblePanels)
	m.focusedPanel = visiblePanels[newIdx]

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
  e           Edit all fields (form)
  x           Delete selected task
  R           Refresh list

Field Editing
  t           Edit title (modal)
  s           Edit status (modal)
  p           Edit priority (modal)
  y           Edit type (modal)
  d           Edit description ($EDITOR)

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

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
