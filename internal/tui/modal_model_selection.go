package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// renderModelSelectionModal renders the model selection modal
func (m *DashboardModel) renderModelSelectionModal() string {
	// Calculate dimensions - smaller modal
	modalWidth := min(m.width-16, 60)  // Smaller width
	modalHeight := min(m.height-8, 20) // Smaller height

	// Account for borders and headers
	contentWidth := modalWidth - 4   // Modal borders
	contentHeight := modalHeight - 4 // Header + status

	// Build model list content with scrolling
	var modelLines []string

	if len(m.availableModelsList) == 0 {
		modelLines = append(modelLines, "No models available")
	} else {
		// Calculate visible range based on available height
		// Reserve space for header (1), status bar (1), borders (2), scroll indicators (2)
		maxVisible := max(5, contentHeight-4) // At least 5 models visible
		totalModels := len(m.availableModelsList)

		// Calculate scroll position to keep selected model visible
		startIdx := 0
		if m.selectedModelIndex >= maxVisible {
			// Keep selected model in the middle if possible
			startIdx = m.selectedModelIndex - maxVisible/2
			if startIdx+maxVisible > totalModels {
				startIdx = totalModels - maxVisible
			}
		}
		if startIdx < 0 {
			startIdx = 0
		}

		endIdx := startIdx + maxVisible
		if endIdx > totalModels {
			endIdx = totalModels
		}

		// Add scroll indicator at top
		if startIdx > 0 {
			scrollUpStyle := lipgloss.NewStyle().Foreground(ColorGray)
			modelLines = append(modelLines, scrollUpStyle.Render(fmt.Sprintf("  ↑ %d more models above", startIdx)))
		}

		// Show visible models
		for i := startIdx; i < endIdx; i++ {
			model := m.availableModelsList[i]
			prefix := "  "
			if i == m.selectedModelIndex {
				prefix = "► "
			}

			// Highlight current model
			displayModel := model
			if model == m.aiModelName {
				displayModel = model + " (current)"
			}

			// Truncate long model names
			maxModelLen := contentWidth - 6 // Account for prefix and padding
			if len(displayModel) > maxModelLen {
				displayModel = displayModel[:maxModelLen-3] + "..."
			}

			line := prefix + displayModel

			// Style the line
			if i == m.selectedModelIndex {
				// Highlight selected model
				selectedStyle := lipgloss.NewStyle().
					Foreground(ColorBlue).
					Bold(true)
				line = selectedStyle.Render(line)
			} else if model == m.aiModelName {
				// Highlight current model
				currentStyle := lipgloss.NewStyle().
					Foreground(ColorGreen)
				line = currentStyle.Render(line)
			}

			modelLines = append(modelLines, line)
		}

		// Add scroll indicator at bottom
		if endIdx < totalModels {
			scrollDownStyle := lipgloss.NewStyle().Foreground(ColorGray)
			modelLines = append(modelLines, scrollDownStyle.Render(fmt.Sprintf("  ↓ %d more models below", totalModels-endIdx)))
		}
	}

	// Create content pane
	contentPane := lipgloss.NewStyle().
		Width(contentWidth).
		Height(contentHeight).
		Border(lipgloss.NormalBorder()).
		BorderForeground(ColorBlue).
		Render(strings.Join(modelLines, "\n"))

	// Header
	header := lipgloss.NewStyle().
		Width(contentWidth).
		Foreground(ColorBlue).
		Bold(true).
		Render(fmt.Sprintf("Select AI Model (%s)", m.aiServiceName))

	// Status bar
	statusBar := lipgloss.NewStyle().
		Foreground(ColorGray).
		Render("↑↓/Wheel: Navigate • PgUp/PgDn: Page • Enter: Select • ESC: Cancel")

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