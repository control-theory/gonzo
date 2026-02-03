package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// prepareColumnConfigModal prepares the column config modal with current state
func (m *DashboardModel) prepareColumnConfigModal(selectedEntry *LogEntry) {
	// Store original state for ESC cancellation
	m.columnConfigOriginal = make([]ColumnConfig, len(m.activeColumns))
	copy(m.columnConfigOriginal, m.activeColumns)

	// Build available columns list from defaults + discovered + selected entry attributes
	m.buildAvailableColumns(selectedEntry)

	m.columnConfigSelected = 1 // Start at first column (skip header)
	m.columnConfigScrollOffset = 0
}

// buildAvailableColumns builds the list of all available columns
func (m *DashboardModel) buildAvailableColumns(selectedEntry *LogEntry) {
	// Start with default columns (but we'll update enabled state from activeColumns)
	defaults := getDefaultAvailableColumns()

	// Build a map of currently active columns for quick lookup
	activeMap := make(map[string]bool)
	for _, col := range m.activeColumns {
		if col.Enabled {
			activeMap[col.Key] = true
		}
	}

	// Update default columns' enabled state based on what's currently active
	// A column is enabled ONLY if it's in activeColumns
	m.availableColumns = make([]ColumnConfig, 0, len(defaults))
	for _, col := range defaults {
		col.Enabled = activeMap[col.Key]
		m.availableColumns = append(m.availableColumns, col)
	}

	// Collect all discovered attribute keys
	allKeys := make(map[string]bool)
	for key := range m.discoveredAttributes {
		allKeys[key] = true
	}

	// Add attributes from selected entry
	if selectedEntry != nil {
		for key := range selectedEntry.Attributes {
			allKeys[key] = true
		}
	}

	// Filter out keys that are already default columns
	defaultKeys := map[string]bool{
		"timestamp":     true,
		"severity":      true,
		"host.name":     true,
		"service.name":  true,
		"k8s.namespace": true,
		"k8s.pod":       true,
		"message":       true,
	}

	// Collect non-default keys and sort them
	var additionalKeys []string
	for key := range allKeys {
		if !defaultKeys[key] {
			additionalKeys = append(additionalKeys, key)
		}
	}
	sort.Strings(additionalKeys)

	// Add discovered columns
	for _, key := range additionalKeys {
		// Check if this column is currently active
		enabled := activeMap[key]

		// Create a readable label from the key
		label := formatColumnLabel(key)

		m.availableColumns = append(m.availableColumns, ColumnConfig{
			Key:       key,
			Label:     label,
			Width:     15, // Default width for discovered columns
			Enabled:   enabled,
			IsDefault: false,
		})
	}
}

// formatColumnLabel creates a human-readable label from an attribute key
func formatColumnLabel(key string) string {
	// Replace common separators with spaces
	label := strings.ReplaceAll(key, ".", " ")
	label = strings.ReplaceAll(label, "_", " ")
	label = strings.ReplaceAll(label, "-", " ")

	// Title case each word
	words := strings.Fields(label)
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + word[1:]
		}
	}

	return strings.Join(words, " ")
}

// applyColumnConfig applies the column configuration from available to active
func (m *DashboardModel) applyColumnConfig() {
	var newActive []ColumnConfig

	for _, col := range m.availableColumns {
		if col.Enabled {
			newActive = append(newActive, col)
		}
	}

	// If no columns selected, keep at least the message column
	if len(newActive) == 0 {
		newActive = append(newActive, ColumnConfig{Key: "message", Label: "Message", Width: 0, Enabled: true, IsDefault: true})
	}

	m.activeColumns = newActive
}

// restoreColumnConfig restores the original column configuration (for ESC)
func (m *DashboardModel) restoreColumnConfig() {
	m.activeColumns = make([]ColumnConfig, len(m.columnConfigOriginal))
	copy(m.activeColumns, m.columnConfigOriginal)
}

