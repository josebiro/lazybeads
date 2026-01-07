package models

import "time"

// Task represents a beads issue
type Task struct {
	ID              string     `json:"id"`
	Title           string     `json:"title"`
	Description     string     `json:"description,omitempty"`
	Status          string     `json:"status"`
	Priority        int        `json:"priority"`
	Type            string     `json:"issue_type"`
	Labels          []string   `json:"labels,omitempty"`
	Assignee        string     `json:"assignee,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	CreatedBy       string     `json:"created_by,omitempty"`
	UpdatedAt       time.Time  `json:"updated_at"`
	ClosedAt        *time.Time `json:"closed_at,omitempty"`
	DueDate         *time.Time `json:"due_date,omitempty"`
	DeferUntil      *time.Time `json:"defer_until,omitempty"`
	BlockedBy       []string   `json:"blocked_by,omitempty"`
	Blocks          []string   `json:"blocks,omitempty"`
	DependencyCount int        `json:"dependency_count,omitempty"`
	DependentCount  int        `json:"dependent_count,omitempty"`
}

// PriorityString returns a short priority label
func (t Task) PriorityString() string {
	switch t.Priority {
	case 0:
		return "P0"
	case 1:
		return "P1"
	case 2:
		return "P2"
	case 3:
		return "P3"
	case 4:
		return "P4"
	default:
		return "P?"
	}
}

// StatusIcon returns a status indicator
func (t Task) StatusIcon() string {
	switch t.Status {
	case "open":
		return "○"
	case "in_progress":
		return "◐"
	case "closed":
		return "●"
	default:
		return "?"
	}
}

// IsBlocked returns true if task has blockers
func (t Task) IsBlocked() bool {
	return len(t.BlockedBy) > 0
}
