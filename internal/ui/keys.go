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
	Edit    key.Binding
	Delete  key.Binding
	Close   key.Binding
	Refresh key.Binding

	// Filtering
	Filter     key.Binding
	FilterDone key.Binding
	Ready      key.Binding
	Open       key.Binding
	All        key.Binding

	// UI
	Help      key.Binding
	Quit      key.Binding
	Cancel    key.Binding
	Submit    key.Binding
	Tab       key.Binding
	PrevView  key.Binding
	NextView  key.Binding
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
			key.WithKeys("a"),
			key.WithHelp("a", "add"),
		),
		Edit: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "edit"),
		),
		Delete: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "delete"),
		),
		Close: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "close"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("R"),
			key.WithHelp("R", "refresh"),
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
		All: key.NewBinding(
			key.WithKeys("A"),
			key.WithHelp("A", "all"),
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
		k.Up,
		k.Select,
		k.Add,
		k.Edit,
		k.Delete,
		k.Filter,
		k.Help,
		k.Quit,
	}
}

// FullHelp returns keybindings for expanded help view
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Top, k.Bottom, k.PageUp, k.PageDown},
		{k.Select, k.Add, k.Edit, k.Delete, k.Close, k.Refresh},
		{k.Filter, k.Ready, k.Open, k.All},
		{k.Help, k.Quit, k.Cancel},
	}
}
