package tui

import (
	"strings"
)

// SeverityCounts tracks log counts by severity level for a time interval
type SeverityCounts struct {
	Trace    int
	Debug    int
	Info     int
	Warn     int
	Error    int
	Fatal    int
	Critical int
	Unknown  int
	Total    int
}

// AddCount adds a count for the given severity level
func (sc *SeverityCounts) AddCount(severity string) {
	normalizedSeverity := normalizeSeverityLevel(severity)
	switch normalizedSeverity {
	case "TRACE":
		sc.Trace++
	case "DEBUG":
		sc.Debug++
	case "INFO":
		sc.Info++
	case "WARN":
		sc.Warn++
	case "ERROR":
		sc.Error++
	case "FATAL":
		sc.Fatal++
	case "CRITICAL":
		sc.Critical++
	default:
		sc.Unknown++
	}
	sc.Total++
}

// normalizeSeverityLevel normalizes severity levels to standard format
func normalizeSeverityLevel(severity string) string {
	normalized := strings.ToUpper(strings.TrimSpace(severity))
	switch normalized {
	case "TRACE", "TRC":
		return "TRACE"
	case "DEBUG", "DBG", "DEBG":
		return "DEBUG"
	case "INFO", "INFORMATION", "INF":
		return "INFO"
	case "WARN", "WARNING", "WRNG", "WRN":
		return "WARN"
	case "ERROR", "ERR":
		return "ERROR"
	case "FATAL", "FTL":
		return "FATAL"
	case "CRITICAL", "CRIT", "CRT":
		return "CRITICAL"
	default:
		return "UNKNOWN"
	}
}

// NewSeverityCountsFromEntries creates SeverityCounts from a slice of log entries
func NewSeverityCountsFromEntries(entries []LogEntry) *SeverityCounts {
	counts := &SeverityCounts{}
	for _, entry := range entries {
		counts.AddCount(entry.Severity)
	}
	return counts
}
