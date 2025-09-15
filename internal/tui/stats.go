package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/control-theory/gonzo/internal/logger"
)

// renderStatsContent renders the detailed statistics content
func (m *DashboardModel) renderStatsContent(contentWidth int) string {
	var sections []string

	// Title section
	titleStyle := lipgloss.NewStyle().
		Foreground(ColorBlue).
		Bold(true).
		Align(lipgloss.Center).
		Width(contentWidth)

	sections = append(sections, titleStyle.Render("Log Analysis Statistics"))
	sections = append(sections, "")

	// Calculate width for side-by-side sections (with spacing)
	halfWidth := (contentWidth - 3) / 2 // -3 for spacing between columns

	// Row 1: General Statistics | Severity Distribution (side by side)
	generalStats := m.renderStatsSection("General Statistics", []StatItem{
		{"Total Logs Processed", fmt.Sprintf("%d", m.statsTotalLogsEver)},
		{"Logs in Buffer", fmt.Sprintf("%d", len(m.allLogEntries))},
		{"Filtered Logs Displayed", fmt.Sprintf("%d", len(m.logEntries))},
		{"Total Bytes Processed", m.formatBytes(m.statsTotalBytes)},
		{"Uptime", m.formatUptime()},
		{"Current Processing Rate", m.formatCurrentRate()},
		{"Peak Logs per Second", fmt.Sprintf("%.1f", m.statsPeakLogsPerSec)},
	}, halfWidth)

	// Severity Statistics with visual bar chart
	severityStats := m.calculateSeverityStats()
	severitySection := m.renderSeveritySection(severityStats, halfWidth)

	// Combine general and severity side by side
	row1 := m.combineSideBySide(generalStats, severitySection)
	sections = append(sections, row1)

	// Host Statistics Section
	hostStats := m.calculateHostStats()
	if len(hostStats) > 0 {
		sections = append(sections, m.renderStatsSection("Top Hosts", hostStats[:min(10, len(hostStats))], contentWidth))
	}

	// Row 2: Top Services | Pattern Analysis (side by side)
	serviceStats := m.calculateServiceStats()
	var row2 string

	if len(serviceStats) > 0 {
		servicesSection := m.renderStatsSection("Top Services", serviceStats[:min(10, len(serviceStats))], halfWidth)

		// Pattern Statistics Section (if available)
		if m.drain3Manager != nil {
			patternCount, totalLogs := m.drain3Manager.GetStats()
			if patternCount > 0 {
				patternStats := []StatItem{
					{"Unique Patterns Detected", fmt.Sprintf("%d", patternCount)},
					{"Pattern Compression Ratio", fmt.Sprintf("%.1f:1", float64(totalLogs)/float64(patternCount))},
					{"Logs Analyzed for Patterns", fmt.Sprintf("%d", totalLogs)},
				}
				patternSection := m.renderStatsSection("Pattern Analysis", patternStats, halfWidth)
				row2 = m.combineSideBySide(servicesSection, patternSection)
			} else {
				row2 = servicesSection
			}
		} else {
			row2 = servicesSection
		}
		sections = append(sections, row2)
	}

	// Attribute Statistics Section (formatted with columns)
	attributeStats := m.calculateAttributeStatsFormatted()
	logger.Debugf("[STATS] calculateAttributeStatsFormatted returned %d items", len(attributeStats))
	logger.Debugf("[STATS] lifetimeAttrCounts has %d items", len(m.lifetimeAttrCounts))
	if len(attributeStats) > 0 {
		sections = append(sections, m.renderAttributeSection(attributeStats[:min(15, len(attributeStats))], contentWidth))
	}

	return strings.Join(sections, "\n")
}

// StatItem represents a statistics key-value pair
type StatItem struct {
	Key   string
	Value string
}

// renderStatsSection renders a section of statistics with consistent formatting (using dashboard styling)
func (m *DashboardModel) renderStatsSection(title string, items []StatItem, width int) string {
	// Use chartTitleStyle for consistent title formatting
	titleContent := chartTitleStyle.Render(title)

	var contentLines []string

	// Calculate the maximum key length for alignment
	maxKeyLen := 0
	for _, item := range items {
		if len(item.Key) > maxKeyLen {
			maxKeyLen = len(item.Key)
		}
	}

	// Add padding
	maxKeyLen += 3

	// Render each statistic item with aligned values
	for _, item := range items {
		keyStyle := lipgloss.NewStyle().
			Foreground(ColorWhite).
			Width(maxKeyLen).
			Align(lipgloss.Left)

		valueStyle := lipgloss.NewStyle().
			Foreground(ColorBlue).
			Bold(true)

		line := fmt.Sprintf("%s %s",
			keyStyle.Render(item.Key+":"),
			valueStyle.Render(item.Value))
		contentLines = append(contentLines, line)
	}

	content := strings.Join(contentLines, "\n")

	// Use sectionStyle for consistent section formatting with borders
	sectionContent := lipgloss.JoinVertical(lipgloss.Left, titleContent, content)

	return sectionStyle.
		Width(width).
		Render(sectionContent)
}

