package tui

import (
	"fmt"
	"strings"

	"github.com/control-theory/gonzo/internal/memory"

	"github.com/charmbracelet/lipgloss"
)

// calculateAttributesContentLines calculates lines needed for attributes chart content
func (m *DashboardModel) calculateAttributesContentLines() int {
	// Always use a consistent minimum height regardless of data availability
	minLines := 8
	if m.width < 80 {
		minLines = 5 // Smaller minimum for narrow terminals
	}

	// Use lifetime attribute data instead of snapshot
	lifetimeAttrs := m.getLifetimeAttributeEntries()
	if len(lifetimeAttrs) == 0 {
		return minLines
	}

	// Use actual data count but never go below minimum
	maxItems := min(len(lifetimeAttrs), 10)
	if m.width < 80 {
		maxItems = min(maxItems, 5)
	}

	return max(maxItems, minLines)
}

// renderAttributesChart renders the attributes chart
func (m *DashboardModel) renderAttributesChart(width, height int) string {
	// Use MaxHeight instead of Height to prevent empty space
	style := sectionStyle.Width(width).Height(height)
	if m.activeSection == SectionAttributes {
		style = activeSectionStyle.Width(width).Height(height)
	}

	title := chartTitleStyle.Render("Top Attributes")

	var content string
	lifetimeAttrs := m.getLifetimeAttributeEntries()
	if len(lifetimeAttrs) > 0 {
		content = m.renderAttributesContent(width, lifetimeAttrs)
	} else {
		content = helpStyle.Render("No data available")
	}

	return style.Render(lipgloss.JoinVertical(lipgloss.Left, title, content))
}

// renderAttributesContent renders the attributes chart content
func (m *DashboardModel) renderAttributesContent(chartWidth int, attributes []*memory.AttributeStatsEntry) string {
	maxItems := 10
	if m.width < 80 {
		maxItems = 5
	}
	if len(attributes) < maxItems {
		maxItems = len(attributes)
	}

	var lines []string
	selectedIdx := m.selectedIndex[SectionAttributes]

	// Calculate maximum unique count to determine dynamic count field width
	maxUniqueCount := 0
	for _, attr := range attributes {
		if attr.UniqueValueCount > maxUniqueCount {
			maxUniqueCount = attr.UniqueValueCount
		}
	}

	// Calculate width needed for count field based on max unique count
	countFieldWidth := len(fmt.Sprintf("%d", maxUniqueCount))
	if countFieldWidth < 3 {
		countFieldWidth = 3 // Minimum width for readability
	}

	// Calculate dynamic layout based on chart width
	availableWidth := chartWidth - 2               // Account for borders (4) + padding (2)
	fixedOverhead := 4 + (countFieldWidth + 2) + 2 // "%2d. " (4) + " %Nd " (N+2) + "||" (2)
	barWidth := 15                                 // Default bar width
	if availableWidth < 40 {
		barWidth = 8 // Smaller bar for narrow charts
	}

	labelWidth := availableWidth - fixedOverhead - barWidth
	if labelWidth < 8 {
		labelWidth = 8 // Minimum label width
	}

	for i := 0; i < maxItems; i++ {
		entry := attributes[i]

		// Create bar visualization
		filled := int((float64(entry.UniqueValueCount) / float64(maxUniqueCount)) * float64(barWidth))
		if filled == 0 && entry.UniqueValueCount > 0 {
			filled = 1
		}

		bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

		// Truncate long keys to fit dynamic label width
		key := entry.Key
		if len(key) > labelWidth {
			key = key[:labelWidth-3] + "..."
		}

		// Dynamic format string with calculated label width and count field width
		formatStr := fmt.Sprintf("%%2d. %%-%ds %%%dd |%%s|", labelWidth, countFieldWidth)
		line := fmt.Sprintf(formatStr, i+1, key, entry.UniqueValueCount, bar)

		if i == selectedIdx && m.activeSection == SectionAttributes {
			line = lipgloss.NewStyle().
				Background(ColorBlue).
				Foreground(ColorBlack).
				Render(line)
		} else {
			line = lipgloss.NewStyle().
				Foreground(ColorWhite).
				Render(line)
		}

		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}
