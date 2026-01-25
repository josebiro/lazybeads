package app

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"text/template"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/josebiro/lazybeads/internal/beads"
	"github.com/josebiro/lazybeads/internal/config"
	"github.com/josebiro/lazybeads/internal/models"
	"github.com/josebiro/lazybeads/internal/ui"
)

// handleMouseEvent handles all mouse events
func (m *Model) handleMouseEvent(msg tea.MouseMsg) tea.Cmd {
	switch m.mode {
	case ViewList:
		return m.handleListMouse(msg)
	case ViewDetail:
		return m.handleDetailMouse(msg)
	case ViewHelp:
		return m.handleHelpMouse(msg)
	case ViewBoard:
		return m.handleBoardMouse(msg)
	case ViewEditStatus, ViewEditPriority, ViewEditType:
		return m.handleModalMouse(msg)
	}
	return nil
}

// handleListMouse handles mouse events in the list view
func (m *Model) handleListMouse(msg tea.MouseMsg) tea.Cmd {
	// Exit search mode on any click
	if m.searchMode && msg.Action == tea.MouseActionPress {
		m.searchMode = false
		m.searchInput.Blur()
	}

	// Calculate panel boundaries
	panelBounds := m.calculatePanelBounds()

	switch msg.Action {
	case tea.MouseActionPress:
		if msg.Button == tea.MouseButtonLeft {
			// Check which panel was clicked
			for panel, bounds := range panelBounds {
				if m.isPointInBounds(msg.X, msg.Y, bounds) {
					// Focus this panel
					m.focusPanelByType(panel)

					// Calculate which item was clicked (accounting for border)
					itemIndex := msg.Y - bounds.top - 1 // -1 for top border
					if itemIndex >= 0 {
						m.selectItemInPanel(panel, itemIndex)
					}
					break
				}
			}

			// Check if click is in the detail panel (wide mode)
			if m.width >= 80 {
				detailLeft := m.width / 2
				if msg.X >= detailLeft {
					// Clicked in detail area - open detail view if we have a selection
					if m.selected != nil {
						m.updateDetailContent()
						m.mode = ViewDetail
					}
				}
			}
		}

	case tea.MouseActionRelease:
		if msg.Button == tea.MouseButtonWheelUp {
			// Scroll up in the focused panel
			m.scrollFocusedPanel(-3)
		} else if msg.Button == tea.MouseButtonWheelDown {
			// Scroll down in the focused panel
			m.scrollFocusedPanel(3)
		}
	}

	return nil
}

// panelBounds represents the screen bounds of a panel
type panelBounds struct {
	top, bottom, left, right int
}

// calculatePanelBounds calculates the screen bounds for each visible panel
func (m *Model) calculatePanelBounds() map[PanelFocus]panelBounds {
	bounds := make(map[PanelFocus]panelBounds)

	// Panel width is half the screen in wide mode, full width in narrow mode
	var panelWidth int
	if m.width >= 80 {
		panelWidth = m.width / 2
	} else {
		panelWidth = m.width
	}

	currentY := 0

	// In Progress panel (if visible)
	if m.isInProgressVisible() {
		h := m.inProgressPanel.height
		bounds[FocusInProgress] = panelBounds{
			top:    currentY,
			bottom: currentY + h,
			left:   0,
			right:  panelWidth,
		}
		currentY += h
	}

	// Open panel
	h := m.openPanel.height
	bounds[FocusOpen] = panelBounds{
		top:    currentY,
		bottom: currentY + h,
		left:   0,
		right:  panelWidth,
	}
	currentY += h

	// Closed panel
	h = m.closedPanel.height
	bounds[FocusClosed] = panelBounds{
		top:    currentY,
		bottom: currentY + h,
		left:   0,
		right:  panelWidth,
	}

	return bounds
}

// isPointInBounds checks if a point is within the given bounds
func (m *Model) isPointInBounds(x, y int, bounds panelBounds) bool {
	return x >= bounds.left && x < bounds.right && y >= bounds.top && y < bounds.bottom
}

