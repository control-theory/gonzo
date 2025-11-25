package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// renderK8sFilterModal renders the Kubernetes namespace/pod filter modal
func (m *DashboardModel) renderK8sFilterModal() string {
	// Calculate dimensions - wider modal to accommodate long pod names
	modalWidth := min(m.width-10, 120)
	modalHeight := min(m.height-8, 25)

	// Account for borders and headers
	contentWidth := modalWidth - 4
	contentHeight := modalHeight - 5 // Header (1) + tab instructions (1) + status bar (1) + outer border (2)

	// Maximum item width (for truncation) = contentWidth - prefix - status - margin
	maxItemWidth := contentWidth - 6 // "► " (2) + " ✓" (2) + margin (2)

	// Build all filter lines (ONLY the scrollable list, no extra headers)
	var allLines []string

	// Header showing current view
	viewTitle := "Kubernetes Filter - Namespaces"
	if m.k8sActiveView == "pods" {
		viewTitle = "Kubernetes Filter - Pods"
	}

	// Build the list (no extra content - just the items)
	if m.k8sActiveView == "namespaces" {
		// Show namespaces
		allLines = append(allLines, m.renderNamespaceList(maxItemWidth)...)
	} else {
		// Show pods
		allLines = append(allLines, m.renderPodList(maxItemWidth)...)
	}

	// Calculate scroll window (matching model_selection_modal pattern)
	// Reserve space for: borders (2) + scroll indicators (2)
	totalLines := len(allLines)
	maxVisibleLines := contentHeight - 4
	visibleCount := maxVisibleLines
	if visibleCount > totalLines {
		visibleCount = totalLines
	}

	// Ensure selected item is visible by adjusting scroll offset
	if m.k8sFilterSelected < m.k8sScrollOffset {
		m.k8sScrollOffset = m.k8sFilterSelected
	} else if m.k8sFilterSelected >= m.k8sScrollOffset+visibleCount {
		m.k8sScrollOffset = m.k8sFilterSelected - visibleCount + 1
	}

	// Clamp scroll offset
	maxScroll := totalLines - visibleCount
	if maxScroll < 0 {
		maxScroll = 0
	}
	if m.k8sScrollOffset > maxScroll {
		m.k8sScrollOffset = maxScroll
	}
	if m.k8sScrollOffset < 0 {
		m.k8sScrollOffset = 0
	}

	// Extract visible portion (limited by actual visible space minus border)
	visibleLines := allLines[m.k8sScrollOffset:]
	if len(visibleLines) > maxVisibleLines {
		visibleLines = visibleLines[:maxVisibleLines]
	}

	// Add scroll indicators
	scrollInfo := ""
	if totalLines > maxVisibleLines {
		scrollInfo = fmt.Sprintf(" [%d/%d]", m.k8sScrollOffset+1, totalLines)
	}

	// Create content pane with visible lines
	// Don't set Height - let it naturally size to the content to avoid extra padding
	contentText := strings.Join(visibleLines, "\n")
	contentPane := lipgloss.NewStyle().
		Width(contentWidth).
		Border(lipgloss.NormalBorder()).
		BorderForeground(ColorBlue).
		Render(contentText)

	// Header
	activeNamespaces := 0
	for _, enabled := range m.k8sNamespaces {
		if enabled {
			activeNamespaces++
		}
	}
	activePods := 0
	for _, enabled := range m.k8sPods {
		if enabled {
			activePods++
		}
	}
	headerText := fmt.Sprintf("%s (%d namespaces, %d pods selected)%s",
		viewTitle, activeNamespaces, activePods, scrollInfo)
	header := lipgloss.NewStyle().
		Width(contentWidth).
		Foreground(ColorBlue).
		Bold(true).
		Render(headerText)

	// Tab instructions (rendered separately, not in scrollable area)
	tabInstructions := lipgloss.NewStyle().
		Foreground(ColorBlue).
		Render("Tab: Switch between Namespaces / Pods")

	// Status bar
	statusBar := lipgloss.NewStyle().
		Foreground(ColorGray).
		Render("↑↓: Navigate • Space: Toggle • Tab: Switch View • Enter: Apply • ESC: Cancel")

	// Combine all parts (header, tab instructions, content, status)
	modal := lipgloss.JoinVertical(lipgloss.Left, header, tabInstructions, contentPane, statusBar)

	// Add outer border and center
	// Don't set Height - let it naturally size to avoid extra padding at bottom
	finalModal := lipgloss.NewStyle().
		Width(modalWidth).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorBlue).
		Render(modal)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, finalModal)
}

