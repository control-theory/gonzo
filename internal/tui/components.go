package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// renderGonzoBranding renders "Gonzo!" with a green to light blue gradient
func (m *DashboardModel) renderGonzoBranding() string {
	// Define gradient colors from green to light blue
	colors := []string{
		"#49E209", // Green (G)
		"#35DD2F", // (o)
		"#21D955", // (n)
		"#0DD47B", // (z)
		"#00D0A1", // (o)
		"#00CAC7", // (!)
	}

	// "Gonzo!" characters
	chars := []string{"G", "o", "n", "z", "o", "!"}

	var result string
	for i, char := range chars {
		// Use gradient colors, cycling through them
		style := lipgloss.NewStyle().
			Background(ColorNavy).
			Foreground(lipgloss.Color(colors[i])).Bold(true)
		result += style.Render(char)
	}

	return result
}

// renderStatusLine renders the status/help line at the bottom of the screen
func (m *DashboardModel) renderStatusLine() string {
	// Create base style for the status line
	baseStyle := lipgloss.NewStyle().
		Background(ColorNavy).
		Foreground(ColorWhite)

	var statusText string
	var leftText string
	var rightText string

	// Determine available width categories
	veryNarrow := m.width < 60
	narrow := m.width < 80
	medium := m.width < 120

	// Build left section (current section indicator)
	sectionNames := map[Section]string{
		SectionWords:        "Words",
		SectionAttributes:   "Attrs",
		SectionDistribution: "Patterns",
		SectionCounts:       "Counts",
		SectionLogs:         "Logs",
		SectionFilter:       "Filter",
	}

	if name, ok := sectionNames[m.activeSection]; ok && !m.filterActive && !m.searchActive {
		if veryNarrow {
			// Use abbreviated names for very narrow terminals
			leftText = name[:min(5, len(name))]
		} else {
			leftText = fmt.Sprintf("[%s]", name)
		}
	}

	// Build center section (status/help text) - dynamically adjust based on width
	if m.filterActive {
		if narrow {
			statusText = "Enter: Apply • ESC: Cancel"
		} else {
			statusText = "Type regex pattern • Enter: Apply • ESC: Cancel"
		}
	} else if m.searchActive {
		if narrow {
			statusText = "Enter: Apply • ESC: Cancel"
		} else {
			statusText = "Type search term • Enter: Apply • ESC: Cancel"
		}
	} else if m.activeSection == SectionLogs {
		if veryNarrow {
			statusText = "?: Help • ↑↓ Nav • Enter"
		} else if narrow {
			statusText = "?: Help • ↑↓ Navigate • Enter: Details"
		} else if medium {
			statusText = "?: Help • ↑↓: Navigate • Home/End • PgUp/Dn • Enter: Details"
		} else {
			statusText = "?: Help • Wheel: scroll • ↑↓: Navigate • Home: Top • End: Latest • PgUp/PgDn: Page • Enter: Details"
		}
	} else if m.showModal {
		statusText = "ESC: Close"
	} else if m.showHelp {
		statusText = "ESC: Close Help"
	} else {
		// Default status showing main actions
		if veryNarrow {
			statusText = "Tab • Space • i • ? • q"
		} else if narrow {
			statusText = "?: Help • Tab: Nav • Space: Pause • i: Stats • q: Quit"
		} else if medium {
			statusText = "Tab: Navigate • Space: Pause • i: Stats • Enter: Select • q: Quit"
		} else {
			statusText = "?: Help • Click sections • Wheel: scroll • Space: Pause • Tab: Navigate • i: Stats • Enter: Select • q: Quit"
		}
	}

	// Build right section (status info and branding)
	var statusInfo string

	// Check for version updates (only if version checker is enabled)
	var versionUpdateInfo string
	if m.versionChecker != nil {
		if updateInfo := m.versionChecker.GetUpdateInfoNonBlocking(); updateInfo != nil && updateInfo.UpdateAvailable {
			versionUpdateInfo = fmt.Sprintf("🔄 v%s available", updateInfo.LatestVersion)
		}
	}

	if !m.filterActive && !m.searchActive && !m.showModal && !m.showHelp {
		if m.viewPaused {
			statusInfo = "⏸"
		} else if !veryNarrow {
			intervalStr := m.formatDuration(m.updateInterval)
			if narrow {
				statusInfo = intervalStr
			} else {
				statusInfo = fmt.Sprintf("Update: %s", intervalStr)
			}
		}
	}

	// Add branding (show unless terminal is very narrow)
	branding := ""
	if m.width >= 30 { // Show branding unless terminal is very narrow
		branding = m.renderGonzoBranding()
	}

	// Combine status info, version update, and branding
	var rightParts []string
	if statusInfo != "" {
		rightParts = append(rightParts, statusInfo)
	}
	if versionUpdateInfo != "" {
		rightParts = append(rightParts, versionUpdateInfo)
	}
	if branding != "" && m.width >= 30 {
		rightParts = append(rightParts, branding)
	}

	if len(rightParts) > 0 {
		rightText = strings.Join(rightParts, "  ")
	}

	// Calculate dynamic widths based on available space using visible width
	leftWidth := lipgloss.Width(leftText) + 2   // Add some padding
	rightWidth := lipgloss.Width(rightText) + 2 // Add some padding

	// Ensure minimum widths don't exceed terminal width
	if leftWidth+rightWidth >= m.width {
		// Terminal too narrow, just show what fits
		availableWidth := m.width
		if availableWidth < 20 {
			// Extremely narrow - just show section name
			return baseStyle.Width(m.width).Render(leftText)
		}
		// Show abbreviated content
		leftWidth = min(10, availableWidth/3)
		rightWidth = min(15, availableWidth/3)
	}

	// Calculate center width (remaining space)
	centerWidth := m.width - leftWidth - rightWidth
	if centerWidth < 0 {
		centerWidth = 0
	}

	// Apply styles with calculated widths
	leftStyle := baseStyle.Align(lipgloss.Left).Width(leftWidth)
	centerStyle := baseStyle.Align(lipgloss.Center).Width(centerWidth)
	rightStyle := baseStyle.Align(lipgloss.Right).Width(rightWidth)

	// Truncate content if necessary to prevent wrapping
	if lipgloss.Width(leftText) > leftWidth {
		leftText = leftText[:max(0, leftWidth-1)]
	}
	if lipgloss.Width(statusText) > centerWidth {
		statusText = statusText[:max(0, centerWidth-1)]
	}
	if lipgloss.Width(rightText) > rightWidth {
		// Don't truncate styled text as it would break ANSI codes
		// Instead, only show what fits based on priority
		if statusInfo != "" && m.width < 50 {
			rightText = statusInfo // Drop branding if too narrow
		} else if m.width < 40 {
			rightText = "" // Drop everything if extremely narrow
		}
	}

	leftPart := leftStyle.Render(leftText)
	centerPart := centerStyle.Render(statusText)
	rightPart := rightStyle.Render(rightText)

	return lipgloss.JoinHorizontal(lipgloss.Top, leftPart, centerPart, rightPart)
}