// focusPanelByType focuses the specified panel
func (m *Model) focusPanelByType(panel PanelFocus) {
	// Track if we're leaving/entering the Closed panel for collapse handling
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

	// Set focus on new panel
	m.focusedPanel = panel
	switch panel {
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
		m.closedPanel.SetCollapsed(true)
		m.updateSizes()
	} else if !wasClosedFocused && nowClosedFocused {
		m.closedPanel.SetCollapsed(false)
		m.updateSizes()
	}

	m.selected = m.getSelectedTask()
}

// selectItemInPanel selects an item by index in the specified panel
func (m *Model) selectItemInPanel(panel PanelFocus, index int) {
	switch panel {
	case FocusInProgress:
		m.inProgressPanel.SelectIndex(index)
	case FocusOpen:
		m.openPanel.SelectIndex(index)
	case FocusClosed:
		m.closedPanel.SelectIndex(index)
	}
	m.selected = m.getSelectedTask()
}

// scrollFocusedPanel scrolls the focused panel by the given amount
func (m *Model) scrollFocusedPanel(amount int) {
	switch m.focusedPanel {
	case FocusInProgress:
		m.inProgressPanel.ScrollBy(amount)
	case FocusOpen:
		m.openPanel.ScrollBy(amount)
	case FocusClosed:
		m.closedPanel.ScrollBy(amount)
	}
	m.selected = m.getSelectedTask()
}

// handleDetailMouse handles mouse events in the detail view
func (m *Model) handleDetailMouse(msg tea.MouseMsg) tea.Cmd {
	switch msg.Action {
	case tea.MouseActionPress:
		if msg.Button == tea.MouseButtonLeft {
			// Click anywhere to go back to list
			m.mode = ViewList
		}
	case tea.MouseActionRelease:
		if msg.Button == tea.MouseButtonWheelUp {
			m.detail.LineUp(3)
		} else if msg.Button == tea.MouseButtonWheelDown {
			m.detail.LineDown(3)
		}
	}
	return nil
}

// handleHelpMouse handles mouse events in the help view
func (m *Model) handleHelpMouse(msg tea.MouseMsg) tea.Cmd {
	switch msg.Action {
	case tea.MouseActionPress:
		if msg.Button == tea.MouseButtonLeft {
			// Click anywhere to close help
			m.helpViewport.GotoTop()
			m.mode = ViewList
		}
	case tea.MouseActionRelease:
		if msg.Button == tea.MouseButtonWheelUp {
			m.helpViewport.LineUp(3)
		} else if msg.Button == tea.MouseButtonWheelDown {
			m.helpViewport.LineDown(3)
		}
	}
	return nil
}

