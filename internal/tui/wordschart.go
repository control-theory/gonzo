package tui

import (
	"fmt"
	"strings"

	"github.com/control-theory/gonzo/internal/memory"

	"github.com/charmbracelet/lipgloss"
)

// calculateWordsContentLines calculates lines needed for words chart content
func (m *DashboardModel) calculateWordsContentLines() int {
	// Always use a consistent minimum height regardless of data availability
	minLines := 8
	if m.width < 80 {
		minLines = 5 // Smaller minimum for narrow terminals
	}

	// Use lifetime word data instead of snapshot
	lifetimeWords := m.getLifetimeWordEntries()
	if len(lifetimeWords) == 0 {
		return minLines
	}

	// Use actual data count but never go below minimum
	maxItems := min(len(lifetimeWords), 10)
	if m.width < 80 {
		maxItems = min(maxItems, 5)
	}

	return max(maxItems, minLines)
}

// renderWordsChart renders the words frequency chart
func (m *DashboardModel) renderWordsChart(width, height int) string {
	// Use MaxHeight instead of Height to prevent empty space
	style := sectionStyle.Width(width).Height(height)
	if m.activeSection == SectionWords {
		style = activeSectionStyle.Width(width).Height(height)
	}

	title := chartTitleStyle.Render("Top Words")

	var content string
	lifetimeWords := m.getLifetimeWordEntries()
	if len(lifetimeWords) > 0 {
		content = m.renderWordsContent(width, lifetimeWords)
	} else {
		content = helpStyle.Render("No data available")
	}

	return style.Render(lipgloss.JoinVertical(lipgloss.Left, title, content))
}

// renderWordsContent renders the words chart content
func (m *DashboardModel) renderWordsContent(chartWidth int, words []*memory.FrequencyEntry) string {
	maxItems := 10
	if m.width < 80 {
		maxItems = 5
	}
	if len(words) < maxItems {
		maxItems = len(words)
	}

	var lines []string
	selectedIdx := m.selectedIndex[SectionWords]

	// Calculate maximum count to determine dynamic count field width
	maxCount := int64(0)
	for _, entry := range words {
		if entry.Count > maxCount {
			maxCount = entry.Count
		}
	}

	// Calculate width needed for count field based on max count
	countFieldWidth := len(fmt.Sprintf("%d", maxCount))
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
		entry := words[i]

		// Create bar visualization
		maxCount := words[0].Count
		filled := int((float64(entry.Count) / float64(maxCount)) * float64(barWidth))
		if filled == 0 && entry.Count > 0 {
			filled = 1
		}

		bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

		// Dynamic format string with calculated label width and count field width
		formatStr := fmt.Sprintf("%%2d. %%-%ds %%%dd |%%s|", labelWidth, countFieldWidth)
		line := fmt.Sprintf(formatStr, i+1, entry.Term, entry.Count, bar)

		if i == selectedIdx && m.activeSection == SectionWords {
			line = lipgloss.NewStyle().
				Background(ColorBlue).
				Foreground(ColorWhite).
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
