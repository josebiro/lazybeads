package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

// ModalType defines the type of modal
type ModalType int

const (
	ModalTypeInput ModalType = iota
	ModalTypeSelect
)

// SelectOption represents an option in a select modal
type SelectOption struct {
	Label string
	Value string
}

// Modal represents a centered overlay modal
type Modal struct {
	Type     ModalType
	Title    string
	Subtitle string // e.g., issue ID to show context
	Width    int

	// For input modals
	Input textinput.Model

	// For select modals
	Options  []SelectOption
	Selected int
}

// NewInputModal creates a new text input modal
func NewInputModal(title string, value string, width int) Modal {
	ti := textinput.New()
	ti.SetValue(value)
	ti.Focus()
	ti.CharLimit = 200
	ti.Width = width - 6 // Account for borders and padding

	return Modal{
		Type:  ModalTypeInput,
		Title: title,
		Width: width,
		Input: ti,
	}
}

// NewSelectModal creates a new select modal
func NewSelectModal(title string, options []SelectOption, currentValue string, width int) Modal {
	selected := 0
	for i, opt := range options {
		if opt.Value == currentValue {
			selected = i
			break
		}
	}

	return Modal{
		Type:     ModalTypeSelect,
		Title:    title,
		Width:    width,
		Options:  options,
		Selected: selected,
	}
}

// MoveUp moves selection up in select modal
func (m *Modal) MoveUp() {
	if m.Type == ModalTypeSelect && m.Selected > 0 {
		m.Selected--
	}
}

// MoveDown moves selection down in select modal
func (m *Modal) MoveDown() {
	if m.Type == ModalTypeSelect && m.Selected < len(m.Options)-1 {
		m.Selected++
	}
}

// SelectedValue returns the currently selected value
func (m Modal) SelectedValue() string {
	if m.Type == ModalTypeSelect && m.Selected >= 0 && m.Selected < len(m.Options) {
		return m.Options[m.Selected].Value
	}
	return ""
}

// InputValue returns the input value
func (m Modal) InputValue() string {
	return m.Input.Value()
}

// View renders the modal
func (m Modal) View() string {
	var content strings.Builder

	// Modal border style
	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorPrimary).
		Padding(1, 2)

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorPrimary).
		MarginBottom(1)

	// Build content based on type
	content.WriteString(titleStyle.Render(m.Title) + "\n")
	if m.Subtitle != "" {
		subtitleStyle := lipgloss.NewStyle().Foreground(ColorMuted).Italic(true)
		content.WriteString(subtitleStyle.Render(m.Subtitle) + "\n")
	}
	content.WriteString("\n")

	if m.Type == ModalTypeInput {
		// Text input
		inputStyle := lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(ColorSecondary).
			Padding(0, 1).
			Width(m.Width - 8)
		content.WriteString(inputStyle.Render(m.Input.View()) + "\n\n")
		content.WriteString(HelpDescStyle.Render("enter: save  esc: cancel"))
	} else {
		// Select options
		for i, opt := range m.Options {
			var line string
			if i == m.Selected {
				line = lipgloss.NewStyle().
					Foreground(ColorPrimary).
					Bold(true).
					Render("> " + opt.Label)
			} else {
				line = lipgloss.NewStyle().
					Foreground(ColorWhite).
					Render("  " + opt.Label)
			}
			content.WriteString(line + "\n")
		}
		content.WriteString("\n")
		// Show number keys hint for Priority modal
		if m.Title == "Priority" {
			content.WriteString(HelpDescStyle.Render("0-4: select  j/k: navigate  enter: select  esc: cancel"))
		} else {
			content.WriteString(HelpDescStyle.Render("j/k: navigate  enter: select  esc: cancel"))
		}
	}

	// Apply border and return (centering is handled by the caller)
	return borderStyle.Width(m.Width).Render(content.String())
}