// handleBoardMouse handles mouse events in the board view
func (m *Model) handleBoardMouse(msg tea.MouseMsg) tea.Cmd {
	// Determine if we're in wide mode (must match viewBoard)
	wideMode := m.width >= 160

	// Calculate board area width - give board 2/3 of space in wide mode
	boardWidth := m.width
	if wideMode {
		boardWidth = (m.width * 2) / 3
	}

	// Column dimensions (must match viewBoard calculations)
	colWidth := (boardWidth - 8) / 3
	if colWidth < 15 {
		colWidth = 15
	}

	// Header takes 3 lines (title + blank + headers)
	headerLines := 3
	// Column content starts after header
	colTop := headerLines

	switch msg.Action {
	case tea.MouseActionPress:
		if msg.Button == tea.MouseButtonLeft {
			// Check if click is in the board area (left half in wide mode)
			if wideMode && msg.X >= boardWidth {
				// Click in detail panel area - could open full detail view
				if m.selected != nil {
					m.updateDetailContent()
					m.mode = ViewDetail
					return m.loadComments(m.selected.ID)
				}
				return nil
			}

			// Determine which column was clicked
			clickedColumn := -1
			if msg.X < colWidth {
				clickedColumn = 0 // Open
			} else if msg.X < colWidth*2 {
				clickedColumn = 1 // In Progress
			} else if msg.X < colWidth*3 {
				clickedColumn = 2 // Closed
			}

			if clickedColumn >= 0 {
				// Calculate which row was clicked (accounting for header and borders)
				clickedRow := msg.Y - colTop - 1 // -1 for column border

				// Get task count for clicked column
				var columnCount int
				for _, t := range m.tasks {
					switch clickedColumn {
					case 0:
						if t.Status == "open" {
							columnCount++
						}
					case 1:
						if t.Status == "in_progress" {
							columnCount++
						}
					case 2:
						if t.Status == "closed" {
							columnCount++
						}
					}
				}

				// Update selection if valid
				if clickedRow >= 0 && clickedRow < columnCount {
					m.boardColumn = clickedColumn
					m.boardRow = clickedRow
					m.selected = m.getBoardSelectedTask()
				} else if clickedRow >= 0 {
					// Clicked in column but below tasks - just focus the column
					m.boardColumn = clickedColumn
					if columnCount > 0 {
						m.boardRow = columnCount - 1
					} else {
						m.boardRow = 0
					}
					m.selected = m.getBoardSelectedTask()
				}
			}
		}

	case tea.MouseActionRelease:
		// Handle scroll wheel in board view
		if msg.Button == tea.MouseButtonWheelUp {
			if m.boardRow > 0 {
				m.boardRow--
				m.selected = m.getBoardSelectedTask()
			}
		} else if msg.Button == tea.MouseButtonWheelDown {
			// Get count for current column
			var columnCount int
			for _, t := range m.tasks {
				switch m.boardColumn {
				case 0:
					if t.Status == "open" {
						columnCount++
					}
				case 1:
					if t.Status == "in_progress" {
						columnCount++
					}
				case 2:
					if t.Status == "closed" {
						columnCount++
					}
				}
			}
			if m.boardRow < columnCount-1 {
				m.boardRow++
				m.selected = m.getBoardSelectedTask()
			}
		}
	}

	return nil
}

// handleModalMouse handles mouse events in modal dialogs
func (m *Model) handleModalMouse(msg tea.MouseMsg) tea.Cmd {
	if msg.Action != tea.MouseActionPress || msg.Button != tea.MouseButtonLeft {
		return nil
	}

	// Calculate modal bounds (centered on screen)
	modalWidth := 40
	modalHeight := len(m.modal.Options) + 4 // header + options + padding
	modalLeft := (m.width - modalWidth) / 2
	modalTop := (m.height - modalHeight) / 2

	// Check if click is outside modal (dismiss)
	if msg.X < modalLeft || msg.X >= modalLeft+modalWidth ||
		msg.Y < modalTop || msg.Y >= modalTop+modalHeight {
		m.mode = ViewList
		return nil
	}

	// Check if click is on an option
	optionStart := modalTop + 2 // After header
	clickedOption := msg.Y - optionStart
	if clickedOption >= 0 && clickedOption < len(m.modal.Options) {
		m.modal.Selected = clickedOption
		// Apply the selection
		if m.selected != nil {
			value := m.modal.SelectedValue()
			taskID := m.selected.ID
			m.mode = ViewList
			return m.applyModalSelection(taskID, value)
		}
	}

	return nil
}

