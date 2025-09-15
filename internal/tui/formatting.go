package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// formatLogEntry formats a log entry with colors
func (m *DashboardModel) formatLogEntry(entry LogEntry, availableWidth int, isSelected bool) string {
	// Format source indicator (2 chars + 2 spaces for alignment)
	sourceIndicator := fmt.Sprintf("%s%d  ", entry.SourceType, entry.SourceID)
	if entry.SourceType == "" {
		sourceIndicator = "??  "
	}

	// Use original timestamp if available, otherwise use receive time
	var timestamp string
	var timeFormat string
	if m.showFullDate {
		// Full date format: MM/DD/YYYY HH:MM:SS (24hr)
		timeFormat = "01/02/2006 15:04:05"
	} else {
		// Time only format: HH:MM:SS
		timeFormat = "15:04:05"
	}

	if !entry.OrigTimestamp.IsZero() {
		timestamp = entry.OrigTimestamp.Format(timeFormat)
	} else {
		timestamp = entry.Timestamp.Format(timeFormat)
	}

	// If selected, apply selection style to entire row
	if isSelected {
		// Format the entire row without individual component styling
		severity := fmt.Sprintf("%-5s", entry.Severity)

		var logLine string
		if m.showColumns {
			// Extract host.name and service.name from OTLP attributes
			// Try both dot and underscore versions (different sources use different conventions)
			host := entry.Attributes["host.name"]
			if host == "" {
				host = entry.Attributes["host_name"]
			}
			service := entry.Attributes["service.name"]
			if service == "" {
				service = entry.Attributes["service_name"]
			}

			// Truncate to fit column width
			if len(host) > 12 {
				host = host[:9] + "..."
			}
			if len(service) > 16 {
				service = service[:13] + "..."
			}

			// Format fixed-width columns
			hostCol := fmt.Sprintf("%-12s", host)
			serviceCol := fmt.Sprintf("%-16s", service)

			// Calculate remaining space for message
			// Adjust for timestamp width: 8 chars for time only, 19 chars for full date
			timestampWidth := 8
			if m.showFullDate {
				timestampWidth = 19
			}
			// Account for source indicator (2 chars + 1 space)
			// Use same calculation as non-selected: availableWidth - sourceWidth - (timestampWidth + 10) - columnsWidth
			sourceWidth := 3
			columnsWidth := 30 // 12 + 16 + 2 spaces
			maxMessageLen := availableWidth - sourceWidth - (timestampWidth + 10) - columnsWidth
			if maxMessageLen < 10 {
				maxMessageLen = 10
			}

			message := entry.Message
			if len(message) > maxMessageLen {
				message = message[:maxMessageLen-3] + "..."
			}

			logLine = fmt.Sprintf("%s%s %-5s %s %s %s", sourceIndicator, timestamp, severity, hostCol, serviceCol, message)
		} else {
			// Calculate space for message - adjust for timestamp width
			timestampWidth := 8
			if m.showFullDate {
				timestampWidth = 19
			}
			// Account for source indicator (2 chars + 1 space)
			sourceWidth := 3
			maxMessageLen := availableWidth - sourceWidth - (timestampWidth + 10)
			if maxMessageLen < 10 {
				maxMessageLen = 10
			}

			message := entry.Message
			if len(message) > maxMessageLen {
				message = message[:maxMessageLen-3] + "..."
			}

			logLine = fmt.Sprintf("%s%s %-5s %s", sourceIndicator, timestamp, severity, message)
		}

		// Apply selection style to entire line
		selectedStyle := lipgloss.NewStyle().
			Background(ColorBlue).
			Foreground(ColorWhite)
		return selectedStyle.Render(logLine)
	}

	// Normal (non-selected) formatting with individual component colors
	severityColor := GetSeverityColor(entry.Severity)

	// Style the source indicator with a distinct color
	sourceColor := lipgloss.AdaptiveColor{Light: "#666666", Dark: "#999999"}
	switch entry.SourceType {
	case "L":
		sourceColor = lipgloss.AdaptiveColor{Light: "#00AA00", Dark: "#00FF00"} // Green for Loki
	case "D":
		sourceColor = lipgloss.AdaptiveColor{Light: "#0066CC", Dark: "#0099FF"} // Blue for Docker
	case "V":
		sourceColor = lipgloss.AdaptiveColor{Light: "#AA00AA", Dark: "#FF00FF"} // Magenta for VMLogs
	case "O":
		sourceColor = lipgloss.AdaptiveColor{Light: "#AA6600", Dark: "#FFAA00"} // Orange for OTLP
	case "F":
		sourceColor = lipgloss.AdaptiveColor{Light: "#666600", Dark: "#AAAA00"} // Yellow for Files
	case "I":
		sourceColor = lipgloss.AdaptiveColor{Light: "#666666", Dark: "#AAAAAA"} // Gray for stdin
	}

	styledSource := lipgloss.NewStyle().
		Foreground(sourceColor).
		Render(sourceIndicator)

	styledSeverity := lipgloss.NewStyle().
		Foreground(severityColor).
		Bold(true).
		Render(fmt.Sprintf("%-5s", entry.Severity))

	styledTimestamp := lipgloss.NewStyle().
		Foreground(ColorGray).
		Render(timestamp)

	// Extract Host and Service columns if enabled
	var hostCol, serviceCol string
	columnsWidth := 0
	if m.showColumns {
		// Extract host.name and service.name from OTLP attributes
		// Try both dot and underscore versions (different sources use different conventions)
		host := entry.Attributes["host.name"]
		if host == "" {
			host = entry.Attributes["host_name"]
		}
		service := entry.Attributes["service.name"]
		if service == "" {
			service = entry.Attributes["service_name"]
		}

		// Truncate to fit column width (12 chars / 16 chars)
		if len(host) > 12 {
			host = host[:9] + "..."
		}
		if len(service) > 16 {
			service = service[:13] + "..."
		}

		// Style the columns
		hostCol = lipgloss.NewStyle().
			Foreground(ColorGreen).
			Render(fmt.Sprintf("%-12s", host))

		serviceCol = lipgloss.NewStyle().
			Foreground(ColorBlue).
			Render(fmt.Sprintf("%-16s", service))

		columnsWidth = 30 // 12 + 16 + 2 spaces
	}

	// Truncate message if too long
	message := entry.Message

	// Adjust for timestamp width
	timestampWidth := 8
	if m.showFullDate {
		timestampWidth = 19
	}
	// Account for source indicator (2 chars + 1 space)
	sourceWidth := 3
	maxMessageLen := availableWidth - sourceWidth - (timestampWidth + 10) - columnsWidth // Account for source, timestamp, severity, and columns
	if maxMessageLen < 10 {
		maxMessageLen = 10 // Absolute minimum
	}
	if len(message) > maxMessageLen {
		message = message[:maxMessageLen-3] + "..."
	}

	// Apply search term highlighting to message (word-level highlighting)
	if m.searchTerm != "" {
		message = m.highlightText(message, m.searchTerm)
	}

	// Create the complete log line
	var logLine string
	if m.showColumns {
		logLine = fmt.Sprintf("%s%s %s %s %s %s", styledSource, styledTimestamp, styledSeverity, hostCol, serviceCol, message)
	} else {
		logLine = fmt.Sprintf("%s%s %s %s", styledSource, styledTimestamp, styledSeverity, message)
	}

	return logLine
}

