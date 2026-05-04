package engine

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/control-theory/gonzo/internal/ai"
	"github.com/control-theory/gonzo/internal/memory"
	"github.com/control-theory/gonzo/internal/tui"
)

// Engine is the shared, concurrency-safe analysis state that the TUI
// and web dashboard read from. The processing pipeline
// writes via Ingest(); all Query*/Get* methods acquire a read lock.
type Engine struct {
	mu sync.RWMutex

	// Log buffer (ring buffer)
	allLogEntries []tui.LogEntry
	maxLogBuffer  int

	// Severity tracking
	lifetimeSeverityCounts map[string]int64
	countsHistory          []tui.SeverityCounts
	severityTimeSeries     []SeverityTimePoint // timestamped 1-second severity history

	// Heatmap data (minute-by-minute severity counts)
	heatmapData []tui.HeatmapMinute

	// Pattern clustering per severity
	drain3BySeverity map[string]*tui.Drain3Manager
	// Combined drain3 for all logs
	drain3All *tui.Drain3Manager

	// Service tracking per severity
	servicesBySeverity map[string][]tui.ServiceCount

	// Dimension tracking for QueryInsightsParams
	lifetimeHostCounts    map[string]int64
	lifetimeServiceCounts map[string]int64
	lifetimeAttrKeyCounts map[string]map[string]int64

	// Frequency snapshot (latest)
	snapshot *memory.FrequencySnapshot

	// Word/attribute tracking for classes
	lifetimeWordCounts map[string]int64
	lifetimeAttrCounts map[string]int64
	stopWords          map[string]bool

	// Stream tracking
	streams map[string]*StreamInfo

	// Statistics
	statsStartTime  time.Time
	totalLogsEver   int
	totalBytes      int64

	// AI client for summary queries
	aiClient ai.Client

	// Config
	useLogTime bool
}

// NewEngine creates a new Engine with the given configuration.
func NewEngine(maxLogBuffer int, stopWords map[string]bool, aiClient ai.Client, useLogTime bool) *Engine {
	return &Engine{
		allLogEntries:          make([]tui.LogEntry, 0, maxLogBuffer),
		maxLogBuffer:           maxLogBuffer,
		lifetimeSeverityCounts: make(map[string]int64),
		countsHistory:          make([]tui.SeverityCounts, 0),
		severityTimeSeries:     make([]SeverityTimePoint, 0, 120),
		heatmapData:            make([]tui.HeatmapMinute, 0),
		drain3BySeverity:       tui.InitializeDrain3BySeverity(),
		drain3All:              tui.NewDrain3Manager(),
		servicesBySeverity:     make(map[string][]tui.ServiceCount),
		lifetimeHostCounts:     make(map[string]int64),
		lifetimeServiceCounts:  make(map[string]int64),
		lifetimeAttrKeyCounts:  make(map[string]map[string]int64),
		lifetimeWordCounts:     make(map[string]int64),
		lifetimeAttrCounts:     make(map[string]int64),
		stopWords:              stopWords,
		streams:                make(map[string]*StreamInfo),
		statsStartTime:         time.Now(),
		aiClient:               aiClient,
		useLogTime:             useLogTime,
	}
}

// Ingest adds a new log entry to the engine. Called from the processing pipeline.
// This is the only write path — it acquires a write lock.
func (e *Engine) Ingest(entry tui.LogEntry) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Add to buffer
	e.allLogEntries = append(e.allLogEntries, entry)
	if len(e.allLogEntries) > e.maxLogBuffer {
		e.allLogEntries = e.allLogEntries[1:]
	}

	// Update statistics
	e.totalLogsEver++
	e.totalBytes += int64(len(entry.RawLine))

	// Update lifetime severity counts
	e.lifetimeSeverityCounts[entry.Severity]++

	// Update host counts
	if host := entry.Attributes["host"]; host != "" {
		e.lifetimeHostCounts[host]++
	}

	// Update service counts
	serviceName := getServiceName(entry)
	if serviceName != "" && serviceName != "unknown" {
		e.lifetimeServiceCounts[serviceName]++
	}

	// Update per-attribute-key value counts
	for key, value := range entry.Attributes {
		attrKey := fmt.Sprintf("%s=%s", key, value)
		if len(attrKey) < 200 {
			e.lifetimeAttrCounts[attrKey]++
		}
		if e.lifetimeAttrKeyCounts[key] == nil {
			e.lifetimeAttrKeyCounts[key] = make(map[string]int64)
		}
		e.lifetimeAttrKeyCounts[key][value]++
	}

	// Update word counts
	words := strings.Fields(strings.ToLower(entry.Message))
	for _, word := range words {
		if len(word) >= 2 && len(word) <= 50 {
			word = strings.Trim(word, ".,!?;:()[]{}\"'")
			if len(word) >= 3 && !e.stopWords[word] {
				e.lifetimeWordCounts[word]++
			}
		}
	}

	// Update heatmap
	e.updateHeatmapData(entry)

	// Update services by severity
	e.updateServicesBySeverity(entry)

	// Update drain3 patterns
	if drain3Instance, exists := e.drain3BySeverity[entry.Severity]; exists && drain3Instance != nil {
		drain3Instance.AddLogMessage(entry.Message)
	}
	if e.drain3All != nil {
		e.drain3All.AddLogMessage(entry.Message)
	}

	// Update stream tracking
	e.updateStreamTracking(entry)
}

