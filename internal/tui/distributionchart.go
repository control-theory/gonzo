package tui

// calculateDistributionContentLines calculates lines needed for distribution chart content
func (m *DashboardModel) calculateDistributionContentLines() int {
	// Now used for drain3 patterns chart - match counts chart sizing
	return 8 // Increased from 7 to match log counts chart height
}

// renderDistributionChart renders the frequency distribution chart

// renderDistributionContent renders the distribution chart content
