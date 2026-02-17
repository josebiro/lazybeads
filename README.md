# BusyBeads (aka bb)

A terminal user interface for managing [beads](https://github.com/steveyegge/beads) issues, inspired by [LazyGit](https://github.com/jesseduffield/lazygit).

Navigate, create, and manage your project issues without leaving the terminal.

Originally forked from [LazyBeads](https://github.com/codegangsta/lazybeads) by [Jeremy Saenz](https://github.com/codegangsta). BusyBeads has since diverged significantly with hierarchical tree views, board view, dependency management, and other features.

<img width="1405" height="814" alt="Screenshot 2026-02-17 120848" src="https://github.com/user-attachments/assets/90dcf4c2-e413-4a48-b8bf-5cf9b4d633eb" />

<img width="1404" height="808" alt="Screenshot 2026-02-17 120908" src="https://github.com/user-attachments/assets/f8ab222d-adc9-4aa1-bca8-515c5cbef2a4" />

## Features

- **Three-panel layout** - See In Progress, Open, and Closed issues at a glance
- **Hierarchical tree view** - Expand/collapse epics to view child tasks and subtasks
- **Board view** - Kanban-style columns (Blocked, Open, Ready, In Progress, Done)
- **Vim-style navigation** - `j/k` to move, `h/l` to switch panels
- **Mouse support** - Click to select, open details, or toggle tree nodes
- **Quick editing** - Edit title, status, priority, type, description, or notes with single keystrokes
- **Filter & search** - Filter by text (`/`), or preset views: ready, open, closed, all
- **Sorting** - Cycle through sort modes (priority, updated)
- **Dependencies** - Add and remove blockers between issues
- **Comments** - View and add comments inline
- **Detail view** - Press `Enter` to see full issue details with comments
- **External editor** - Edit descriptions and notes with `$EDITOR`
- **Custom commands** - Define your own keybindings with template variables

## Installation

### Prerequisites

- Go 1.25+
- [`bd` CLI](https://github.com/steveyegge/beads) installed and available in PATH

### From source

```bash
go install github.com/josebiro/bb@latest
```

Or clone and build:

```bash
git clone https://github.com/josebiro/bb
cd bb
make install
```

## Usage

Run `bb` in any directory with beads initialized:

```bash
cd your-project
bb
```

If beads isn't initialized, you'll be prompted to set it up.

### Validation mode

Verify the bd CLI integration works:

```bash
bb --check
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

### Editing (from list or detail view)

| Key | Action |
|-----|--------|
| `e` | Edit title |
| `s` | Edit status |
| `p` | Edit priority |
| `t` | Edit type |
| `d` | Edit description |
| `n` | Edit notes |
| `y` | Copy issue ID to clipboard |

### Comments & Dependencies

| Key | Action |
|-----|--------|
| `C` | Add comment |
| `B` | Add blocker |
| `D` | Remove blocker |

### Tree View

| Key | Action |
|-----|--------|
| `Space` | Expand / collapse epic children |

### Filtering & Sorting

| Key | Action |
|-----|--------|
| `/` | Start text filter |
| `r` | Show ready issues |
| `o` | Show open issues |
| `O` | Show closed issues |
| `A` | Show all issues |
| `S` | Cycle sort mode |

### Views

| Key | Action |
|-----|--------|
| `b` | Toggle board view |
| `?` | Show help |
| `Esc` | Go back / cancel |
| `q` | Quit |

## Configuration

bb looks for a configuration file at:

1. `$BB_CONFIG` (if set, direct path to config file)
2. `~/.config/bb/config.yml` (default)

### Custom commands

Define custom keybindings that execute shell commands. Template variables from the selected issue are available.

```yaml
customCommands:
  - key: "w"
    description: "Start work in new tmux pane"
    context: "list"  # list, detail, or global
    command: "tmux split-window -h 'claude --issue {{.ID}}'"

  - key: "W"
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

## Project Structure

```
bb/
├── main.go              # Entry point, CLI flags
├── internal/
│   ├── app/             # Bubble Tea application model
│   ├── beads/           # bd CLI wrapper
│   ├── config/          # Configuration loading
│   ├── models/          # Data models and hierarchy utils
│   └── ui/              # UI components and styles
└── .beads/              # Issue storage (managed by bd)
```

## Acknowledgments

bb is built on [LazyBeads](https://github.com/codegangsta/lazybeads) by [Jeremy Saenz](https://github.com/codegangsta), which provided the original three-panel TUI design and bd CLI integration.

## Related Projects

- [beads](https://github.com/steveyegge/beads) - Git-native issue tracking
- [LazyBeads](https://github.com/codegangsta/lazybeads) - The original TUI for beads
- [LazyGit](https://github.com/jesseduffield/lazygit) - Terminal UI for git
- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - TUI framework for Go

## License

MIT