// IngestSeverityCounts adds interval-level severity counts to history.
func (e *Engine) IngestSeverityCounts(counts tui.SeverityCounts) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.countsHistory = append(e.countsHistory, counts)
	if len(e.countsHistory) > 50 {
		e.countsHistory = e.countsHistory[1:]
	}

	// Also store in timestamped time series for the web severity chart
	point := SeverityTimePoint{
		Timestamp: time.Now().Unix(),
		Counts: map[string]int{
			"FATAL": counts.Fatal,
			"ERROR": counts.Error,
			"WARN":  counts.Warn,
			"INFO":  counts.Info,
			"DEBUG": counts.Debug,
			"TRACE": counts.Trace,
		},
		Total: counts.Total,
	}
	e.severityTimeSeries = append(e.severityTimeSeries, point)
	// Keep up to 120 seconds (2 minutes)
	if len(e.severityTimeSeries) > 120 {
		e.severityTimeSeries = e.severityTimeSeries[len(e.severityTimeSeries)-120:]
	}
}

// UpdateFrequencySnapshot updates the latest frequency snapshot.
func (e *Engine) UpdateFrequencySnapshot(snapshot *memory.FrequencySnapshot) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.snapshot = snapshot
}

// Reset clears all engine state.
func (e *Engine) Reset() {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.allLogEntries = e.allLogEntries[:0]
	e.lifetimeSeverityCounts = make(map[string]int64)
	e.countsHistory = e.countsHistory[:0]
	e.severityTimeSeries = e.severityTimeSeries[:0]
	e.heatmapData = e.heatmapData[:0]
	e.drain3BySeverity = tui.InitializeDrain3BySeverity()
	e.drain3All = tui.NewDrain3Manager()
	e.servicesBySeverity = make(map[string][]tui.ServiceCount)
	e.lifetimeHostCounts = make(map[string]int64)
	e.lifetimeServiceCounts = make(map[string]int64)
	e.lifetimeAttrKeyCounts = make(map[string]map[string]int64)
	e.lifetimeWordCounts = make(map[string]int64)
	e.lifetimeAttrCounts = make(map[string]int64)
	e.streams = make(map[string]*StreamInfo)
	e.snapshot = nil
	e.totalLogsEver = 0
	e.totalBytes = 0
	e.statsStartTime = time.Now()
}

// --- Read-path: State Accessors (for TUI backward compat) ---

// GetAllLogEntries returns a copy of all log entries.
func (e *Engine) GetAllLogEntries() []tui.LogEntry {
	e.mu.RLock()
	defer e.mu.RUnlock()
	result := make([]tui.LogEntry, len(e.allLogEntries))
	copy(result, e.allLogEntries)
	return result
}

// GetLogEntryCount returns the number of log entries in the buffer.
func (e *Engine) GetLogEntryCount() int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return len(e.allLogEntries)
}

// GetHeatmapData returns a copy of heatmap data.
func (e *Engine) GetHeatmapData() []tui.HeatmapMinute {
	e.mu.RLock()
	defer e.mu.RUnlock()
	result := make([]tui.HeatmapMinute, len(e.heatmapData))
	copy(result, e.heatmapData)
	return result
}

// GetSeverityTimeSeries returns timestamped per-second severity counts.
// When no filters are provided, returns the pre-aggregated global series.
// When filters are present (e.g. search), builds the series from filtered log entries.
func (e *Engine) GetSeverityTimeSeries() []SeverityTimePoint {
	e.mu.RLock()
	defer e.mu.RUnlock()
	result := make([]SeverityTimePoint, len(e.severityTimeSeries))
	copy(result, e.severityTimeSeries)
	return result
}

// QuerySeverityTimeSeries returns per-second severity counts filtered by the given filters.
// This recomputes the time series from raw log entries to support search/stream filtering.
func (e *Engine) QuerySeverityTimeSeries(_ context.Context, filters InsightsFilters) []SeverityTimePoint {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// If no meaningful filters, return the pre-aggregated series
	if filters.Search == nil && filters.Severity == nil && filters.Start == nil && filters.End == nil {
		result := make([]SeverityTimePoint, len(e.severityTimeSeries))
		copy(result, e.severityTimeSeries)
		return result
	}

	filtered := e.filterEntries(filters)
	if len(filtered) == 0 {
		return nil
	}

	// Bucket entries by second
	buckets := make(map[int64]map[string]int)
	for _, entry := range filtered {
		ts := entry.Timestamp.Unix()
		if e.useLogTime && !entry.OrigTimestamp.IsZero() {
			ts = entry.OrigTimestamp.Unix()
		}
		if buckets[ts] == nil {
			buckets[ts] = make(map[string]int)
		}
		sev := strings.ToUpper(entry.Severity)
		if sev == "" {
			sev = "INFO"
		}
		buckets[ts][sev]++
	}

	// Sort timestamps
	timestamps := make([]int64, 0, len(buckets))
	for ts := range buckets {
		timestamps = append(timestamps, ts)
	}
	sort.Slice(timestamps, func(i, j int) bool { return timestamps[i] < timestamps[j] })

	// Build result
	result := make([]SeverityTimePoint, 0, len(timestamps))
	for _, ts := range timestamps {
		counts := buckets[ts]
		total := 0
		for _, c := range counts {
			total += c
		}
		result = append(result, SeverityTimePoint{
			Timestamp: ts,
			Counts:    counts,
			Total:     total,
		})
	}
	return result
}

