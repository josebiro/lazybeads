package app

import (
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/josebiro/lazybeads/internal/beads"
	"github.com/josebiro/lazybeads/internal/config"
	"github.com/josebiro/lazybeads/internal/models"
	"github.com/josebiro/lazybeads/internal/ui"
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
	ViewAddComment
	ViewBoard
	ViewAddBlocker
	ViewRemoveBlocker
)

// PanelFocus represents which panel is focused
type PanelFocus int

const (
	FocusInProgress PanelFocus = iota
	FocusOpen
	FocusClosed
	panelCount
)

// FilterMode represents quick filter presets
type FilterMode int

const (
	FilterAll    FilterMode = iota // Show all tasks
	FilterOpen                     // Show only open tasks
	FilterClosed                   // Show only closed tasks
	FilterReady                    // Show only ready tasks (no blockers)
)

// String returns the display name for the filter mode
func (f FilterMode) String() string {
	switch f {
	case FilterAll:
		return "All"
	case FilterOpen:
		return "Open"
	case FilterClosed:
		return "Closed"
	case FilterReady:
		return "Ready"
	default:
		return "?"
	}
}

// SortMode represents how tasks are sorted
type SortMode int

const (
	SortDefault    SortMode = iota // Priority then updated
	SortCreatedAsc                 // Oldest first
	SortCreatedDesc                // Newest first
	SortPriority                   // Priority only
	SortUpdated                    // Most recently updated first
	sortModeCount
)

// String returns the display name for the sort mode
func (s SortMode) String() string {
	switch s {
	case SortDefault:
		return "Default"
	case SortCreatedAsc:
		return "Created ↑"
	case SortCreatedDesc:
		return "Created ↓"
	case SortPriority:
		return "Priority"
	case SortUpdated:
		return "Updated"
	default:
		return "?"
	}
}

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
	detail       viewport.Model
	helpViewport viewport.Model
	filterText   textinput.Model

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

	// Modal state for field editing
	modal ui.Modal

	// Filter state
	filterQuery string
	filterMode  FilterMode       // quick filter mode (All/Open/Closed/Ready)
	searchMode  bool             // true when inline search is active
	searchInput textinput.Model  // text input for inline search in status bar

	// Sort mode
	sortMode SortMode

	// Board view state
	boardColumn  int      // 0=Open, 1=In Progress, 2=Closed
	boardRow     int      // Selected row within the column
	previousMode ViewMode // Track where user came from (for returning from detail view)

	// Status message (flash notification)
	statusMsg string

	// Task lookup map for O(1) access by ID (used for linked issue display)
	tasksMap map[string]*models.Task

	// Comments for selected task
	comments     []models.Comment
	commentInput textinput.Model

	// Blocker selection (for add/remove blocker modals)
	blockerOptions []string // List of issue IDs to choose from

	// Custom commands from config
	customCommands []config.CustomCommand
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
	closedPanel.SetCollapsed(true) // Start collapsed since not focused

	// Initialize detail viewport
	vp := viewport.New(0, 0)

	// Initialize help viewport
	helpVp := viewport.New(0, 0)

	// Initialize filter input (legacy - can be removed)
	filter := textinput.New()
	filter.Placeholder = "Search tasks..."
	filter.CharLimit = 100

	// Initialize inline search input for status bar
	searchInput := textinput.New()
	searchInput.Prompt = ""
	searchInput.CharLimit = 100
	searchInput.Width = 30

	// Initialize form inputs
	formTitle := textinput.New()
	formTitle.Prompt = ""
	formTitle.Placeholder = "Enter a brief, descriptive title for this task"
	formTitle.CharLimit = 200

	formDesc := textinput.New()
	formDesc.Prompt = ""
	formDesc.Placeholder = "Add details, context, or acceptance criteria (optional)"
	formDesc.CharLimit = 1000

	// Initialize comment input
	commentInput := textinput.New()
	commentInput.Prompt = ""
	commentInput.Placeholder = "Enter your comment..."
	commentInput.CharLimit = 1000

	// Load config (ignore errors, use empty config)
	cfg, _ := config.Load()
	var customCmds []config.CustomCommand
	if cfg != nil {
		customCmds = cfg.CustomCommands
	}

	// Build key map with custom commands
	keys := ui.DefaultKeyMap()
	keys.CustomCommands = buildCustomCommandBindings(customCmds)

	return Model{
		client:          beads.NewClient(),
		keys:            keys,
		help:            h,
		mode:            ViewList,
		focusedPanel:    FocusInProgress,
		inProgressPanel: inProgressPanel,
		openPanel:       openPanel,
		closedPanel:     closedPanel,
		detail:          vp,
		helpViewport:    helpVp,
		filterText:      filter,
		searchInput:     searchInput,
		formTitle:       formTitle,
		formDesc:        formDesc,
		formPriority:    2,
		formType:        "feature",
		commentInput:    commentInput,
		customCommands:  customCmds,
	}
}

