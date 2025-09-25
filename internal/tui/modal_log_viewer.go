package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// renderLogViewerModal renders the log viewer in a fullscreen modal
func (m *DashboardModel) renderLogViewerModal() string {
	// Calculate modal dimensions - leave space for borders
	modalWidth := m.width - 4   // Leave margin for borders
	modalHeight := m.height - 2 // Leave margin for borders

	// Inner content dimensions (accounting for borders)
	contentWidth := modalWidth - 2   // -2 for left/right borders
	contentHeight := modalHeight - 2 // -2 for top/bottom borders

	// Reserve space for header and status
	headerHeight := 1
	statusHeight := 1
	logAreaHeight := contentHeight - headerHeight - statusHeight

	// Get log content without border wrapper
	logLines := m.renderLogScrollContent(logAreaHeight, contentWidth)

	// Create header
	header := lipgloss.NewStyle().
		Foreground(ColorBlue).
		Bold(true).
		Width(contentWidth).
		Render("Log Viewer")

	// Create log content area with fixed height
	logArea := lipgloss.NewStyle().
		Width(contentWidth).
		Height(logAreaHeight).
		Render(lipgloss.JoinVertical(lipgloss.Left, logLines...))

	// Create status line with filter/search indicators
	var statusLeft string

	// Check for active filter/search (including while being typed)
	hasActiveFilter := m.filterActive || m.filterRegex != nil || m.filterInput.Value() != ""
	hasActiveSearch := m.searchActive || m.searchTerm != "" || m.searchInput.Value() != ""

	// Build status message
	statusParts := []string{fmt.Sprintf("Total: %d", len(m.logEntries))}

	if m.viewPaused {
		statusParts = append(statusParts, "⏸ PAUSED")
	}

	if hasActiveFilter {
		if m.filterActive {
			// Currently editing filter
			filterValue := m.filterInput.Value()
			if filterValue == "" {
				statusParts = append(statusParts, "🔍 Filter: (editing...)")
			} else {
				statusParts = append(statusParts, fmt.Sprintf("🔍 Filter: [%s] (editing)", filterValue))
			}
		} else if m.filterRegex != nil {
			// Filter applied
			statusParts = append(statusParts, fmt.Sprintf("🔍 Filter: [%s] (%d/%d)",
				m.filterInput.Value(), len(m.logEntries), len(m.allLogEntries)))
		}
	}

	if hasActiveSearch {
		if m.searchActive {
			// Currently editing search
			searchValue := m.searchInput.Value()
			if searchValue == "" {
				statusParts = append(statusParts, "🔎 Search: (editing...)")
			} else {
				statusParts = append(statusParts, fmt.Sprintf("🔎 Search: [%s] (editing)", searchValue))
			}
		} else if m.searchTerm != "" {
			// Search applied
			statusParts = append(statusParts, fmt.Sprintf("🔎 Search: [%s]", m.searchTerm))
		}
	}

	statusLeft = strings.Join(statusParts, " | ")

	// Create concise help text that fits
	helpText := "ESC:Close ↑↓:Nav Enter:Details /:Filter s:Search c:Columns"

	// Calculate available space for each side
	leftWidth := lipgloss.Width(statusLeft)
	rightWidth := lipgloss.Width(helpText)

	// If combined width exceeds available space, truncate
	if leftWidth+rightWidth+2 > contentWidth {
		// Prioritize showing status on left, truncate help on right
		availableForRight := contentWidth - leftWidth - 2
		if availableForRight < 20 {
			// If very little space, just show essential help
			helpText = "ESC:Close ?:Help"
		} else if availableForRight < 40 {
			helpText = "ESC:Close ↑↓:Nav /:Filter"
		}
	}

	// Create properly sized status sections
	padding := contentWidth - lipgloss.Width(statusLeft) - lipgloss.Width(helpText)
	if padding < 0 {
		padding = 0
	}

	statusBar := lipgloss.NewStyle().
		Foreground(ColorGray).
		Width(contentWidth).
		MaxWidth(contentWidth).
		Height(statusHeight).
		MaxHeight(statusHeight).
		Render(statusLeft + strings.Repeat(" ", padding) + helpText)

	// Combine all content
	content := lipgloss.JoinVertical(lipgloss.Left,
		header,
		logArea,
		statusBar,
	)

	// Apply border to the content - don't set height to allow content to define size
	modal := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(ColorBlue).
		Width(modalWidth).
		Render(content)

	// Center the modal on screen
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, modal)
}