// Helper functions for statistics calculations
func (m *DashboardModel) calculateSeverityStats() []StatItem {
	var stats []StatItem
	severityOrder := []string{"FATAL", "ERROR", "WARN", "INFO", "DEBUG", "TRACE"}
	colors := map[string]lipgloss.Color{
		"FATAL": ColorRed, "ERROR": ColorRed, "WARN": ColorOrange,
		"INFO": ColorBlue, "DEBUG": ColorGray, "TRACE": ColorGray,
	}

	// Use lifetime total instead of buffer size for accurate percentages
	total := m.statsTotalLogsEver
	if total == 0 {
		total = 1 // Avoid division by zero
	}

	for _, sev := range severityOrder {
		count := m.lifetimeSeverityCounts[sev] // Use lifetime data
		percentage := float64(count) * 100.0 / float64(total)
		valueStyle := lipgloss.NewStyle().Foreground(colors[sev])
		value := valueStyle.Render(fmt.Sprintf("%d (%.1f%%)", count, percentage))
		stats = append(stats, StatItem{sev, value})
	}

	return stats
}

func (m *DashboardModel) calculateHostStats() []StatItem {
	// Use lifetime host counts directly
	hostCounts := make(map[string]int)
	for host, count := range m.lifetimeHostCounts {
		hostCounts[host] = int(count)
	}

	return m.sortAndFormatStats(hostCounts)
}

func (m *DashboardModel) calculateServiceStats() []StatItem {
	// Use lifetime service counts directly
	serviceCounts := make(map[string]int)
	for service, count := range m.lifetimeServiceCounts {
		serviceCounts[service] = int(count)
	}

	return m.sortAndFormatStats(serviceCounts)
}

func (m *DashboardModel) sortAndFormatStats(counts map[string]int) []StatItem {
	type kv struct {
		Key   string
		Value int
	}

	var kvList []kv
	for k, v := range counts {
		kvList = append(kvList, kv{k, v})
	}

	sort.Slice(kvList, func(i, j int) bool {
		// Primary sort: by count (descending)
		if kvList[i].Value != kvList[j].Value {
			return kvList[i].Value > kvList[j].Value
		}
		// Secondary sort: by key name (ascending) when counts are equal
		return kvList[i].Key < kvList[j].Key
	})

	var stats []StatItem
	total := m.statsTotalLogsEver // Use lifetime total for accurate percentages

	for _, kv := range kvList {
		percentage := float64(kv.Value) * 100.0 / float64(total)
		value := fmt.Sprintf("%d (%.1f%%)", kv.Value, percentage)
		stats = append(stats, StatItem{kv.Key, value})
	}

	return stats
}

func (m *DashboardModel) formatBytes(bytes int64) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	}
	if bytes < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(bytes)/1024)
	}
	if bytes < 1024*1024*1024 {
		return fmt.Sprintf("%.1f MB", float64(bytes)/(1024*1024))
	}
	return fmt.Sprintf("%.1f GB", float64(bytes)/(1024*1024*1024))
}

func (m *DashboardModel) formatUptime() string {
	if m.statsStartTime.IsZero() {
		return "0s"
	}
	duration := time.Since(m.statsStartTime)

	if duration < time.Minute {
		return fmt.Sprintf("%.0fs", duration.Seconds())
	}
	if duration < time.Hour {
		return fmt.Sprintf("%.1fm", duration.Minutes())
	}
	return fmt.Sprintf("%.1fh", duration.Hours())
}

func (m *DashboardModel) formatCurrentRate() string {
	// If no historical data yet, show current second rate
	if len(m.statsRecentCounts) == 0 {
		return fmt.Sprintf("%.1f logs/sec", float64(m.statsLogsThisSecond))
	}

	// Calculate average over recent window (last 5 seconds for more responsive rate)
	totalLogs := 0
	validSeconds := 0
	cutoffTime := time.Now().Add(-5 * time.Second)

	// Count logs from recent complete seconds
	for i, timestamp := range m.statsRecentTimes {
		if timestamp.After(cutoffTime) {
			totalLogs += m.statsRecentCounts[i]
			validSeconds++
		}
	}

	// Always include current partial second (even if 0)
	totalLogs += m.statsLogsThisSecond
	validSeconds++

	if validSeconds == 0 {
		return "0.0 logs/sec"
	}

	// Calculate rate over the window
	rate := float64(totalLogs) / float64(validSeconds)
	return fmt.Sprintf("%.1f logs/sec", rate)
}