// renderNamespaceList renders the list of namespaces
func (m *DashboardModel) renderNamespaceList(maxItemWidth int) []string {
	var lines []string

	// Add "All Namespaces" option at the top
	allNamespacesPrefix := "  "
	if m.k8sFilterSelected == 0 {
		allNamespacesPrefix = "► "
	}
	allSelected := true
	for _, enabled := range m.k8sNamespaces {
		if !enabled {
			allSelected = false
			break
		}
	}
	selectAllStatus := ""
	if allSelected {
		selectAllStatus = " ✓"
	}
	selectAllLine := allNamespacesPrefix + "All Namespaces" + selectAllStatus

	// Style the select all line
	if m.k8sFilterSelected == 0 {
		selectedStyle := lipgloss.NewStyle().
			Foreground(ColorBlue).
			Bold(true)
		selectAllLine = selectedStyle.Render(selectAllLine)
	}
	lines = append(lines, selectAllLine)

	// Add separator
	lines = append(lines, "")

	// Get sorted namespace names (use helper for consistency)
	namespaces := m.getSortedNamespaces()

	// Add individual namespaces (starting from index 2 after "All" and separator)
	for i, ns := range namespaces {
		listIndex := i + 2
		prefix := "  "
		if m.k8sFilterSelected == listIndex {
			prefix = "► "
		}

		// Show selection status
		status := ""
		if m.k8sNamespaces[ns] {
			status = " ✓"
		}

		// Truncate namespace name if too long
		displayName := ns
		if len(displayName) > maxItemWidth {
			displayName = displayName[:maxItemWidth-3] + "..."
		}

		line := prefix + displayName + status

		// Apply selection styling
		if m.k8sFilterSelected == listIndex {
			selectedStyle := lipgloss.NewStyle().
				Foreground(ColorBlue).
				Bold(true)
			line = selectedStyle.Render(line)
		}

		lines = append(lines, line)
	}

	return lines
}

// renderPodList renders the list of pods
func (m *DashboardModel) renderPodList(maxItemWidth int) []string {
	var lines []string

	// Add "All Pods" option at the top
	allPodsPrefix := "  "
	if m.k8sFilterSelected == 0 {
		allPodsPrefix = "► "
	}
	allSelected := true
	for _, enabled := range m.k8sPods {
		if !enabled {
			allSelected = false
			break
		}
	}
	selectAllStatus := ""
	if allSelected {
		selectAllStatus = " ✓"
	}
	selectAllLine := allPodsPrefix + "All Pods" + selectAllStatus

	// Style the select all line
	if m.k8sFilterSelected == 0 {
		selectedStyle := lipgloss.NewStyle().
			Foreground(ColorBlue).
			Bold(true)
		selectAllLine = selectedStyle.Render(selectAllLine)
	}
	lines = append(lines, selectAllLine)

	// Add separator
	lines = append(lines, "")

	// Get sorted pod names (use helper for consistency)
	pods := m.getSortedPods()

	// Add individual pods (starting from index 2 after "All" and separator)
	for i, pod := range pods {
		listIndex := i + 2
		prefix := "  "
		if m.k8sFilterSelected == listIndex {
			prefix = "► "
		}

		// Show selection status
		status := ""
		if m.k8sPods[pod] {
			status = " ✓"
		}

		// Truncate pod name if too long
		displayName := pod
		if len(displayName) > maxItemWidth {
			displayName = displayName[:maxItemWidth-3] + "..."
		}

		line := prefix + displayName + status

		// Apply selection styling
		if m.k8sFilterSelected == listIndex {
			selectedStyle := lipgloss.NewStyle().
				Foreground(ColorBlue).
				Bold(true)
			line = selectedStyle.Render(line)
		}

		lines = append(lines, line)
	}

	if len(pods) == 0 {
		lines = append(lines, lipgloss.NewStyle().
			Foreground(ColorGray).
			Italic(true).
			Render("  No pods available"))
	}

	return lines
}

