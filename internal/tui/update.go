package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

// Update handles messages
func (m *DashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.initializeCharts()

	case tea.KeyMsg:
		return m.handleKeyPress(msg)

	case tea.MouseMsg:
		return m.handleMouseEvent(msg)

	case UpdateMsg:
		return m.handleUpdate(msg)

	case ManualResetMsg:
		// Handle manual reset - the actual reset will be done in the app layer
		// Just pass it up the chain
		return m, func() tea.Msg { return msg }

	case TickMsg:
		// Update processing rate statistics on every tick
		m.updateProcessingRateStats()
		
		// Only refresh charts when not paused
		if !m.viewPaused {
			m.updateCharts()
		}

		// Animate spinner when AI is analyzing (even when paused)
		if m.aiAnalyzing {
			m.aiSpinnerFrame = (m.aiSpinnerFrame + 1) % 4
			// Update modal content to show animated spinner
			if m.currentLogEntry != nil {
				m.modalContent = m.formatLogDetails(*m.currentLogEntry, 60)
				m.modalReady = false // Force viewport update
			}
		}
		
		// Animate chat spinner separately
		if m.chatAiAnalyzing {
			m.chatSpinnerFrame = (m.chatSpinnerFrame + 1) % 4
			// Update the working message in chat history with new spinner
			if len(m.chatHistory) > 0 {
				lastIdx := len(m.chatHistory) - 1
				// Check if the last message is a working indicator
				if strings.HasPrefix(m.chatHistory[lastIdx], "AI:") && strings.Contains(m.chatHistory[lastIdx], "Working on it...") {
					m.chatHistory[lastIdx] = fmt.Sprintf("AI: %s Working on it...", m.getChatSpinner())
				}
			}
		}

		// Continue periodic ticks
		return m, tea.Tick(m.updateInterval, func(t time.Time) tea.Msg {
			return TickMsg(t)
		})

	case AIAnalysisMsg:
		if msg.IsChat {
			// Handle chat AI response
			m.chatAiAnalyzing = false
			
			// Remove the "Working on it..." message (should be the last one)
			if len(m.chatHistory) > 0 {
				lastIdx := len(m.chatHistory) - 1
				if strings.HasPrefix(m.chatHistory[lastIdx], "AI:") && strings.Contains(m.chatHistory[lastIdx], "Working on it...") {
					// Remove the working message
					m.chatHistory = m.chatHistory[:lastIdx]
				}
			}
			
			// Add the actual response
			if msg.Error != nil {
				m.chatHistory = append(m.chatHistory, fmt.Sprintf("AI: Error: %v", msg.Error))
			} else {
				m.chatHistory = append(m.chatHistory, fmt.Sprintf("AI: %s", msg.Result))
			}
			m.chatAutoScroll = true  // Enable auto-scroll for new AI response
		} else {
			// Handle info section AI analysis
			m.aiAnalyzing = false
			if msg.Error != nil {
				m.aiAnalysisResult = fmt.Sprintf("Error: %v", msg.Error)
			} else {
				m.aiAnalysisResult = msg.Result
			}
		}
		// Update modal content with new analysis (only for non-chat responses)
		if m.currentLogEntry != nil && !msg.IsChat {
			m.modalContent = m.formatLogDetails(*m.currentLogEntry, 60)
			m.modalReady = false // Reset viewport for new content
		}
		// If chat mode is active, refocus the input after analysis
		if m.chatActive {
			m.chatInput.Focus()
		}
		return m, nil
	}

	return m, tea.Batch(cmds...)
}

