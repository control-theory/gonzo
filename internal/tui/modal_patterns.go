package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// renderPatternsModal renders the patterns modal showing all log patterns
func (m *DashboardModel) renderPatternsModal() string {
	// Calculate dimensions
	modalWidth := m.width - 8   // Leave 4 chars margin on each side
	modalHeight := m.height - 4 // Leave 2 lines margin top and bottom

	// Account for borders and headers
	contentWidth := modalWidth - 4   // Modal borders
	contentHeight := modalHeight - 4 // Header + status

	// Update viewport
	m.infoViewport.Width = contentWidth
	m.infoViewport.Height = contentHeight

	// Get pattern content and set it to viewport
	patternsContent := m.renderAllPatternsContent(contentWidth)
	m.infoViewport.SetContent(patternsContent)

	// Create content pane
	contentPane := lipgloss.NewStyle().
		Width(contentWidth).
		Height(contentHeight).
		Border(lipgloss.NormalBorder()).
		BorderForeground(ColorGray).
		Render(m.infoViewport.View())

	// Get pattern stats for the title
	patternCount, totalLogs := 0, 0
	if m.drain3Manager != nil {
		patternCount, totalLogs = m.drain3Manager.GetStats()
	}

	// Build title with stats
	titleText := "All Log Patterns"
	if patternCount > 0 {
		titleText = fmt.Sprintf("All Log Patterns (%d patterns from %d logs)", patternCount, totalLogs)
	}

	// Header
	header := lipgloss.NewStyle().
		Width(contentWidth).
		Foreground(ColorBlue).
		Bold(true).
		Render(titleText)

	// Status bar
	statusBar := lipgloss.NewStyle().
		Foreground(ColorGray).
		Render("↑↓/Wheel: Scroll • PgUp/PgDn: Page • ESC: Close")

	// Combine all parts
	modal := lipgloss.JoinVertical(lipgloss.Left, header, contentPane, statusBar)

	// Add outer border and center
	finalModal := lipgloss.NewStyle().
		Width(modalWidth).
		Height(modalHeight).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorBlue).
		Render(modal)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, finalModal)
}

// renderAllPatternsContent renders all patterns in the same chart style format
func (m *DashboardModel) renderAllPatternsContent(contentWidth int) string {
	if m.drain3Manager == nil {
		return helpStyle.Render("Pattern extraction not available")
	}

	// Get all patterns (no limit)
	patterns := m.drain3Manager.GetTopPatterns(0) // 0 = get all patterns

	if len(patterns) == 0 {
		return helpStyle.Render("No patterns extracted yet")
	}

	// Calculate the maximum count for bar scaling
	maxCount := 0
	for _, p := range patterns {
		if p.Count > maxCount {
			maxCount = p.Count
		}
	}

	// Build the pattern list with mini bar charts
	var lines []string

	// Calculate available width for the template text
	// Format: [bar] count% | template
	// Reserve space for: bar(15) + count%(8) + separators(3) = 26
	templateWidth := contentWidth - 26
	if templateWidth < 20 {
		templateWidth = 20
	}

	for i, pattern := range patterns {
		// Create a mini bar for each pattern
		barWidth := 12
		fillWidth := int(float64(pattern.Count) * float64(barWidth) / float64(maxCount))
		if fillWidth == 0 && pattern.Count > 0 {
			fillWidth = 1 // Ensure at least 1 char for non-zero counts
		}

		// Create the bar visual
		bar := strings.Repeat("█", fillWidth) + strings.Repeat("░", barWidth-fillWidth)

		// Format percentage
		percentage := fmt.Sprintf("%5.1f%%", pattern.Percentage)

		// Truncate template if needed
		template := pattern.Template
		if len(template) > templateWidth {
			template = template[:templateWidth-3] + "..."
		}

		// Color code based on frequency (high frequency = more important)
		var barColor lipgloss.Style
		if i < 3 {
			// Top 3 patterns - red/orange for high frequency
			barColor = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
		} else if i < 6 {
			// Middle patterns - yellow
			barColor = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
		} else {
			// Lower patterns - blue/gray
			barColor = lipgloss.NewStyle().Foreground(lipgloss.Color("12"))
		}

		// Format the line
		line := fmt.Sprintf("%s %s │ %s",
			barColor.Render(bar),
			lipgloss.NewStyle().Foreground(ColorGray).Render(percentage),
			lipgloss.NewStyle().Foreground(ColorWhite).Render(template),
		)

		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}