// renderColumnConfigModal renders the column configuration modal
func (m *DashboardModel) renderColumnConfigModal() string {
	// Status bar text determines the minimum width
	statusBarText := "↑↓/jk: Navigate • Space/x: Toggle • Enter: Apply • ESC: Cancel"
	statusBarWidth := len(statusBarText)

	// Calculate dimensions
	// We want: outer border (2) + padding (3 each side) + inner box
	// Inner box width should match status bar width
	sidePadding := 3
	modalWidth := statusBarWidth + 2 + (sidePadding * 2) // 2 for outer border + padding each side
	if modalWidth > m.width-4 {
		modalWidth = m.width - 4 // Leave some screen margin
	}

	modalHeight := m.height - 4 // Leave small margin

	// Inner box should be same width as status bar (they align)
	innerBoxContentWidth := statusBarWidth - 2 // -2 for inner box border
	if innerBoxContentWidth < 20 {
		innerBoxContentWidth = 20
	}

	contentHeight := modalHeight - 6 // Account for header, outer borders, status bar

	// Build all lines first
	var allLines []string

	// Count sections for navigation
	defaultCount := 0
	discoveredCount := 0
	for _, col := range m.availableColumns {
		if col.IsDefault {
			defaultCount++
		} else {
			discoveredCount++
		}
	}

	// Build content
	lineIdx := 0

	// Default columns section header
	sectionHeaderStyle := lipgloss.NewStyle().
		Foreground(ColorGray).
		Bold(true)
	allLines = append(allLines, sectionHeaderStyle.Render("Default Columns"))
	lineIdx++

	// Render default columns
	for _, col := range m.availableColumns {
		if !col.IsDefault {
			continue
		}

		isSelected := m.columnConfigSelected == lineIdx
		line := m.renderColumnConfigLine(col, isSelected, innerBoxContentWidth)
		allLines = append(allLines, line)
		lineIdx++
	}

	// Add separator and discovered columns section if any
	if discoveredCount > 0 {
		allLines = append(allLines, "")
		lineIdx++

		allLines = append(allLines, sectionHeaderStyle.Render("Discovered Attributes"))
		lineIdx++

		for _, col := range m.availableColumns {
			if col.IsDefault {
				continue
			}

			isSelected := m.columnConfigSelected == lineIdx
			line := m.renderColumnConfigLine(col, isSelected, innerBoxContentWidth)
			allLines = append(allLines, line)
			lineIdx++
		}
	}

	// Apply scroll offset - ensure selected item is visible
	visibleLines := contentHeight
	if visibleLines < 3 {
		visibleLines = 3
	}

	// Adjust scroll to keep selected item in view
	if m.columnConfigSelected < m.columnConfigScrollOffset {
		m.columnConfigScrollOffset = m.columnConfigSelected
	}
	if m.columnConfigSelected >= m.columnConfigScrollOffset+visibleLines {
		m.columnConfigScrollOffset = m.columnConfigSelected - visibleLines + 1
	}

	// Get visible slice of lines
	startIdx := m.columnConfigScrollOffset
	endIdx := startIdx + visibleLines
	if startIdx < 0 {
		startIdx = 0
	}
	if endIdx > len(allLines) {
		endIdx = len(allLines)
	}
	if startIdx >= len(allLines) {
		startIdx = max(0, len(allLines)-1)
	}

	visibleContent := allLines[startIdx:endIdx]

	// Add scroll indicators
	scrollIndicator := ""
	if m.columnConfigScrollOffset > 0 {
		scrollIndicator = " ▲"
	}
	if endIdx < len(allLines) {
		if scrollIndicator != "" {
			scrollIndicator += " ▼"
		} else {
			scrollIndicator = " ▼"
		}
	}

	// Create content pane (inner box)
	content := strings.Join(visibleContent, "\n")
	contentPane := lipgloss.NewStyle().
		Width(innerBoxContentWidth).
		Height(contentHeight).
		Border(lipgloss.NormalBorder()).
		BorderForeground(ColorBlue).
		Render(content)

	// Header with count - same width as status bar, centered
	activeCount := 0
	for _, col := range m.availableColumns {
		if col.Enabled {
			activeCount++
		}
	}
	headerText := fmt.Sprintf("Configure Columns (%d/%d active)%s", activeCount, len(m.availableColumns), scrollIndicator)
	header := lipgloss.NewStyle().
		Width(statusBarWidth).
		Foreground(ColorBlue).
		Bold(true).
		Align(lipgloss.Center).
		Render(headerText)

	// Status bar - same width, centered
	statusBar := lipgloss.NewStyle().
		Width(statusBarWidth).
		Foreground(ColorGray).
		Align(lipgloss.Center).
		Render(statusBarText)

	// Combine all parts with center alignment
	modal := lipgloss.JoinVertical(lipgloss.Center, header, contentPane, statusBar)

	// Add outer border - use explicit padding to center content
	finalModal := lipgloss.NewStyle().
		Width(modalWidth).
		Height(modalHeight).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorBlue).
		Align(lipgloss.Center, lipgloss.Center).
		Render(modal)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, finalModal)
}