func (m *Model) handleKeyPress(msg tea.KeyMsg) tea.Cmd {
	// If in search mode, handle search keys first
	if m.searchMode {
		return m.handleSearchKeys(msg)
	}

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
	case ViewFilter:
		return m.handleFilterKeys(msg)
	case ViewAddComment:
		return m.handleAddCommentKeys(msg)
	case ViewBoard:
		return m.handleBoardKeys(msg)
	case ViewAddBlocker:
		return m.handleAddBlockerKeys(msg)
	case ViewRemoveBlocker:
		return m.handleRemoveBlockerKeys(msg)
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
			m.comments = nil // Clear old comments
			m.updateDetailContent()
			m.mode = ViewDetail
			return m.loadComments(task.ID)
		}

	case key.Matches(msg, m.keys.Add):
		m.resetForm()
		m.editing = false
		m.mode = ViewForm
		m.formTitle.Focus()

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
			m.modal = ui.NewModalInput("Edit Title", task.ID, task.Title)
			m.mode = ViewEditTitle
		}

	case key.Matches(msg, m.keys.EditStatus):
		if task := m.getSelectedTask(); task != nil {
			options := []ui.ModalOption{
				{Label: "open", Value: "open", Shortcut: "o"},
				{Label: "in_progress", Value: "in_progress", Shortcut: "i"},
				{Label: "closed", Value: "closed", Shortcut: "c"},
			}
			m.modal = ui.NewModalSelect("Edit Status", task.ID, options, task.Status)
			m.mode = ViewEditStatus
		}

	case key.Matches(msg, m.keys.EditPriority):
		if task := m.getSelectedTask(); task != nil {
			options := []ui.ModalOption{
				{Label: "P0 - Critical", Value: "0", Shortcut: "0"},
				{Label: "P1 - High", Value: "1", Shortcut: "1"},
				{Label: "P2 - Medium", Value: "2", Shortcut: "2"},
				{Label: "P3 - Low", Value: "3", Shortcut: "3"},
				{Label: "P4 - Backlog", Value: "4", Shortcut: "4"},
			}
			m.modal = ui.NewModalSelect("Edit Priority", task.ID, options, fmt.Sprintf("%d", task.Priority))
			m.mode = ViewEditPriority
		}

	case key.Matches(msg, m.keys.EditType):
		if task := m.getSelectedTask(); task != nil {
			options := []ui.ModalOption{
				{Label: "task", Value: "task", Shortcut: "t"},
				{Label: "bug", Value: "bug", Shortcut: "b"},
				{Label: "feature", Value: "feature", Shortcut: "f"},
				{Label: "epic", Value: "epic", Shortcut: "e"},
				{Label: "chore", Value: "chore", Shortcut: "r"},
			}
			m.modal = ui.NewModalSelect("Edit Type", task.ID, options, task.Type)
			m.mode = ViewEditType
		}

	case key.Matches(msg, m.keys.EditDescription):
		if task := m.getSelectedTask(); task != nil {
			return m.editDescriptionInEditor(task)
		}

	case key.Matches(msg, m.keys.AddComment):
		if task := m.getSelectedTask(); task != nil {
			m.commentInput.SetValue("")
			m.commentInput.Focus()
			m.mode = ViewAddComment
			return m.commentInput.Focus()
		}

	case key.Matches(msg, m.keys.AddBlocker):
		if task := m.getSelectedTask(); task != nil {
			// Build list of potential blockers (all other open tasks)
			var options []ui.ModalOption
			for _, t := range m.tasks {
				if t.ID != task.ID && t.Status != "closed" {
					// Check if already blocking
					alreadyBlocking := false
					for _, b := range task.BlockedBy {
						if b == t.ID {
							alreadyBlocking = true
							break
						}
					}
					if !alreadyBlocking {
						label := fmt.Sprintf("%s - %s", t.ID, t.Title)
						if len(label) > 50 {
							label = label[:47] + "..."
						}
						options = append(options, ui.ModalOption{
							Label: label,
							Value: t.ID,
						})
					}
				}
			}
			if len(options) == 0 {
				m.statusMsg = "No available tasks to add as blocker"
				return tea.Tick(statusFlashDuration, func(t time.Time) tea.Msg {
					return clearStatusMsg{}
				})
			}
			m.modal = ui.NewModalSelect("Add Blocker", task.ID, options, "")
			m.mode = ViewAddBlocker
		}

	case key.Matches(msg, m.keys.RemoveBlocker):
		if task := m.getSelectedTask(); task != nil {
			if len(task.BlockedBy) == 0 {
				m.statusMsg = "No blockers to remove"
				return tea.Tick(statusFlashDuration, func(t time.Time) tea.Msg {
					return clearStatusMsg{}
				})
			}
			// Build list of current blockers
			var options []ui.ModalOption
			for _, blockerID := range task.BlockedBy {
				label := blockerID
				// Try to get title from tasks map
				if blockerTask, ok := m.tasksMap[blockerID]; ok {
					label = fmt.Sprintf("%s - %s", blockerID, blockerTask.Title)
					if len(label) > 50 {
						label = label[:47] + "..."
					}
				}
				options = append(options, ui.ModalOption{
					Label: label,
					Value: blockerID,
				})
			}
			m.modal = ui.NewModalSelect("Remove Blocker", task.ID, options, "")
			m.mode = ViewRemoveBlocker
		}

	case key.Matches(msg, m.keys.Filter):
		// Enter inline search mode in status bar
		m.searchMode = true
		m.searchInput.SetValue(m.filterQuery)
		m.searchInput.Focus()
		return m.searchInput.Focus() // Return blink command

	case key.Matches(msg, m.keys.CopyID):
		if task := m.getSelectedTask(); task != nil {
			taskID := task.ID
			return func() tea.Msg {
				err := clipboard.WriteAll(taskID)
				return clipboardCopiedMsg{text: taskID, err: err}
			}
		}

	case key.Matches(msg, m.keys.Sort):
		// Cycle through sort modes
		m.sortMode = (m.sortMode + 1) % sortModeCount
		m.distributeTasks()

	case key.Matches(msg, m.keys.Board):
		// Switch to board view
		m.boardColumn = 0
		m.boardRow = 0
		m.mode = ViewBoard

	case key.Matches(msg, m.keys.Open):
		// Toggle open filter (show open + in_progress only)
		if m.filterMode == FilterOpen {
			m.filterMode = FilterAll
		} else {
			m.filterMode = FilterOpen
		}
		m.distributeTasks()

	case key.Matches(msg, m.keys.Closed):
		// Toggle closed filter (show closed only)
		if m.filterMode == FilterClosed {
			m.filterMode = FilterAll
		} else {
			m.filterMode = FilterClosed
		}
		m.distributeTasks()

	case key.Matches(msg, m.keys.Ready):
		// Toggle ready filter (show tasks without blockers)
		if m.filterMode == FilterReady {
			m.filterMode = FilterAll
		} else {
			m.filterMode = FilterReady
		}
		m.distributeTasks()

	case key.Matches(msg, m.keys.All):
		// Clear filter mode
		m.filterMode = FilterAll
		m.distributeTasks()

	default:
		// Check custom commands
		if cmd := m.matchCustomCommand(msg, "list"); cmd != nil {
			return cmd
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
	default:
		// Check custom commands
		if cmd := m.matchCustomCommand(msg, "detail"); cmd != nil {
			return cmd
		}
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
		// Reset scroll position when closing help
		m.helpViewport.GotoTop()
		m.mode = ViewList
	case key.Matches(msg, m.keys.Up):
		m.helpViewport.LineUp(1)
	case key.Matches(msg, m.keys.Down):
		m.helpViewport.LineDown(1)
	case key.Matches(msg, m.keys.PageUp):
		m.helpViewport.HalfViewUp()
	case key.Matches(msg, m.keys.PageDown):
		m.helpViewport.HalfViewDown()
	case key.Matches(msg, m.keys.Top):
		m.helpViewport.GotoTop()
	case key.Matches(msg, m.keys.Bottom):
		m.helpViewport.GotoBottom()
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
			newTitle := strings.TrimSpace(m.modal.InputValue())
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
	if m.modal.SelectByShortcut(key) {
		// Shortcut matched, apply immediately
		if m.selected != nil {
			value := m.modal.SelectedValue()
			taskID := m.selected.ID
			m.mode = ViewList
			return m.applyModalSelection(taskID, value)
		}
	}

	switch key {
	case "k", "up":
		m.modal.MoveUp()
	case "j", "down":
		m.modal.MoveDown()
	case "enter":
		if m.selected != nil {
			value := m.modal.SelectedValue()
			taskID := m.selected.ID
			m.mode = ViewList
			return m.applyModalSelection(taskID, value)
		}
		m.mode = ViewList
	case "esc":
		m.mode = ViewList
	}
	return nil
}

