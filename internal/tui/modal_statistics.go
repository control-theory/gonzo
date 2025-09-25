package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// renderStatsModal renders the statistics modal showing comprehensive log stats
func (m *DashboardModel) renderStatsModal() string {
	// Calculate dimensions
	modalWidth := m.width - 8   // Leave 4 chars margin on each side
	modalHeight := m.height - 4 // Leave 2 lines margin top and bottom

	// Account for borders and headers
	contentWidth := modalWidth - 4   // Modal borders
	contentHeight := modalHeight - 4 // Header + status

	// Update viewport
	m.infoViewport.Width = contentWidth
	m.infoViewport.Height = contentHeight

	// Get statistics content and set it to viewport
	statsContent := m.renderStatsContent(contentWidth)
	m.infoViewport.SetContent(statsContent)

	// Create content pane
	contentPane := lipgloss.NewStyle().
		Width(contentWidth).
		Height(contentHeight).
		Border(lipgloss.NormalBorder()).
		BorderForeground(ColorGray).
		Render(m.infoViewport.View())

	// Header with title
	header := lipgloss.NewStyle().
		Width(contentWidth).
		Foreground(ColorBlue).
		Bold(true).
		Render("Log Statistics")

	// Status bar
	statusBar := lipgloss.NewStyle().
		Foreground(ColorGray).
		Render("↑↓/Wheel: Scroll • PgUp/PgDn: Page • i: Toggle Stats • ESC: Close")

	// Combine all parts
	modal := lipgloss.JoinVertical(lipgloss.Left, header, contentPane, statusBar)

	// Add outer border and center
	finalModal := lipgloss.NewStyle().
		Width(modalWidth).
		Height(modalHeight).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorBlue).
		Render(modal)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, finalModal)
}

// Note: renderStatsContent is defined in stats.go