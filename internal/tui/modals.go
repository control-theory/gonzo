package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// renderModalOverlay renders modal using full screen with padding
func (m *DashboardModel) renderModalOverlay() string {
	// Check for patterns modal first
	if m.showPatternsModal {
		return m.renderPatternsModal()
	}

	// Check for counts modal
	if m.showCountsModal {
		return m.renderCountsModal()
	}

	// Check for model selection modal
	if m.showModelSelectionModal {
		return m.renderModelSelectionModal()
	}

	// Check for severity filter modal
	if m.showSeverityFilterModal {
		return m.renderSeverityFilterModal()
	}

	// Check if this is a log details modal
	isLogDetailsModal := m.currentLogEntry != nil

	if isLogDetailsModal {
		return m.renderSplitModal()
	} else {
		return m.renderSingleModal()
	}
}

// renderSplitModal renders log details modal with proper Bubbles split layout
func (m *DashboardModel) renderSplitModal() string {
	// Calculate dimensions
	modalWidth := m.width - 8   // 4 chars margin on each side
	modalHeight := m.height - 6 // 3 lines margin top and bottom

	// Account for borders and headers
	contentWidth := modalWidth - 4   // Modal borders
	contentHeight := modalHeight - 6 // Header + status

	// Split layout: 70% info, 30% chat
	infoWidth := int(float64(contentWidth)*0.7) - 1 // -1 for separator
	chatWidth := contentWidth - infoWidth - 1

	// Update viewport sizes
	m.infoViewport.Width = infoWidth
	m.infoViewport.Height = contentHeight
	m.chatViewport.Width = chatWidth
	m.chatViewport.Height = contentHeight

	// Update content with proper text wrapping
	if m.currentLogEntry != nil {
		// Account for minimal border padding (viewport handles most of its own spacing)
		contentAreaWidth := infoWidth - 2
		if contentAreaWidth < 10 {
			contentAreaWidth = 10
		}
		infoContent := m.formatLogDetails(*m.currentLogEntry, contentAreaWidth)
		wrappedInfoContent := m.wrapTextToWidth(infoContent, contentAreaWidth)
		m.infoViewport.SetContent(wrappedInfoContent)
	}

	// Prepare chat content with proper text wrapping
	var chatContent strings.Builder

	// Show chat history with proper colors
	if len(m.chatHistory) > 0 {
		for i, msg := range m.chatHistory {
			if i > 0 {
				chatContent.WriteString("\n")
			}

			// Apply colors and wrap text to viewport width
			var styledMsg string
			// Account for viewport's internal rendering - use most of the width
			msgWidth := chatWidth - 2 // Minimal padding for clean display

			if strings.HasPrefix(msg, "You:") {
				// User messages in light gray
				userStyle := lipgloss.NewStyle().Foreground(ColorGray)
				wrappedMsg := m.wrapTextToWidth(msg, msgWidth)
				styledMsg = userStyle.Render(wrappedMsg)
			} else {
				// AI messages in blue
				aiStyle := lipgloss.NewStyle().Foreground(ColorBlue)
				wrappedMsg := m.wrapTextToWidth(msg, msgWidth)
				styledMsg = aiStyle.Render(wrappedMsg)
			}

			chatContent.WriteString(styledMsg)
		}
	}

	// Add chat input section
	if len(m.chatHistory) > 0 {
		chatContent.WriteString("\n\n")
	}

	// Show chat input or prompt text
	if m.modalActiveSection == "chat" && m.chatActive {
		chatContent.WriteString(m.chatInput.View())
	} else {
		// Wrap prompt text to viewport width
		msgWidth := chatWidth - 2
		if len(m.chatHistory) == 0 {
			promptText := m.wrapTextToWidth("No chat yet.\nTab here to start chatting", msgWidth)
			chatContent.WriteString(promptText)
		} else {
			promptText := m.wrapTextToWidth("Tab here to continue chatting", msgWidth)
			chatContent.WriteString(promptText)
		}
	}

	// Set pre-wrapped content to viewport (no double-wrapping)
	m.chatViewport.SetContent(chatContent.String())

	// Only auto-scroll to bottom when flagged (new content added)
	if m.chatAutoScroll {
		m.chatViewport.GotoBottom()
		m.chatAutoScroll = false // Reset flag after auto-scrolling
	}

	// Create side-by-side layout using lipgloss
	infoPane := lipgloss.NewStyle().
		Width(infoWidth).
		Height(contentHeight).
		Border(lipgloss.NormalBorder()).
		BorderForeground(func() lipgloss.Color {
			if m.modalActiveSection == "info" {
				return ColorBlue
			}
			return ColorGray
		}()).
		Render(m.infoViewport.View())

	chatPane := lipgloss.NewStyle().
		Width(chatWidth).
		Height(contentHeight).
		Border(lipgloss.NormalBorder()).
		BorderForeground(func() lipgloss.Color {
			if m.modalActiveSection == "chat" {
				return ColorBlue
			}
			return ColorGray
		}()).
		Render(m.chatViewport.View())

	// Combine panes horizontally
	content := lipgloss.JoinHorizontal(lipgloss.Top, infoPane, chatPane)

	// Add header with AI status
	headerTitle := "Log Detail Modal"
	var aiStatus string
	if m.aiConfigured {
		aiStatus = fmt.Sprintf("%s: %s", m.aiServiceName, m.aiModelName)
	} else {
		aiStatus = "AI Not Available: Config Error"
	}

	// Create header with title on left and AI status on right
	headerLeft := lipgloss.NewStyle().
		Foreground(ColorBlue).
		Bold(true).
		Render(headerTitle)

	headerRight := lipgloss.NewStyle().
		Foreground(func() lipgloss.Color {
			if m.aiConfigured {
				return ColorGreen
			}
			return ColorOrange
		}()).
		Render(aiStatus)

	// Calculate spacing between left and right
	headerSpacing := contentWidth - len(headerTitle) - len(aiStatus)
	if headerSpacing < 1 {
		headerSpacing = 1
	}

	header := lipgloss.JoinHorizontal(lipgloss.Top,
		headerLeft,
		strings.Repeat(" ", headerSpacing),
		headerRight,
	)

	// Add tab indicators
	infoTab := "Details"
	chatTab := "Chat"
	if m.modalActiveSection == "info" {
		infoTab = "► " + infoTab
		chatTab = "  " + chatTab
	} else {
		infoTab = "  " + infoTab
		chatTab = "► " + chatTab
	}

	tabs := lipgloss.JoinHorizontal(lipgloss.Left,
		lipgloss.NewStyle().Foreground(func() lipgloss.Color {
			if m.modalActiveSection == "info" {
				return ColorGreen
			}
			return ColorGray
		}()).Render(infoTab),
		strings.Repeat(" ", contentWidth-len(infoTab)-len(chatTab)),
		lipgloss.NewStyle().Foreground(func() lipgloss.Color {
			if m.modalActiveSection == "chat" {
				return ColorGreen
			}
			return ColorGray
		}()).Render(chatTab),
	)

	// Status bar
	statusBar := m.renderModalStatusBar()

	// Combine all parts
	modal := lipgloss.JoinVertical(lipgloss.Left, header, tabs, content, statusBar)

	// Add outer border and center
	finalModal := lipgloss.NewStyle().
		Width(modalWidth).
		Height(modalHeight).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorBlue).
		Render(modal)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, finalModal)
}