// renderColumnConfigLine renders a single column config line
func (m *DashboardModel) renderColumnConfigLine(col ColumnConfig, isSelected bool, maxWidth int) string {
	prefix := "  "
	if isSelected {
		prefix = "► "
	}

	// Show checkbox status
	checkbox := "[ ]"
	if col.Enabled {
		checkbox = "[✓]"
	}

	// Calculate available width for label
	// prefix (2) + checkbox (3) + space (1) = 6 chars overhead
	labelMaxWidth := maxWidth - 6
	if labelMaxWidth < 10 {
		labelMaxWidth = 10
	}

	label := col.Label
	if len(label) > labelMaxWidth {
		// Truncate with ellipsis - remove 6 chars total (3 for ... + 3 more for clarity)
		if labelMaxWidth > 6 {
			label = label[:labelMaxWidth-6] + "..."
		} else {
			label = label[:labelMaxWidth]
		}
	}

	line := fmt.Sprintf("%s%s %s", prefix, checkbox, label)

	// Apply styling
	if isSelected {
		selectedStyle := lipgloss.NewStyle().
			Foreground(ColorBlue).
			Bold(true)
		return selectedStyle.Render(line)
	}

	return line
}

// getColumnConfigLineCount returns the total number of navigable lines in the modal
func (m *DashboardModel) getColumnConfigLineCount() int {
	count := 1 // "Default Columns" header

	defaultCount := 0
	discoveredCount := 0
	for _, col := range m.availableColumns {
		if col.IsDefault {
			defaultCount++
		} else {
			discoveredCount++
		}
	}

	count += defaultCount

	if discoveredCount > 0 {
		count += 2 // separator + "Discovered Attributes" header
		count += discoveredCount
	}

	return count
}

// isColumnConfigHeaderLine checks if the current line is a header (non-selectable)
func (m *DashboardModel) isColumnConfigHeaderLine(lineIdx int) bool {
	// Line 0 is "Default Columns" header
	if lineIdx == 0 {
		return true
	}

	// Count default columns
	defaultCount := 0
	for _, col := range m.availableColumns {
		if col.IsDefault {
			defaultCount++
		}
	}

	// After defaults: separator at defaultCount+1, header at defaultCount+2
	discoveredCount := 0
	for _, col := range m.availableColumns {
		if !col.IsDefault {
			discoveredCount++
		}
	}

	if discoveredCount > 0 {
		separatorLine := 1 + defaultCount
		headerLine := separatorLine + 1
		if lineIdx == separatorLine || lineIdx == headerLine {
			return true
		}
	}

	return false
}

// getColumnConfigColumnIndex returns the index in availableColumns for a given line index
func (m *DashboardModel) getColumnConfigColumnIndex(lineIdx int) int {
	if lineIdx == 0 {
		return -1 // Header
	}

	defaultCount := 0
	for _, col := range m.availableColumns {
		if col.IsDefault {
			defaultCount++
		}
	}

	// Lines 1 to defaultCount are default columns
	if lineIdx >= 1 && lineIdx <= defaultCount {
		return lineIdx - 1
	}

	discoveredCount := 0
	for _, col := range m.availableColumns {
		if !col.IsDefault {
			discoveredCount++
		}
	}

	if discoveredCount > 0 {
		// After separator (defaultCount+1) and header (defaultCount+2)
		// Discovered columns start at line defaultCount+3
		discoveredStart := defaultCount + 3
		if lineIdx >= discoveredStart {
			discoveredIdx := lineIdx - discoveredStart
			// Find the actual index in availableColumns
			idx := 0
			for i, col := range m.availableColumns {
				if !col.IsDefault {
					if idx == discoveredIdx {
						return i
					}
					idx++
				}
			}
		}
	}

	return -1
}

// toggleColumnConfigSelection toggles the currently selected column
func (m *DashboardModel) toggleColumnConfigSelection() {
	colIdx := m.getColumnConfigColumnIndex(m.columnConfigSelected)
	if colIdx < 0 || colIdx >= len(m.availableColumns) {
		return
	}

	col := &m.availableColumns[colIdx]
	col.Enabled = !col.Enabled
}
