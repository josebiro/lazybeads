package beads

import (
	"bytes"
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

// runBD executes a bd command and returns stdout, returning a descriptive
// error that includes stderr output when the command fails.
func runBD(args ...string) ([]byte, error) {
	cmd := exec.Command("bd", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		msg := stderr.String()
		if msg != "" {
			return nil, fmt.Errorf("bd %s failed: %w: %s", args[0], err, strings.TrimSpace(msg))
		}
		return nil, fmt.Errorf("bd %s failed: %w", args[0], err)
	}
	return out, nil
}

// parseTasks unmarshals JSON output into a task slice. It handles both a
// bare JSON array and an object wrapper (e.g. {"issues": [...]}).
func parseTasks(out []byte) ([]models.Task, error) {
	out = bytes.TrimSpace(out)
	if len(out) == 0 {
		return nil, nil
	}

	// Try bare array first.
	var tasks []models.Task
	if err := json.Unmarshal(out, &tasks); err == nil {
		return tasks, nil
	}

	// Try object wrapper with common key names.
	var wrapper map[string]json.RawMessage
	if err := json.Unmarshal(out, &wrapper); err != nil {
		preview := string(out)
		if len(preview) > 200 {
			preview = preview[:200] + "..."
		}
		return nil, fmt.Errorf("failed to parse bd output as JSON: %w\nraw output: %s", err, preview)
	}

	for _, key := range []string{"issues", "data", "results", "items"} {
		if raw, ok := wrapper[key]; ok {
			if err := json.Unmarshal(raw, &tasks); err == nil {
				return tasks, nil
			}
		}
	}

	// Fallback: try the first array-valued key.
	for _, raw := range wrapper {
		if len(raw) > 0 && raw[0] == '[' {
			if err := json.Unmarshal(raw, &tasks); err == nil {
				return tasks, nil
			}
		}
	}

	preview := string(out)
	if len(preview) > 200 {
		preview = preview[:200] + "..."
	}
	return nil, fmt.Errorf("bd JSON output has unexpected structure: %s", preview)
}

// List returns tasks with optional filters
func (c *Client) List(filters ...string) ([]models.Task, error) {
	args := []string{"list", "--json", "--flat"}
	args = append(args, filters...)

	out, err := runBD(args...)
	if err != nil {
		return nil, err
	}

	return parseTasks(out)
}

// ListOpen returns all open tasks
func (c *Client) ListOpen() ([]models.Task, error) {
	return c.List("--status=open")
}

// Ready returns tasks with no blockers
func (c *Client) Ready() ([]models.Task, error) {
	out, err := runBD("ready", "--json")
	if err != nil {
		return nil, err
	}

	return parseTasks(out)
}

// Show returns details for a specific task
func (c *Client) Show(id string) (*models.Task, error) {
	out, err := runBD("show", id, "--json")
	if err != nil {
		return nil, err
	}

	// bd show may return an array with single item or a single object.
	tasks, parseErr := parseTasks(out)
	if parseErr == nil {
		if len(tasks) == 0 {
			return nil, fmt.Errorf("task not found: %s", id)
		}
		return &tasks[0], nil
	}

	// Try single object.
	out = bytes.TrimSpace(out)
	var task models.Task
	if err := json.Unmarshal(out, &task); err != nil {
		return nil, parseErr // return original error
	}
	return &task, nil
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

	out, err := runBD(args...)
	if err != nil {
		return nil, err
	}

	// bd create may return a single object or a wrapped object.
	out = bytes.TrimSpace(out)
	if len(out) == 0 {
		return nil, fmt.Errorf("bd create returned empty output")
	}

	var task models.Task
	if err := json.Unmarshal(out, &task); err == nil && task.ID != "" {
		return &task, nil
	}

	// Fallback: try parsing as array (in case format changed).
	tasks, parseErr := parseTasks(out)
	if parseErr == nil && len(tasks) > 0 {
		return &tasks[0], nil
	}

	preview := string(out)
	if len(preview) > 200 {
		preview = preview[:200] + "..."
	}
	return nil, fmt.Errorf("failed to parse bd create output: %s", preview)
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

	_, err := runBD(args...)
	return err
}

// Close marks a task as completed
func (c *Client) Close(id string, reason string) error {
	args := []string{"close", id}
	if reason != "" {
		args = append(args, "--reason", reason)
	}

	_, err := runBD(args...)
	return err
}

// Delete removes a task
func (c *Client) Delete(id string) error {
	_, err := runBD("delete", id, "--force")
	return err
}

// GetComments returns all comments for a task
func (c *Client) GetComments(id string) ([]models.Comment, error) {
	out, err := runBD("comments", id, "--json")
	if err != nil {
		return nil, err
	}

	out = bytes.TrimSpace(out)
	if len(out) == 0 {
		return nil, nil
	}

	// Try bare array first.
	var comments []models.Comment
	if err := json.Unmarshal(out, &comments); err == nil {
		return comments, nil
	}

	// Try object wrapper.
	var wrapper map[string]json.RawMessage
	if err := json.Unmarshal(out, &wrapper); err == nil {
		for _, key := range []string{"comments", "data", "results", "items"} {
			if raw, ok := wrapper[key]; ok {
				if err := json.Unmarshal(raw, &comments); err == nil {
					return comments, nil
				}
			}
		}
	}

	preview := string(out)
	if len(preview) > 200 {
		preview = preview[:200] + "..."
	}
	return nil, fmt.Errorf("failed to parse bd comments output: %s", preview)
}

// AddComment adds a comment to a task
func (c *Client) AddComment(id string, text string) error {
	_, err := runBD("comments", "add", id, text)
	return err
}

// AddBlocker adds a dependency (blocker blocks blockee)
func (c *Client) AddBlocker(blockee string, blocker string) error {
	_, err := runBD("dep", "add", blockee, blocker)
	return err
}

// RemoveBlocker removes a dependency
func (c *Client) RemoveBlocker(blockee string, blocker string) error {
	_, err := runBD("dep", "rm", blockee, blocker)
	return err
}
