package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// renderCountsModal renders the log counts modal with heatmap and analysis sections
func (m *DashboardModel) renderCountsModal() string {
	// Calculate dimensions
	modalWidth := m.width - 8   // Leave 4 chars margin on each side
	modalHeight := m.height - 4 // Leave 2 lines margin top and bottom

	// Account for borders and headers
	contentWidth := modalWidth - 4   // Modal borders
	contentHeight := modalHeight - 4 // Header + status

	// Update viewport
	m.infoViewport.Width = contentWidth
	m.infoViewport.Height = contentHeight

	// Get counts modal content and set it to viewport
	countsContent := m.renderCountsModalContent(contentWidth)
	m.infoViewport.SetContent(countsContent)

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
		Render("Log Counts Analysis")

	// Status bar
	statusBar := lipgloss.NewStyle().
		Foreground(ColorGray).
		Render("↑↓/Wheel: Scroll • PgUp/PgDn: Page • ESC: Close")

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

// renderCountsModalContent renders the content for the counts modal
func (m *DashboardModel) renderCountsModalContent(contentWidth int) string {
	var sections []string

	// Title section
	titleStyle := lipgloss.NewStyle().
		Foreground(ColorBlue).
		Bold(true).
		Align(lipgloss.Center).
		Width(contentWidth)

	sections = append(sections, titleStyle.Render("Log Activity Analysis"))
	sections = append(sections, "")

	// Heatmap section - full width
	heatmapSection := m.renderHeatmapSection(contentWidth)
	sections = append(sections, heatmapSection)
	sections = append(sections, "")

	// Calculate width for side-by-side sections
	halfWidth := (contentWidth - 3) / 2 // -3 for spacing between columns

	// Side-by-side sections: Patterns by Severity | Services by Severity
	patternsSection := m.renderPatternsBySeveritySection(halfWidth)
	servicesSection := m.renderServicesBySeveritySection(halfWidth)

	sideBySide := lipgloss.JoinHorizontal(lipgloss.Top, patternsSection, servicesSection)
	sections = append(sections, sideBySide)

	return strings.Join(sections, "\n")
}

// renderHeatmapSection renders the severity heatmap chart
func (m *DashboardModel) renderHeatmapSection(width int) string {
	// Use chartTitleStyle for consistent title formatting
	titleContent := chartTitleStyle.Render("Severity Activity Heatmap (Last 60 Minutes)")

	var contentLines []string

	// Create real heatmap from actual log data
	now := time.Now()

	// Always render the heatmap structure, even with no data
	// Create time axis header aligned with data (1 character per minute)
	timeHeader := "Time (mins ago):"

	// Build time header with proper 5-minute intervals
	// Create header showing every 5 minutes: 60, 55, 50, 45, 40, 35, 30, 25, 20, 15, 10, 5, 0
	dataHeader := ""
	for i := 60; i >= 0; i-- {
		if i%5 == 0 { // Show every 5 minutes
			if i >= 10 {
				// For 2-digit numbers, show both digits but only use space for tens digit position
				if i%10 == 0 { // Show full number at multiples of 10
					dataHeader += fmt.Sprintf("%2d", i)
					if i > 0 { // Skip next character since we used 2 chars
						i--
					}
				} else {
					dataHeader += " " // Just space for 5, 15, 25, etc.
				}
			} else {
				dataHeader += fmt.Sprintf("%d", i) // Single digit: 5, 0
			}
		} else {
			dataHeader += " " // Empty space for non-labeled minutes
		}
	}

	timeHeader += dataHeader
	contentLines = append(contentLines, timeHeader)
	contentLines = append(contentLines, strings.Repeat("─", len(timeHeader)))

	// Get severity order and colors
	severities := []string{"FATAL", "ERROR", "WARN", "INFO", "DEBUG", "TRACE"}
	colors := map[string]lipgloss.Color{
		"FATAL": ColorRed, "ERROR": ColorRed, "WARN": ColorOrange,
		"INFO": ColorBlue, "DEBUG": ColorGray, "TRACE": ColorGray,
	}

	// Calculate max count per severity for individual scaling
	maxCounts := make(map[string]int)
	totalCounts := make(map[string]int)
	for _, severity := range severities {
		maxCounts[severity] = 1 // Start with 1 to avoid division by zero
		totalCounts[severity] = 0
	}

	for _, minute := range m.heatmapData {
		for _, severity := range severities {
			var count int
			switch severity {
			case "FATAL":
				count = minute.Counts.Fatal + minute.Counts.Critical
			case "ERROR":
				count = minute.Counts.Error
			case "WARN":
				count = minute.Counts.Warn
			case "INFO":
				count = minute.Counts.Info
			case "DEBUG":
				count = minute.Counts.Debug
			case "TRACE":
				count = minute.Counts.Trace
			}
			totalCounts[severity] += count
			if count > maxCounts[severity] {
				maxCounts[severity] = count
			}
		}
	}

	// Calculate total counts for each severity over the 60-minute window
	severityTotals := make(map[string]int)
	for _, severity := range severities {
		total := 0
		for _, minute := range m.heatmapData {
			// Only count minutes within the last 60 minutes
			if minute.Timestamp.After(now.Add(-60 * time.Minute)) {
				switch severity {
				case "FATAL":
					total += minute.Counts.Fatal + minute.Counts.Critical
				case "ERROR":
					total += minute.Counts.Error
				case "WARN":
					total += minute.Counts.Warn
				case "INFO":
					total += minute.Counts.Info
				case "DEBUG":
					total += minute.Counts.Debug
				case "TRACE":
					total += minute.Counts.Trace
				}
			}
		}
		severityTotals[severity] = total
	}

	// Render each severity level row
	for _, severity := range severities {
		// Create severity label with total count
		severityWithCount := fmt.Sprintf("%s (%d)", severity, severityTotals[severity])
		coloredLabel := lipgloss.NewStyle().Foreground(getSeverityColor(severity)).Bold(true).Render(fmt.Sprintf("%-12s", severityWithCount))

		// Align data with time header - "Time (mins ago):" is 16 chars, so we need 16 chars total
		line := coloredLabel + "    " // 12 + 4 = 16 to match header

		// For each minute in the last 60 minutes
		// i=0 represents the current minute and will show real-time updates
		for i := 60; i >= 0; i-- {
			minuteTime := now.Add(time.Duration(-i) * time.Minute).Truncate(time.Minute)

			// Find data for this exact minute
			var minuteActivity int
			found := false
			for _, minute := range m.heatmapData {
				if minute.Timestamp.Equal(minuteTime) {
					found = true
					switch severity {
					case "FATAL":
						minuteActivity = minute.Counts.Fatal + minute.Counts.Critical
					case "ERROR":
						minuteActivity = minute.Counts.Error
					case "WARN":
						minuteActivity = minute.Counts.Warn
					case "INFO":
						minuteActivity = minute.Counts.Info
					case "DEBUG":
						minuteActivity = minute.Counts.Debug
					case "TRACE":
						minuteActivity = minute.Counts.Trace
					}
					break
				}
			}

			// Convert to visual representation using per-severity scaling
			var symbol string
			if !found || minuteActivity == 0 {
				symbol = "." // Single dot for no data
			} else {
				intensity := float64(minuteActivity) / float64(maxCounts[severity])
				if intensity > 0.7 {
					symbol = "█"
				} else if intensity > 0.4 {
					symbol = "▓"
				} else if intensity > 0.1 {
					symbol = "▒"
				} else {
					symbol = "░"
				}

			}

			// Apply color styling only if there's data
			if found && minuteActivity > 0 {
				styledSymbol := lipgloss.NewStyle().Foreground(colors[severity]).Render(symbol)
				line += styledSymbol
			} else {
				line += symbol // No color for dots
			}
		}

		contentLines = append(contentLines, line)
	}

	contentLines = append(contentLines, "")
	contentLines = append(contentLines, "Legend: █ High Activity  ▓ Medium Activity  ▒ Low Activity  . No Activity")

	content := strings.Join(contentLines, "\n")

	// Use sectionStyle for consistent section formatting with borders
	sectionContent := lipgloss.JoinVertical(lipgloss.Left, titleContent, content)

	return sectionStyle.
		Width(width).
		Render(sectionContent)
}

// renderPatternsBySeveritySection renders patterns grouped by severity using drain3 data
func (m *DashboardModel) renderPatternsBySeveritySection(width int) string {
	// Use chartTitleStyle for consistent title formatting
	titleContent := chartTitleStyle.Render("Top Patterns by Severity")

	var contentLines []string

	// Get patterns from severity-specific drain3 instances
	severities := []string{"FATAL", "ERROR", "WARN", "INFO", "DEBUG", "TRACE"}

	hasAnyData := false
	for _, severity := range severities {
		if drain3Instance, exists := m.drain3BySeverity[severity]; exists && drain3Instance != nil {
			patterns := drain3Instance.GetTopPatterns(3) // Get top 3 patterns for this severity
			if len(patterns) > 0 {
				hasAnyData = true

				// Severity header
				severityStyle := lipgloss.NewStyle().Foreground(getSeverityColor(severity)).Bold(true)
				contentLines = append(contentLines, severityStyle.Render(severity+":"))

				// Show patterns for this severity
				for i, pattern := range patterns {
					line := fmt.Sprintf("  %d. %s (%d)", i+1, pattern.Template, pattern.Count)
					contentLines = append(contentLines, line)
				}
				contentLines = append(contentLines, "")
			}
		}
	}

	if !hasAnyData {
		contentLines = append(contentLines, helpStyle.Render("No patterns detected yet..."))
	}

	content := strings.Join(contentLines, "\n")

	// Use sectionStyle for consistent section formatting with borders
	sectionContent := lipgloss.JoinVertical(lipgloss.Left, titleContent, content)

	return sectionStyle.
		Width(width).
		Render(sectionContent)
}

// renderServicesBySeveritySection renders services grouped by severity
func (m *DashboardModel) renderServicesBySeveritySection(width int) string {
	// Use chartTitleStyle for consistent title formatting
	titleContent := chartTitleStyle.Render("Top Services by Severity")

	var contentLines []string

	// Use real service data grouped by severity
	severities := []string{"FATAL", "ERROR", "WARN", "INFO", "DEBUG", "TRACE"}

	hasAnyData := false
	for _, severity := range severities {
		services := m.servicesBySeverity[severity]
		if len(services) > 0 {
			hasAnyData = true

			// Severity header
			severityStyle := lipgloss.NewStyle().Foreground(getSeverityColor(severity)).Bold(true)
			contentLines = append(contentLines, severityStyle.Render(severity+":"))

			// Show top 3 services for this severity
			for i, service := range services {
				if i >= 3 {
					break // Only show top 3
				}
				line := fmt.Sprintf("  %d. %s (%d)", i+1, service.Service, service.Count)
				contentLines = append(contentLines, line)
			}
			contentLines = append(contentLines, "")
		}
	}

	if !hasAnyData {
		contentLines = append(contentLines, helpStyle.Render("No service data available yet..."))
	}

	content := strings.Join(contentLines, "\n")

	// Use sectionStyle for consistent section formatting with borders
	sectionContent := lipgloss.JoinVertical(lipgloss.Left, titleContent, content)

	return sectionStyle.
		Width(width).
		Render(sectionContent)
}