package ui

import "github.com/charmbracelet/lipgloss"

// Colors - lazygit-inspired theme
var (
	ColorPrimary    = lipgloss.Color("2")       // Green (selected/active)
	ColorSecondary  = lipgloss.Color("4")       // Blue (options/help keys)
	ColorAccent     = lipgloss.Color("6")       // Cyan (search/accent)
	ColorWarning    = lipgloss.Color("3")       // Yellow
	ColorDanger     = lipgloss.Color("1")       // Red
	ColorMuted      = lipgloss.Color("8")       // Bright black (gray)
	ColorWhite      = lipgloss.Color("7")       // White
	ColorMagenta    = lipgloss.Color("5")       // Magenta
	ColorBorder     = lipgloss.Color("8")       // Gray border
)

// Priority colors
var PriorityColors = map[int]lipgloss.Color{
	0: ColorDanger,    // P0 - Critical (red)
	1: ColorWarning,   // P1 - High (yellow)
	2: ColorSecondary, // P2 - Medium (blue)
	3: ColorMuted,     // P3 - Low (gray)
	4: ColorMuted,     // P4 - Backlog (gray)
}

// Status colors
var StatusColors = map[string]lipgloss.Color{
	"open":        ColorPrimary, // Green
	"in_progress": ColorWarning, // Yellow
	"closed":      ColorMuted,   // Gray
}

// Base styles
var (
	// App container
	AppStyle = lipgloss.NewStyle().
			Padding(0, 1)

	// Title bar
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary).
			Padding(0, 1)

	// Panel styles
	PanelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorder).
			Padding(0, 1)

	FocusedPanelStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorPrimary).
				Bold(true).
				Padding(0, 1)

	PanelTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorWhite).
			MarginBottom(1)

	// Task list item styles
	TaskItemStyle = lipgloss.NewStyle().
			PaddingLeft(2)

	SelectedTaskStyle = lipgloss.NewStyle().
				PaddingLeft(1).
				Foreground(ColorAccent).
				Bold(true)

	TaskIDStyle = lipgloss.NewStyle().
			Foreground(ColorMuted).
			Width(12)

	TaskTitleStyle = lipgloss.NewStyle().
			Foreground(ColorWhite)

	// Status bar
	StatusBarStyle = lipgloss.NewStyle().
			Foreground(ColorMuted).
			Padding(0, 1).
			MarginTop(1)

	// Help bar at bottom
	HelpBarStyle = lipgloss.NewStyle().
			Foreground(ColorMuted).
			Padding(0, 1)

	HelpKeyStyle = lipgloss.NewStyle().
			Foreground(ColorSecondary).
			Bold(true)

	HelpDescStyle = lipgloss.NewStyle().
			Foreground(ColorWhite)

	// Detail view
	DetailLabelStyle = lipgloss.NewStyle().
				Foreground(ColorSecondary).
				Bold(true).
				Width(12)

	DetailValueStyle = lipgloss.NewStyle().
				Foreground(ColorWhite)

	// Form styles
	FormLabelStyle = lipgloss.NewStyle().
			Foreground(ColorSecondary).
			Bold(true).
			MarginRight(1)

	FormInputStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorder).
			Padding(0, 1)

	FormInputFocusedStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorPrimary).
				Padding(0, 1)

	// Overlay/modal
	OverlayStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorPrimary).
			Padding(1, 2)

	// Error/message styles
	ErrorStyle = lipgloss.NewStyle().
			Foreground(ColorDanger).
			Bold(true)

	SuccessStyle = lipgloss.NewStyle().
			Foreground(ColorPrimary)
)

// PriorityStyle returns a styled priority string
func PriorityStyle(priority int) lipgloss.Style {
	color, ok := PriorityColors[priority]
	if !ok {
		color = ColorMuted
	}
	return lipgloss.NewStyle().
		Foreground(color).
		Bold(priority <= 1) // Bold for P0/P1
}

// StatusStyle returns a styled status string
func StatusStyle(status string) lipgloss.Style {
	color, ok := StatusColors[status]
	if !ok {
		color = ColorMuted
	}
	return lipgloss.NewStyle().Foreground(color)
}
