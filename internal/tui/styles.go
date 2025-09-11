package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// Color palette - these will be updated by the skin system
var (
	ColorBlue     = lipgloss.Color("#0f93fc")
	ColorGreen    = lipgloss.Color("#49E209")
	ColorNavy     = lipgloss.Color("#081C39")
	ColorGray     = lipgloss.Color("#BCBEC0")
	ColorDarkGray = lipgloss.Color("#2D2D2D") // Dark gray for modal backgrounds
	ColorBlack    = lipgloss.Color("#000000")
	ColorWhite    = lipgloss.Color("#FFFFFF")
	ColorRed      = lipgloss.Color("#FF6B6B")
	ColorYellow   = lipgloss.Color("#FFD93D")
	ColorOrange   = lipgloss.Color("#FF8C42")
	ColorPink     = lipgloss.Color("#FF69B4")
)

// Shared styles used across multiple view components
// These will be recreated when colors change
var (
	sectionStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(ColorGray).
			Padding(0, 1).
			Margin(0) // Remove horizontal margins to use more space

	activeSectionStyle = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder()).
				BorderForeground(ColorBlue).
				Padding(0, 1).
				Margin(0) // Remove horizontal margins to use more space

	helpStyle = lipgloss.NewStyle().
			Foreground(ColorGray).
			Italic(true).
			Padding(1)

	chartTitleStyle = lipgloss.NewStyle().
			Foreground(ColorBlue).
			Bold(true).
			Align(lipgloss.Center)
)