// handleMouseEvent processes mouse interactions
func (m *DashboardModel) handleMouseEvent(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	// Handle mouse events in modals
	if m.showModal {
		return m.handleModalMouseEvent(msg)
	}

	// Handle mouse events in model selection modal
	if m.showModelSelectionModal {
		return m.handleModelSelectionMouseEvent(msg)
	}
	
	// Handle mouse events in help modal
	if m.showHelp {
		return m.handleHelpModalMouseEvent(msg)
	}
	
	// Handle mouse events in patterns modal
	if m.showPatternsModal {
		return m.handlePatternsModalMouseEvent(msg)
	}
	
	// Handle mouse events in statistics modal
	if m.showStatsModal {
		return m.handleStatsModalMouseEvent(msg)
	}

	// Handle mouse events in counts modal
	if m.showCountsModal {
		return m.handleCountsModalMouseEvent(msg)
	}
	
	// Handle mouse events in log viewer modal
	if m.showLogViewerModal {
		return m.handleLogViewerModalMouseEvent(msg)
	}
	
	// Skip mouse events for input modes
	if m.filterActive || m.searchActive {
		return m, nil
	}

	switch msg.Action {
	case tea.MouseActionPress:
		switch msg.Button {
		case tea.MouseButtonLeft:
			// Handle left mouse button clicks to switch sections
			return m.handleMouseClick(msg.X, msg.Y)

		case tea.MouseButtonWheelUp:
			// Scroll wheel up = move selection up (like up arrow), or down if reversed
			if m.reverseScrollWheel {
				m.moveSelection(-1)
			} else {
				m.moveSelection(1)
			}
			return m, nil

		case tea.MouseButtonWheelDown:
			// Scroll wheel down = move selection down (like down arrow), or up if reversed
			if m.reverseScrollWheel {
				m.moveSelection(1)
			} else {
				m.moveSelection(-1)
			}
			return m, nil
		}
	}

	return m, nil
}

// handleModalMouseEvent processes mouse interactions within modals
func (m *DashboardModel) handleModalMouseEvent(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	// Handle single modals (like Top Values) when there's no log entry
	if m.currentLogEntry == nil {
		switch msg.Action {
		case tea.MouseActionPress:
			switch msg.Button {
			case tea.MouseButtonWheelUp:
				// Scroll up in single modal, or down if reversed
				if m.reverseScrollWheel {
					m.infoViewport.ScrollDown(1)
				} else {
					m.infoViewport.ScrollUp(1)
				}
				return m, nil

			case tea.MouseButtonWheelDown:
				// Scroll down in single modal, or up if reversed
				if m.reverseScrollWheel {
					m.infoViewport.ScrollUp(1)
				} else {
					m.infoViewport.ScrollDown(1)
				}
				return m, nil
			}
		}
		return m, nil
	}

	switch msg.Action {
	case tea.MouseActionPress:
		switch msg.Button {
		case tea.MouseButtonLeft:
			// Handle clicks to switch between modal sections
			return m.handleModalClick(msg.X, msg.Y)

		case tea.MouseButtonWheelUp:
			// Priority: if chat is active, always scroll chat viewport
			if m.chatActive {
				if m.reverseScrollWheel {
					m.chatViewport.ScrollDown(1)
				} else {
					m.chatViewport.ScrollUp(1)
				}
			} else if m.modalActiveSection == "info" {
				if m.reverseScrollWheel {
					m.infoViewport.ScrollDown(1)
				} else {
					m.infoViewport.ScrollUp(1)
				}
			} else if m.modalActiveSection == "chat" {
				if m.reverseScrollWheel {
					m.chatViewport.ScrollDown(1)
				} else {
					m.chatViewport.ScrollUp(1)
				}
			}
			return m, nil

		case tea.MouseButtonWheelDown:
			// Priority: if chat is active, always scroll chat viewport
			if m.chatActive {
				if m.reverseScrollWheel {
					m.chatViewport.ScrollUp(1)
				} else {
					m.chatViewport.ScrollDown(1)
				}
			} else if m.modalActiveSection == "info" {
				if m.reverseScrollWheel {
					m.infoViewport.ScrollUp(1)
				} else {
					m.infoViewport.ScrollDown(1)
				}
			} else if m.modalActiveSection == "chat" {
				if m.reverseScrollWheel {
					m.chatViewport.ScrollUp(1)
				} else {
					m.chatViewport.ScrollDown(1)
				}
			}
			return m, nil
		}
	}

	return m, nil
}