// combineSideBySide combines two sections side by side (using lipgloss like the main dashboard)
func (m *DashboardModel) combineSideBySide(left, right string) string {
	// Use lipgloss.JoinHorizontal for consistent layout like the main dashboard
	return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
}

// renderSeveritySection renders severity statistics with a visual bar chart (using dashboard styling)
func (m *DashboardModel) renderSeveritySection(stats []StatItem, width int) string {
	// Use chartTitleStyle for consistent title formatting
	titleContent := chartTitleStyle.Render("Severity Distribution")

	var contentLines []string

	// Calculate the maximum key length for alignment
	maxKeyLen := 0
	for _, item := range stats {
		if len(item.Key) > maxKeyLen {
			maxKeyLen = len(item.Key)
		}
	}

	// Add padding
	maxKeyLen += 3

	// Render each statistic item with aligned values
	for _, item := range stats {
		keyStyle := lipgloss.NewStyle().
			Foreground(ColorWhite).
			Width(maxKeyLen).
			Align(lipgloss.Left)

		line := fmt.Sprintf("%s %s",
			keyStyle.Render(item.Key+":"),
			item.Value) // Value already has color styling
		contentLines = append(contentLines, line)
	}

	contentLines = append(contentLines, "") // Add spacing

	// Add horizontal stacked bar chart
	contentLines = append(contentLines, m.renderSeverityBarChart(width-4)) // Account for section padding

	content := strings.Join(contentLines, "\n")

	// Use sectionStyle for consistent section formatting with borders
	sectionContent := lipgloss.JoinVertical(lipgloss.Left, titleContent, content)

	return sectionStyle.
		Width(width).
		Render(sectionContent)
}

// renderSeverityBarChart creates a horizontal stacked bar chart for severity distribution
func (m *DashboardModel) renderSeverityBarChart(width int) string {
	// Use full available width for the bar
	barWidth := width - 2 // Account for border characters │ │

	severityOrder := []string{"FATAL", "ERROR", "WARN", "INFO", "DEBUG", "TRACE"}
	colors := map[string]lipgloss.Color{
		"FATAL": ColorRed, "ERROR": ColorRed, "WARN": ColorOrange,
		"INFO": ColorBlue, "DEBUG": ColorGray, "TRACE": ColorGray,
	}

	// Build the bar
	var bar string
	total := m.statsTotalLogsEver
	if total == 0 {
		// Empty bar if no logs
		bar = strings.Repeat("░", barWidth)
	} else {
		remainingWidth := barWidth
		for _, sev := range severityOrder {
			count := m.lifetimeSeverityCounts[sev]
			if count > 0 && remainingWidth > 0 {
				// Calculate width for this severity
				segmentWidth := int(float64(count) * float64(barWidth) / float64(total))
				if segmentWidth == 0 && count > 0 {
					segmentWidth = 1 // At least 1 char for non-zero counts
				}
				if segmentWidth > remainingWidth {
					segmentWidth = remainingWidth
				}

				// Add colored segment
				style := lipgloss.NewStyle().Foreground(colors[sev])
				bar += style.Render(strings.Repeat("█", segmentWidth))
				remainingWidth -= segmentWidth
			}
		}

		// Fill any remaining space with empty bar
		if remainingWidth > 0 {
			bar += strings.Repeat("░", remainingWidth)
		}
	}

	return fmt.Sprintf("│%s│", bar)
}

// AttributeStatFormatted represents a formatted attribute statistic
type AttributeStatFormatted struct {
	Key        string
	Value      string
	Count      int
	Percentage float64
}

// calculateAttributeStatsFormatted returns formatted attribute statistics with separated key/value
func (m *DashboardModel) calculateAttributeStatsFormatted() []AttributeStatFormatted {
	// Convert lifetime attribute counts to formatted stats
	type kv struct {
		Key   string
		Value string
		Count int
	}

	var kvList []kv
	for attrKey, count := range m.lifetimeAttrCounts {
		// Skip entries that don't follow key=value format
		parts := strings.SplitN(attrKey, "=", 2)
		if len(parts) != 2 {
			continue
		}

		// Don't skip any keys for now - show all attributes
		// if parts[0] == "host" || parts[0] == "service.name" || parts[0] == "service" {
		// 	continue
		// }

		kvList = append(kvList, kv{
			Key:   parts[0],
			Value: parts[1],
			Count: int(count),
		})
	}

	// Sort by count (descending) then by key+value (ascending)
	sort.Slice(kvList, func(i, j int) bool {
		if kvList[i].Count != kvList[j].Count {
			return kvList[i].Count > kvList[j].Count
		}
		if kvList[i].Key != kvList[j].Key {
			return kvList[i].Key < kvList[j].Key
		}
		return kvList[i].Value < kvList[j].Value
	})

	// Convert to formatted stats
	var stats []AttributeStatFormatted
	total := m.statsTotalLogsEver

	for _, kv := range kvList {
		percentage := float64(kv.Count) * 100.0 / float64(total)
		stats = append(stats, AttributeStatFormatted{
			Key:        kv.Key,
			Value:      kv.Value,
			Count:      kv.Count,
			Percentage: percentage,
		})
	}

	return stats
}

