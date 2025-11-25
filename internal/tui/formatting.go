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
			// Check if this is a k8s log (has k8s.namespace or k8s.pod attributes)
			namespace := entry.Attributes["k8s.namespace"]
			pod := entry.Attributes["k8s.pod"]
			isK8s := namespace != "" || pod != ""

			var col1Str, col2Str string
			var columnsWidth int

			if isK8s {
				// K8s mode: show namespace and pod (both truncated to 20 chars)
				if len(namespace) > 20 {
					namespace = namespace[:17] + "..."
				}
				if len(pod) > 20 {
					pod = pod[:17] + "..."
				}

				// Format fixed-width columns
				col1Str = fmt.Sprintf("%-20s", namespace)
				col2Str = fmt.Sprintf("%-20s", pod)
				columnsWidth = 42 // 20 + 20 + 2 spaces
			} else {
				// Normal mode: show host.name and service.name from OTLP attributes
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
				col1Str = fmt.Sprintf("%-12s", host)
				col2Str = fmt.Sprintf("%-16s", service)
				columnsWidth = 30 // 12 + 16 + 2 spaces
			}

			// Calculate remaining space for message
			// Use same calculation as non-selected: availableWidth - 18 - columnsWidth
			maxMessageLen := availableWidth - 18 - columnsWidth
			if maxMessageLen < 10 {
				maxMessageLen = 10
			}

			message := entry.Message
			if len(message) > maxMessageLen {
				message = message[:maxMessageLen-3] + "..."
			}

			logLine = fmt.Sprintf("%s %-5s %s %s %s", timestamp, severity, col1Str, col2Str, message)
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

	// Extract columns if enabled (K8s or Host/Service)
	var col1, col2 string
	columnsWidth := 0
	if m.showColumns {
		// Check if this is a k8s log (has k8s.namespace or k8s.pod attributes)
		namespace := entry.Attributes["k8s.namespace"]
		pod := entry.Attributes["k8s.pod"]
		isK8s := namespace != "" || pod != ""

		if isK8s {
			// K8s mode: show namespace and pod (both truncated to 20 chars)
			if len(namespace) > 20 {
				namespace = namespace[:17] + "..."
			}
			if len(pod) > 20 {
				pod = pod[:17] + "..."
			}

			// Style the k8s columns
			col1 = lipgloss.NewStyle().
				Foreground(ColorGreen).
				Render(fmt.Sprintf("%-20s", namespace))

			col2 = lipgloss.NewStyle().
				Foreground(ColorBlue).
				Render(fmt.Sprintf("%-20s", pod))

			columnsWidth = 42 // 20 + 20 + 2 spaces
		} else {
			// Normal mode: show host.name and service.name from OTLP attributes
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
			col1 = lipgloss.NewStyle().
				Foreground(ColorGreen).
				Render(fmt.Sprintf("%-12s", host))

			col2 = lipgloss.NewStyle().
				Foreground(ColorBlue).
				Render(fmt.Sprintf("%-16s", service))

			columnsWidth = 30 // 12 + 16 + 2 spaces
		}
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
		logLine = fmt.Sprintf("%s %s %s %s %s", styledTimestamp, styledSeverity, col1, col2, message)
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
		// Use lipgloss.Width to get visual width (ignoring ANSI sequences)
		if lipgloss.Width(line) <= width {
			wrappedLines = append(wrappedLines, line)
			continue
		}

		// Character-based wrapping - don't split by words for log content
		remaining := line
		for len(remaining) > 0 {
			// Find the maximum characters that fit within width
			maxChars := min(len(remaining), width)

			// Adjust maxChars to fit within visual width
			for maxChars > 0 && lipgloss.Width(remaining[:maxChars]) > width {
				maxChars--
			}

			// Try to fit more characters if possible
			for maxChars < len(remaining) && lipgloss.Width(remaining[:maxChars+1]) <= width {
				maxChars++
			}

			if maxChars <= 0 {
				maxChars = 1 // At least one character
			}

			chunk := remaining[:maxChars]
			wrappedLines = append(wrappedLines, chunk)
			remaining = remaining[maxChars:]
		}
	}

	return strings.Join(wrappedLines, "\n")
}