// handleModalClick processes mouse clicks within modals to switch sections
func (m *DashboardModel) handleModalClick(x, _ int) (tea.Model, tea.Cmd) {
	// Calculate modal layout to determine which section was clicked
	// Based on renderSplitModal layout: 70% info, 30% chat

	modalWidth := m.width - 8      // 4 chars margin on each side
	contentWidth := modalWidth - 4 // Modal borders

	// Split layout: 70% info, 30% chat
	infoWidth := int(float64(contentWidth)*0.7) - 1 // -1 for separator

	// Calculate section boundaries (relative to modal area)
	// Modal is centered, so calculate offset
	modalStartX := 4 // Left margin

	// Check if click is in info section (left side) or chat section (right side)
	relativeX := x - modalStartX

	if relativeX >= 0 && relativeX < infoWidth {
		// Clicked in info section
		if m.modalActiveSection != "info" {
			m.modalActiveSection = "info"
			// Exit chat mode when switching away from chat
			if m.chatActive {
				m.chatActive = false
				m.chatInput.Blur()
			}
		}
	} else if relativeX >= infoWidth {
		// Clicked in chat section
		if m.modalActiveSection != "chat" {
			m.modalActiveSection = "chat"
			// Check if AI is configured before enabling chat
			if !m.aiConfigured {
				// Show error in chat area instead of enabling chat
				chatError := fmt.Sprintf("AI Chat Not Available\n\nError: %s\n\nTo configure AI:\n• Set OPENAI_API_KEY environment variable\n• For local AI: Set OPENAI_API_BASE\n• Use --ai-model flag to specify model", m.aiErrorMessage)
				m.chatHistory = []string{fmt.Sprintf("System: %s", chatError)}
				m.chatAutoScroll = true  // Enable auto-scroll for error message
				return m, nil
			}
			// Automatically enter chat mode when clicking on chat section
			if m.currentLogEntry != nil && m.aiClient != nil {
				m.chatActive = true
				m.chatInput.Focus()
				return m, textarea.Blink
			}
		}
	}

	return m, nil
}

// handleModelSelectionMouseEvent processes mouse interactions in model selection modal
func (m *DashboardModel) handleModelSelectionMouseEvent(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	switch msg.Action {
	case tea.MouseActionPress:
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			// Scroll up - move selection up by 1, or down if reversed
			if m.reverseScrollWheel {
				m.selectedModelIndex = min(len(m.availableModelsList)-1, m.selectedModelIndex+1)
			} else {
				m.selectedModelIndex = max(0, m.selectedModelIndex-1)
			}
			return m, nil

		case tea.MouseButtonWheelDown:
			// Scroll down - move selection down by 1, or up if reversed
			if m.reverseScrollWheel {
				m.selectedModelIndex = max(0, m.selectedModelIndex-1)
			} else {
				m.selectedModelIndex = min(len(m.availableModelsList)-1, m.selectedModelIndex+1)
			}
			return m, nil
		}
	}
	
	return m, nil
}

// handleHelpModalMouseEvent processes mouse interactions in help modal
func (m *DashboardModel) handleHelpModalMouseEvent(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	switch msg.Action {
	case tea.MouseActionPress:
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			// Scroll up in help modal, or down if reversed
			if m.reverseScrollWheel {
				m.infoViewport.ScrollDown(1)
			} else {
				m.infoViewport.ScrollUp(1)
			}
			return m, nil

		case tea.MouseButtonWheelDown:
			// Scroll down in help modal, or up if reversed
			if m.reverseScrollWheel {
				m.infoViewport.ScrollUp(1)
			} else {
				m.infoViewport.ScrollDown(1)
			}
			return m, nil
		}
	}

	return m, nil
}

// handlePatternsModalMouseEvent processes mouse interactions in patterns modal
func (m *DashboardModel) handlePatternsModalMouseEvent(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	switch msg.Action {
	case tea.MouseActionPress:
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			// Scroll up in patterns modal, or down if reversed
			if m.reverseScrollWheel {
				m.infoViewport.ScrollDown(1)
			} else {
				m.infoViewport.ScrollUp(1)
			}
			return m, nil

		case tea.MouseButtonWheelDown:
			// Scroll down in patterns modal, or up if reversed
			if m.reverseScrollWheel {
				m.infoViewport.ScrollUp(1)
			} else {
				m.infoViewport.ScrollDown(1)
			}
			return m, nil
		}
	}

	return m, nil
}

// handleStatsModalMouseEvent processes mouse interactions in statistics modal
func (m *DashboardModel) handleStatsModalMouseEvent(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	switch msg.Action {
	case tea.MouseActionPress:
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			// Scroll up in statistics modal, or down if reversed
			if m.reverseScrollWheel {
				m.infoViewport.ScrollDown(1)
			} else {
				m.infoViewport.ScrollUp(1)
			}
			return m, nil

		case tea.MouseButtonWheelDown:
			// Scroll down in statistics modal, or up if reversed
			if m.reverseScrollWheel {
				m.infoViewport.ScrollUp(1)
			} else {
				m.infoViewport.ScrollDown(1)
			}
			return m, nil
		}
	}

	return m, nil
}