// GetDrain3BySeverity returns the drain3 map (not a copy — callers should not modify).
func (e *Engine) GetDrain3BySeverity() map[string]*tui.Drain3Manager {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.drain3BySeverity
}

// GetServicesBySeverity returns the services map.
func (e *Engine) GetServicesBySeverity() map[string][]tui.ServiceCount {
	e.mu.RLock()
	defer e.mu.RUnlock()
	// Return a shallow copy
	result := make(map[string][]tui.ServiceCount, len(e.servicesBySeverity))
	for k, v := range e.servicesBySeverity {
		copied := make([]tui.ServiceCount, len(v))
		copy(copied, v)
		result[k] = copied
	}
	return result
}

// GetCountsHistory returns a copy of the counts history.
func (e *Engine) GetCountsHistory() []tui.SeverityCounts {
	e.mu.RLock()
	defer e.mu.RUnlock()
	result := make([]tui.SeverityCounts, len(e.countsHistory))
	copy(result, e.countsHistory)
	return result
}

// GetFrequencySnapshot returns the latest frequency snapshot.
func (e *Engine) GetFrequencySnapshot() *memory.FrequencySnapshot {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.snapshot
}

// GetLifetimeSeverityCounts returns a copy of lifetime severity counts.
func (e *Engine) GetLifetimeSeverityCounts() map[string]int64 {
	e.mu.RLock()
	defer e.mu.RUnlock()
	result := make(map[string]int64, len(e.lifetimeSeverityCounts))
	for k, v := range e.lifetimeSeverityCounts {
		result[k] = v
	}
	return result
}

// GetLifetimeHostCounts returns a copy of lifetime host counts.
func (e *Engine) GetLifetimeHostCounts() map[string]int64 {
	e.mu.RLock()
	defer e.mu.RUnlock()
	result := make(map[string]int64, len(e.lifetimeHostCounts))
	for k, v := range e.lifetimeHostCounts {
		result[k] = v
	}
	return result
}

// GetLifetimeServiceCounts returns a copy of lifetime service counts.
func (e *Engine) GetLifetimeServiceCounts() map[string]int64 {
	e.mu.RLock()
	defer e.mu.RUnlock()
	result := make(map[string]int64, len(e.lifetimeServiceCounts))
	for k, v := range e.lifetimeServiceCounts {
		result[k] = v
	}
	return result
}

// GetLifetimeAttrCounts returns a copy of lifetime attribute counts.
func (e *Engine) GetLifetimeAttrCounts() map[string]int64 {
	e.mu.RLock()
	defer e.mu.RUnlock()
	result := make(map[string]int64, len(e.lifetimeAttrCounts))
	for k, v := range e.lifetimeAttrCounts {
		result[k] = v
	}
	return result
}

// GetLifetimeWordCounts returns a copy of lifetime word counts.
func (e *Engine) GetLifetimeWordCounts() map[string]int64 {
	e.mu.RLock()
	defer e.mu.RUnlock()
	result := make(map[string]int64, len(e.lifetimeWordCounts))
	for k, v := range e.lifetimeWordCounts {
		result[k] = v
	}
	return result
}

// GetLifetimeAttrKeyCounts returns a copy of per-key attribute value counts.
func (e *Engine) GetLifetimeAttrKeyCounts() map[string]map[string]int64 {
	e.mu.RLock()
	defer e.mu.RUnlock()
	result := make(map[string]map[string]int64, len(e.lifetimeAttrKeyCounts))
	for k, v := range e.lifetimeAttrKeyCounts {
		inner := make(map[string]int64, len(v))
		for ik, iv := range v {
			inner[ik] = iv
		}
		result[k] = inner
	}
	return result
}

// GetStats returns engine statistics.
func (e *Engine) GetStats() EngineStats {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return EngineStats{
		StartTime:     e.statsStartTime,
		TotalLogsEver: e.totalLogsEver,
		TotalBytes:    e.totalBytes,
		BufferSize:    e.maxLogBuffer,
		BufferUsed:    len(e.allLogEntries),
	}
}

// GetStreams returns all tracked streams.
func (e *Engine) GetStreams() []StreamInfo {
	e.mu.RLock()
	defer e.mu.RUnlock()
	result := make([]StreamInfo, 0, len(e.streams))
	for _, s := range e.streams {
		result = append(result, *s)
	}
	return result
}

// --- Read-path: Query Methods (for Web API) ---