// buildCustomCommandBindings creates key bindings from custom commands
func buildCustomCommandBindings(cmds []config.CustomCommand) []key.Binding {
	var bindings []key.Binding
	for _, cmd := range cmds {
		bindings = append(bindings, key.NewBinding(
			key.WithKeys(cmd.Key),
			key.WithHelp(cmd.Key, cmd.Description),
		))
	}
	return bindings
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

	case tea.MouseMsg:
		cmd := m.handleMouseEvent(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)

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
			// If in search mode, exit search mode and clear filter
			if m.searchMode {
				m.searchMode = false
				m.searchInput.Blur()
				m.filterQuery = ""
				m.searchInput.SetValue("")
				m.distributeTasks()
				return m, nil
			}
			// Handle escape based on current mode
			switch m.mode {
			case ViewDetail:
				// Return to where we came from (board or list)
				if m.previousMode == ViewBoard {
					m.mode = ViewBoard
				} else {
					m.mode = ViewList
				}
				m.previousMode = ViewList // Reset
				return m, nil
			case ViewList:
				// In list mode, clear filter if active
				if m.filterQuery != "" {
					m.filterQuery = ""
					m.distributeTasks()
					return m, nil
				}
				return m, nil
			default:
				// Other modes: go back to list
				m.mode = ViewList
				return m, nil
			}
		}

		prevMode := m.mode
		prevSearchMode := m.searchMode
		cmd := m.handleKeyPress(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		// If mode changed or search mode was just activated, don't pass key to child components
		if m.mode != prevMode || (m.searchMode && !prevSearchMode) {
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

	case commentsLoadedMsg:
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.comments = msg.comments
		}

	case commentAddedMsg:
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.statusMsg = "Comment added!"
			cmds = append(cmds, tea.Tick(statusFlashDuration, func(t time.Time) tea.Msg {
				return clearStatusMsg{}
			}))
			// Reload comments
			if m.selected != nil {
				cmds = append(cmds, m.loadComments(m.selected.ID))
			}
		}
		m.mode = ViewList

	case clipboardCopiedMsg:
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.statusMsg = "Copied!"
			cmds = append(cmds, tea.Tick(statusFlashDuration, func(t time.Time) tea.Msg {
				return clearStatusMsg{}
			}))
		}

	case blockerAddedMsg:
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.statusMsg = "Blocker added!"
			cmds = append(cmds, tea.Tick(statusFlashDuration, func(t time.Time) tea.Msg {
				return clearStatusMsg{}
			}))
		}
		m.mode = ViewList
		cmds = append(cmds, m.loadTasks())

	case blockerRemovedMsg:
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.statusMsg = "Blocker removed!"
			cmds = append(cmds, tea.Tick(statusFlashDuration, func(t time.Time) tea.Msg {
				return clearStatusMsg{}
			}))
		}
		m.mode = ViewList
		cmds = append(cmds, m.loadTasks())

	case clearStatusMsg:
		m.statusMsg = ""
	}

	// Update child components
	switch m.mode {
	case ViewList:
		// If in search mode, update the search input
		if m.searchMode {
			var cmd tea.Cmd
			m.searchInput, cmd = m.searchInput.Update(msg)
			cmds = append(cmds, cmd)
			// Update filter query in real-time
			m.filterQuery = m.searchInput.Value()
			m.distributeTasks()
		} else {
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
		}
		// Sync selected item with detail panel
		m.selected = m.getSelectedTask()
	case ViewDetail:
		var cmd tea.Cmd
		m.detail, cmd = m.detail.Update(msg)
		cmds = append(cmds, cmd)
	case ViewForm:
		cmds = append(cmds, m.updateForm(msg))
	case ViewEditTitle:
		// Update text input in modal
		var cmd tea.Cmd
		m.modal.Input, cmd = m.modal.Input.Update(msg)
		cmds = append(cmds, cmd)
	case ViewFilter:
		// Update text input in modal for filter
		var cmd tea.Cmd
		m.modal.Input, cmd = m.modal.Input.Update(msg)
		cmds = append(cmds, cmd)
	case ViewHelp:
		// Update help viewport for scrolling
		var cmd tea.Cmd
		m.helpViewport, cmd = m.helpViewport.Update(msg)
		cmds = append(cmds, cmd)
	case ViewAddComment:
		// Update comment input
		var cmd tea.Cmd
		m.commentInput, cmd = m.commentInput.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) updateSizes() {
	// Reserve space for help bar (1 line) + margins
	contentHeight := m.height - 2
	if contentHeight < 0 {
		contentHeight = 0
	}

	// Determine how many panels are visible
	visiblePanels := m.getVisiblePanels()
	numPanels := len(visiblePanels)
	if numPanels == 0 {
		numPanels = 1 // Shouldn't happen, but avoid division by zero
	}

	// Account for newlines between panels when joined vertically
	// JoinVertical adds (numPanels - 1) newlines between panels
	joinedHeight := contentHeight - (numPanels - 1)
	if joinedHeight < numPanels {
		joinedHeight = numPanels // Minimum 1 line per panel
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

	// Check if Closed panel is collapsed (only when not focused)
	closedCollapsed := m.closedPanel.IsCollapsed()
	collapsedHeight := 3 // Collapsed panel takes 3 lines (top border + 1 content + bottom border)

	// Calculate available height for expanded panels
	availableHeight := joinedHeight
	numExpandedPanels := numPanels
	if closedCollapsed {
		availableHeight -= collapsedHeight
		numExpandedPanels--
	}
	if numExpandedPanels < 1 {
		numExpandedPanels = 1
	}

	// Calculate panel heights - distribute evenly with remainder going to first panels
	panelHeight := availableHeight / numExpandedPanels
	remainder := availableHeight % numExpandedPanels
	if panelHeight < 4 {
		panelHeight = 4
	}

	// Distribute heights to visible panels
	expandedPanelIndex := 0
	for _, panel := range visiblePanels {
		switch panel {
		case FocusInProgress:
			h := panelHeight
			if expandedPanelIndex < remainder {
				h++
			}
			m.inProgressPanel.SetSize(panelWidth, h)
			expandedPanelIndex++
		case FocusOpen:
			h := panelHeight
			if expandedPanelIndex < remainder {
				h++
			}
			m.openPanel.SetSize(panelWidth, h)
			expandedPanelIndex++
		case FocusClosed:
			if closedCollapsed {
				m.closedPanel.SetSize(panelWidth, collapsedHeight)
			} else {
				h := panelHeight
				if expandedPanelIndex < remainder {
					h++
				}
				m.closedPanel.SetSize(panelWidth, h)
				expandedPanelIndex++
			}
		}
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

	// Update help viewport size
	// Help view: title (2 lines) + content + help bar (1 line)
	helpHeight := m.height - 4
	if helpHeight < 5 {
		helpHeight = 5
	}
	m.helpViewport.Width = m.width - 4
	m.helpViewport.Height = helpHeight
}

func (m *Model) distributeTasks() {
	// Build task lookup map for O(1) access (used for linked issue display)
	m.tasksMap = make(map[string]*models.Task)
	for i := range m.tasks {
		m.tasksMap[m.tasks[i].ID] = &m.tasks[i]
	}

	var inProgress, open, closed []models.Task
	filterLower := strings.ToLower(m.filterQuery)
	for _, t := range m.tasks {
		// Apply text filter if set
		if filterLower != "" {
			titleLower := strings.ToLower(t.Title)
			idLower := strings.ToLower(t.ID)
			if !strings.Contains(titleLower, filterLower) && !strings.Contains(idLower, filterLower) {
				continue
			}
		}

		// Apply quick filter mode
		switch m.filterMode {
		case FilterOpen:
			if t.Status == "closed" {
				continue
			}
		case FilterClosed:
			if t.Status != "closed" {
				continue
			}
		case FilterReady:
			// Ready = open/in_progress AND not blocked
			if t.Status == "closed" || len(t.BlockedBy) > 0 {
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

	// Apply sorting based on current sort mode
	sortTasks := func(tasks []models.Task) {
		switch m.sortMode {
		case SortDefault:
			// Priority (ascending), then updated (descending)
			sort.Slice(tasks, func(i, j int) bool {
				if tasks[i].Priority != tasks[j].Priority {
					return tasks[i].Priority < tasks[j].Priority
				}
				return tasks[i].UpdatedAt.After(tasks[j].UpdatedAt)
			})
		case SortCreatedAsc:
			sort.Slice(tasks, func(i, j int) bool {
				return tasks[i].CreatedAt.Before(tasks[j].CreatedAt)
			})
		case SortCreatedDesc:
			sort.Slice(tasks, func(i, j int) bool {
				return tasks[i].CreatedAt.After(tasks[j].CreatedAt)
			})
		case SortPriority:
			sort.Slice(tasks, func(i, j int) bool {
				return tasks[i].Priority < tasks[j].Priority
			})
		case SortUpdated:
			sort.Slice(tasks, func(i, j int) bool {
				return tasks[i].UpdatedAt.After(tasks[j].UpdatedAt)
			})
		}
	}

	sortTasks(inProgress)
	sortTasks(open)

	// Sort closed tasks by ClosedAt (most recently closed first) - always
	sort.Slice(closed, func(i, j int) bool {
		if closed[i].ClosedAt == nil && closed[j].ClosedAt == nil {
			return false
		}
		if closed[i].ClosedAt == nil {
			return false
		}
		if closed[j].ClosedAt == nil {
			return true
		}
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

// getBoardSelectedTask returns the currently selected task in board view
// Board has 4 columns: 0=Blocked, 1=Ready, 2=In Progress, 3=Done
func (m *Model) getBoardSelectedTask() *models.Task {
	var blockedTasks, readyTasks, inProgressTasks, doneTasks []models.Task
	for _, t := range m.tasks {
		switch t.Status {
		case "open":
			// Split open tasks into Blocked vs Ready
			if t.IsBlocked() {
				blockedTasks = append(blockedTasks, t)
			} else {
				readyTasks = append(readyTasks, t)
			}
		case "in_progress":
			inProgressTasks = append(inProgressTasks, t)
		case "closed":
			doneTasks = append(doneTasks, t)
		}
	}

	var tasks []models.Task
	switch m.boardColumn {
	case 0:
		tasks = blockedTasks
	case 1:
		tasks = readyTasks
	case 2:
		tasks = inProgressTasks
	case 3:
		tasks = doneTasks
	}

	if m.boardRow >= 0 && m.boardRow < len(tasks) {
		return &tasks[m.boardRow]
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

	// Track if we're leaving the Closed panel
	wasClosedFocused := m.focusedPanel == FocusClosed

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

	// Handle Closed panel collapse/expand
	nowClosedFocused := m.focusedPanel == FocusClosed
	if wasClosedFocused && !nowClosedFocused {
		// Leaving Closed panel - collapse it
		m.closedPanel.SetCollapsed(true)
		m.updateSizes()
	} else if !wasClosedFocused && nowClosedFocused {
		// Entering Closed panel - expand it
		m.closedPanel.SetCollapsed(false)
		m.updateSizes()
	}

	// Update selected task for detail panel
	m.selected = m.getSelectedTask()
}