// handleCountsModalMouseEvent processes mouse interactions in counts modal
func (m *DashboardModel) handleCountsModalMouseEvent(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	switch msg.Action {
	case tea.MouseActionPress:
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			// Scroll up in counts modal, or down if reversed
			if m.reverseScrollWheel {
				m.infoViewport.ScrollDown(1)
			} else {
				m.infoViewport.ScrollUp(1)
			}
			return m, nil

		case tea.MouseButtonWheelDown:
			// Scroll down in counts modal, or up if reversed
			if m.reverseScrollWheel {
				m.infoViewport.ScrollUp(1)
			} else {
				m.infoViewport.ScrollDown(1)
			}
			return m, nil
		}
	}

	return m, nil
}

// handleLogViewerModalMouseEvent processes mouse interactions in log viewer modal
func (m *DashboardModel) handleLogViewerModalMouseEvent(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	switch msg.Action {
	case tea.MouseActionPress:
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			// Scroll up in log viewer - should behave like scrolling content up (move to later/newer logs)
			if m.reverseScrollWheel {
				// Reversed: wheel up goes to earlier logs
				if m.selectedLogIndex > 0 {
					m.selectedLogIndex--
				}
			} else {
				// Normal: wheel up goes to later logs (like scrolling list up)
				if m.selectedLogIndex < len(m.logEntries)-1 {
					m.selectedLogIndex++
				}
			}
			return m, nil

		case tea.MouseButtonWheelDown:
			// Scroll down in log viewer - should behave like scrolling content down (move to earlier/older logs)
			if m.reverseScrollWheel {
				// Reversed: wheel down goes to later logs
				if m.selectedLogIndex < len(m.logEntries)-1 {
					m.selectedLogIndex++
				}
			} else {
				// Normal: wheel down goes to earlier logs (like scrolling list down)
				if m.selectedLogIndex > 0 {
					m.selectedLogIndex--
				}
			}
			return m, nil
		}
	}

	// Ignore all other mouse events in the log viewer modal
	return m, nil
}

// handleMouseClick processes mouse clicks to switch between sections
func (m *DashboardModel) handleMouseClick(x, y int) (tea.Model, tea.Cmd) {
	// Calculate section boundaries based on screen layout
	// The dashboard uses a 2x2 grid layout with logs at the bottom

	if m.width <= 0 || m.height <= 0 {
		return m, nil
	}

	// Calculate grid dimensions
	gridWidth := m.width / 2
	gridHeight := m.calculateRequiredChartsHeight() / 2

	// Define section boundaries (approximate)
	switch {
	case x < gridWidth && y < gridHeight:
		// Top-left: Words section
		m.activeSection = SectionWords

	case x >= gridWidth && y < gridHeight:
		// Top-right: Attributes section
		m.activeSection = SectionAttributes

	case x < gridWidth && y >= gridHeight && y < gridHeight*2:
		// Bottom-left: Distribution/Patterns section
		m.activeSection = SectionDistribution

	case x >= gridWidth && y >= gridHeight && y < gridHeight*2:
		// Bottom-right: Counts section
		m.activeSection = SectionCounts

	case y >= gridHeight*2:
		// Bottom area: Logs section
		m.activeSection = SectionLogs
	}

	return m, nil
}