func (m *Model) applyModalSelection(taskID, value string) tea.Cmd {
	// Determine what field to update based on modal title
	switch m.modal.Title {
	case "Edit Status":
		return func() tea.Msg {
			err := m.client.Update(taskID, beads.UpdateOptions{
				Status: value,
			})
			return taskUpdatedMsg{err: err}
		}
	case "Edit Priority":
		priority := 2
		fmt.Sscanf(value, "%d", &priority)
		return func() tea.Msg {
			err := m.client.Update(taskID, beads.UpdateOptions{
				Priority: &priority,
			})
			return taskUpdatedMsg{err: err}
		}
	case "Edit Type":
		return func() tea.Msg {
			err := m.client.Update(taskID, beads.UpdateOptions{
				Type: value,
			})
			return taskUpdatedMsg{err: err}
		}
	}
	return nil
}

func (m *Model) handleSearchKeys(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "enter":
		// Confirm filter and exit search mode (keep filter active)
		m.searchMode = false
		m.searchInput.Blur()
		m.filterQuery = strings.TrimSpace(m.searchInput.Value())
		m.distributeTasks()
		return nil
	case "backspace":
		// If input is empty, exit search mode without clearing existing filter
		if m.searchInput.Value() == "" {
			m.searchMode = false
			m.searchInput.Blur()
			return nil
		}
		// Otherwise let the textinput handle backspace normally
		return nil
	}
	// Let the textinput handle all other keys
	return nil
}

