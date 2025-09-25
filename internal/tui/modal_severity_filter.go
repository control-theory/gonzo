package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// renderSeverityFilterModal renders the severity filter selection modal
func (m *DashboardModel) renderSeverityFilterModal() string {
	// Calculate dimensions - smaller modal
	modalWidth := min(m.width-16, 50)  // Smaller width for severity list
	modalHeight := min(m.height-8, 18) // Smaller height

	// Account for borders and headers
	contentWidth := modalWidth - 4   // Modal borders
	contentHeight := modalHeight - 4 // Header + status

	// Define severity levels in order (most critical first)
	severityLevels := []string{"FATAL", "CRITICAL", "ERROR", "WARN", "INFO", "DEBUG", "TRACE", "UNKNOWN"}

	// Build severity list content
	var severityLines []string

	// Add "Select All" option at the top
	selectAllPrefix := "  "
	if m.severityFilterSelected == 0 {
		selectAllPrefix = "► "
	}
	allSelected := true
	for _, severity := range severityLevels {
		if !m.severityFilter[severity] {
			allSelected = false
			break
		}
	}
	selectAllStatus := ""
	if allSelected {
		selectAllStatus = " ✓"
	}
	selectAllLine := selectAllPrefix + "Select All" + selectAllStatus

	// Style the select all line
	if m.severityFilterSelected == 0 {
		selectedStyle := lipgloss.NewStyle().
			Foreground(ColorBlue).
			Bold(true)
		selectAllLine = selectedStyle.Render(selectAllLine)
	}
	severityLines = append(severityLines, selectAllLine)

	// Add "Select None" option
	selectNonePrefix := "  "
	if m.severityFilterSelected == 1 {
		selectNonePrefix = "► "
	}
	noneSelected := true
	for _, severity := range severityLevels {
		if m.severityFilter[severity] {
			noneSelected = false
			break
		}
	}
	selectNoneStatus := ""
	if noneSelected {
		selectNoneStatus = " ✓"
	}
	selectNoneLine := selectNonePrefix + "Select None" + selectNoneStatus

	// Style the select none line
	if m.severityFilterSelected == 1 {
		selectedStyle := lipgloss.NewStyle().
			Foreground(ColorBlue).
			Bold(true)
		selectNoneLine = selectedStyle.Render(selectNoneLine)
	}
	severityLines = append(severityLines, selectNoneLine)

	// Add separator
	severityLines = append(severityLines, "")

	// Add individual severity levels (starting from index 2)
	for i, severity := range severityLevels {
		listIndex := i + 3 // Offset by 3 (select all + select none + separator)
		prefix := "  "
		if m.severityFilterSelected == listIndex {
			prefix = "► "
		}

		// Show selection status
		status := ""
		if m.severityFilter[severity] {
			status = " ✓"
		}

		line := prefix + severity + status

		// Apply severity color and selection styling
		severityColor := getSeverityColor(severity)
		if m.severityFilterSelected == listIndex {
			// Highlight selected item
			selectedStyle := lipgloss.NewStyle().
				Foreground(ColorBlue).
				Bold(true)
			line = selectedStyle.Render(line)
		} else {
			// Use severity color for non-selected items
			severityStyle := lipgloss.NewStyle().
				Foreground(severityColor)
			line = severityStyle.Render(line)
		}

		severityLines = append(severityLines, line)
	}

	// Create content pane
	contentPane := lipgloss.NewStyle().
		Width(contentWidth).
		Height(contentHeight).
		Border(lipgloss.NormalBorder()).
		BorderForeground(ColorBlue).
		Render(strings.Join(severityLines, "\n"))

	// Header
	activeCount := 0
	for _, enabled := range m.severityFilter {
		if enabled {
			activeCount++
		}
	}
	headerText := fmt.Sprintf("Severity Filter (%d/%d active)", activeCount, len(severityLevels))
	header := lipgloss.NewStyle().
		Width(contentWidth).
		Foreground(ColorBlue).
		Bold(true).
		Render(headerText)

	// Status bar
	statusBar := lipgloss.NewStyle().
		Foreground(ColorGray).
		Render("↑↓: Navigate • Space: Toggle • Enter: Apply/Select • ESC: Cancel")

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