// handleUpdate processes data updates
func (m *DashboardModel) handleUpdate(msg UpdateMsg) (tea.Model, tea.Cmd) {
	// Always buffer log entries even when paused
	if msg.NewLogEntry != nil {
		m.addLogEntry(*msg.NewLogEntry)
	}

	// Handle batch log entries
	if len(msg.NewLogBatch) > 0 {
		for _, entry := range msg.NewLogBatch {
			if entry != nil {
				m.addLogEntry(*entry)
			}
		}
	}

	// Only update dashboard data when not paused
	if !m.viewPaused {
		if msg.Snapshot != nil {
			m.snapshot = msg.Snapshot
			m.updateCharts()
		}

		if msg.SeverityCount != nil || msg.ForceCountUpdate {
			// Use SeverityCount if provided, otherwise create empty counts
			counts := msg.SeverityCount
			if counts == nil {
				counts = &SeverityCounts{}
			}
			m.countsHistory = append(m.countsHistory, *counts)
			// Keep only last 50 data points
			if len(m.countsHistory) > 50 {
				m.countsHistory = m.countsHistory[1:]
			}
			// Chart data updated in view rendering
		}
	}

	// Handle drain3 reset (always process this even when paused)
	if msg.ResetDrain3 && m.drain3Manager != nil {
		m.drain3Manager.Reset()
		m.drain3LastProcessed = 0 // Reset tracking
		
		// Also reset all severity-specific drain3 instances
		for _, drain3Instance := range m.drain3BySeverity {
			if drain3Instance != nil {
				drain3Instance.Reset()
			}
		}
	}

	return m, nil
}

// addLogEntry adds a new log entry to the buffer
func (m *DashboardModel) addLogEntry(entry LogEntry) {
	// Always add to the complete unfiltered buffer
	m.allLogEntries = append(m.allLogEntries, entry)
	
	// Update statistics tracking
	m.statsTotalLogsEver++  // Track total logs processed (unlimited)
	m.statsTotalBytes += int64(len(entry.RawLine))
	
	// Update lifetime statistics (unlimited tracking)
	m.updateLifetimeStats(entry)
	
	// Update heatmap data for counts modal
	m.updateHeatmapData(entry)
	
	// Update services data for counts modal (patterns will be derived from drain3)
	m.updateCountsModalServices(entry)
	
	// Track logs for the current second
	m.statsLogsThisSecond++

	// Maintain buffer size for complete buffer
	if len(m.allLogEntries) > m.maxLogBuffer {
		m.allLogEntries = m.allLogEntries[1:]
		// Adjust drain3 tracking if we removed an entry
		if m.drain3LastProcessed > 0 {
			m.drain3LastProcessed--
		}
	}

	// Only process through drain3 and update view when not paused
	if !m.viewPaused {
		// Process through drain3 for pattern extraction
		if m.drain3Manager != nil {
			m.drain3Manager.AddLogMessage(entry.Message)
			m.drain3LastProcessed = len(m.allLogEntries) // Track that we've processed up to here
		}

		// Update filtered view
		m.updateFilteredView()
	}
}

// updateFilteredView regenerates the filtered log entries view
func (m *DashboardModel) updateFilteredView() {
	oldSelection := m.selectedLogIndex

	// Clear current filtered view
	m.logEntries = m.logEntries[:0]

	// Apply filter to all entries
	for _, entry := range m.allLogEntries {
		// Check regex filter (if any) - search in message, attributes keys, and attribute values
		passesRegexFilter := m.filterRegex == nil || m.matchesFilter(entry)

		// Check severity filter (if active)
		// Normalize severity to match filter keys
		normalizedSeverity := normalizeSeverityLevel(entry.Severity)
		passesSeverityFilter := !m.severityFilterActive || m.severityFilter[normalizedSeverity]

		// Include entry only if it passes both filters
		if passesRegexFilter && passesSeverityFilter {
			m.logEntries = append(m.logEntries, entry)
		}
	}

	// Update selection based on auto-scroll setting
	if m.logAutoScroll {
		// Auto-scroll enabled: always go to latest entry
		m.selectedLogIndex = max(0, len(m.logEntries)-1)
	} else {
		// Auto-scroll disabled: try to maintain current position
		if oldSelection < len(m.logEntries) {
			m.selectedLogIndex = oldSelection
		} else {
			m.selectedLogIndex = max(0, len(m.logEntries)-1)
		}
	}
}

// initializeCharts sets up the charts based on current dimensions
func (m *DashboardModel) initializeCharts() {
	if m.width <= 0 || m.height <= 0 {
		return
	}

	m.chartsInitialized = true
}

// updateCharts updates chart data from snapshot
func (m *DashboardModel) updateCharts() {
	// Chart data is handled in the view rendering
	// This method is kept for consistency with the interface
}