func (m *Model) handleFilterKeys(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "enter":
		// Apply filter and return to list
		m.filterQuery = strings.TrimSpace(m.modal.InputValue())
		m.distributeTasks()
		m.mode = ViewList
	case "esc":
		// Cancel and return to list (don't change filter)
		m.mode = ViewList
	}
	return nil
}

func (m *Model) handleAddCommentKeys(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "enter":
		// Submit comment
		comment := strings.TrimSpace(m.commentInput.Value())
		if comment != "" && m.selected != nil {
			taskID := m.selected.ID
			m.commentInput.Blur()
			m.mode = ViewList
			return func() tea.Msg {
				err := m.client.AddComment(taskID, comment)
				return commentAddedMsg{err: err}
			}
		}
		m.mode = ViewList
	case "esc":
		m.commentInput.Blur()
		m.mode = ViewList
	}
	return nil
}

func (m *Model) handleAddBlockerKeys(msg tea.KeyMsg) tea.Cmd {
	key := msg.String()

	switch key {
	case "k", "up":
		m.modal.MoveUp()
	case "j", "down":
		m.modal.MoveDown()
	case "enter":
		if m.selected != nil {
			blockerID := m.modal.SelectedValue()
			taskID := m.selected.ID
			m.mode = ViewList
			return func() tea.Msg {
				err := m.client.AddBlocker(taskID, blockerID)
				return blockerAddedMsg{err: err}
			}
		}
		m.mode = ViewList
	case "esc":
		m.mode = ViewList
	}
	return nil
}

func (m *Model) handleRemoveBlockerKeys(msg tea.KeyMsg) tea.Cmd {
	key := msg.String()

	switch key {
	case "k", "up":
		m.modal.MoveUp()
	case "j", "down":
		m.modal.MoveDown()
	case "enter":
		if m.selected != nil {
			blockerID := m.modal.SelectedValue()
			taskID := m.selected.ID
			m.mode = ViewList
			return func() tea.Msg {
				err := m.client.RemoveBlocker(taskID, blockerID)
				return blockerRemovedMsg{err: err}
			}
		}
		m.mode = ViewList
	case "esc":
		m.mode = ViewList
	}
	return nil
}

