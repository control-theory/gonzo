package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const (
	// MaxColumnWidth is the maximum width for any column
	MaxColumnWidth = 100
)

// columnPart represents a column and its value for formatting
type columnPart struct {
	col   ColumnConfig
	value string
	width int // calculated width for this column
}

// calculateEffectiveColumnWidth returns the effective width for a column
// Min width = label length, Max width = 64 characters
func (m *DashboardModel) calculateEffectiveColumnWidth(col ColumnConfig, value string) int {
	minWidth := len(col.Label)
	if minWidth < 3 {
		minWidth = 3 // Absolute minimum
	}

	// For columns with defined width, use it as a hint but respect min/max
	width := col.Width
	if width == 0 {
		// No defined width - use content length
		width = len(value)
	}

	// Apply min constraint (header must be visible)
	if width < minWidth {
		width = minWidth
	}

	// Apply max constraint (only when width limit is enabled)
	if m.columnWidthLimitEnabled && width > MaxColumnWidth {
		width = MaxColumnWidth
	}

	return width
}

// calculateColumnWidths calculates the width for each enabled column based on content.
// Widths only grow (never shrink) and are capped at MaxColumnWidth.
// Returns a map of column key to width.
func (m *DashboardModel) calculateColumnWidths(entries []LogEntry) map[string]int {
	widths := make(map[string]int)

	for _, col := range m.activeColumns {
		if !col.Enabled {
			continue
		}
		// Start from the highest of: label length, defined Width hint, previously seen max
		minWidth := len(col.Label)
		if minWidth < 3 {
			minWidth = 3
		}
		w := minWidth
		if col.Width > w {
			w = col.Width
		}
		if m.columnMaxWidths[col.Key] > w {
			w = m.columnMaxWidths[col.Key]
		}
		widths[col.Key] = w
	}

	// Sample entries and grow widths (but never shrink)
	sampleSize := min(len(entries), 100)
	for i := 0; i < sampleSize; i++ {
		entry := entries[i]
		for _, col := range m.activeColumns {
			if !col.Enabled {
				continue
			}
			value := m.getColumnValue(entry, col.Key)
			if len(value) > widths[col.Key] {
				widths[col.Key] = len(value)
			}
		}
	}

	// Cap at MaxColumnWidth (only when limit is enabled) and persist new maxima
	for key := range widths {
		if m.columnWidthLimitEnabled && widths[key] > MaxColumnWidth {
			widths[key] = MaxColumnWidth
		}
		if widths[key] > m.columnMaxWidths[key] {
			m.columnMaxWidths[key] = widths[key]
		}
	}

	return widths
}

// formatLogEntry formats a log entry with colors using dynamic columns
func (m *DashboardModel) formatLogEntry(entry LogEntry, availableWidth int, isSelected bool, columnWidths map[string]int, horizontalOffset int) string {
	// Build the log line using active columns
	var parts []columnPart

	// Collect parts with pre-calculated widths
	for _, col := range m.activeColumns {
		if !col.Enabled {
			continue
		}

		value := m.getColumnValue(entry, col.Key)
		width := columnWidths[col.Key]
		if width == 0 {
			width = m.calculateEffectiveColumnWidth(col, value)
		}

		parts = append(parts, columnPart{col, value, width})
	}

	// Always use per-column styling which handles horizontal scrolling properly
	return m.formatLogEntryStyled(entry, parts, availableWidth, isSelected, horizontalOffset)
}

