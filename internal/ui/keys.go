package ui

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines all keybindings
type KeyMap struct {
	// Navigation
	Up       key.Binding
	Down     key.Binding
	Top      key.Binding
	Bottom   key.Binding
	PageUp   key.Binding
	PageDown key.Binding

	// Actions
	Select  key.Binding
	Add     key.Binding
	Delete  key.Binding
	Refresh key.Binding

	// Field-specific editing
	EditTitle       key.Binding
	EditStatus      key.Binding
	EditPriority    key.Binding
	EditType        key.Binding
	EditDescription key.Binding
	EditNotes       key.Binding
	AddComment      key.Binding
	CopyID          key.Binding

	// Dependency management
	AddBlocker    key.Binding
	RemoveBlocker key.Binding

	// Filtering
	Filter     key.Binding
	FilterDone key.Binding
	Ready      key.Binding
	Open       key.Binding
	Closed     key.Binding
	All        key.Binding

	// Sorting
	Sort key.Binding

	// Tree
	ToggleExpand key.Binding

	// Views
	Board key.Binding

	// UI
	Help      key.Binding
	Quit      key.Binding
	Cancel    key.Binding
	Submit    key.Binding
	Tab       key.Binding
	ShiftTab  key.Binding
	PrevView  key.Binding
	NextView  key.Binding

	// Custom commands (loaded from config)
	CustomCommands []key.Binding
}

// DefaultKeyMap returns the default keybindings
func DefaultKeyMap() KeyMap {
	return KeyMap{
		// Navigation (vim-style)
		Up: key.NewBinding(
			key.WithKeys("k", "up"),
			key.WithHelp("j/k", "navigate"),
		),
		Down: key.NewBinding(
			key.WithKeys("j", "down"),
			key.WithHelp("", ""),
		),
		Top: key.NewBinding(
			key.WithKeys("g"),
			key.WithHelp("g/G", "top/bottom"),
		),
		Bottom: key.NewBinding(
			key.WithKeys("G"),
			key.WithHelp("", ""),
		),
		PageUp: key.NewBinding(
			key.WithKeys("ctrl+u", "pgup"),
			key.WithHelp("^u/^d", "page up/down"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("ctrl+d", "pgdown"),
			key.WithHelp("", ""),
		),

		// Actions
		Select: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "details"),
		),
		Add: key.NewBinding(
			key.WithKeys("a", "c"),
			key.WithHelp("a/c", "add"),
		),
		Delete: key.NewBinding(
			key.WithKeys("x"),
			key.WithHelp("x", "delete"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("R"),
			key.WithHelp("R", "refresh"),
		),

		// Field-specific editing
		EditTitle: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "edit title"),
		),
		EditStatus: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "edit status"),
		),
		EditPriority: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "edit priority"),
		),
		EditType: key.NewBinding(
			key.WithKeys("t"),
			key.WithHelp("t", "edit type"),
		),
		CopyID: key.NewBinding(
			key.WithKeys("y"),
			key.WithHelp("y", "copy id"),
		),
		EditDescription: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "edit description"),
		),
		EditNotes: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "edit notes"),
		),
		AddComment: key.NewBinding(
			key.WithKeys("C"),
			key.WithHelp("C", "add comment"),
		),

		// Dependency management
		AddBlocker: key.NewBinding(
			key.WithKeys("B"),
			key.WithHelp("B", "add blocker"),
		),
		RemoveBlocker: key.NewBinding(
			key.WithKeys("D"),
			key.WithHelp("D", "remove blocker"),
		),

		// Filtering
		Filter: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "filter"),
		),
		FilterDone: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("", ""),
		),
		Ready: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "ready"),
		),
		Open: key.NewBinding(
			key.WithKeys("o"),
			key.WithHelp("o", "open"),
		),
		Closed: key.NewBinding(
			key.WithKeys("O"),
			key.WithHelp("O", "closed"),
		),
		All: key.NewBinding(
			key.WithKeys("A"),
			key.WithHelp("A", "all"),
		),

		// Sorting
		Sort: key.NewBinding(
			key.WithKeys("S"),
			key.WithHelp("S", "cycle sort"),
		),

		// Tree
		ToggleExpand: key.NewBinding(
			key.WithKeys(" "),
			key.WithHelp("space", "expand/collapse"),
		),

		// Views
		Board: key.NewBinding(
			key.WithKeys("b"),
			key.WithHelp("b", "board view"),
		),

		// UI
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Cancel: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "cancel"),
		),
		Submit: key.NewBinding(
			key.WithKeys("ctrl+s"),
			key.WithHelp("^s", "submit"),
		),
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "next field"),
		),
		ShiftTab: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("shift+tab", "prev field"),
		),
		PrevView: key.NewBinding(
			key.WithKeys("h", "left"),
			key.WithHelp("h/l", "prev/next view"),
		),
		NextView: key.NewBinding(
			key.WithKeys("l", "right"),
			key.WithHelp("", ""),
		),
	}
}

// ShortHelp returns keybindings for compact help view
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		k.Select,
		k.Add,
		k.EditTitle,
		k.EditStatus,
		k.EditDescription,
		k.EditNotes,
		k.Delete,
		k.Help,
		k.Quit,
	}
}

// FullHelp returns keybindings for expanded help view
func (k KeyMap) FullHelp() [][]key.Binding {
	groups := [][]key.Binding{
		{k.Up, k.Down, k.Top, k.Bottom, k.PageUp, k.PageDown},
		{k.Select, k.Add, k.Delete, k.Refresh},
		{k.EditTitle, k.EditStatus, k.EditPriority, k.EditType, k.EditDescription, k.EditNotes},
		{k.AddComment, k.CopyID, k.AddBlocker, k.RemoveBlocker},
		{k.Filter, k.Ready, k.Open, k.Closed, k.All, k.Sort},
		{k.Board, k.Help, k.Quit, k.Cancel},
	}
	// Add custom commands as a separate group if present
	if len(k.CustomCommands) > 0 {
		groups = append(groups, k.CustomCommands)
	}
	return groups
}