// QueryLogSamples returns log entries matching the given filters.
func (e *Engine) QueryLogSamples(_ context.Context, filters InsightsFilters) ([]LogSample, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	filtered := e.filterEntries(filters)

	// Apply limit
	limit := len(filtered)
	if filters.Limit != nil && *filters.Limit > 0 && *filters.Limit < limit {
		limit = *filters.Limit
	}

	// Take the most recent entries (from the end)
	start := len(filtered) - limit
	if start < 0 {
		start = 0
	}

	samples := make([]LogSample, 0, limit)
	for _, entry := range filtered[start:] {
		ts := entry.Timestamp.Unix()
		if e.useLogTime && !entry.OrigTimestamp.IsZero() {
			ts = entry.OrigTimestamp.Unix()
		}
		samples = append(samples, LogSample{
			Timestamp:  ts,
			Severity:   entry.Severity,
			Message:    entry.Message,
			Attributes: entry.Attributes,
			RawLine:    entry.RawLine,
		})
	}

	return samples, nil
}

// QuerySeverityData returns severity distribution grouped by the specified dimension.
func (e *Engine) QuerySeverityData(_ context.Context, filters InsightsFilters) ([]SeverityGroup, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	filtered := e.filterEntries(filters)
	groupBy := filters.GroupBy
	if groupBy == "" {
		groupBy = "service"
	}

	groups := make(map[string]map[string]int)
	for _, entry := range filtered {
		groupVal := e.getGroupValue(entry, groupBy)
		if groups[groupVal] == nil {
			groups[groupVal] = make(map[string]int)
		}
		groups[groupVal][entry.Severity]++
	}

	result := make([]SeverityGroup, 0, len(groups))
	for groupVal, counts := range groups {
		total := 0
		for _, c := range counts {
			total += c
		}
		result = append(result, SeverityGroup{
			GroupValue: groupVal,
			Counts:     counts,
			Total:      total,
		})
	}

	// Sort by total descending
	sort.Slice(result, func(i, j int) bool {
		return result[i].Total > result[j].Total
	})

	return result, nil
}

// QuerySentimentData returns sentiment distribution derived from severity.
func (e *Engine) QuerySentimentData(_ context.Context, filters InsightsFilters) (*SentimentData, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	filtered := e.filterEntries(filters)
	groupBy := filters.GroupBy
	if groupBy == "" {
		groupBy = "service"
	}

	// Group entries by time bucket (1-second) and group_by dimension
	type bucketKey struct {
		second   int64
		groupVal string
	}
	buckets := make(map[bucketKey][]float64)
	groupValueSet := make(map[string]bool)

	for _, entry := range filtered {
		groupVal := e.getGroupValue(entry, groupBy)
		groupValueSet[groupVal] = true

		ts := entry.Timestamp
		if e.useLogTime && !entry.OrigTimestamp.IsZero() {
			ts = entry.OrigTimestamp
		}
		sec := ts.Unix()

		key := bucketKey{second: sec, groupVal: groupVal}
		sentiment := severityToSentiment(entry.Severity)
		buckets[key] = append(buckets[key], sentiment)
	}

	// Build response
	groupValues := make([]string, 0, len(groupValueSet))
	for gv := range groupValueSet {
		groupValues = append(groupValues, gv)
	}
	sort.Strings(groupValues)

	sentimentBuckets := make([]SentimentBucket, 0, len(buckets))
	for key, sentiments := range buckets {
		avg := 0.0
		for _, s := range sentiments {
			avg += s
		}
		avg /= float64(len(sentiments))

		sentimentBuckets = append(sentimentBuckets, SentimentBucket{
			Timestamp:  key.second,
			GroupValue: key.groupVal,
			Sentiment:  avg,
			LogCount:   len(sentiments),
		})
	}

	return &SentimentData{
		GroupValues: groupValues,
		Buckets:     sentimentBuckets,
	}, nil
}

// QueryPatterns returns drain3 patterns grouped by the specified dimension.
func (e *Engine) QueryPatterns(_ context.Context, filters InsightsFilters) ([]PatternGroup, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	groupBy := filters.GroupBy
	if groupBy == "" {
		groupBy = "service"
	}

	limit := 20
	if filters.Limit != nil && *filters.Limit > 0 {
		limit = *filters.Limit
	}

	// For severity grouping, use the pre-built drain3 instances
	if groupBy == "severity" {
		result := make([]PatternGroup, 0)
		for severity, dm := range e.drain3BySeverity {
			if dm == nil {
				continue
			}
			patterns := dm.GetTopPatterns(limit)
			if len(patterns) == 0 {
				continue
			}
			items := make([]PatternItem, len(patterns))
			for i, p := range patterns {
				items[i] = PatternItem{
					Pattern:    p.Template,
					Count:      p.Count,
					Percentage: p.Percentage,
				}
			}
			result = append(result, PatternGroup{
				GroupValue: severity,
				Patterns:   items,
			})
		}
		return result, nil
	}

	// For other groupings, use the combined drain3 and return as single group
	patterns := e.drain3All.GetTopPatterns(limit)
	items := make([]PatternItem, len(patterns))
	for i, p := range patterns {
		items[i] = PatternItem{
			Pattern:    p.Template,
			Count:      p.Count,
			Percentage: p.Percentage,
		}
	}

	return []PatternGroup{{
		GroupValue: "all",
		Patterns:   items,
	}}, nil
}