// formatLogEntryStyled formats a log entry with per-column styling, supporting horizontal scroll and selection
func (m *DashboardModel) formatLogEntryStyled(entry LogEntry, parts []columnPart, availableWidth int, isSelected bool, horizontalOffset int) string {
	// Build the full raw line to calculate positions for horizontal scrolling
	type styledSegment struct {
		start int    // start position in the raw line
		end   int    // end position in the raw line
		text  string // padded text value
		col   ColumnConfig
	}

	var segments []styledSegment
	currentPos := 0

	for i, p := range parts {
		width := p.width
		if width <= 0 {
			continue
		}

		value := p.value
		// Truncate if needed (with ellipsis)
		if len(value) > width {
			if width > 3 {
				value = value[:width-3] + "..."
			} else {
				value = value[:width]
			}
		}
		paddedValue := fmt.Sprintf("%-*s", width, value)

		// Add space separator between columns (except before first)
		if i > 0 && len(segments) > 0 {
			currentPos++ // account for the space separator
		}

		segments = append(segments, styledSegment{
			start: currentPos,
			end:   currentPos + len(paddedValue),
			text:  paddedValue,
			col:   p.col,
		})
		currentPos += len(paddedValue)
	}

	totalWidth := currentPos

	// Calculate visible range after horizontal offset
	visibleStart := horizontalOffset
	visibleEnd := horizontalOffset + availableWidth
	if visibleStart > totalWidth {
		visibleStart = totalWidth
	}
	if visibleEnd > totalWidth {
		visibleEnd = totalWidth
	}

	// Build the visible line by processing each segment
	var result strings.Builder
	outputPos := 0

	// Define selected style once for reuse
	selectedStyle := lipgloss.NewStyle().
		Background(ColorBlue).
		Foreground(ColorWhite)

	for i, seg := range segments {
		// Add space separator between columns
		if i > 0 {
			spacePos := seg.start - 1
			if spacePos >= visibleStart && spacePos < visibleEnd {
				if isSelected {
					result.WriteString(selectedStyle.Render(" "))
				} else {
					result.WriteString(" ")
				}
				outputPos++
			}
		}

		// Calculate the visible portion of this segment
		segVisibleStart := max(0, visibleStart-seg.start)
		segVisibleEnd := min(len(seg.text), visibleEnd-seg.start)

		if segVisibleStart >= segVisibleEnd {
			continue // segment is not visible
		}

		visibleText := seg.text[segVisibleStart:segVisibleEnd]

		// Apply styling based on column type and selection state
		var styled string
		if isSelected {
			styled = selectedStyle.Render(visibleText)
		} else {
			// Apply per-column styling with search highlighting
			styled = m.styleColumnValue(visibleText, seg.col.Key, entry)
		}

		result.WriteString(styled)
		outputPos += len(visibleText)
	}

	// Pad to available width for selected rows
	if isSelected && outputPos < availableWidth {
		padding := strings.Repeat(" ", availableWidth-outputPos)
		result.WriteString(selectedStyle.Render(padding))
	}

	return result.String()
}

// styleColumnValue applies appropriate styling for a column value, including search highlighting
func (m *DashboardModel) styleColumnValue(value string, colKey string, entry LogEntry) string {
	// Get the base style for this column type
	var baseStyle lipgloss.Style
	switch colKey {
	case "timestamp":
		baseStyle = lipgloss.NewStyle().Foreground(ColorGray)
	case "severity":
		baseStyle = lipgloss.NewStyle().Foreground(GetSeverityColor(entry.Severity)).Bold(true)
	case "host.name", "k8s.namespace":
		baseStyle = lipgloss.NewStyle().Foreground(ColorGreen)
	case "service.name", "k8s.pod":
		baseStyle = lipgloss.NewStyle().Foreground(ColorBlue)
	case "message":
		baseStyle = lipgloss.NewStyle().Foreground(ColorWhite)
	default:
		baseStyle = lipgloss.NewStyle().Foreground(ColorWhite)
	}

	// Apply search highlighting if active
	if m.searchTerm != "" {
		return m.highlightTextWithBaseStyle(value, m.searchTerm, baseStyle)
	}

	return baseStyle.Render(value)
}

// getColumnValue extracts the value for a column from a log entry
func (m *DashboardModel) getColumnValue(entry LogEntry, key string) string {
	switch key {
	case "timestamp":
		return m.getDisplayTimestamp(entry).Format("15:04:05")
	case "severity":
		return entry.Severity
	case "message":
		return entry.Message
	default:
		// Look up in attributes
		if val, ok := entry.Attributes[key]; ok {
			return val
		}
		return ""
	}
}

// highlightTextWithBaseStyle highlights search term within text, applying a base style to non-highlighted portions
func (m *DashboardModel) highlightTextWithBaseStyle(text, searchTerm string, baseStyle lipgloss.Style) string {
	if searchTerm == "" {
		return baseStyle.Render(text)
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
			// No more matches, append the rest with base style
			if lastIndex < len(text) {
				result.WriteString(baseStyle.Render(text[lastIndex:]))
			}
			break
		}

		// Calculate actual position in original text
		actualIndex := lastIndex + index

		// Append text before match with base style
		if actualIndex > lastIndex {
			result.WriteString(baseStyle.Render(text[lastIndex:actualIndex]))
		}

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
