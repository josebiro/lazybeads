package app

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/josebiro/lazybeads/internal/models"
)

const pollInterval = 2 * time.Second
const statusFlashDuration = 1 * time.Second

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

// clipboardCopiedMsg is sent when text is copied to clipboard
type clipboardCopiedMsg struct {
	text string
	err  error
}

// clearStatusMsg clears the status flash message
type clearStatusMsg struct{}

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