// updateK8sNamespacesFromLogs scans log entries for k8s.namespace attributes
func (m *DashboardModel) updateK8sNamespacesFromLogs() {
	if m.k8sNamespaces == nil {
		m.k8sNamespaces = make(map[string]bool)
	}

	// Scan all logs for k8s.namespace attribute
	for _, entry := range m.allLogEntries {
		if ns, ok := entry.Attributes["k8s.namespace"]; ok && ns != "" {
			if _, exists := m.k8sNamespaces[ns]; !exists {
				// New namespace found, enable it by default
				m.k8sNamespaces[ns] = true
			}
		}
	}
}

// updateK8sPodsFromLogs scans log entries for k8s.pod attributes
func (m *DashboardModel) updateK8sPodsFromLogs() {
	if m.k8sPods == nil {
		m.k8sPods = make(map[string]bool)
	}

	// Scan all logs for k8s.pod attribute (filtered by selected namespaces)
	for _, entry := range m.allLogEntries {
		ns, hasNs := entry.Attributes["k8s.namespace"]
		pod, hasPod := entry.Attributes["k8s.pod"]

		// Only include pods from selected namespaces
		if hasPod && pod != "" {
			if !hasNs || m.k8sNamespaces[ns] {
				// Format: namespace/pod for clarity
				podKey := pod
				if hasNs {
					podKey = ns + "/" + pod
				}
				if _, exists := m.k8sPods[podKey]; !exists {
					// New pod found, enable it by default
					m.k8sPods[podKey] = true
				}
			}
		}
	}
}

// updateK8sNamespacesFromAPI queries Kubernetes API for available namespaces
func (m *DashboardModel) updateK8sNamespacesFromAPI() {
	// If no K8s source available, fall back to scanning logs
	if m.k8sSource == nil {
		m.updateK8sNamespacesFromLogs()
		return
	}

	// Query Kubernetes API for namespaces
	namespaces, err := m.k8sSource.ListNamespaces()
	if err != nil {
		// Fallback to scanning logs if API query fails
		m.updateK8sNamespacesFromLogs()
		return
	}

	// Preserve existing selections if we already have namespaces
	if len(m.k8sNamespaces) > 0 {
		// Preserve selections: keep existing state for namespaces that still exist
		for ns := range namespaces {
			if existingSelected, exists := m.k8sNamespaces[ns]; exists {
				// Namespace already in our list - keep user's selection
				namespaces[ns] = existingSelected
			}
			// New namespaces will use the default from API (selected by default)
		}
	}

	// Update namespaces map
	m.k8sNamespaces = namespaces
}

// updateK8sPodsFromAPI queries Kubernetes API for available pods
func (m *DashboardModel) updateK8sPodsFromAPI() {
	// If no K8s source available, fall back to scanning logs
	if m.k8sSource == nil {
		m.updateK8sPodsFromLogs()
		return
	}

	// Build map of only selected namespaces to pass to API
	selectedNamespaces := make(map[string]bool)
	for ns, selected := range m.k8sNamespaces {
		if selected {
			selectedNamespaces[ns] = true
		}
	}

	// Query Kubernetes API for pods (only from selected namespaces)
	pods, err := m.k8sSource.ListPods(selectedNamespaces)
	if err != nil {
		// Fallback to scanning logs if API query fails
		m.updateK8sPodsFromLogs()
		return
	}

	// Preserve existing pod selections if we already have pods
	if len(m.k8sPods) > 0 {
		// Preserve selections: keep existing state for pods that still exist
		for pod := range pods {
			if existingSelected, exists := m.k8sPods[pod]; exists {
				// Pod already in our list - keep user's selection
				pods[pod] = existingSelected
			}
			// New pods will use the default from API (selected by default)
		}
	}

	// Update pods map
	m.k8sPods = pods
}

// getSortedNamespaces returns a sorted list of namespace names
func (m *DashboardModel) getSortedNamespaces() []string {
	namespaces := make([]string, 0, len(m.k8sNamespaces))
	for ns := range m.k8sNamespaces {
		namespaces = append(namespaces, ns)
	}
	sort.Strings(namespaces)
	return namespaces
}

// getSortedPods returns a sorted list of pod names
func (m *DashboardModel) getSortedPods() []string {
	pods := make([]string, 0, len(m.k8sPods))
	for pod := range m.k8sPods {
		pods = append(pods, pod)
	}
	sort.Strings(pods)
	return pods
}
