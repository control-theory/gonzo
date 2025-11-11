package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// hasFilterOrSearch returns true if a filter or search is active or applied
func (m *DashboardModel) hasFilterOrSearch() bool {
	return m.filterActive || m.searchActive || 
		m.filterRegex != nil || m.filterInput.Value() != "" || 
		m.searchTerm != "" || m.searchInput.Value() != ""
}

// View renders the dashboard
func (m *DashboardModel) View() string {
	if m.width <= 0 || m.height <= 0 {
		return "Initializing dashboard..."
	}

	// Show help modal
	if m.showHelp {
		return m.renderHelpModal()
	}

	// Show patterns modal
	if m.showPatternsModal {
		return m.renderPatternsModal()
	}
	
	// Show statistics modal
	if m.showStatsModal {
		return m.renderStatsModal()
	}
	
	// Show counts modal
	if m.showCountsModal {
		return m.renderCountsModal()
	}
	
	// Show severity filter modal (check before log viewer so it can overlay)
	if m.showSeverityFilterModal {
		return m.renderSeverityFilterModal()
	}

	// Show log viewer modal (fullscreen log viewer)
	if m.showLogViewerModal {
		return m.renderLogViewerModal()
	}
	
	// Show model selection modal
	if m.showModelSelectionModal {
		return m.renderModelSelectionModal()
	}

	// Show detail modal - use lipgloss overlay
	if m.showModal {
		return m.renderModalOverlay()
	}

	// Main dashboard layout
	return m.renderDashboard()
}

// renderDashboard renders the main dashboard layout
func (m *DashboardModel) renderDashboard() string {
	// Ensure minimum height
	if m.height < 20 {
		return "Terminal too small. Resize to at least 20 lines."
	}

	// Calculate required space for charts dynamically
	requiredChartsHeight := m.calculateRequiredChartsHeight()

	// Filter/Search height depends on whether filter or search is applied (or being edited)
	filterHeight := 0 // No space when inactive
	if m.hasFilterOrSearch() {
		filterHeight = 1 // Single row for filter/search
	}

	// Reserve space for status line at bottom
	statusLineHeight := 1

	// Use full height for proper layout
	usableHeight := m.height - statusLineHeight - 2 // Use full height minus status line (minus 2 because.. I have no idea why)
	logsHeight := usableHeight - requiredChartsHeight - filterHeight

	// Final allocation - trust the math
	chartsHeight := requiredChartsHeight

	// Ensure minimum log space without breaking layout
	if logsHeight < 3 {
		logsHeight = 3 // Absolute minimum
	}

	// Layout calculations complete

	// Top section: 2x2 grid of charts (VERY constrained height)
	topSection := m.renderChartsGrid(chartsHeight)

	// Middle section: Filter (only when active)
	var sections []string
	sections = append(sections, topSection)

	if m.hasFilterOrSearch() {
		filterSection := m.renderFilter()
		sections = append(sections, filterSection)
	}

	// Bottom section: Log scroll
	logsSection := m.renderLogScroll(logsHeight)
	sections = append(sections, logsSection)

	// Combine sections with strict height constraints
	mainContent := lipgloss.JoinVertical(lipgloss.Left, sections...)

	// Add status line at the very bottom
	statusLine := m.renderStatusLine()

	// Combine main content with status line
	result := lipgloss.JoinVertical(
		lipgloss.Left,
		mainContent,
		statusLine,
	)

	// Apply final height constraint to entire dashboard
	finalStyle := lipgloss.NewStyle().
		Height(m.height).
		MaxWidth(m.width)

	return finalStyle.Render(result)
}

