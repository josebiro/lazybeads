package beads

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/josebiro/bb/internal/models"
)

// Client wraps the bd CLI commands
type Client struct{}

// NewClient creates a new beads client
func NewClient() *Client {
	return &Client{}
}

// IsInitialized checks if beads is initialized in current directory
func (c *Client) IsInitialized() bool {
	_, err := os.Stat(".beads")
	return err == nil
}

// Init initializes beads in current directory
func (c *Client) Init() error {
	cmd := exec.Command("bd", "init")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

// List returns tasks with optional filters
func (c *Client) List(filters ...string) ([]models.Task, error) {
	args := []string{"list", "--json"}
	args = append(args, filters...)

	out, err := exec.Command("bd", args...).Output()
	if err != nil {
		return nil, fmt.Errorf("bd list failed: %w", err)
	}

	var tasks []models.Task
	if err := json.Unmarshal(out, &tasks); err != nil {
		return nil, fmt.Errorf("failed to parse bd list output: %w", err)
	}

	return tasks, nil
}

// ListOpen returns all open tasks
func (c *Client) ListOpen() ([]models.Task, error) {
	return c.List("--status=open")
}

// Ready returns tasks with no blockers
func (c *Client) Ready() ([]models.Task, error) {
	args := []string{"ready", "--json"}

	out, err := exec.Command("bd", args...).Output()
	if err != nil {
		return nil, fmt.Errorf("bd ready failed: %w", err)
	}

	var tasks []models.Task
	if err := json.Unmarshal(out, &tasks); err != nil {
		return nil, fmt.Errorf("failed to parse bd ready output: %w", err)
	}

	return tasks, nil
}

// Show returns details for a specific task
func (c *Client) Show(id string) (*models.Task, error) {
	out, err := exec.Command("bd", "show", id, "--json").Output()
	if err != nil {
		return nil, fmt.Errorf("bd show failed: %w", err)
	}

	// bd show returns an array with single item
	var tasks []models.Task
	if err := json.Unmarshal(out, &tasks); err != nil {
		return nil, fmt.Errorf("failed to parse bd show output: %w", err)
	}

	if len(tasks) == 0 {
		return nil, fmt.Errorf("task not found: %s", id)
	}

	return &tasks[0], nil
}

// CreateOptions holds options for creating a task
type CreateOptions struct {
	Title       string
	Description string
	Type        string // task, bug, feature, epic, chore
	Priority    int    // 0-4
	Labels      []string
}

// Create creates a new task
func (c *Client) Create(opts CreateOptions) (*models.Task, error) {
	args := []string{"create", "--title", opts.Title, "--json"}

	if opts.Type != "" {
		args = append(args, "--type", opts.Type)
	}
	if opts.Priority >= 0 && opts.Priority <= 4 {
		args = append(args, "--priority", fmt.Sprintf("%d", opts.Priority))
	}
	if opts.Description != "" {
		args = append(args, "-d", opts.Description)
	}
	if len(opts.Labels) > 0 {
		args = append(args, "-l", strings.Join(opts.Labels, ","))
	}

	out, err := exec.Command("bd", args...).Output()
	if err != nil {
		return nil, fmt.Errorf("bd create failed: %w", err)
	}

	// bd create returns a single task object
	var task models.Task
	if err := json.Unmarshal(out, &task); err != nil {
		return nil, fmt.Errorf("failed to parse bd create output: %w", err)
	}

	return &task, nil
}

// UpdateOptions holds options for updating a task
type UpdateOptions struct {
	Status      string
	Priority    *int
	Title       string
	Assignee    string
	Type        string
	Description string
	Notes       string
}

// Update modifies an existing task
func (c *Client) Update(id string, opts UpdateOptions) error {
	args := []string{"update", id}

	if opts.Status != "" {
		args = append(args, "--status", opts.Status)
	}
	if opts.Priority != nil {
		args = append(args, "--priority", fmt.Sprintf("%d", *opts.Priority))
	}
	if opts.Title != "" {
		args = append(args, "--title", opts.Title)
	}
	if opts.Assignee != "" {
		args = append(args, "--assignee", opts.Assignee)
	}
	if opts.Type != "" {
		args = append(args, "--type", opts.Type)
	}
	if opts.Description != "" {
		args = append(args, "-d", opts.Description)
	}
	if opts.Notes != "" {
		args = append(args, "--notes", opts.Notes)
	}

	cmd := exec.Command("bd", args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("bd update failed: %w", err)
	}

	return nil
}

// Close marks a task as completed
func (c *Client) Close(id string, reason string) error {
	args := []string{"close", id}
	if reason != "" {
		args = append(args, "--reason", reason)
	}

	cmd := exec.Command("bd", args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("bd close failed: %w", err)
	}

	return nil
}

// Delete removes a task
func (c *Client) Delete(id string) error {
	cmd := exec.Command("bd", "delete", id, "--force")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("bd delete failed: %w", err)
	}

	return nil
}

// GetComments returns all comments for a task
func (c *Client) GetComments(id string) ([]models.Comment, error) {
	out, err := exec.Command("bd", "comments", id, "--json").Output()
	if err != nil {
		return nil, fmt.Errorf("bd comments failed: %w", err)
	}

	var comments []models.Comment
	if err := json.Unmarshal(out, &comments); err != nil {
		return nil, fmt.Errorf("failed to parse bd comments output: %w", err)
	}

	return comments, nil
}

// AddComment adds a comment to a task
func (c *Client) AddComment(id string, text string) error {
	cmd := exec.Command("bd", "comments", "add", id, text)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("bd comments add failed: %w", err)
	}

	return nil
}

// AddBlocker adds a dependency (blocker blocks blockee)
func (c *Client) AddBlocker(blockee string, blocker string) error {
	cmd := exec.Command("bd", "dep", "add", blockee, blocker)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("bd dep add failed: %w", err)
	}

	return nil
}

// RemoveBlocker removes a dependency
func (c *Client) RemoveBlocker(blockee string, blocker string) error {
	cmd := exec.Command("bd", "dep", "rm", blockee, blocker)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("bd dep rm failed: %w", err)
	}

	return nil
}
