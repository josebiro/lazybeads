package ui

import (
	"github.com/charmbracelet/glamour"
)

var mdRenderer *glamour.TermRenderer

func init() {
	// Initialize markdown renderer with dark style
	r, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(0), // We'll handle wrapping ourselves
	)
	if err != nil {
		// Fallback: no rendering
		return
	}
	mdRenderer = r
}

// RenderMarkdown renders markdown text to styled terminal output
func RenderMarkdown(text string, width int) string {
	if mdRenderer == nil || text == "" {
		return text
	}

	// Create a new renderer with the specific width
	r, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return text
	}

	rendered, err := r.Render(text)
	if err != nil {
		return text
	}

	return rendered
}
