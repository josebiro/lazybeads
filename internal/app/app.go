package app

import (
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"lazybeads/internal/beads"
	"lazybeads/internal/models"
	"lazybeads/internal/ui"
)

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
	ViewFilter
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

	// Filter state
	filterQuery string

	// Status message (flash notification)
	statusMsg string
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
			// In list mode, clear filter if active
			if m.filterQuery != "" {
				m.filterQuery = ""
				m.distributeTasks()
				return m, nil
			}
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

	case clipboardCopiedMsg:
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.statusMsg = "Copied!"
			cmds = append(cmds, tea.Tick(statusFlashDuration, func(t time.Time) tea.Msg {
				return clearStatusMsg{}
			}))
		}

	case clearStatusMsg:
		m.statusMsg = ""
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
	case ViewFilter:
		// Update text input in inline bar for filter
		var cmd tea.Cmd
		m.inlineBar.Input, cmd = m.inlineBar.Input.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
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
	filterLower := strings.ToLower(m.filterQuery)
	for _, t := range m.tasks {
		// Apply filter if set
		if filterLower != "" {
			titleLower := strings.ToLower(t.Title)
			idLower := strings.ToLower(t.ID)
			if !strings.Contains(titleLower, filterLower) && !strings.Contains(idLower, filterLower) {
				continue
			}
		}
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
