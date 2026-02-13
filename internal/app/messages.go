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
	tasks    []models.Task
	readyIDs map[string]bool
	err      error
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

// clipboardCopiedMsg is sent when text is copied to clipboard
type clipboardCopiedMsg struct {
	text string
	err  error
}

// clearStatusMsg clears the status flash message
type clearStatusMsg struct{}

// commentsLoadedMsg is sent when comments are loaded for a task
type commentsLoadedMsg struct {
	comments []models.Comment
	err      error
}

// commentAddedMsg is sent when a comment is added
type commentAddedMsg struct {
	err error
}

// blockerAddedMsg is sent when a blocker is added
type blockerAddedMsg struct {
	err error
}

// blockerRemovedMsg is sent when a blocker is removed
type blockerRemovedMsg struct {
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
		// Use --limit=0 to bypass the default 50-task limit
		tasks, err := m.client.List("--all", "--limit=0")
		if err != nil {
			return tasksLoadedMsg{err: err}
		}

		// Also load ready task IDs for board view column categorization
		readyIDs := make(map[string]bool)
		readyTasks, readyErr := m.client.Ready()
		if readyErr == nil {
			for _, t := range readyTasks {
				readyIDs[t.ID] = true
			}
		}

		return tasksLoadedMsg{tasks: tasks, readyIDs: readyIDs, err: err}
	}
}

// loadComments creates a command to load comments for a task
func (m Model) loadComments(taskID string) tea.Cmd {
	return func() tea.Msg {
		comments, err := m.client.GetComments(taskID)
		return commentsLoadedMsg{comments: comments, err: err}
	}
}