// QueryClasses returns log classification distribution.
func (e *Engine) QueryClasses(_ context.Context, filters InsightsFilters) ([]ClassItem, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	filtered := e.filterEntries(filters)

	// Classify logs by severity as a simple categorization
	classCounts := make(map[string]int)
	for _, entry := range filtered {
		classCounts[entry.Severity]++
	}

	total := len(filtered)
	result := make([]ClassItem, 0, len(classCounts))
	for name, count := range classCounts {
		pct := 0.0
		if total > 0 {
			pct = float64(count) * 100.0 / float64(total)
		}
		result = append(result, ClassItem{
			Name:       name,
			Count:      count,
			Percentage: pct,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Count > result[j].Count
	})

	if filters.Limit != nil && *filters.Limit > 0 && len(result) > *filters.Limit {
		result = result[:*filters.Limit]
	}

	return result, nil
}

// QuerySummary returns an AI-generated summary of the logs.
func (e *Engine) QuerySummary(_ context.Context, filters InsightsFilters) (string, error) {
	e.mu.RLock()
	entries := e.filterEntries(filters)
	severityCounts := make(map[string]int)
	for _, entry := range entries {
		severityCounts[entry.Severity]++
	}
	totalLogs := len(entries)
	aiClient := e.aiClient
	e.mu.RUnlock()

	if aiClient == nil {
		// Return a statistical summary when AI is not configured
		return e.generateStatisticalSummary(totalLogs, severityCounts, entries), nil
	}

	// Build a prompt for the AI
	var sb strings.Builder
	sb.WriteString("Analyze these log entries and provide a brief summary of key issues, patterns, and recommendations:\n\n")
	sb.WriteString(fmt.Sprintf("Total logs: %d\n", totalLogs))
	sb.WriteString("Severity distribution:\n")
	for sev, count := range severityCounts {
		sb.WriteString(fmt.Sprintf("  %s: %d\n", sev, count))
	}
	sb.WriteString("\nRecent log samples:\n")

	// Include up to 50 recent entries
	sampleCount := min(50, len(entries))
	start := len(entries) - sampleCount
	for _, entry := range entries[start:] {
		sb.WriteString(fmt.Sprintf("[%s] %s\n", entry.Severity, entry.Message))
	}

	result, err := aiClient.AnalyzeLog(sb.String(), "SUMMARY", time.Now().Format(time.RFC3339), nil)
	if err != nil {
		return "", fmt.Errorf("AI analysis failed: %w", err)
	}

	return result, nil
}

// GetSentimentHeatmap returns an ASCII heatmap of sentiment over time.
func (e *Engine) GetSentimentHeatmap(_ context.Context, filters InsightsFilters) (string, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	filtered := e.filterEntries(filters)
	groupBy := filters.GroupBy
	if groupBy == "" {
		groupBy = "service"
	}

	// Build minute-by-group sentiment map
	type cellKey struct {
		minute   time.Time
		groupVal string
	}
	cells := make(map[cellKey][]float64)
	groupValueSet := make(map[string]bool)
	var minTime, maxTime time.Time

	for _, entry := range filtered {
		groupVal := e.getGroupValue(entry, groupBy)
		groupValueSet[groupVal] = true

		ts := entry.Timestamp
		if e.useLogTime && !entry.OrigTimestamp.IsZero() {
			ts = entry.OrigTimestamp
		}
		minute := ts.Truncate(time.Minute)

		if minTime.IsZero() || minute.Before(minTime) {
			minTime = minute
		}
		if maxTime.IsZero() || minute.After(maxTime) {
			maxTime = minute
		}

		key := cellKey{minute: minute, groupVal: groupVal}
		cells[key] = append(cells[key], severityToSentiment(entry.Severity))
	}

	if len(cells) == 0 {
		return "No data available for heatmap.", nil
	}

	// Build ASCII heatmap
	groupValues := make([]string, 0, len(groupValueSet))
	for gv := range groupValueSet {
		groupValues = append(groupValues, gv)
	}
	sort.Strings(groupValues)

	// Generate time columns (every 5 minutes)
	var sb strings.Builder
	sb.WriteString("Sentiment Heatmap (+ positive, . neutral, - negative)\n\n")

	// Determine label width
	maxLabelLen := 0
	for _, gv := range groupValues {
		if len(gv) > maxLabelLen {
			maxLabelLen = len(gv)
		}
	}
	if maxLabelLen > 20 {
		maxLabelLen = 20
	}

	for _, gv := range groupValues {
		label := gv
		if len(label) > maxLabelLen {
			label = label[:maxLabelLen]
		}
		sb.WriteString(fmt.Sprintf("%-*s │", maxLabelLen, label))

		// Walk through time buckets
		t := minTime
		for !t.After(maxTime) {
			key := cellKey{minute: t, groupVal: gv}
			if sentiments, ok := cells[key]; ok {
				avg := 0.0
				for _, s := range sentiments {
					avg += s
				}
				avg /= float64(len(sentiments))
				sb.WriteByte(sentimentToChar(avg))
			} else {
				sb.WriteByte(' ')
			}
			t = t.Add(time.Minute)
		}
		sb.WriteByte('\n')
	}

	return sb.String(), nil
}

// GetAnomalies detects anomalies in the log data.
func (e *Engine) GetAnomalies(_ context.Context, filters InsightsFilters) (string, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	filtered := e.filterEntries(filters)
	if len(filtered) == 0 {
		return "No log data available for anomaly detection.", nil
	}

	var sb strings.Builder
	sb.WriteString("# Anomaly Detection Report\n\n")

	// Check for fatal/critical logs
	fatalCount := 0
	errorCount := 0
	for _, entry := range filtered {
		switch entry.Severity {
		case "FATAL", "CRITICAL":
			fatalCount++
		case "ERROR":
			errorCount++
		}
	}

	if fatalCount > 0 {
		sb.WriteString(fmt.Sprintf("## Fatal/Critical Logs Detected\n- **%d** fatal/critical log entries found\n\n", fatalCount))
	}

	// Check error rate
	totalCount := len(filtered)
	if totalCount > 0 {
		errorRate := float64(errorCount) * 100.0 / float64(totalCount)
		if errorRate > 10 {
			sb.WriteString(fmt.Sprintf("## High Error Rate\n- Error rate: **%.1f%%** (%d/%d logs)\n\n", errorRate, errorCount, totalCount))
		}
	}

	// Check for error spikes (compare last 5 min vs overall rate)
	now := time.Now()
	recentCutoff := now.Add(-5 * time.Minute)
	recentErrors := 0
	recentTotal := 0
	for _, entry := range filtered {
		ts := entry.Timestamp
		if e.useLogTime && !entry.OrigTimestamp.IsZero() {
			ts = entry.OrigTimestamp
		}
		if ts.After(recentCutoff) {
			recentTotal++
			if entry.Severity == "ERROR" || entry.Severity == "FATAL" || entry.Severity == "CRITICAL" {
				recentErrors++
			}
		}
	}

	if recentTotal > 0 {
		recentErrorRate := float64(recentErrors) * 100.0 / float64(recentTotal)
		overallErrorRate := float64(errorCount+fatalCount) * 100.0 / float64(totalCount)
		if recentErrorRate > overallErrorRate*2 && recentErrors > 5 {
			sb.WriteString(fmt.Sprintf("## Error Spike Detected\n- Last 5 minutes: **%.1f%%** error rate (%d errors in %d logs)\n- Overall: **%.1f%%** error rate\n\n",
				recentErrorRate, recentErrors, recentTotal, overallErrorRate))
		}
	}

	// Check sentiment variance by service
	type serviceStats struct {
		sentiments []float64
	}
	serviceMap := make(map[string]*serviceStats)
	for _, entry := range filtered {
		svc := getServiceName(entry)
		if svc == "" || svc == "unknown" {
			continue
		}
		if serviceMap[svc] == nil {
			serviceMap[svc] = &serviceStats{}
		}
		serviceMap[svc].sentiments = append(serviceMap[svc].sentiments, severityToSentiment(entry.Severity))
	}

	for svc, stats := range serviceMap {
		if len(stats.sentiments) < 10 {
			continue
		}
		mean := 0.0
		for _, s := range stats.sentiments {
			mean += s
		}
		mean /= float64(len(stats.sentiments))

		variance := 0.0
		for _, s := range stats.sentiments {
			diff := s - mean
			variance += diff * diff
		}
		variance /= float64(len(stats.sentiments))

		if mean < -0.3 {
			sb.WriteString(fmt.Sprintf("## Negative Sentiment: %s\n- Mean sentiment: **%.2f** (%.0f logs)\n- Variance: %.2f\n\n",
				svc, mean, float64(len(stats.sentiments)), variance))
		}
	}

	if sb.Len() == len("# Anomaly Detection Report\n\n") {
		sb.WriteString("No anomalies detected in the current data.\n")
	}

	return sb.String(), nil
}

// QueryInsightsParams returns available filter dimension values.
func (e *Engine) QueryInsightsParams(_ context.Context, _ InsightsFilters) (*InsightsParams, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	params := &InsightsParams{
		TotalLogs: int64(e.totalLogsEver),
	}

	// Collect unique values from lifetime counts
	for host := range e.lifetimeHostCounts {
		params.Hosts = append(params.Hosts, host)
	}
	for svc := range e.lifetimeServiceCounts {
		params.Services = append(params.Services, svc)
	}
	for sev := range e.lifetimeSeverityCounts {
		params.Severities = append(params.Severities, sev)
	}

	// Collect namespace, pod, deployment, environment, cluster from attributes
	dimKeys := map[string]*[]string{
		"k8s.namespace":  &params.Namespaces,
		"k8s.pod":        &params.Pods,
		"k8s.deployment": &params.Deployments,
		"environment":    &params.Environments,
		"env":            &params.Environments,
		"cluster":        &params.Clusters,
	}
	for key, target := range dimKeys {
		if vals, ok := e.lifetimeAttrKeyCounts[key]; ok {
			for val := range vals {
				*target = append(*target, val)
			}
		}
	}

	// Time range
	if len(e.allLogEntries) > 0 {
		first := e.allLogEntries[0].Timestamp.Unix()
		last := e.allLogEntries[len(e.allLogEntries)-1].Timestamp.Unix()
		params.OldestLog = &first
		params.NewestLog = &last
	}

	// Sort all slices
	sort.Strings(params.Hosts)
	sort.Strings(params.Services)
	sort.Strings(params.Severities)
	sort.Strings(params.Namespaces)
	sort.Strings(params.Pods)
	sort.Strings(params.Deployments)
	sort.Strings(params.Environments)
	sort.Strings(params.Clusters)

	return params, nil
}

// --- Internal helpers ---

// filterEntries applies InsightsFilters to the in-memory log buffer.
// Must be called with at least a read lock held.
func (e *Engine) filterEntries(filters InsightsFilters) []tui.LogEntry {
	result := make([]tui.LogEntry, 0, len(e.allLogEntries))

	for _, entry := range e.allLogEntries {
		ts := entry.Timestamp
		if e.useLogTime && !entry.OrigTimestamp.IsZero() {
			ts = entry.OrigTimestamp
		}

		// Time range filter
		if filters.Start != nil && ts.Unix() < *filters.Start {
			continue
		}
		if filters.End != nil && ts.Unix() > *filters.End {
			continue
		}

		// Severity filter
		if filters.Severity != nil && *filters.Severity != "" {
			if !strings.EqualFold(entry.Severity, *filters.Severity) {
				continue
			}
		}

		// Search filter
		if filters.Search != nil && *filters.Search != "" {
			search := strings.ToLower(*filters.Search)
			if !strings.Contains(strings.ToLower(entry.Message), search) &&
				!strings.Contains(strings.ToLower(entry.RawLine), search) {
				found := false
				for _, v := range entry.Attributes {
					if strings.Contains(strings.ToLower(v), search) {
						found = true
						break
					}
				}
				if !found {
					continue
				}
			}
		}

		// Dimension filters
		if !e.matchesDimensionFilter(entry, filters) {
			continue
		}

		result = append(result, entry)
	}

	return result
}

// matchesDimensionFilter checks if an entry matches the dimension filters.
func (e *Engine) matchesDimensionFilter(entry tui.LogEntry, filters InsightsFilters) bool {
	if filters.Namespaces != nil && len(*filters.Namespaces) > 0 {
		ns := entry.Attributes["k8s.namespace"]
		if !containsString(*filters.Namespaces, ns) {
			return false
		}
	}
	if filters.Pods != nil && len(*filters.Pods) > 0 {
		pod := entry.Attributes["k8s.pod"]
		if !containsString(*filters.Pods, pod) {
			return false
		}
	}
	if filters.Hosts != nil && len(*filters.Hosts) > 0 {
		host := entry.Attributes["host"]
		if !containsString(*filters.Hosts, host) {
			return false
		}
	}
	if filters.Services != nil && len(*filters.Services) > 0 {
		svc := getServiceName(entry)
		if !containsString(*filters.Services, svc) {
			return false
		}
	}
	if filters.Environments != nil && len(*filters.Environments) > 0 {
		env := entry.Attributes["environment"]
		if env == "" {
			env = entry.Attributes["env"]
		}
		if !containsString(*filters.Environments, env) {
			return false
		}
	}
	if filters.Clusters != nil && len(*filters.Clusters) > 0 {
		cluster := entry.Attributes["cluster"]
		if !containsString(*filters.Clusters, cluster) {
			return false
		}
	}
	if filters.Deployments != nil && len(*filters.Deployments) > 0 {
		deploy := entry.Attributes["k8s.deployment"]
		if !containsString(*filters.Deployments, deploy) {
			return false
		}
	}
	return true
}

// getGroupValue extracts the group_by dimension value from a log entry.
func (e *Engine) getGroupValue(entry tui.LogEntry, groupBy string) string {
	switch groupBy {
	case "service":
		return getServiceName(entry)
	case "host":
		if h := entry.Attributes["host"]; h != "" {
			return h
		}
		return "unknown"
	case "severity":
		return entry.Severity
	case "namespace":
		if ns := entry.Attributes["k8s.namespace"]; ns != "" {
			return ns
		}
		return "default"
	case "pod":
		if pod := entry.Attributes["k8s.pod"]; pod != "" {
			return pod
		}
		return "unknown"
	case "deployment":
		if dep := entry.Attributes["k8s.deployment"]; dep != "" {
			return dep
		}
		return "unknown"
	case "env", "environment":
		if env := entry.Attributes["environment"]; env != "" {
			return env
		}
		if env := entry.Attributes["env"]; env != "" {
			return env
		}
		return "default"
	case "cluster":
		if cl := entry.Attributes["cluster"]; cl != "" {
			return cl
		}
		return "default"
	case "category":
		return entry.Severity // Use severity as category for local
	default:
		return getServiceName(entry)
	}
}

// updateHeatmapData updates minute-by-minute heatmap data.
func (e *Engine) updateHeatmapData(entry tui.LogEntry) {
	ts := entry.Timestamp
	if e.useLogTime && !entry.OrigTimestamp.IsZero() {
		ts = entry.OrigTimestamp
	}
	entryTime := ts.Truncate(time.Minute)

	// Find or create the minute entry
	var targetMinute *tui.HeatmapMinute
	for i := range e.heatmapData {
		if e.heatmapData[i].Timestamp.Equal(entryTime) {
			targetMinute = &e.heatmapData[i]
			break
		}
	}

	if targetMinute == nil {
		e.heatmapData = append(e.heatmapData, tui.HeatmapMinute{
			Timestamp: entryTime,
			Counts:    tui.SeverityCounts{},
		})
		targetMinute = &e.heatmapData[len(e.heatmapData)-1]
	}

	// Update severity count
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
	targetMinute.Counts.Total++

	// Prune old data (keep 6 hours)
	cutoffTime := time.Now().Add(-6 * time.Hour)
	filtered := e.heatmapData[:0]
	for _, minute := range e.heatmapData {
		if minute.Timestamp.After(cutoffTime) {
			filtered = append(filtered, minute)
		}
	}
	e.heatmapData = filtered
}

// updateServicesBySeverity updates the services-by-severity tracking.
func (e *Engine) updateServicesBySeverity(entry tui.LogEntry) {
	severity := entry.Severity
	if severity == "" {
		severity = "UNKNOWN"
	}

	serviceName := getServiceName(entry)
	if serviceName == "" || serviceName == "unknown" {
		return
	}

	if e.servicesBySeverity[severity] == nil {
		e.servicesBySeverity[severity] = make([]tui.ServiceCount, 0)
	}

	found := false
	for i := range e.servicesBySeverity[severity] {
		if e.servicesBySeverity[severity][i].Service == serviceName {
			e.servicesBySeverity[severity][i].Count++
			found = true
			break
		}
	}

	if !found {
		e.servicesBySeverity[severity] = append(e.servicesBySeverity[severity], tui.ServiceCount{
			Service: serviceName,
			Count:   1,
		})
	}

	// Keep top 10 sorted
	services := e.servicesBySeverity[severity]
	sort.Slice(services, func(i, j int) bool {
		return services[i].Count > services[j].Count
	})
	if len(services) > 10 {
		e.servicesBySeverity[severity] = services[:10]
	}
}

// updateStreamTracking updates stream info for a log entry.
func (e *Engine) updateStreamTracking(entry tui.LogEntry) {
	// Determine source and stream from attributes
	source := "stdin"
	stream := "stdin"

	if ns := entry.Attributes["k8s.namespace"]; ns != "" {
		source = "k8s"
		pod := entry.Attributes["k8s.pod"]
		if pod != "" {
			stream = ns + "/" + pod
		} else {
			stream = ns
		}
	} else if svc := entry.Attributes["service.name"]; svc != "" {
		source = "otlp"
		stream = svc
	} else if filename := entry.Attributes["source_file"]; filename != "" {
		source = "file"
		stream = filename
	}

	key := source + ":" + stream
	if s, ok := e.streams[key]; ok {
		s.LogCount++
		s.LastSeen = time.Now()
	} else {
		e.streams[key] = &StreamInfo{
			Source:   source,
			Stream:   stream,
			LogCount: 1,
			LastSeen: time.Now(),
			Active:   true,
		}
	}
}

// generateStatisticalSummary creates a text summary when AI is not configured.
func (e *Engine) generateStatisticalSummary(totalLogs int, severityCounts map[string]int, entries []tui.LogEntry) string {
	var sb strings.Builder
	sb.WriteString("## Log Analysis Summary (Statistical)\n\n")
	sb.WriteString(fmt.Sprintf("**Total logs analyzed:** %d\n\n", totalLogs))

	sb.WriteString("### Severity Distribution\n")
	for sev, count := range severityCounts {
		pct := float64(count) * 100.0 / float64(max(totalLogs, 1))
		sb.WriteString(fmt.Sprintf("- %s: %d (%.1f%%)\n", sev, count, pct))
	}

	sb.WriteString("\n### Top Patterns\n")
	patterns := e.drain3All.GetTopPatterns(5)
	for i, p := range patterns {
		sb.WriteString(fmt.Sprintf("%d. `%s` (%d occurrences, %.1f%%)\n", i+1, p.Template, p.Count, p.Percentage))
	}

	if len(entries) > 0 {
		uptime := time.Since(e.statsStartTime).Round(time.Second)
		sb.WriteString(fmt.Sprintf("\n**Uptime:** %s | **Buffer:** %d/%d\n", uptime, len(e.allLogEntries), e.maxLogBuffer))
	}

	sb.WriteString("\n*Note: AI analysis is not configured. Set up an AI provider for deeper insights.*\n")
	return sb.String()
}

// --- Utility functions ---

func getServiceName(entry tui.LogEntry) string {
	if svc := entry.Attributes["service"]; svc != "" {
		return svc
	}
	if svc := entry.Attributes["service.name"]; svc != "" {
		return svc
	}
	if svc := entry.Attributes["serviceName"]; svc != "" {
		return svc
	}
	if svc := entry.Attributes["app"]; svc != "" {
		return svc
	}
	if svc := entry.Attributes["application"]; svc != "" {
		return svc
	}
	if host := entry.Attributes["host"]; host != "" {
		return "host:" + host
	}
	return "unknown"
}

func severityToSentiment(severity string) float64 {
	switch strings.ToUpper(severity) {
	case "FATAL", "CRITICAL":
		return -1.0
	case "ERROR":
		return -0.6
	case "WARN", "WARNING":
		return 0.0
	case "INFO":
		return 0.5
	case "DEBUG":
		return 0.7
	case "TRACE":
		return 0.9
	default:
		return 0.0
	}
}

func sentimentToChar(sentiment float64) byte {
	if sentiment > 0.3 {
		return '+'
	} else if sentiment < -0.3 {
		return '-'
	}
	return '.'
}

func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if strings.EqualFold(item, s) {
			return true
		}
	}
	return false
}

