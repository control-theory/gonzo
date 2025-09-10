package tui

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// formatLogEntry formats a log entry with colors
func (m *DashboardModel) formatLogEntry(entry LogEntry, availableWidth int, isSelected bool) string {
	// Use receive time for display
	timestamp := entry.Timestamp.Format("15:04:05")

	// If selected, apply selection style to entire row
	if isSelected {
		// Format the entire row without individual component styling
		severity := fmt.Sprintf("%-5s", entry.Severity)

		var logLine string
		if m.showColumns {
			// Extract host.name and service.name from OTLP attributes
			host := entry.Attributes["host.name"]
			service := entry.Attributes["service.name"]

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
			// Use same calculation as non-selected: availableWidth - 18 - columnsWidth
			columnsWidth := 30 // 12 + 16 + 2 spaces
			maxMessageLen := availableWidth - 18 - columnsWidth
			if maxMessageLen < 10 {
				maxMessageLen = 10
			}

			message := entry.Message
			if len(message) > maxMessageLen {
				message = message[:maxMessageLen-3] + "..."
			}

			logLine = fmt.Sprintf("%s %-5s %s %s %s", timestamp, severity, hostCol, serviceCol, message)
		} else {
			// Calculate space for message - use same as non-selected: availableWidth - 18
			maxMessageLen := availableWidth - 18
			if maxMessageLen < 10 {
				maxMessageLen = 10
			}

			message := entry.Message
			if len(message) > maxMessageLen {
				message = message[:maxMessageLen-3] + "..."
			}

			logLine = fmt.Sprintf("%s %-5s %s", timestamp, severity, message)
		}

		// Apply selection style to entire line
		selectedStyle := lipgloss.NewStyle().
			Background(ColorBlue).
			Foreground(ColorWhite)
		return selectedStyle.Render(logLine)
	}

	// Normal (non-selected) formatting with individual component colors
	severityColor := GetSeverityColor(entry.Severity)

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
		host := entry.Attributes["host.name"]
		service := entry.Attributes["service.name"]

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

	maxMessageLen := availableWidth - 18 - columnsWidth // Account for timestamp, severity, and columns
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
		logLine = fmt.Sprintf("%s %s %s %s %s", styledTimestamp, styledSeverity, hostCol, serviceCol, message)
	} else {
		logLine = fmt.Sprintf("%s %s %s", styledTimestamp, styledSeverity, message)
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
func (m *DashboardModel) containsWord(text, word string) bool {
	if word == "" {
		return false
	}

	// Convert both to lowercase for case-insensitive matching
	lowerText := strings.ToLower(text)
	lowerWord := strings.ToLower(word)

	// Use regex to match word boundaries - this ensures we match whole words
	// even when they're surrounded by punctuation
	pattern := `\b` + regexp.QuoteMeta(lowerWord) + `\b`
	matched, err := regexp.MatchString(pattern, lowerText)
	if err != nil {
		// Fallback to simple contains if regex fails
		return strings.Contains(lowerText, lowerWord)
	}

	return matched
}

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
