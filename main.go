package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/josebiro/bb/internal/app"
	"github.com/josebiro/bb/internal/beads"
	"github.com/josebiro/bb/internal/config"
)

func main() {
	checkMode := flag.Bool("check", false, "Run headless validation (test bd CLI integration)")
	configMode := flag.Bool("config", false, "Show config loading status and diagnostics")
	flag.Parse()

	// Config diagnostics mode (runs before beads check)
	if *configMode {
		showConfigStatus()
		return
	}

	client := beads.NewClient()

	// Check if beads is initialized
	if !client.IsInitialized() {
		if *checkMode {
			fmt.Println("FAIL: Beads is not initialized in this directory")
			os.Exit(1)
		}

		fmt.Println("Beads is not initialized in this directory.")
		fmt.Print("Would you like to initialize it now? [y/N] ")

		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))

		if response == "y" || response == "yes" {
			if err := client.Init(); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to initialize beads: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("Beads initialized successfully!")
		} else {
			fmt.Println("Run 'bd init' to initialize beads, then try again.")
			os.Exit(0)
		}
	}

	// Headless validation mode
	if *checkMode {
		runCheck(client)
		return
	}

	// Create and run the TUI application
	p := tea.NewProgram(
		app.New(),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running bb: %v\n", err)
		os.Exit(1)
	}
}

// runCheck performs headless validation of the beads client
func runCheck(client *beads.Client) {
	fmt.Println("Running bb validation...")
	fmt.Println()

	failed := false

	// Test 1: List tasks
	fmt.Print("  List tasks: ")
	tasks, err := client.List()
	if err != nil {
		fmt.Printf("FAIL (%v)\n", err)
		failed = true
	} else {
		fmt.Printf("OK (%d tasks)\n", len(tasks))
	}

	// Test 2: List open tasks
	fmt.Print("  List open tasks: ")
	openTasks, err := client.ListOpen()
	if err != nil {
		fmt.Printf("FAIL (%v)\n", err)
		failed = true
	} else {
		fmt.Printf("OK (%d open)\n", len(openTasks))
	}

	// Test 3: Ready tasks
	fmt.Print("  Ready tasks: ")
	readyTasks, err := client.Ready()
	if err != nil {
		fmt.Printf("FAIL (%v)\n", err)
		failed = true
	} else {
		fmt.Printf("OK (%d ready)\n", len(readyTasks))
	}

	// Test 4: Create task
	fmt.Print("  Create task: ")
	task, err := client.Create(beads.CreateOptions{
		Title:    "__bb_check_task__",
		Type:     "task",
		Priority: 4,
	})
	if err != nil {
		fmt.Printf("FAIL (%v)\n", err)
		failed = true
	} else {
		fmt.Printf("OK (created %s)\n", task.ID)

		// Test 5: Show task
		fmt.Print("  Show task: ")
		shown, err := client.Show(task.ID)
		if err != nil {
			fmt.Printf("FAIL (%v)\n", err)
			failed = true
		} else if shown.ID != task.ID {
			fmt.Printf("FAIL (ID mismatch)\n")
			failed = true
		} else {
			fmt.Println("OK")
		}

		// Test 6: Update task
		fmt.Print("  Update task: ")
		err = client.Update(task.ID, beads.UpdateOptions{
			Status: "in_progress",
		})
		if err != nil {
			fmt.Printf("FAIL (%v)\n", err)
			failed = true
		} else {
			fmt.Println("OK")
		}

		// Test 7: Close task
		fmt.Print("  Close task: ")
		err = client.Close(task.ID, "check completed")
		if err != nil {
			fmt.Printf("FAIL (%v)\n", err)
			failed = true
		} else {
			fmt.Println("OK")
		}

		// Test 8: Delete task
		fmt.Print("  Delete task: ")
		err = client.Delete(task.ID)
		if err != nil {
			fmt.Printf("FAIL (%v)\n", err)
			failed = true
		} else {
			fmt.Println("OK")
		}
	}

	fmt.Println()
	if failed {
		fmt.Println("VALIDATION FAILED")
		os.Exit(1)
	}
	fmt.Println("All checks passed!")
}

// showConfigStatus displays configuration loading diagnostics
func showConfigStatus() {
	fmt.Println("Config Status")

	// Show BB_CONFIG env var
	envValue := os.Getenv("BB_CONFIG")
	if envValue == "" {
		fmt.Println("  BB_CONFIG: (not set)")
	} else {
		fmt.Printf("  BB_CONFIG: %s\n", envValue)
	}

	// Show resolved config path
	configPath := config.ConfigPath()
	fmt.Printf("  Config path:      %s\n", configPath)

	// Check if file exists
	_, statErr := os.Stat(configPath)
	fileExists := statErr == nil
	if fileExists {
		fmt.Println("  File exists:      yes")
	} else {
		fmt.Println("  File exists:      no")
	}

	// Attempt to parse and show status
	var cfg *config.Config
	var parseErr error
	if fileExists {
		cfg, parseErr = config.Load()
		if parseErr != nil {
			fmt.Printf("  Parse status:     error (%v)\n", parseErr)
		} else {
			fmt.Println("  Parse status:     ok")
		}
	} else {
		fmt.Println("  Parse status:     n/a (no config file)")
	}

	fmt.Println()

	// Show custom commands
	if cfg != nil && len(cfg.CustomCommands) > 0 {
		fmt.Printf("Custom Commands (%d loaded)\n", len(cfg.CustomCommands))
		for _, cmd := range cfg.CustomCommands {
			fmt.Printf("  %s  %q  (%s)\n", cmd.Key, cmd.Description, cmd.Context)
		}
	} else {
		fmt.Println("Custom Commands (0 loaded)")
		fmt.Println("  (none)")
	}
}