// renderAttributeSection renders the attribute statistics section with columnar format (using dashboard styling)
func (m *DashboardModel) renderAttributeSection(stats []AttributeStatFormatted, width int) string {
	// Use chartTitleStyle for consistent title formatting
	titleContent := chartTitleStyle.Render("Top Attributes")

	var contentLines []string

	// Calculate column widths using full available width
	availableWidth := width - 4 // Account for section borders and padding
	
	// Reserve fixed space for count column and separators
	countColumnWidth := 15 // Fixed width for "Count (%)" column
	separatorWidth := 6    // " │ " separators (2 * 3 chars)
	
	// Use remaining space for key and value columns
	keyValueWidth := availableWidth - countColumnWidth - separatorWidth
	
	// Find actual max lengths in the data
	actualMaxKeyLen := 0
	actualMaxValueLen := 0
	for _, stat := range stats {
		if len(stat.Key) > actualMaxKeyLen {
			actualMaxKeyLen = len(stat.Key)
		}
		if len(stat.Value) > actualMaxValueLen {
			actualMaxValueLen = len(stat.Value)
		}
	}
	
	// Distribute available space between key and value columns
	// Give them proportional space based on their actual content, with reasonable minimums
	minKeyLen := max(8, min(actualMaxKeyLen, 15))  // At least 8, prefer actual up to 15
	minValueLen := max(12, min(actualMaxValueLen, 20)) // At least 12, prefer actual up to 20
	
	var maxKeyLen, maxValueLen int
	
	if minKeyLen + minValueLen <= keyValueWidth {
		// If we have enough space, distribute the extra proportionally
		extraSpace := keyValueWidth - minKeyLen - minValueLen
		if actualMaxKeyLen + actualMaxValueLen > 0 {
			keyRatio := float64(actualMaxKeyLen) / float64(actualMaxKeyLen + actualMaxValueLen)
			maxKeyLen = minKeyLen + int(float64(extraSpace) * keyRatio)
			maxValueLen = minValueLen + int(float64(extraSpace) * (1 - keyRatio))
		} else {
			// Equal split if no actual data
			maxKeyLen = minKeyLen + extraSpace/2
			maxValueLen = minValueLen + extraSpace/2
		}
	} else {
		// If we don't have enough space, scale down proportionally
		ratio := float64(keyValueWidth) / float64(minKeyLen + minValueLen)
		maxKeyLen = max(8, int(float64(minKeyLen) * ratio))
		maxValueLen = max(8, int(float64(minValueLen) * ratio))
	}

	// Header
	headerStyle := lipgloss.NewStyle().Foreground(ColorWhite).Bold(true)
	header := fmt.Sprintf("%-*s │ %-*s │ %s",
		maxKeyLen, "Key",
		maxValueLen, "Value",
		"Count (%)")
	contentLines = append(contentLines, headerStyle.Render(header))

	// Divider line
	dividerStyle := lipgloss.NewStyle().Foreground(ColorGray)
	contentLines = append(contentLines, dividerStyle.Render(strings.Repeat("─", len(header))))

	// Render each attribute
	for _, stat := range stats {
		key := stat.Key
		if len(key) > maxKeyLen {
			key = key[:maxKeyLen-3] + "..."
		}

		value := stat.Value
		if len(value) > maxValueLen {
			value = value[:maxValueLen-3] + "..."
		}

		keyStyle := lipgloss.NewStyle().Foreground(ColorWhite)
		valueStyle := lipgloss.NewStyle().Foreground(ColorBlue)
		countStyle := lipgloss.NewStyle().Foreground(ColorGreen).Bold(true)

		line := fmt.Sprintf("%s │ %s │ %s",
			keyStyle.Render(fmt.Sprintf("%-*s", maxKeyLen, key)),
			valueStyle.Render(fmt.Sprintf("%-*s", maxValueLen, value)),
			countStyle.Render(fmt.Sprintf("%d (%.1f%%)", stat.Count, stat.Percentage)))

		contentLines = append(contentLines, line)
	}

	content := strings.Join(contentLines, "\n")

	// Use sectionStyle for consistent section formatting with borders
	sectionContent := lipgloss.JoinVertical(lipgloss.Left, titleContent, content)

	return sectionStyle.
		Width(width).
		Render(sectionContent)
}