// highlightText highlights search term within text (for 's' command)
func (m *DashboardModel) highlightText(text, searchTerm string) string {
	if searchTerm == "" {
		return text
	}

	// Case-insensitive search
	lowerText := strings.ToLower(text)
	lowerSearch := strings.ToLower(searchTerm)

	// Find all occurrences
	var result strings.Builder
	lastIndex := 0

	for {
		index := strings.Index(lowerText[lastIndex:], lowerSearch)
		if index == -1 {
			// No more matches, append the rest
			result.WriteString(text[lastIndex:])
			break
		}

		// Calculate actual position in original text
		actualIndex := lastIndex + index

		// Append text before match
		result.WriteString(text[lastIndex:actualIndex])

		// Append highlighted match
		highlightStyle := lipgloss.NewStyle().
			Background(ColorYellow). // Yellow for word highlighting
			Foreground(ColorBlack).
			Bold(true)

		result.WriteString(highlightStyle.Render(text[actualIndex : actualIndex+len(searchTerm)]))

		// Move past this match
		lastIndex = actualIndex + len(searchTerm)
	}

	return result.String()
}

// containsWord checks if a word appears in text using word boundary matching
// This matches how words are extracted for frequency analysis

// wrapTextToWidth wraps text to fit within the specified width
func (m *DashboardModel) wrapTextToWidth(text string, width int) string {
	if width <= 0 {
		return text
	}

	lines := strings.Split(text, "\n")
	var wrappedLines []string

	for _, line := range lines {
		// If line is shorter than width, add as-is
		if len(line) <= width {
			wrappedLines = append(wrappedLines, line)
			continue
		}

		// Wrap long lines
		words := strings.Fields(line)
		if len(words) == 0 {
			wrappedLines = append(wrappedLines, line)
			continue
		}

		currentLine := ""
		for _, word := range words {
			// If adding this word would exceed width, start new line
			testLine := currentLine
			if testLine != "" {
				testLine += " "
			}
			testLine += word

			if len(testLine) > width {
				// If current line has content, save it and start new line with current word
				if currentLine != "" {
					wrappedLines = append(wrappedLines, currentLine)
					currentLine = word
				} else {
					// Single word is longer than width, truncate it
					currentLine = word
					if len(currentLine) > width {
						currentLine = currentLine[:width-3] + "..."
					}
				}
			} else {
				currentLine = testLine
			}
		}

		// Add remaining content
		if currentLine != "" {
			wrappedLines = append(wrappedLines, currentLine)
		}
	}

	return strings.Join(wrappedLines, "\n")
}