// renderFilter renders the filter or search input section
func (m *DashboardModel) renderFilter() string {
	var title, content string
	var styleColor lipgloss.Color

	// Check what to display based on active state and applied filters/searches
	if m.filterActive {
		// Actively editing filter
		title = "🔍 Filter (editing)"
		content = m.filterInput.View()
		styleColor = ColorGreen
		if m.filterRegex != nil {
			content += fmt.Sprintf(" | Showing: %d/%d entries", len(m.logEntries), len(m.allLogEntries))
		}
	} else if m.searchActive {
		// Actively editing search
		title = "🔎 Search (editing)"
		content = m.searchInput.View()
		styleColor = ColorYellow
		if m.searchTerm != "" {
			content += fmt.Sprintf(" | Highlighting: %q", m.searchTerm)
		}
	} else if m.filterRegex != nil || m.filterInput.Value() != "" {
		// Filter applied but not editing - show the filter value
		title = "🔍 Filter"
		content = fmt.Sprintf("[%s]", m.filterInput.Value())
		styleColor = ColorGreen
		content += fmt.Sprintf(" | Showing: %d/%d entries", len(m.logEntries), len(m.allLogEntries))
		content += " | Press '/' to edit"
	} else if m.searchTerm != "" || m.searchInput.Value() != "" {
		// Search applied but not editing - show the search term
		title = "🔎 Search"
		searchValue := m.searchTerm
		if searchValue == "" {
			searchValue = m.searchInput.Value()
		}
		content = fmt.Sprintf("[%s]", searchValue)
		styleColor = ColorYellow
		content += fmt.Sprintf(" | Highlighting: %q", searchValue)
		content += " | Press 's' to edit"
	} else {
		// Nothing active or applied
		return ""
	}

	// Minimal style without borders for filter/search
	minimalFilterStyle := lipgloss.NewStyle().
		Foreground(styleColor).
		Padding(0, 1)

	return minimalFilterStyle.Render(title + " " + content)
}