func (m *Model) handleBoardKeys(msg tea.KeyMsg) tea.Cmd {
	// Get tasks for each column to know bounds
	var openCount, inProgressCount, closedCount int
	for _, t := range m.tasks {
		switch t.Status {
		case "open":
			openCount++
		case "in_progress":
			inProgressCount++
		case "closed":
			closedCount++
		}
	}

	// Get count for current column
	currentColumnCount := func() int {
		switch m.boardColumn {
		case 0:
			return openCount
		case 1:
			return inProgressCount
		case 2:
			return closedCount
		}
		return 0
	}

	// Track if selection changed for syncing detail panel
	selectionChanged := false

	switch {
	case key.Matches(msg, m.keys.PrevView): // h/left - move to previous column
		if m.boardColumn > 0 {
			m.boardColumn--
			// Clamp row to new column's bounds
			newCount := currentColumnCount()
			if m.boardRow >= newCount {
				m.boardRow = newCount - 1
			}
			if m.boardRow < 0 {
				m.boardRow = 0
			}
			selectionChanged = true
		}

	case key.Matches(msg, m.keys.NextView): // l/right - move to next column
		if m.boardColumn < 2 {
			m.boardColumn++
			// Clamp row to new column's bounds
			newCount := currentColumnCount()
			if m.boardRow >= newCount {
				m.boardRow = newCount - 1
			}
			if m.boardRow < 0 {
				m.boardRow = 0
			}
			selectionChanged = true
		}

	case key.Matches(msg, m.keys.Up): // k/up - move up in column
		if m.boardRow > 0 {
			m.boardRow--
			selectionChanged = true
		}

	case key.Matches(msg, m.keys.Down): // j/down - move down in column
		count := currentColumnCount()
		if m.boardRow < count-1 {
			m.boardRow++
			selectionChanged = true
		}

	case key.Matches(msg, m.keys.Top): // g - go to top
		if m.boardRow != 0 {
			m.boardRow = 0
			selectionChanged = true
		}

	case key.Matches(msg, m.keys.Bottom): // G - go to bottom
		count := currentColumnCount()
		if count > 0 && m.boardRow != count-1 {
			m.boardRow = count - 1
			selectionChanged = true
		}

	case key.Matches(msg, m.keys.Select): // enter - view task details (full screen in narrow mode)
		task := m.getBoardSelectedTask()
		if task != nil {
			m.selected = task
			m.comments = nil
			m.updateDetailContent()
			m.mode = ViewDetail
			return m.loadComments(task.ID)
		}

	case key.Matches(msg, m.keys.Board): // b - toggle back to list view
		m.mode = ViewList

	case key.Matches(msg, m.keys.Help):
		m.mode = ViewHelp

	case key.Matches(msg, m.keys.Cancel): // esc - back to list
		m.mode = ViewList
	}

	// Sync selected task with detail panel after navigation
	if selectionChanged {
		m.selected = m.getBoardSelectedTask()
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

// matchCustomCommand checks if the key matches any custom command for the given context
func (m *Model) matchCustomCommand(msg tea.KeyMsg, context string) tea.Cmd {
	keyStr := msg.String()
	for _, cmd := range m.customCommands {
		if cmd.Key == keyStr && (cmd.Context == context || cmd.Context == "global") {
			return m.executeCustomCommand(cmd)
		}
	}
	return nil
}

// executeCustomCommand renders and executes a custom command
func (m *Model) executeCustomCommand(cmd config.CustomCommand) tea.Cmd {
	task := m.getSelectedTask()
	if task == nil {
		return nil
	}

	// Render command template
	rendered, err := m.renderCommandTemplate(cmd.Command, task)
	if err != nil {
		m.err = fmt.Errorf("template error: %w", err)
		return nil
	}

	// Execute command non-blocking (for tmux commands)
	c := exec.Command("sh", "-c", rendered)
	if err := c.Start(); err != nil {
		m.err = fmt.Errorf("failed to execute command: %w", err)
	}

	return nil
}

// shellEscape escapes a string for safe use in shell commands
// Escapes single quotes, double quotes, backticks, and dollar signs
func shellEscape(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, `'`, `'\''`)
	s = strings.ReplaceAll(s, "`", "\\`")
	s = strings.ReplaceAll(s, `$`, `\$`)
	return s
}

// renderCommandTemplate renders the command template with task data
func (m *Model) renderCommandTemplate(cmdTemplate string, task *models.Task) (string, error) {
	funcMap := template.FuncMap{
		"sh": shellEscape,
	}

	tmpl, err := template.New("cmd").Funcs(funcMap).Parse(cmdTemplate)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, task); err != nil {
		return "", err
	}

	return buf.String(), nil
}