// renderSingleModal renders non-log-details modal with simple single layout
func (m *DashboardModel) renderSingleModal() string {
	// Calculate dimensions
	modalWidth := m.width - 8   // 4 chars margin on each side
	modalHeight := m.height - 6 // 3 lines margin top and bottom

	// Account for borders and headers
	contentWidth := modalWidth - 4   // Modal borders
	contentHeight := modalHeight - 4 // Header + status

	// Update viewport
	m.infoViewport.Width = contentWidth
	m.infoViewport.Height = contentHeight
	m.infoViewport.SetContent(m.modalContent)

	// Create content pane
	contentPane := lipgloss.NewStyle().
		Width(contentWidth).
		Height(contentHeight).
		Border(lipgloss.NormalBorder()).
		BorderForeground(ColorGray).
		Render(m.infoViewport.View())

	// Header
	header := lipgloss.NewStyle().
		Width(contentWidth).
		Foreground(ColorBlue).
		Bold(true).
		Render("Top Values")

	// Status bar
	statusBar := m.renderModalStatusBar()

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

// renderModalStatusBar renders the status bar for modals
func (m *DashboardModel) renderModalStatusBar() string {
	var statusItems []string

	if m.currentLogEntry != nil {
		// Split modal help text
		statusItems = append(statusItems, "Tab/Click: Switch panes (Details/Chat)")

		if m.modalActiveSection == "chat" && m.chatActive {
			statusItems = append(statusItems, "Enter: Send message", "ESC: Stop typing")
		} else {
			if m.aiClient != nil {
				statusItems = append(statusItems, "i: AI Analysis")
			}
			// Add wrapping toggle for log details modal
			if m.attributeWrappingEnabled {
				statusItems = append(statusItems, "w: Disable wrapping")
			} else {
				statusItems = append(statusItems, "w: Enable wrapping")
			}
			statusItems = append(statusItems, "↑↓/Wheel: Scroll", "PgUp/PgDn: Page")
		}
	} else {
		// Single modal help text (like Top Values modal)
		statusItems = append(statusItems, "↑↓/Wheel: Scroll", "PgUp/PgDn: Page")
	}

	// Always show close option
	statusItems = append(statusItems, "ESC: Close")

	// Format status bar
	statusStyle := lipgloss.NewStyle().
		Foreground(ColorGray)

	return statusStyle.Render(strings.Join(statusItems, " • "))
}

// renderHelpModal renders the help modal using full screen
func (m *DashboardModel) renderHelpModal() string {
	// Calculate dimensions
	modalWidth := m.width - 8   // Leave 4 chars margin on each side
	modalHeight := m.height - 4 // Leave 2 lines margin top and bottom

	// Account for borders and headers
	contentWidth := modalWidth - 4   // Modal borders
	contentHeight := modalHeight - 4 // Header + status

	// Update viewport
	m.infoViewport.Width = contentWidth
	m.infoViewport.Height = contentHeight

	// Get help content and wrap it properly
	helpContent := m.renderHelpModalContent()
	wrappedContent := m.wrapTextToWidth(helpContent, contentWidth)
	m.infoViewport.SetContent(wrappedContent)

	// Create content pane
	contentPane := lipgloss.NewStyle().
		Width(contentWidth).
		Height(contentHeight).
		Border(lipgloss.NormalBorder()).
		BorderForeground(ColorGray).
		Render(m.infoViewport.View())

	// Header
	header := lipgloss.NewStyle().
		Width(contentWidth).
		Foreground(ColorBlue).
		Bold(true).
		Render("🎯 Help & Documentation")

	// Status bar
	statusBar := lipgloss.NewStyle().
		Foreground(ColorGray).
		Render("↑↓/Wheel: Scroll • PgUp/PgDn: Page • ?/h: Toggle Help • ESC: Close")

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

// renderHelpModalContent returns the help modal content without positioning
func (m *DashboardModel) renderHelpModalContent() string {
	helpContent := `🎯 Log Analyzer Dashboard Help

NAVIGATION:
  Tab/Shift+Tab  - Navigate between sections
  Mouse Click    - Click on any section to switch to it
  ↑/↓ or k/j     - Move selection within section
  Mouse Wheel    - Scroll up/down to navigate selections
  Enter          - Show details for selected item
  Escape         - Close modal/exit filter mode

ACTIONS:
  /              - Activate filter (regex supported)
  s              - Search and highlight text in logs
  Ctrl+f         - Open severity filter modal
  f              - Open fullscreen log viewer modal
  Space          - Pause/unpause UI updates
  c              - Toggle Host/Service columns in log view
  r              - Reset all data (manual reset)
  u/U            - Cycle update intervals (forward/backward)
  i              - Show comprehensive statistics modal
  i              - AI analysis (when viewing log details)
  m              - Switch AI model (shows available models)
  ? or h         - Toggle this help
  q/Ctrl+C       - Quit

LOG VIEWER NAVIGATION:
  Home           - Jump to top of log buffer (stops auto-scroll)
  End            - Jump to latest logs (resumes auto-scroll)
  PgUp/PgDn      - Navigate by pages (10 entries at a time)
  ↑/↓ or k/j     - Navigate individual entries with smart auto-scroll

SECTIONS:
  Words          - Most frequent words in logs
  Attributes     - OTLP attributes by unique value count
  Log Patterns   - Common log message patterns (Drain3)
  Counts         - Log counts over time
  Logs           - Navigate and inspect individual log entries

FILTER & SEARCH:
  Filter (/): Type regex patterns to filter displayed logs
  Search (s): Type text to highlight in displayed logs
  Severity (Ctrl+f): Filter by log severity levels
  Examples: "error", "k8s.*pod", "severity.*INFO"

AI ANALYSIS:
  Set environment variables for AI-powered log analysis:
  • OPENAI_API_KEY   - Your API key (required)
  • OPENAI_API_BASE  - Custom endpoint (optional)
  
  Examples:
  • OpenAI: export OPENAI_API_KEY=sk-your-key
  • LM Studio: export OPENAI_API_BASE=http://localhost:1234/v1
  • Ollama: export OPENAI_API_BASE=http://localhost:11434/v1
  
  Press 'i' in log detail modal for AI insights.
  Press 'm' anywhere to switch between available models.
`

	return lipgloss.NewStyle().
		Width(65).
		Render(helpContent)
}

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

// renderStatsModal renders the statistics modal showing comprehensive log stats
func (m *DashboardModel) renderStatsModal() string {
	// Calculate dimensions
	modalWidth := m.width - 8   // Leave 4 chars margin on each side
	modalHeight := m.height - 4 // Leave 2 lines margin top and bottom

	// Account for borders and headers
	contentWidth := modalWidth - 4   // Modal borders
	contentHeight := modalHeight - 4 // Header + status

	// Update viewport
	m.infoViewport.Width = contentWidth
	m.infoViewport.Height = contentHeight

	// Get statistics content and set it to viewport
	statsContent := m.renderStatsContent(contentWidth)
	m.infoViewport.SetContent(statsContent)

	// Create content pane
	contentPane := lipgloss.NewStyle().
		Width(contentWidth).
		Height(contentHeight).
		Border(lipgloss.NormalBorder()).
		BorderForeground(ColorGray).
		Render(m.infoViewport.View())

	// Header with title
	header := lipgloss.NewStyle().
		Width(contentWidth).
		Foreground(ColorBlue).
		Bold(true).
		Render("Log Statistics")

	// Status bar
	statusBar := lipgloss.NewStyle().
		Foreground(ColorGray).
		Render("↑↓/Wheel: Scroll • PgUp/PgDn: Page • i: Toggle Stats • ESC: Close")

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

// renderCountsModal renders the log counts modal with heatmap and analysis sections
func (m *DashboardModel) renderCountsModal() string {
	// Calculate dimensions
	modalWidth := m.width - 8   // Leave 4 chars margin on each side
	modalHeight := m.height - 4 // Leave 2 lines margin top and bottom

	// Account for borders and headers
	contentWidth := modalWidth - 4   // Modal borders
	contentHeight := modalHeight - 4 // Header + status

	// Update viewport
	m.infoViewport.Width = contentWidth
	m.infoViewport.Height = contentHeight

	// Get counts modal content and set it to viewport
	countsContent := m.renderCountsModalContent(contentWidth)
	m.infoViewport.SetContent(countsContent)

	// Create content pane
	contentPane := lipgloss.NewStyle().
		Width(contentWidth).
		Height(contentHeight).
		Border(lipgloss.NormalBorder()).
		BorderForeground(ColorGray).
		Render(m.infoViewport.View())

	// Header with title
	header := lipgloss.NewStyle().
		Width(contentWidth).
		Foreground(ColorBlue).
		Bold(true).
		Render("Log Counts Analysis")

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

// renderCountsModalContent renders the content for the counts modal
func (m *DashboardModel) renderCountsModalContent(contentWidth int) string {
	var sections []string

	// Title section
	titleStyle := lipgloss.NewStyle().
		Foreground(ColorBlue).
		Bold(true).
		Align(lipgloss.Center).
		Width(contentWidth)

	sections = append(sections, titleStyle.Render("Log Activity Analysis"))
	sections = append(sections, "")

	// Heatmap section - full width
	heatmapSection := m.renderHeatmapSection(contentWidth)
	sections = append(sections, heatmapSection)
	sections = append(sections, "")

	// Calculate width for side-by-side sections
	halfWidth := (contentWidth - 3) / 2 // -3 for spacing between columns

	// Side-by-side sections: Patterns by Severity | Services by Severity
	patternsSection := m.renderPatternsBySeveritySection(halfWidth)
	servicesSection := m.renderServicesBySeveritySection(halfWidth)

	sideBySide := lipgloss.JoinHorizontal(lipgloss.Top, patternsSection, servicesSection)
	sections = append(sections, sideBySide)

	return strings.Join(sections, "\n")
}

// renderHeatmapSection renders the severity heatmap chart
func (m *DashboardModel) renderHeatmapSection(width int) string {
	// Use chartTitleStyle for consistent title formatting
	titleContent := chartTitleStyle.Render("Severity Activity Heatmap (Last 60 Minutes)")

	var contentLines []string

	// Create real heatmap from actual log data
	now := time.Now()

	// Always render the heatmap structure, even with no data
	// Create time axis header aligned with data (1 character per minute)
	timeHeader := "Time (mins ago):"

	// Build time header with proper 5-minute intervals
	// Create header showing every 5 minutes: 60, 55, 50, 45, 40, 35, 30, 25, 20, 15, 10, 5, 0
	dataHeader := ""
	for i := 60; i >= 0; i-- {
		if i%5 == 0 { // Show every 5 minutes
			if i >= 10 {
				// For 2-digit numbers, show both digits but only use space for tens digit position
				if i%10 == 0 { // Show full number at multiples of 10
					dataHeader += fmt.Sprintf("%2d", i)
					if i > 0 { // Skip next character since we used 2 chars
						i--
					}
				} else {
					dataHeader += " " // Just space for 5, 15, 25, etc.
				}
			} else {
				dataHeader += fmt.Sprintf("%d", i) // Single digit: 5, 0
			}
		} else {
			dataHeader += " " // Empty space for non-labeled minutes
		}
	}

	timeHeader += dataHeader
	contentLines = append(contentLines, timeHeader)
	contentLines = append(contentLines, strings.Repeat("─", len(timeHeader)))

	// Get severity order and colors
	severities := []string{"FATAL", "ERROR", "WARN", "INFO", "DEBUG", "TRACE"}
	colors := map[string]lipgloss.Color{
		"FATAL": ColorRed, "ERROR": ColorRed, "WARN": ColorOrange,
		"INFO": ColorBlue, "DEBUG": ColorGray, "TRACE": ColorGray,
	}

	// Calculate max count per severity for individual scaling
	maxCounts := make(map[string]int)
	totalCounts := make(map[string]int)
	for _, severity := range severities {
		maxCounts[severity] = 1 // Start with 1 to avoid division by zero
		totalCounts[severity] = 0
	}

	for _, minute := range m.heatmapData {
		for _, severity := range severities {
			var count int
			switch severity {
			case "FATAL":
				count = minute.Counts.Fatal + minute.Counts.Critical
			case "ERROR":
				count = minute.Counts.Error
			case "WARN":
				count = minute.Counts.Warn
			case "INFO":
				count = minute.Counts.Info
			case "DEBUG":
				count = minute.Counts.Debug
			case "TRACE":
				count = minute.Counts.Trace
			}
			totalCounts[severity] += count
			if count > maxCounts[severity] {
				maxCounts[severity] = count
			}
		}
	}

	// Calculate total counts for each severity over the 60-minute window
	severityTotals := make(map[string]int)
	for _, severity := range severities {
		total := 0
		for _, minute := range m.heatmapData {
			// Only count minutes within the last 60 minutes
			if minute.Timestamp.After(now.Add(-60 * time.Minute)) {
				switch severity {
				case "FATAL":
					total += minute.Counts.Fatal + minute.Counts.Critical
				case "ERROR":
					total += minute.Counts.Error
				case "WARN":
					total += minute.Counts.Warn
				case "INFO":
					total += minute.Counts.Info
				case "DEBUG":
					total += minute.Counts.Debug
				case "TRACE":
					total += minute.Counts.Trace
				}
			}
		}
		severityTotals[severity] = total
	}

	// Render each severity level row
	for _, severity := range severities {
		// Create severity label with total count
		severityWithCount := fmt.Sprintf("%s (%d)", severity, severityTotals[severity])
		coloredLabel := lipgloss.NewStyle().Foreground(getSeverityColor(severity)).Bold(true).Render(fmt.Sprintf("%-12s", severityWithCount))

		// Align data with time header - "Time (mins ago):" is 16 chars, so we need 16 chars total
		line := coloredLabel + "    " // 12 + 4 = 16 to match header

		// For each minute in the last 60 minutes
		// i=0 represents the current minute and will show real-time updates
		for i := 60; i >= 0; i-- {
			minuteTime := now.Add(time.Duration(-i) * time.Minute).Truncate(time.Minute)

			// Find data for this exact minute
			var minuteActivity int
			found := false
			for _, minute := range m.heatmapData {
				if minute.Timestamp.Equal(minuteTime) {
					found = true
					switch severity {
					case "FATAL":
						minuteActivity = minute.Counts.Fatal + minute.Counts.Critical
					case "ERROR":
						minuteActivity = minute.Counts.Error
					case "WARN":
						minuteActivity = minute.Counts.Warn
					case "INFO":
						minuteActivity = minute.Counts.Info
					case "DEBUG":
						minuteActivity = minute.Counts.Debug
					case "TRACE":
						minuteActivity = minute.Counts.Trace
					}
					break
				}
			}

			// Convert to visual representation using per-severity scaling
			var symbol string
			if !found || minuteActivity == 0 {
				symbol = "." // Single dot for no data
			} else {
				intensity := float64(minuteActivity) / float64(maxCounts[severity])
				if intensity > 0.7 {
					symbol = "█"
				} else if intensity > 0.4 {
					symbol = "▓"
				} else if intensity > 0.1 {
					symbol = "▒"
				} else {
					symbol = "░"
				}

			}

			// Apply color styling only if there's data
			if found && minuteActivity > 0 {
				styledSymbol := lipgloss.NewStyle().Foreground(colors[severity]).Render(symbol)
				line += styledSymbol
			} else {
				line += symbol // No color for dots
			}
		}

		contentLines = append(contentLines, line)
	}

	contentLines = append(contentLines, "")
	contentLines = append(contentLines, "Legend: █ High Activity  ▓ Medium Activity  ▒ Low Activity  . No Activity")

	content := strings.Join(contentLines, "\n")

	// Use sectionStyle for consistent section formatting with borders
	sectionContent := lipgloss.JoinVertical(lipgloss.Left, titleContent, content)

	return sectionStyle.
		Width(width).
		Render(sectionContent)
}

// renderPatternsBySeveritySection renders patterns grouped by severity using drain3 data
func (m *DashboardModel) renderPatternsBySeveritySection(width int) string {
	// Use chartTitleStyle for consistent title formatting
	titleContent := chartTitleStyle.Render("Top Patterns by Severity")

	var contentLines []string

	// Get patterns from severity-specific drain3 instances
	severities := []string{"FATAL", "ERROR", "WARN", "INFO", "DEBUG", "TRACE"}

	hasAnyData := false
	for _, severity := range severities {
		if drain3Instance, exists := m.drain3BySeverity[severity]; exists && drain3Instance != nil {
			patterns := drain3Instance.GetTopPatterns(3) // Get top 3 patterns for this severity
			if len(patterns) > 0 {
				hasAnyData = true

				// Severity header
				severityStyle := lipgloss.NewStyle().Foreground(getSeverityColor(severity)).Bold(true)
				contentLines = append(contentLines, severityStyle.Render(severity+":"))

				// Show patterns for this severity
				for i, pattern := range patterns {
					line := fmt.Sprintf("  %d. %s (%d)", i+1, pattern.Template, pattern.Count)
					contentLines = append(contentLines, line)
				}
				contentLines = append(contentLines, "")
			}
		}
	}

	if !hasAnyData {
		contentLines = append(contentLines, helpStyle.Render("No patterns detected yet..."))
	}

	content := strings.Join(contentLines, "\n")

	// Use sectionStyle for consistent section formatting with borders
	sectionContent := lipgloss.JoinVertical(lipgloss.Left, titleContent, content)

	return sectionStyle.
		Width(width).
		Render(sectionContent)
}

// renderServicesBySeveritySection renders services grouped by severity
func (m *DashboardModel) renderServicesBySeveritySection(width int) string {
	// Use chartTitleStyle for consistent title formatting
	titleContent := chartTitleStyle.Render("Top Services by Severity")

	var contentLines []string

	// Use real service data grouped by severity
	severities := []string{"FATAL", "ERROR", "WARN", "INFO", "DEBUG", "TRACE"}

	hasAnyData := false
	for _, severity := range severities {
		services := m.servicesBySeverity[severity]
		if len(services) > 0 {
			hasAnyData = true

			// Severity header
			severityStyle := lipgloss.NewStyle().Foreground(getSeverityColor(severity)).Bold(true)
			contentLines = append(contentLines, severityStyle.Render(severity+":"))

			// Show top 3 services for this severity
			for i, service := range services {
				if i >= 3 {
					break // Only show top 3
				}
				line := fmt.Sprintf("  %d. %s (%d)", i+1, service.Service, service.Count)
				contentLines = append(contentLines, line)
			}
			contentLines = append(contentLines, "")
		}
	}

	if !hasAnyData {
		contentLines = append(contentLines, helpStyle.Render("No service data available yet..."))
	}

	content := strings.Join(contentLines, "\n")

	// Use sectionStyle for consistent section formatting with borders
	sectionContent := lipgloss.JoinVertical(lipgloss.Left, titleContent, content)

	return sectionStyle.
		Width(width).
		Render(sectionContent)
}

// getSeverityColor returns the appropriate color for a severity level
func getSeverityColor(severity string) lipgloss.Color {
	switch severity {
	case "FATAL", "CRITICAL":
		return ColorRed
	case "ERROR":
		return ColorRed
	case "WARN":
		return ColorOrange
	case "INFO":
		return ColorBlue
	case "DEBUG", "TRACE":
		return ColorGray
	default:
		return ColorWhite
	}
}

// renderLogViewerModal renders the log viewer in a fullscreen modal
func (m *DashboardModel) renderLogViewerModal() string {
	// Calculate modal dimensions - leave space for borders
	modalWidth := m.width - 4   // Leave margin for borders
	modalHeight := m.height - 2 // Leave margin for borders

	// Inner content dimensions (accounting for borders)
	contentWidth := modalWidth - 2   // -2 for left/right borders
	contentHeight := modalHeight - 2 // -2 for top/bottom borders

	// Reserve space for header and status
	headerHeight := 1
	statusHeight := 1
	logAreaHeight := contentHeight - headerHeight - statusHeight

	// Get log content without border wrapper
	logLines := m.renderLogScrollContent(logAreaHeight, contentWidth)

	// Create header
	header := lipgloss.NewStyle().
		Foreground(ColorBlue).
		Bold(true).
		Width(contentWidth).
		Render("Log Viewer")

	// Create log content area with fixed height
	logArea := lipgloss.NewStyle().
		Width(contentWidth).
		Height(logAreaHeight).
		Render(lipgloss.JoinVertical(lipgloss.Left, logLines...))

	// Create status line with filter/search indicators
	var statusLeft string

	// Check for active filter/search (including while being typed)
	hasActiveFilter := m.filterActive || m.filterRegex != nil || m.filterInput.Value() != ""
	hasActiveSearch := m.searchActive || m.searchTerm != "" || m.searchInput.Value() != ""

	// Build status message
	statusParts := []string{fmt.Sprintf("Total: %d", len(m.logEntries))}

	if m.viewPaused {
		statusParts = append(statusParts, "⏸ PAUSED")
	}

	if hasActiveFilter {
		if m.filterActive {
			// Currently editing filter
			filterValue := m.filterInput.Value()
			if filterValue == "" {
				statusParts = append(statusParts, "🔍 Filter: (editing...)")
			} else {
				statusParts = append(statusParts, fmt.Sprintf("🔍 Filter: [%s] (editing)", filterValue))
			}
		} else if m.filterRegex != nil {
			// Filter applied
			statusParts = append(statusParts, fmt.Sprintf("🔍 Filter: [%s] (%d/%d)",
				m.filterInput.Value(), len(m.logEntries), len(m.allLogEntries)))
		}
	}

	if hasActiveSearch {
		if m.searchActive {
			// Currently editing search
			searchValue := m.searchInput.Value()
			if searchValue == "" {
				statusParts = append(statusParts, "🔎 Search: (editing...)")
			} else {
				statusParts = append(statusParts, fmt.Sprintf("🔎 Search: [%s] (editing)", searchValue))
			}
		} else if m.searchTerm != "" {
			// Search applied
			statusParts = append(statusParts, fmt.Sprintf("🔎 Search: [%s]", m.searchTerm))
		}
	}

	statusLeft = strings.Join(statusParts, " | ")

	// Create concise help text that fits
	helpText := "ESC:Close ↑↓:Nav Enter:Details /:Filter s:Search c:Columns"

	// Calculate available space for each side
	leftWidth := lipgloss.Width(statusLeft)
	rightWidth := lipgloss.Width(helpText)

	// If combined width exceeds available space, truncate
	if leftWidth+rightWidth+2 > contentWidth {
		// Prioritize showing status on left, truncate help on right
		availableForRight := contentWidth - leftWidth - 2
		if availableForRight < 20 {
			// If very little space, just show essential help
			helpText = "ESC:Close ?:Help"
		} else if availableForRight < 40 {
			helpText = "ESC:Close ↑↓:Nav /:Filter"
		}
	}

	// Create properly sized status sections
	padding := contentWidth - lipgloss.Width(statusLeft) - lipgloss.Width(helpText)
	if padding < 0 {
		padding = 0
	}

	statusBar := lipgloss.NewStyle().
		Foreground(ColorGray).
		Width(contentWidth).
		MaxWidth(contentWidth).
		Height(statusHeight).
		MaxHeight(statusHeight).
		Render(statusLeft + strings.Repeat(" ", padding) + helpText)

	// Combine all content
	content := lipgloss.JoinVertical(lipgloss.Left,
		header,
		logArea,
		statusBar,
	)

	// Apply border to the content - don't set height to allow content to define size
	modal := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(ColorBlue).
		Width(modalWidth).
		Render(content)

	// Center the modal on screen
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, modal)
}

// renderSeverityFilterModal renders the severity filter selection modal
func (m *DashboardModel) renderSeverityFilterModal() string {
	// Calculate dimensions - smaller modal
	modalWidth := min(m.width-16, 50)  // Smaller width for severity list
	modalHeight := min(m.height-8, 18) // Smaller height

	// Account for borders and headers
	contentWidth := modalWidth - 4   // Modal borders
	contentHeight := modalHeight - 4 // Header + status

	// Define severity levels in order (most critical first)
	severityLevels := []string{"FATAL", "CRITICAL", "ERROR", "WARN", "INFO", "DEBUG", "TRACE", "UNKNOWN"}

	// Build severity list content
	var severityLines []string

	// Add "Select All" option at the top
	selectAllPrefix := "  "
	if m.severityFilterSelected == 0 {
		selectAllPrefix = "► "
	}
	allSelected := true
	for _, severity := range severityLevels {
		if !m.severityFilter[severity] {
			allSelected = false
			break
		}
	}
	selectAllStatus := ""
	if allSelected {
		selectAllStatus = " ✓"
	}
	selectAllLine := selectAllPrefix + "Select All" + selectAllStatus

	// Style the select all line
	if m.severityFilterSelected == 0 {
		selectedStyle := lipgloss.NewStyle().
			Foreground(ColorBlue).
			Bold(true)
		selectAllLine = selectedStyle.Render(selectAllLine)
	}
	severityLines = append(severityLines, selectAllLine)

	// Add "Select None" option
	selectNonePrefix := "  "
	if m.severityFilterSelected == 1 {
		selectNonePrefix = "► "
	}
	noneSelected := true
	for _, severity := range severityLevels {
		if m.severityFilter[severity] {
			noneSelected = false
			break
		}
	}
	selectNoneStatus := ""
	if noneSelected {
		selectNoneStatus = " ✓"
	}
	selectNoneLine := selectNonePrefix + "Select None" + selectNoneStatus

	// Style the select none line
	if m.severityFilterSelected == 1 {
		selectedStyle := lipgloss.NewStyle().
			Foreground(ColorBlue).
			Bold(true)
		selectNoneLine = selectedStyle.Render(selectNoneLine)
	}
	severityLines = append(severityLines, selectNoneLine)

	// Add separator
	severityLines = append(severityLines, "")

	// Add individual severity levels (starting from index 2)
	for i, severity := range severityLevels {
		listIndex := i + 3 // Offset by 3 (select all + select none + separator)
		prefix := "  "
		if m.severityFilterSelected == listIndex {
			prefix = "► "
		}

		// Show selection status
		status := ""
		if m.severityFilter[severity] {
			status = " ✓"
		}

		line := prefix + severity + status

		// Apply severity color and selection styling
		severityColor := getSeverityColor(severity)
		if m.severityFilterSelected == listIndex {
			// Highlight selected item
			selectedStyle := lipgloss.NewStyle().
				Foreground(ColorBlue).
				Bold(true)
			line = selectedStyle.Render(line)
		} else {
			// Use severity color for non-selected items
			severityStyle := lipgloss.NewStyle().
				Foreground(severityColor)
			line = severityStyle.Render(line)
		}

		severityLines = append(severityLines, line)
	}

	// Create content pane
	contentPane := lipgloss.NewStyle().
		Width(contentWidth).
		Height(contentHeight).
		Border(lipgloss.NormalBorder()).
		BorderForeground(ColorBlue).
		Render(strings.Join(severityLines, "\n"))

	// Header
	activeCount := 0
	for _, enabled := range m.severityFilter {
		if enabled {
			activeCount++
		}
	}
	headerText := fmt.Sprintf("Severity Filter (%d/%d active)", activeCount, len(severityLevels))
	header := lipgloss.NewStyle().
		Width(contentWidth).
		Foreground(ColorBlue).
		Bold(true).
		Render(headerText)

	// Status bar
	statusBar := lipgloss.NewStyle().
		Foreground(ColorGray).
		Render("↑↓: Navigate • Space: Toggle • Enter: Apply/Select • ESC: Cancel")

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