// updateLifetimeStats updates lifetime statistics for a log entry
func (m *DashboardModel) updateLifetimeStats(entry LogEntry) {
	// Update severity counts
	m.lifetimeSeverityCounts[entry.Severity]++
	
	// Update host counts
	if host, exists := entry.Attributes["host"]; exists && host != "" {
		m.lifetimeHostCounts[host]++
	}
	
	// Update service counts
	if service, exists := entry.Attributes["service.name"]; exists && service != "" {
		m.lifetimeServiceCounts[service]++
	} else if service, exists := entry.Attributes["service"]; exists && service != "" {
		m.lifetimeServiceCounts[service]++
	}
	
	// Update attribute counts
	for key, value := range entry.Attributes {
		// Skip common keys that we handle separately for some stats
		attrKey := fmt.Sprintf("%s=%s", key, value)
		if len(attrKey) < 200 { // Only include reasonable length attributes
			m.lifetimeAttrCounts[attrKey]++
		}
		
		// Update per-attribute-key value counts (for dashboard charts)
		if m.lifetimeAttrKeyCounts[key] == nil {
			m.lifetimeAttrKeyCounts[key] = make(map[string]int64)
		}
		m.lifetimeAttrKeyCounts[key][value]++
	}
	
	// Update word counts (simplified word extraction for performance)
	words := strings.Fields(strings.ToLower(entry.Message))
	for _, word := range words {
		// Simple cleanup: only count words that are alphanumeric and reasonable length
		if len(word) >= 2 && len(word) <= 50 {
			// Remove common punctuation
			word = strings.Trim(word, ".,!?;:()[]{}\"'")
			// Check minimum length and stopwords filter
			if len(word) >= 3 && !m.stopWords[word] {
				m.lifetimeWordCounts[word]++
			}
		}
	}
}

// updateProcessingRateStats updates the processing rate statistics on every update cycle
func (m *DashboardModel) updateProcessingRateStats() {
	now := time.Now()
	
	// Check if a second has passed
	if now.Sub(m.statsLastSecond) >= time.Second {
		// Store the count for the completed second
		if m.statsLogsThisSecond > 0 {
			rate := float64(m.statsLogsThisSecond)
			if rate > m.statsPeakLogsPerSec {
				m.statsPeakLogsPerSec = rate
			}
			
			// Add to sliding window
			m.statsRecentCounts = append(m.statsRecentCounts, m.statsLogsThisSecond)
			m.statsRecentTimes = append(m.statsRecentTimes, m.statsLastSecond)
		} else {
			// Even if no logs, add 0 to maintain the window
			m.statsRecentCounts = append(m.statsRecentCounts, 0)
			m.statsRecentTimes = append(m.statsRecentTimes, m.statsLastSecond)
		}
		
		// Keep only last 10 seconds
		if len(m.statsRecentCounts) > 10 {
			m.statsRecentCounts = m.statsRecentCounts[1:]
			m.statsRecentTimes = m.statsRecentTimes[1:]
		}
		
		// Reset for new second
		m.statsLastSecond = now
		m.statsLogsThisSecond = 0
	}
	
	// Clean up old entries from the sliding window (older than 10 seconds)
	cutoffTime := now.Add(-10 * time.Second)
	for len(m.statsRecentTimes) > 0 && m.statsRecentTimes[0].Before(cutoffTime) {
		m.statsRecentCounts = m.statsRecentCounts[1:]
		m.statsRecentTimes = m.statsRecentTimes[1:]
	}
}

// matchesFilter checks if a log entry matches the current regex filter
// It searches in the message, attribute keys, and attribute values
func (m *DashboardModel) matchesFilter(entry LogEntry) bool {
	if m.filterRegex == nil {
		return true
	}

	// Check the raw log line (message)
	if m.filterRegex.MatchString(entry.RawLine) {
		return true
	}

	// Check the processed message
	if m.filterRegex.MatchString(entry.Message) {
		return true
	}

	// Check all attribute keys and values
	for key, value := range entry.Attributes {
		// Check if the attribute key matches
		if m.filterRegex.MatchString(key) {
			return true
		}
		// Check if the attribute value matches
		if m.filterRegex.MatchString(value) {
			return true
		}
	}

	return false
}

