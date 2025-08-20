package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/control-theory/gonzo/internal/memory"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
)

// formatAttributesTable formats attributes using Bubbles table component
func (m *DashboardModel) formatAttributesTable(attributes map[string]string, maxWidth int) string {
	if len(attributes) == 0 {
		return ""
	}

	// Sort keys for consistent display
	keys := make([]string, 0, len(attributes))
	for k := range attributes {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Calculate column widths
	keyWidth := maxWidth / 3              // ~33% for keys
	valueWidth := maxWidth - keyWidth - 4 // remainder for values, minus borders

	if keyWidth < 10 {
		keyWidth = 10
	}
	if valueWidth < 10 {
		valueWidth = 10
	}

	// Create table columns
	columns := []table.Column{
		{Title: "Name", Width: keyWidth},
		{Title: "Value", Width: valueWidth},
	}

	// Create table rows
	rows := []table.Row{}
	for _, key := range keys {
		value := attributes[key]
		// Truncate long values to fit
		if len(value) > valueWidth-3 {
			value = value[:valueWidth-3] + "..."
		}
		// Truncate long keys to fit
		displayKey := key
		if len(displayKey) > keyWidth-3 {
			displayKey = displayKey[:keyWidth-3] + "..."
		}
		rows = append(rows, table.Row{displayKey, value})
	}

	// Create and configure table
	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithHeight(len(rows)+2), // +2 for header and padding
		table.WithWidth(maxWidth),
		table.WithFocused(false), // Disable focus to prevent selection
	)

	// Style the table
	styles := table.DefaultStyles()
	styles.Header = lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorBlue).
		Width(keyWidth)
	styles.Cell = lipgloss.NewStyle()     // Default cell style
	styles.Selected = lipgloss.NewStyle() // No selection highlighting

	t.SetStyles(styles)
	t.Blur() // Ensure table is not focused

	return t.View()
}

// formatLogDetails formats detailed log entry information for modal display
func (m *DashboardModel) formatLogDetails(entry LogEntry, maxWidth int) string {
	// Define styles
	headerStyle := lipgloss.NewStyle().
		Foreground(ColorBlue).
		Bold(true)

	labelStyle := lipgloss.NewStyle().
		Foreground(ColorGray).
		Width(12)

	valueStyle := lipgloss.NewStyle().
		Foreground(ColorWhite)

	severityStyle := lipgloss.NewStyle().
		Bold(true)

	// Color severity based on level
	normalizedSeverity := normalizeSeverityLevel(entry.Severity)
	switch normalizedSeverity {
	case "ERROR", "FATAL", "CRITICAL":
		severityStyle = severityStyle.Foreground(ColorRed)
	case "WARN":
		severityStyle = severityStyle.Foreground(ColorYellow)
	case "INFO":
		severityStyle = severityStyle.Foreground(ColorGreen)
	case "DEBUG", "TRACE":
		severityStyle = severityStyle.Foreground(ColorGray)
	default:
		severityStyle = severityStyle.Foreground(ColorWhite)
	}

	var details strings.Builder

	// Header
	details.WriteString(headerStyle.Render("Log Entry Details") + "\n\n")

	// Basic information - show both timestamps
	details.WriteString(labelStyle.Render("Received:") + " " +
		valueStyle.Render(entry.Timestamp.Format("2006-01-02 15:04:05.000")) + "\n")
	
	// Show original timestamp if available and different from receive time
	if !entry.OrigTimestamp.IsZero() {
		details.WriteString(labelStyle.Render("Log Time:") + " " +
			valueStyle.Render(entry.OrigTimestamp.Format("2006-01-02 15:04:05.000")) + "\n")
	}
	
	details.WriteString(labelStyle.Render("Severity:") + " " +
		severityStyle.Render(entry.Severity) + "\n")
	details.WriteString(labelStyle.Render("Message:") + "\n" +
		valueStyle.Render(entry.Message) + "\n")

	// Attributes table
	if len(entry.Attributes) > 0 {
		details.WriteString("\n" + headerStyle.Render("Attributes") + "\n")
		details.WriteString(m.formatAttributesTable(entry.Attributes, maxWidth))
	}

	// AI Analysis section
	if m.aiAnalysisResult != "" && m.aiAnalysisResult != "Analyzing..." {
		details.WriteString("\n" + headerStyle.Render("🤖 AI Analysis") + "\n")
		details.WriteString(valueStyle.Render(m.aiAnalysisResult) + "\n")
	} else if m.aiAnalyzing {
		details.WriteString("\n" + headerStyle.Render("🤖 AI Analysis") + "\n")
		spinnerText := fmt.Sprintf("%s Analyzing log entry...", m.getSpinner())
		details.WriteString(lipgloss.NewStyle().Foreground(ColorYellow).Render(spinnerText) + "\n")
	}

	// Chat history is now handled separately in the chat pane

	return details.String()
}

