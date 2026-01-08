package app

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"text/template"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"lazybeads/internal/beads"
	"lazybeads/internal/config"
	"lazybeads/internal/models"
	"lazybeads/internal/ui"
)

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
