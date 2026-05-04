package engine

import "time"

// InsightsFilters defines filters for querying log analysis data.
// For local Gonzo, time range filters apply to the in-memory log buffer.
// Dimension filters (environments, namespaces, etc.) map to log entry attributes.
type InsightsFilters struct {
	Start        *int64    `json:"start,omitempty"`
	End          *int64    `json:"end,omitempty"`
	GroupBy      string    `json:"group_by,omitempty"`
	Environments *[]string `json:"environments,omitempty"`
	Clusters     *[]string `json:"clusters,omitempty"`
	Namespaces   *[]string `json:"namespaces,omitempty"`
	Deployments  *[]string `json:"deployments,omitempty"`
	Pods         *[]string `json:"pods,omitempty"`
	Hosts        *[]string `json:"hosts,omitempty"`
	Services     *[]string `json:"services,omitempty"`
	Search       *string   `json:"search,omitempty"`
	Severity     *string   `json:"severity,omitempty"`
	Limit        *int      `json:"limit,omitempty"`
}

// LogSample is a single log entry for API responses.
type LogSample struct {
	Timestamp  int64             `json:"timestamp"`
	Severity   string            `json:"severity"`
	Message    string            `json:"message"`
	Attributes map[string]string `json:"attributes,omitempty"`
	RawLine    string            `json:"raw_line,omitempty"`
}

// SeverityGroup represents severity distribution for one group_by value.
type SeverityGroup struct {
	GroupValue string         `json:"group_value"`
	Counts     map[string]int `json:"counts"`
	Total      int            `json:"total"`
}

// SentimentBucket represents one time bucket of sentiment data for a group.
type SentimentBucket struct {
	Timestamp   int64   `json:"timestamp"`
	GroupValue  string  `json:"group_value"`
	Sentiment   float64 `json:"sentiment"`
	IsAnomaly   bool    `json:"is_anomaly,omitempty"`
	LogCount    int     `json:"log_count"`
}

// SentimentData is the response format for sentiment queries.
type SentimentData struct {
	GroupValues []string          `json:"group_values"`
	Buckets     []SentimentBucket `json:"buckets"`
}

// PatternItem represents a single log pattern.
type PatternItem struct {
	Pattern    string  `json:"pattern"`
	Count      int     `json:"count"`
	Percentage float64 `json:"percentage"`
	Sample     string  `json:"sample,omitempty"`
}

// PatternGroup represents patterns for one group_by value.
type PatternGroup struct {
	GroupValue string        `json:"group_value"`
	Patterns   []PatternItem `json:"patterns"`
}

// ClassItem represents a log class/category.
type ClassItem struct {
	Name       string  `json:"name"`
	Count      int     `json:"count"`
	Percentage float64 `json:"percentage"`
}

// HeatmapMinuteData represents one minute of heatmap data.
type HeatmapMinuteData struct {
	Timestamp int64          `json:"timestamp"`
	Counts    map[string]int `json:"counts"`
}

// SeverityTimePoint represents severity counts at a point in time (1-second interval).
type SeverityTimePoint struct {
	Timestamp int64          `json:"timestamp"`
	Counts    map[string]int `json:"counts"`
	Total     int            `json:"total"`
}

// AnomalyInfo represents a detected anomaly.
type AnomalyInfo struct {
	Type        string  `json:"type"`
	Severity    string  `json:"severity"`
	Description string  `json:"description"`
	Timestamp   int64   `json:"timestamp"`
	Score       float64 `json:"score"`
}

// InsightsParams describes available filter dimension values.
type InsightsParams struct {
	Environments []string `json:"environments,omitempty"`
	Clusters     []string `json:"clusters,omitempty"`
	Namespaces   []string `json:"namespaces,omitempty"`
	Deployments  []string `json:"deployments,omitempty"`
	Pods         []string `json:"pods,omitempty"`
	Hosts        []string `json:"hosts,omitempty"`
	Services     []string `json:"services,omitempty"`
	Severities   []string `json:"severities,omitempty"`
	Categories   []string `json:"categories,omitempty"`
	TotalLogs    int64    `json:"total_logs"`
	OldestLog    *int64   `json:"oldest_log,omitempty"`
	NewestLog    *int64   `json:"newest_log,omitempty"`
}

// StreamInfo represents an active log input stream.
type StreamInfo struct {
	Source    string    `json:"source"`
	Stream   string    `json:"stream"`
	LogCount int64     `json:"log_count"`
	LastSeen time.Time `json:"last_seen"`
	Active   bool      `json:"active"`
}

// StatusInfo represents server status.
type StatusInfo struct {
	Uptime       string       `json:"uptime"`
	TotalLogs    int64        `json:"total_logs"`
	TotalBytes   int64        `json:"total_bytes"`
	LogRate      float64      `json:"log_rate"`
	Streams      []StreamInfo `json:"streams"`
	BufferSize   int          `json:"buffer_size"`
	BufferUsed   int          `json:"buffer_used"`
	AIConfigured bool         `json:"ai_configured"`
}

// AttributeEntry represents a single top attribute key+value with count.
type AttributeEntry struct {
	Key        string  `json:"key"`
	Value      string  `json:"value"`
	Count      int64   `json:"count"`
	Percentage float64 `json:"percentage"`
}

// EngineStats holds engine-level statistics.
type EngineStats struct {
	StartTime      time.Time `json:"start_time"`
	TotalLogsEver  int       `json:"total_logs_ever"`
	TotalBytes     int64     `json:"total_bytes"`
	BufferSize     int       `json:"buffer_size"`
	BufferUsed     int       `json:"buffer_used"`
}