// formatAttributeValuesModal formats the attribute values modal showing individual values and their counts with full width layout
func (m *DashboardModel) formatAttributeValuesModal(entry *memory.AttributeStatsEntry, maxWidth int) string {
	var modal strings.Builder

	// Title section
	titleStyle := lipgloss.NewStyle().
		Foreground(ColorWhite).
		Bold(true)

	modal.WriteString(titleStyle.Render(fmt.Sprintf("Attribute Values for \"%s\"", entry.Key)) + "\n\n")

	if len(entry.Values) == 0 {
		helpStyle := lipgloss.NewStyle().Foreground(ColorGray).Italic(true)
		modal.WriteString(helpStyle.Render("No values recorded for this attribute.") + "\n")
		return modal.String()
	}

	// Add summary section at the top
	summaryStyle := lipgloss.NewStyle().Foreground(ColorGreen).Bold(true)
	modal.WriteString(summaryStyle.Render("Summary:") + "\n")

	summaryDetailStyle := lipgloss.NewStyle().Foreground(ColorWhite)
	modal.WriteString(summaryDetailStyle.Render(fmt.Sprintf("Total occurrences: %d", entry.TotalCount)) + "\n")
	modal.WriteString(summaryDetailStyle.Render(fmt.Sprintf("Unique values: %d", entry.UniqueValueCount)) + "\n\n")

	// Convert map to sorted slice for consistent display
	type ValueCount struct {
		Value string
		Count int64
	}

	values := make([]ValueCount, 0, len(entry.Values))
	for value, count := range entry.Values {
		values = append(values, ValueCount{Value: value, Count: count})
	}

	// Sort by count (descending), then by value name for ties
	sort.Slice(values, func(i, j int) bool {
		if values[i].Count == values[j].Count {
			return values[i].Value < values[j].Value
		}
		return values[i].Count > values[j].Count
	})

	// Calculate optimal column widths using full available width
	availableWidth := maxWidth

	// Reserve space for count column and separators
	countColumnWidth := 18 // Fixed width for "Count" column including percentage
	separatorWidth := 3    // " │ " separator

	// Use remaining space for value column
	valueColumnWidth := availableWidth - countColumnWidth - separatorWidth

	// Find actual max value length in data
	actualMaxValueLen := 0
	for _, vc := range values {
		if len(vc.Value) > actualMaxValueLen {
			actualMaxValueLen = len(vc.Value)
		}
	}

	// Use full available space for values, but ensure minimum readable width
	maxValueLength := max(valueColumnWidth, 20) // Use full space available, minimum 20

	// Table header
	headerStyle := lipgloss.NewStyle().Foreground(ColorWhite).Bold(true)
	header := fmt.Sprintf("%-*s │ %s", maxValueLength, "Value", "Count")
	modal.WriteString(headerStyle.Render(header) + "\n")

	// Divider line
	dividerStyle := lipgloss.NewStyle().Foreground(ColorGray)
	modal.WriteString(dividerStyle.Render(strings.Repeat("─", len(header))) + "\n")

	// Display ALL values with counts in table format (no artificial limit - let scrolling handle it)
	for _, vc := range values {
		displayValue := vc.Value
		if len(displayValue) > maxValueLength {
			displayValue = displayValue[:maxValueLength-3] + "..."
		}

		// Calculate percentage
		percentage := float64(vc.Count) * 100.0 / float64(entry.TotalCount)

		// Style the value and count
		valueStyle := lipgloss.NewStyle().Foreground(ColorBlue)
		countStyle := lipgloss.NewStyle().Foreground(ColorGreen).Bold(true)

		// Format with proper table alignment
		line := fmt.Sprintf("%s │ %s",
			valueStyle.Render(fmt.Sprintf("%-*s", maxValueLength, displayValue)),
			countStyle.Render(fmt.Sprintf("%d (%.1f%%)", vc.Count, percentage)))

		modal.WriteString(line + "\n")
	}

	return modal.String()
}

// formatDuration formats a duration for user display
func (m *DashboardModel) formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Nanoseconds()/1000000)
	}
	if d < time.Minute {
		if d%time.Second == 0 {
			return fmt.Sprintf("%ds", int(d.Seconds()))
		}
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	if d < time.Hour {
		if d%time.Minute == 0 {
			return fmt.Sprintf("%dm", int(d.Minutes()))
		}
		return fmt.Sprintf("%.1fm", d.Minutes())
	}
	return d.String()
}
