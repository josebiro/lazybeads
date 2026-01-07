package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

// InlineBarType defines the type of inline bar
type InlineBarType int

const (
	InlineBarInput InlineBarType = iota
	InlineBarSelect
)

// InlineBarOption represents an option in a select inline bar
type InlineBarOption struct {
	Label    string
	Value    string
	Shortcut string // Single key shortcut (e.g., "0", "1", "2")
}

// InlineBar represents a bottom bar for inline editing
type InlineBar struct {
	Type     InlineBarType
	Title    string
	Subtitle string // e.g., issue ID

	// For input bars
	Input textinput.Model

	// For select bars
	Options  []InlineBarOption
	Selected int
}

// NewInlineBarInput creates a new text input inline bar
func NewInlineBarInput(title, subtitle, value string, width int) InlineBar {
	ti := textinput.New()
	ti.SetValue(value)
	ti.Focus()
	ti.CharLimit = 200
	ti.Width = width - 20 // Account for title and padding

	return InlineBar{
		Type:     InlineBarInput,
		Title:    title,
		Subtitle: subtitle,
		Input:    ti,
	}
}

// NewInlineBarSelect creates a new select inline bar
func NewInlineBarSelect(title, subtitle string, options []InlineBarOption, currentValue string) InlineBar {
	selected := 0
	for i, opt := range options {
		if opt.Value == currentValue {
			selected = i
			break
		}
	}

	return InlineBar{
		Type:     InlineBarSelect,
		Title:    title,
		Subtitle: subtitle,
		Options:  options,
		Selected: selected,
	}
}

// MoveLeft moves selection left in select bar
func (b *InlineBar) MoveLeft() {
	if b.Type == InlineBarSelect && b.Selected > 0 {
		b.Selected--
	}
}

// MoveRight moves selection right in select bar
func (b *InlineBar) MoveRight() {
	if b.Type == InlineBarSelect && b.Selected < len(b.Options)-1 {
		b.Selected++
	}
}

// SelectByShortcut selects an option by its shortcut key
// Returns true if a shortcut matched
func (b *InlineBar) SelectByShortcut(key string) bool {
	if b.Type != InlineBarSelect {
		return false
	}
	for i, opt := range b.Options {
		if opt.Shortcut == key {
			b.Selected = i
			return true
		}
	}
	return false
}

// SelectedValue returns the currently selected value
func (b InlineBar) SelectedValue() string {
	if b.Type == InlineBarSelect && b.Selected >= 0 && b.Selected < len(b.Options) {
		return b.Options[b.Selected].Value
	}
	return ""
}

// InputValue returns the input value
func (b InlineBar) InputValue() string {
	return b.Input.Value()
}

// View renders the inline bar (single line, full width)
func (b InlineBar) View(width int) string {
	var content strings.Builder

	// Title and subtitle
	titleStyle := lipgloss.NewStyle().
		Foreground(ColorPrimary).
		Bold(true)
	subtitleStyle := lipgloss.NewStyle().
		Foreground(ColorMuted).
		Italic(true)

	content.WriteString(titleStyle.Render(b.Title))
	if b.Subtitle != "" {
		content.WriteString(" ")
		content.WriteString(subtitleStyle.Render(b.Subtitle))
	}
	content.WriteString(": ")

	if b.Type == InlineBarInput {
		// Text input
		inputStyle := lipgloss.NewStyle().
			Foreground(ColorWhite)
		content.WriteString(inputStyle.Render(b.Input.View()))
		content.WriteString("  ")
		content.WriteString(HelpDescStyle.Render("enter:save esc:cancel"))
	} else {
		// Horizontal select options
		for i, opt := range b.Options {
			var optText string
			if opt.Shortcut != "" {
				optText = "[" + opt.Shortcut + "]" + opt.Label
			} else {
				optText = opt.Label
			}

			if i == b.Selected {
				style := lipgloss.NewStyle().
					Foreground(ColorPrimary).
					Bold(true).
					Reverse(true).
					Padding(0, 1)
				content.WriteString(style.Render(optText))
			} else {
				style := lipgloss.NewStyle().
					Foreground(ColorWhite).
					Padding(0, 1)
				content.WriteString(style.Render(optText))
			}
		}
		content.WriteString("  ")
		content.WriteString(HelpDescStyle.Render("h/l:nav enter:select esc:cancel"))
	}

	// Render with background style
	barStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("0")).
		Width(width).
		Padding(0, 1)

	return barStyle.Render(content.String())
}
