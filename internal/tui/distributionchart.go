package tui

import (
	"fmt"
	"strings"

	"github.com/control-theory/gonzo/internal/memory"

	"github.com/charmbracelet/lipgloss"
)

// calculateDistributionContentLines calculates lines needed for distribution chart content
func (m *DashboardModel) calculateDistributionContentLines() int {
	// Now used for drain3 patterns chart - match counts chart sizing
	return 8 // Increased from 7 to match log counts chart height
}

// renderDistributionChart renders the frequency distribution chart
func (m *DashboardModel) renderDistributionChart(width, height int) string {
	// Use MaxHeight instead of Height to prevent empty space
	style := sectionStyle.Width(width).Height(height)
	if m.activeSection == SectionDistribution {
		style = activeSectionStyle.Width(width).Height(height)
	}

	title := chartTitleStyle.Render("Log Patterns")

	var content string
	lifetimeWords := m.getLifetimeWordEntries()
	if len(lifetimeWords) > 0 {
		content = m.renderDistributionContent(width, lifetimeWords)
	} else {
		content = helpStyle.Render("No data available")
	}

	return style.Render(lipgloss.JoinVertical(lipgloss.Left, title, content))
}

// renderDistributionContent renders the distribution chart content
func (m *DashboardModel) renderDistributionContent(chartWidth int, words []*memory.FrequencyEntry) string {
	ranges := []struct {
		min, max int64
		label    string
	}{
		{1, 1, "1 occurrence"},
		{2, 5, "2-5 occurrences"},
		{6, 10, "6-10 occurrences"},
		{11, 25, "11-25 occurrences"},
		{26, 100, "26-100 occurrences"},
		{101, 1000, "101-1000 occurrences"},
		{1001, 999999, "1000+ occurrences"},
	}

	distribution := make([]int, len(ranges))

	for _, word := range words {
		for i, r := range ranges {
			if word.Count >= r.min && word.Count <= r.max {
				distribution[i]++
				break
			}
		}
	}

	maxCount := 0
	for _, count := range distribution {
		if count > maxCount {
			maxCount = count
		}
	}

	// Ensure maxCount is at least 1 to avoid division by zero in bar calculations
	if maxCount == 0 {
		maxCount = 1
	}

	// Calculate width needed for count field based on max count
	countFieldWidth := len(fmt.Sprintf("%d", maxCount))
	if countFieldWidth < 3 {
		countFieldWidth = 3 // Minimum width for readability
	}

	// Calculate dynamic layout based on chart width
	availableWidth := chartWidth - 2           // Account for borders (4) + padding (2)
	fixedOverhead := (countFieldWidth + 2) + 2 // " %Nd " (N+2) + "||" (2) (no numbering for distribution)
	barWidth := 15                             // Default bar width
	if availableWidth < 40 {
		barWidth = 8 // Smaller bar for narrow charts
	}

	labelWidth := availableWidth - fixedOverhead - barWidth
	if labelWidth < 12 {
		labelWidth = 12 // Minimum label width for range descriptions
	}

	var lines []string
	selectedIdx := m.selectedIndex[SectionDistribution]

	for i, r := range ranges {
		count := distribution[i]

		filled := int((float64(count) / float64(maxCount)) * float64(barWidth))
		if filled == 0 && count > 0 {
			filled = 1
		}

		bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

		// Truncate label if too long for dynamic width
		label := r.label
		if len(label) > labelWidth {
			label = label[:labelWidth-3] + "..."
		}

		// Dynamic format string with calculated label width and count field width
		formatStr := fmt.Sprintf("%%-%ds %%%dd |%%s|", labelWidth, countFieldWidth)
		line := fmt.Sprintf(formatStr, label, count, bar)

		if i == selectedIdx && m.activeSection == SectionDistribution {
			line = lipgloss.NewStyle().
				Background(ColorYellow).
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