// updateHeatmapData updates the minute-by-minute heatmap data for the counts modal
func (m *DashboardModel) updateHeatmapData(entry LogEntry) {
	// Now entry.Timestamp is always the receive time, so we can use it directly
	// This ensures the heatmap shows when logs were received, not their original timestamps
	entryTime := entry.Timestamp.Truncate(time.Minute)
	
	// Find or create the heatmap minute entry
	var targetMinute *HeatmapMinute
	
	// Look for existing minute entry
	for i := range m.heatmapData {
		if m.heatmapData[i].Timestamp.Equal(entryTime) {
			targetMinute = &m.heatmapData[i]
			break
		}
	}
	
	// If not found, create new minute entry
	if targetMinute == nil {
		newMinute := HeatmapMinute{
			Timestamp: entryTime,
			Counts:    SeverityCounts{},
		}
		m.heatmapData = append(m.heatmapData, newMinute)
		targetMinute = &m.heatmapData[len(m.heatmapData)-1]
	}
	
	// Update the severity count for this minute
	switch entry.Severity {
	case "TRACE":
		targetMinute.Counts.Trace++
	case "DEBUG":
		targetMinute.Counts.Debug++
	case "INFO":
		targetMinute.Counts.Info++
	case "WARN", "WARNING":
		targetMinute.Counts.Warn++
	case "ERROR":
		targetMinute.Counts.Error++
	case "FATAL":
		targetMinute.Counts.Fatal++
	case "CRITICAL":
		targetMinute.Counts.Critical++
	default:
		targetMinute.Counts.Unknown++
	}
	
	// Update total count
	targetMinute.Counts.Total++
	
	// Keep a larger window of data (6 hours) to accommodate logs with older timestamps
	// The actual 60-minute window filtering will be done during display
	cutoffTime := time.Now().Add(-6 * time.Hour)
	filteredData := make([]HeatmapMinute, 0)
	
	for _, minute := range m.heatmapData {
		if minute.Timestamp.After(cutoffTime) {
			filteredData = append(filteredData, minute)
		}
	}
	
	m.heatmapData = filteredData
}

// updateCountsModalServices updates services data grouped by severity
func (m *DashboardModel) updateCountsModalServices(entry LogEntry) {
	severity := entry.Severity
	if severity == "" {
		severity = "UNKNOWN"
	}
	
	// Update service counts by severity
	serviceName := getServiceName(entry)
	if serviceName != "" {
		if m.servicesBySeverity[severity] == nil {
			m.servicesBySeverity[severity] = make([]ServiceCount, 0)
		}
		
		// Find existing service or create new one
		found := false
		for i := range m.servicesBySeverity[severity] {
			if m.servicesBySeverity[severity][i].Service == serviceName {
				m.servicesBySeverity[severity][i].Count++
				found = true
				break
			}
		}
		
		if !found {
			m.servicesBySeverity[severity] = append(m.servicesBySeverity[severity], ServiceCount{
				Service: serviceName,
				Count:   1,
			})
		}
		
		// Keep only top 10 services per severity and sort
		m.sortAndTrimServiceCounts(severity)
	}
	
	// Feed log to severity-specific drain3 instance
	if drain3Instance, exists := m.drain3BySeverity[entry.Severity]; exists && drain3Instance != nil {
		drain3Instance.AddLogMessage(entry.Message)
	}
}

// getServiceName extracts service name from log entry attributes
func getServiceName(entry LogEntry) string {
	// Try different common service attribute names
	if service, ok := entry.Attributes["service"]; ok {
		return service
	}
	if service, ok := entry.Attributes["service.name"]; ok {
		return service
	}
	if service, ok := entry.Attributes["serviceName"]; ok {
		return service
	}
	if service, ok := entry.Attributes["app"]; ok {
		return service
	}
	if service, ok := entry.Attributes["application"]; ok {
		return service
	}
	
	// Fallback to host if no service specified
	if host, ok := entry.Attributes["host"]; ok {
		return "host:" + host
	}
	
	return "unknown"
}

// sortAndTrimServiceCounts sorts and keeps only top services for a severity
func (m *DashboardModel) sortAndTrimServiceCounts(severity string) {
	services := m.servicesBySeverity[severity]
	if len(services) <= 1 {
		return
	}
	
	// Sort by count (descending)
	for i := 0; i < len(services); i++ {
		for j := i + 1; j < len(services); j++ {
			if services[i].Count < services[j].Count {
				services[i], services[j] = services[j], services[i]
			}
		}
	}
	
	// Keep only top 10
	if len(services) > 10 {
		m.servicesBySeverity[severity] = services[:10]
	}
}

