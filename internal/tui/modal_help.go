package tui

import (
	"github.com/charmbracelet/lipgloss"
)

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
  c              - Toggle columns (Host/Service ↔ Namespace/Pod in k8s mode)
  C              - Configure visible columns (add/remove columns)
  Ctrl+l         - Toggle column width limit (100 char max)
  Ctrl+r         - Reset column widths to accomodate current buffer
  T              - Toggle timestamp mode (Log Time / Receive Time)
  r              - Reset all data (manual reset)
  u/U            - Cycle update intervals (forward/backward)
  i              - Show comprehensive statistics modal
  i              - AI analysis (when viewing log details)
  m              - Switch AI model (shows available models)
  d              - Open Dstl8 Lite web dashboard in browser
  w              - Show what's new (release notes)
  ? or h         - Toggle this help
  q/Ctrl+C       - Quit

LOG VIEWER NAVIGATION:
  Home           - Jump to top of log buffer (stops auto-scroll)
  End            - Jump to latest logs (resumes auto-scroll)
  PgUp/PgDn      - Navigate by pages (10 entries at a time)
  ↑/↓ or k/j     - Navigate individual entries with smart auto-scroll
  ←/→            - Scroll columns horizontally (when content exceeds width)

SECTIONS:
  Words          - Most frequent words in logs
  Attributes     - OTLP attributes by unique value count
  Log Patterns   - Common log message patterns (Drain3)
  Counts         - Log counts over time
  Logs           - Navigate and inspect individual log entries

FILTER & SEARCH:
  Filter (/): Type regex patterns to filter logs (searches message & attributes)
  Search (s): Type text to highlight in displayed logs
  Severity (Ctrl+f): Filter by log severity levels
  Examples: "error", "k8s.*pod", "service.name", "host.name.*prod"

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
