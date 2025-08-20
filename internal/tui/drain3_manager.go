package tui

import (
	"github.com/control-theory/gonzo/internal/drain3"
	"sort"
	"strings"
	"time"

	goDrain "github.com/jaeyo/go-drain3/pkg/drain3"
)

// Drain3Manager manages the drain3 instance for pattern extraction
type Drain3Manager struct {
	drain       *drain3.Drain
	lastReset   time.Time
	totalCount  int
}

// PatternInfo represents a log pattern with its statistics
type PatternInfo struct {
	Template   string
	Count      int
	Percentage float64
}

// NewDrain3Manager creates a new drain3 manager with optimized settings for log pattern extraction
func NewDrain3Manager() *Drain3Manager {
	// Use optimized config for real-time log processing
	config := &drain3.Config{
		Depth:        4,    // Moderate depth for balanced pattern extraction
		SimilarityTh: 0.5,  // 50% similarity threshold - balanced clustering
		MaxChildren:  50,   // Lower for performance in real-time
		MaxClusters:  100,  // Keep top 100 patterns
	}

	return &Drain3Manager{
		drain:     drain3.New(config),
		lastReset: time.Now(),
	}
}

// AddLogMessage processes a log message and extracts its pattern
func (dm *Drain3Manager) AddLogMessage(message string) {
	if dm.drain == nil {
		return
	}

	// Skip empty messages
	if strings.TrimSpace(message) == "" {
		return
	}

	// Add to drain3 for pattern extraction
	_ = dm.drain.AddLogMessage(message)
	dm.totalCount++
}

// GetTopPatterns returns the top N patterns by frequency
func (dm *Drain3Manager) GetTopPatterns(limit int) []PatternInfo {
	if dm.drain == nil {
		return []PatternInfo{}
	}

	clusters := dm.drain.GetClusters()
	if len(clusters) == 0 {
		return []PatternInfo{}
	}

	// Convert to PatternInfo and sort by count
	patterns := make([]PatternInfo, 0, len(clusters))
	for _, cluster := range clusters {
		template := formatTemplate(cluster)
		if template != "" {
			patterns = append(patterns, PatternInfo{
				Template:   template,
				Count:      int(cluster.Size),
				Percentage: 0, // Will calculate after sorting
			})
		}
	}

	// Sort by count (descending), then by template alphabetically
	sort.Slice(patterns, func(i, j int) bool {
		if patterns[i].Count == patterns[j].Count {
			return patterns[i].Template < patterns[j].Template // Secondary sort alphabetically
		}
		return patterns[i].Count > patterns[j].Count
	})

	// Calculate percentages
	total := 0
	for _, p := range patterns {
		total += p.Count
	}
	if total > 0 {
		for i := range patterns {
			patterns[i].Percentage = float64(patterns[i].Count) * 100.0 / float64(total)
		}
	}

	// Limit results (0 means return all)
	if limit > 0 && len(patterns) > limit {
		patterns = patterns[:limit]
	}

	return patterns
}

// formatTemplate formats a drain3 cluster template for display
func formatTemplate(cluster *goDrain.LogCluster) string {
	if cluster == nil || len(cluster.LogTemplateTokens) == 0 {
		return ""
	}

	// Join tokens and replace placeholders with more readable format
	template := strings.Join(cluster.LogTemplateTokens, " ")
	
	// Replace drain3 placeholders (<*>) with more readable ones
	template = strings.ReplaceAll(template, "<*>", "***")
	
	// Truncate very long templates
	if len(template) > 100 {
		template = template[:97] + "..."
	}

	return template
}

// Reset clears the drain3 instance and starts fresh
func (dm *Drain3Manager) Reset() {
	if dm.drain != nil {
		_ = dm.drain.Reset()
		dm.lastReset = time.Now()
		dm.totalCount = 0
	}
}

// GetStats returns statistics about the current patterns
func (dm *Drain3Manager) GetStats() (patternCount int, totalLogs int) {
	if dm.drain == nil {
		return 0, 0
	}

	clusters := dm.drain.GetClusters()
	return len(clusters), dm.totalCount
}

// ShouldReset checks if it's time to reset based on a duration
func (dm *Drain3Manager) ShouldReset(resetInterval time.Duration) bool {
	return time.Since(dm.lastReset) >= resetInterval
}