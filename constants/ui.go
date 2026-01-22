package constants

import "github.com/charmbracelet/lipgloss"

// TezCapital brand colors based on gradient: #4d609b -> #233366 -> #000b2c
// Using AdaptiveColor to support both light and dark terminal backgrounds
var (
	// Primary accent color - lighter blue from the gradient
	// Dark terminals: lighter shade, Light terminals: original brand color
	ColorPrimary = lipgloss.AdaptiveColor{Light: "#233366", Dark: "#4d609b"}

	// Secondary color - mid-tone blue
	ColorSecondary = lipgloss.AdaptiveColor{Light: "#000b2c", Dark: "#233366"}

	// Text colors - adaptive for readability
	ColorText     = lipgloss.AdaptiveColor{Light: "#1a1a2e", Dark: "#E0E6F0"} // Main text
	ColorTextDim  = lipgloss.AdaptiveColor{Light: "#4a5568", Dark: "#8891A8"} // Muted/description text
	ColorTextHint = lipgloss.AdaptiveColor{Light: "#718096", Dark: "#5A6478"} // Help text

	// Cursor/selection indicator - brighter for visibility
	ColorCursor = lipgloss.AdaptiveColor{Light: "#2d3a6d", Dark: "#7B8FBF"}
)

// TUI styles used across the application
var (
	// StyleTitle is used for TUI headers/titles
	StyleTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary).
			MarginBottom(1)

	// StyleSelected is used for currently selected items
	StyleSelected = lipgloss.NewStyle().
			Foreground(ColorCursor).
			Bold(true)

	// StyleNormal is used for unselected items
	StyleNormal = lipgloss.NewStyle().
			Foreground(ColorText)

	// StyleDim is used for descriptions and secondary text
	StyleDim = lipgloss.NewStyle().
			Foreground(ColorTextDim)

	// StyleHelp is used for help text at the bottom
	StyleHelp = lipgloss.NewStyle().
			Foreground(ColorTextHint).
			MarginTop(1)

	// StylePrompt is used for prompt messages
	StylePrompt = lipgloss.NewStyle().
			Foreground(ColorPrimary)

	// StyleHint is used for hints like [Y/n]
	StyleHint = lipgloss.NewStyle().
			Foreground(ColorTextDim)
)
