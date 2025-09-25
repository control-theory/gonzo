package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

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