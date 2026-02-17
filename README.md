# BusyBeads (aka bb)

A terminal user interface for managing [beads](https://github.com/anthropics/beads) issues, inspired by [LazyGit](https://github.com/jesseduffield/lazygit), this was originally a fork of [LazyBeads](https://github.com/codegangsta/lazybeads).

Navigate, create, and manage your project issues without leaving the terminal. Heavily uses charmbracelet libraries for a responsive and intuitive experience.

<img width="2550" height="1362" alt="CleanShot 2026-01-07 at 18 17 53@2x" src="https://github.com/user-attachments/assets/2b6fdc40-4595-41f3-9a73-e3eb9de00cfc" />


## Features

- **Three-panel layout** - See In Progress, Open, and Closed issues at a glance
- **Hierarchical issues** - Expand epics to view child tasks and subtasks
- **Board view** - Visualize issue status kanban-style with color-coded priorities
- **Vim-style navigation** - `j/k` to move, `h/l` to switch panels
- **Mouse support** - Click to select, open details, or trigger actions
- **Quick editing** - Edit title, status, priority, or type with single keystrokes
- **Filter & search** - Use `/` to filter issues by title or ID
- **Detail view** - Press `Enter` to see full issue details
- **External editor** - Edit descriptions with `$EDITOR` (defaults to nano)
- **Custom commands** - Define your own keybindings for workflows

## Installation

### Prerequisites

- Go 1.25+
- `bd` CLI installed and available in PATH

### From source

```bash
go install github.com/youruser/lazybeads@latest
```

Or clone and build:

```bash
git clone https://github.com/youruser/lazybeads
cd lazybeads
go install .
```

## Usage

Run `lazybeads` in any directory with beads initialized:

```bash
cd your-project
lazybeads
```

If beads isn't initialized, you'll be prompted to set it up.

### Validation mode

Verify the bd CLI integration works:

```bash
lazybeads --check
```

## Keybindings

### Navigation

| Key | Action |
|-----|--------|
| `j` / `k` | Move down / up |
| `g` / `G` | Jump to top / bottom |
| `Ctrl+u` / `Ctrl+d` | Page up / down |
| `h` / `l` | Previous / next panel |

### Actions

| Key | Action |
|-----|--------|
| `Enter` | View issue details |
| `a` or `c` | Create new issue |
| `x` | Delete issue |
| `R` | Refresh list |

### Quick edit

| Key | Action |
|-----|--------|
| `t` | Edit title |
| `s` | Edit status |
| `p` | Edit priority |
| `T` | Edit type |
| `d` or `e` | Edit description (opens $EDITOR) |
| `y` | Copy issue ID to clipboard |

### Filter

| Key | Action |
|-----|--------|
| `/` | Start filter |
| `Esc` | Clear filter |

### General

| Key | Action |
|-----|--------|
| `?` | Show help |
| `Esc` | Go back / cancel |
| `q` | Quit |

## Configuration

LazyBeads looks for a configuration file at:

- `$LAZYBEADS_CONFIG` (if set)
- `~/.config/lazybeads/config.yml` (default)

### Custom commands

Define custom keybindings that execute shell commands. Template variables from the selected issue are available.

```yaml
customCommands:
  - key: "w"
    description: "Start work in new tmux pane"
    context: "list"  # list, detail, or global
    command: "tmux split-window -h 'claude --issue {{.ID}}'"

  - key: "b"
    description: "Open in browser"
    context: "list"
    command: "open 'https://your-tracker.com/issues/{{.ID}}'"
```

Available template variables:

- `{{.ID}}` - Issue ID
- `{{.Title}}` - Issue title
- `{{.Status}}` - Status (open, in_progress, closed)
- `{{.Type}}` - Type (task, bug, feature, epic, chore)
- `{{.Priority}}` - Priority (0-4)
- `{{.Description}}` - Full description

## Project structure

```
lazybeads/
├── main.go              # Entry point, CLI flags
├── internal/
│   ├── app/             # Bubble Tea application model
│   ├── beads/           # bd CLI wrapper
│   ├── config/          # Configuration loading
│   ├── models/          # Data models
│   └── ui/              # UI components and styles
└── .beads/              # Issue storage (managed by bd)
```

## Related projects

- [beads](https://github.com/anthropics/beads) - Git-native issue tracking
- [LazyGit](https://github.com/jesseduffield/lazygit) - Terminal UI for git
- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - TUI framework for Go

## License

MIT
