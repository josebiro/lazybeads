package beads

import (
	"os"
	"testing"
)

// These tests require bd to be installed and run in a beads-initialized directory
// Skip if not in a valid environment

func skipIfNoBeads(t *testing.T) {
	t.Helper()
	if _, err := os.Stat(".beads"); os.IsNotExist(err) {
		// Try parent directories up to 3 levels
		for _, dir := range []string{"..", "../..", "../../.."} {
			if _, err := os.Stat(dir + "/.beads"); err == nil {
				if err := os.Chdir(dir); err == nil {
					return
				}
			}
		}
		t.Skip("No .beads directory found, skipping integration test")
	}
}

func TestClient_IsInitialized(t *testing.T) {
	skipIfNoBeads(t)
	client := NewClient()

	if !client.IsInitialized() {
		t.Error("Expected IsInitialized to return true in beads directory")
	}
}

func TestClient_List(t *testing.T) {
	skipIfNoBeads(t)
	client := NewClient()

	tasks, err := client.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	t.Logf("Found %d tasks", len(tasks))
	for _, task := range tasks {
		t.Logf("  - %s: %s (status=%s, priority=%d, type=%s)",
			task.ID, task.Title, task.Status, task.Priority, task.Type)
	}
}

func TestClient_ListOpen(t *testing.T) {
	skipIfNoBeads(t)
	client := NewClient()

	tasks, err := client.ListOpen()
	if err != nil {
		t.Fatalf("ListOpen failed: %v", err)
	}

	t.Logf("Found %d open tasks", len(tasks))
	for _, task := range tasks {
		if task.Status != "open" {
			t.Errorf("Expected status 'open', got '%s' for task %s", task.Status, task.ID)
		}
	}
}

func TestClient_Ready(t *testing.T) {
	skipIfNoBeads(t)
	client := NewClient()

	tasks, err := client.Ready()
	if err != nil {
		t.Fatalf("Ready failed: %v", err)
	}

	t.Logf("Found %d ready tasks", len(tasks))
}

func TestClient_Show(t *testing.T) {
	skipIfNoBeads(t)
	client := NewClient()

	// First get a task ID from list
	tasks, err := client.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(tasks) == 0 {
		t.Skip("No tasks to show")
	}

	task, err := client.Show(tasks[0].ID)
	if err != nil {
		t.Fatalf("Show failed: %v", err)
	}

	if task.ID != tasks[0].ID {
		t.Errorf("Expected ID %s, got %s", tasks[0].ID, task.ID)
	}

	t.Logf("Showed task: %s - %s", task.ID, task.Title)
}

func TestClient_CreateAndDelete(t *testing.T) {
	skipIfNoBeads(t)
	client := NewClient()

	// Create a test task
	task, err := client.Create(CreateOptions{
		Title:       "Test task from client_test.go",
		Description: "This is a test task",
		Type:        "task",
		Priority:    3,
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	t.Logf("Created task: %s - %s", task.ID, task.Title)

	if task.Title != "Test task from client_test.go" {
		t.Errorf("Expected title 'Test task from client_test.go', got '%s'", task.Title)
	}
	if task.Priority != 3 {
		t.Errorf("Expected priority 3, got %d", task.Priority)
	}
	if task.Type != "task" {
		t.Errorf("Expected type 'task', got '%s'", task.Type)
	}

	// Clean up - delete the task
	err = client.Delete(task.ID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	t.Log("Deleted test task")
}

func TestClient_Update(t *testing.T) {
	skipIfNoBeads(t)
	client := NewClient()

	// Create a test task
	task, err := client.Create(CreateOptions{
		Title:    "Update test task",
		Type:     "task",
		Priority: 2,
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	defer client.Delete(task.ID)

	// Update the task
	newPriority := 1
	err = client.Update(task.ID, UpdateOptions{
		Status:   "in_progress",
		Priority: &newPriority,
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// Verify the update
	updated, err := client.Show(task.ID)
	if err != nil {
		t.Fatalf("Show failed: %v", err)
	}

	if updated.Status != "in_progress" {
		t.Errorf("Expected status 'in_progress', got '%s'", updated.Status)
	}
	if updated.Priority != 1 {
		t.Errorf("Expected priority 1, got %d", updated.Priority)
	}

	t.Log("Update test passed")
}

func TestClient_Close(t *testing.T) {
	skipIfNoBeads(t)
	client := NewClient()

	// Create a test task
	task, err := client.Create(CreateOptions{
		Title:    "Close test task",
		Type:     "task",
		Priority: 3,
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	defer client.Delete(task.ID)

	// Close the task
	err = client.Close(task.ID, "Test completed")
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Verify the close
	closed, err := client.Show(task.ID)
	if err != nil {
		t.Fatalf("Show failed: %v", err)
	}

	if closed.Status != "closed" {
		t.Errorf("Expected status 'closed', got '%s'", closed.Status)
	}

	t.Log("Close test passed")
}
