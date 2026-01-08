package app

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"lazybeads/internal/beads"
	"lazybeads/internal/models"
	"lazybeads/internal/ui"
)

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
	case ViewFilter:
		return m.handleFilterKeys(msg)
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

	case key.Matches(msg, m.keys.Filter):
		// Enter filter mode with an inline bar
		m.inlineBar = ui.NewInlineBarInput("Filter", "", m.filterQuery, m.width)
		m.mode = ViewFilter

	case key.Matches(msg, m.keys.CopyID):
		if task := m.getSelectedTask(); task != nil {
			taskID := task.ID
			return func() tea.Msg {
				err := clipboard.WriteAll(taskID)
				return clipboardCopiedMsg{text: taskID, err: err}
			}
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

func (m *Model) handleFilterKeys(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "enter":
		// Apply filter and return to list
		m.filterQuery = strings.TrimSpace(m.inlineBar.InputValue())
		m.distributeTasks()
		m.mode = ViewList
	case "esc":
		// Clear filter and return to list
		m.filterQuery = ""
		m.distributeTasks()
		m.mode = ViewList
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