// renderLogScrollContent generates the log content without border wrapper
func (m *DashboardModel) renderLogScrollContent(height int, logWidth int) []string {
	var logLines []string

	// Add paused indicator and help text when log section is active
	if m.activeSection == SectionLogs {
		pausedStyle := lipgloss.NewStyle().
			Foreground(ColorYellow).
			Bold(true)
		statusLine := pausedStyle.Render("↑/↓ to navigate • Home: Top • End: Latest • PgUp/PgDn: Page • Enter for details")
		logLines = append(logLines, statusLine)
		height-- // Reduce available height for logs
	}

	// Add column headers when columns are enabled
	if m.showColumns {
		timestampHeader := lipgloss.NewStyle().Foreground(ColorWhite).Render("Time    ")
		severityHeader := lipgloss.NewStyle().Foreground(ColorWhite).Render("Level")
		hostHeader := lipgloss.NewStyle().Foreground(ColorWhite).Render("Host        ")
		serviceHeader := lipgloss.NewStyle().Foreground(ColorWhite).Render("Service         ")
		messageHeader := lipgloss.NewStyle().Foreground(ColorWhite).Render("Message")

		headerLine := fmt.Sprintf("%s %s %s %s %s",
			timestampHeader, severityHeader, hostHeader, serviceHeader, messageHeader)
		logLines = append(logLines, headerLine)
		height-- // Reduce available height for logs
	}

	// Show recent log entries
	startIdx := 0
	maxLines := height // Use all remaining space after accounting for paused status and headers
	if maxLines < 1 {
		maxLines = 1
	}

	// When in log section or log viewer modal, don't auto-scroll to latest
	if m.activeSection != SectionLogs && !m.showLogViewerModal && len(m.logEntries) > maxLines {
		startIdx = len(m.logEntries) - maxLines
	} else if m.activeSection == SectionLogs || m.showLogViewerModal {
		// Keep selected log in view
		if m.selectedLogIndex >= 0 && m.selectedLogIndex < len(m.logEntries) {
			// Center selected log if possible
			startIdx = m.selectedLogIndex - maxLines/2
			if startIdx < 0 {
				startIdx = 0
			}
			if startIdx+maxLines > len(m.logEntries) {
				startIdx = max(0, len(m.logEntries)-maxLines)
			}
		}
	}

	for i := startIdx; i < len(m.logEntries) && i < startIdx+maxLines; i++ {
		entry := m.logEntries[i]
		isSelected := (m.activeSection == SectionLogs || m.showLogViewerModal) && i == m.selectedLogIndex
		formatted := m.formatLogEntry(entry, logWidth, isSelected)
		logLines = append(logLines, formatted)
	}

	if len(logLines) <= 1 { // Only status line
		// Add helpful instructions when no logs are available
		instructions := []string{
			"Waiting for log entries...",
			"",
			"💡 To get started:",
			"  • Pipe logs: cat mylog.json | gonzo",
			"  • Stream logs: kubectl logs -f pod | gonzo",
			"  • From file: gonzo -f application.log -f other.log -f 'dir/*.globlog'",
			"",
		}

		// Add current filters section if any are applied
		filterStatus := m.buildFilterStatus()
		if len(filterStatus) > 0 {
			instructions = append(instructions, "🔍 Current filters:")
			instructions = append(instructions, filterStatus...)
			instructions = append(instructions, "")
		}

		instructions = append(instructions, []string{
			"📋 Key commands:",
			"  • ?/h: Show help",
			"  • /: Filter logs (regex)",
			"  • Ctrl+f: Filter logs by severity",
			"  • s: Search and highlight",
			"  • Tab: Navigate sections",
			"  • q: Quit",
		}...)

		// Handle scrolling for instructions if they exceed available height
		availableLines := height - 1 // Reserve space for status line that's already added
		if availableLines < 1 {
			availableLines = 1
		}

		if len(instructions) > availableLines {
			// Add scroll indicators and implement scrolling
			maxScroll := len(instructions) - availableLines + 1 // +1 for scroll indicator space
			if m.instructionsScrollOffset > maxScroll {
				m.instructionsScrollOffset = maxScroll
			}
			if m.instructionsScrollOffset < 0 {
				m.instructionsScrollOffset = 0
			}

			// Add scroll up indicator if not at top
			if m.instructionsScrollOffset > 0 {
				scrollUpIndicator := lipgloss.NewStyle().
					Foreground(ColorGray).
					Render(fmt.Sprintf("  ↑ %d more lines above", m.instructionsScrollOffset))
				logLines = append(logLines, scrollUpIndicator)
				availableLines-- // Use one line for indicator
			}

			// Show visible portion of instructions
			endIdx := m.instructionsScrollOffset + availableLines
			if endIdx > len(instructions) {
				endIdx = len(instructions)
			}

			// Reserve space for bottom scroll indicator if needed
			if endIdx < len(instructions) {
				availableLines-- // Reserve space for bottom indicator
				endIdx = m.instructionsScrollOffset + availableLines
			}

			// Add visible instructions
			visibleInstructions := instructions[m.instructionsScrollOffset:endIdx]
			logLines = append(logLines, visibleInstructions...)

			// Add scroll down indicator if not at bottom
			if endIdx < len(instructions) {
				remaining := len(instructions) - endIdx
				scrollDownIndicator := lipgloss.NewStyle().
					Foreground(ColorGray).
					Render(fmt.Sprintf("  ↓ %d more lines below (use ↑↓ or k/j to scroll)", remaining))
				logLines = append(logLines, scrollDownIndicator)
			}
		} else {
			// All instructions fit, no scrolling needed
			logLines = append(logLines, instructions...)
		}
	}

	return logLines
}

// buildFilterStatus returns a list of currently applied filters for display when no logs are shown
func (m *DashboardModel) buildFilterStatus() []string {
	var filters []string

	// Check severity filter
	if m.severityFilterActive {
		disabledSeverities := []string{}
		enabledSeverities := []string{}

		severityLevels := []string{"FATAL", "CRITICAL", "ERROR", "WARN", "INFO", "DEBUG", "TRACE", "UNKNOWN"}
		for _, severity := range severityLevels {
			if enabled, exists := m.severityFilter[severity]; exists {
				if enabled {
					enabledSeverities = append(enabledSeverities, severity)
				} else {
					disabledSeverities = append(disabledSeverities, severity)
				}
			}
		}

		if len(enabledSeverities) > 0 && len(enabledSeverities) < len(severityLevels) {
			if len(enabledSeverities) <= 3 {
				filters = append(filters, "  • Severity: Only showing "+joinWithCommas(enabledSeverities))
			} else {
				filters = append(filters, "  • Severity: Hiding "+joinWithCommas(disabledSeverities))
			}
		} else if len(enabledSeverities) == 0 {
			filters = append(filters, "  • Severity: All severities disabled (no logs will show)")
		}
	}

	// Check regex filter
	if m.filterRegex != nil {
		pattern := m.filterInput.Value()
		if pattern == "" && m.filterRegex != nil {
			pattern = m.filterRegex.String()
		}
		if pattern != "" {
			filters = append(filters, "  • Regex filter: "+pattern)
		}
	}

	// Check search term
	if m.searchTerm != "" {
		filters = append(filters, "  • Search highlight: "+m.searchTerm)
	}

	// Add instructions for clearing filters if any are active
	if len(filters) > 0 {
		filters = append(filters, "")
		filters = append(filters, "  💡 To clear filters:")
		if m.severityFilterActive {
			filters = append(filters, "    • Ctrl+F → Select All → Enter (enable all severities)")
		}
		if m.filterRegex != nil {
			filters = append(filters, "    • / → Backspace/Delete → Enter (clear regex)")
		}
		if m.searchTerm != "" {
			filters = append(filters, "    • s → Backspace/Delete → Enter (clear search)")
		}
	}

	return filters
}

// joinWithCommas joins a slice of strings with commas and "and" before the last item
func joinWithCommas(items []string) string {
	if len(items) == 0 {
		return ""
	}
	if len(items) == 1 {
		return items[0]
	}
	if len(items) == 2 {
		return items[0] + " and " + items[1]
	}

	result := ""
	for i, item := range items {
		if i == len(items)-1 {
			result += "and " + item
		} else {
			result += item + ", "
		}
	}
	return result
}

// renderLogScroll renders the scrolling log section
func (m *DashboardModel) renderLogScroll(height int) string {
	// Use most of terminal width for logs
	logWidth := m.width - 2 // Account for borders and minimal padding
	if logWidth < 40 {
		logWidth = 40 // Higher minimum for readability
	}

	// Highlight border when log section is active
	borderColor := ColorNavy
	if m.activeSection == SectionLogs {
		borderColor = ColorBlue
	}

	style := sectionStyle.
		Width(logWidth).
		Height(height).
		Border(lipgloss.NormalBorder()).
		BorderForeground(borderColor)

	// Get log content
	logLines := m.renderLogScrollContent(height, logWidth)

	return style.Render(lipgloss.JoinVertical(lipgloss.Left, logLines...))
}
