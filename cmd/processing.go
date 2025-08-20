package main

import (
	"strings"

	"github.com/control-theory/gonzo/internal/analyzer"
	"github.com/control-theory/gonzo/internal/otlplog"
	"github.com/control-theory/gonzo/internal/tui"
)

// processLogLine processes a single log line and updates frequency memory
func (m *simpleTuiModel) processLogLine(line string) {
	// Early filter: Skip OTLP collector logs about traces/metrics processing
	if isOTLPSignalLog(line) {
		return // Skip processing this line entirely
	}
	
	// Handle multi-line JSON accumulation
	if m.tryAccumulateJSON(line) {
		return // Line was accumulated, wait for complete JSON
	}
	
	// Count only lines that pass the filter
	m.logCount++
	
	// Detect format
	format := m.formatDetector.DetectFormat(line)

	var result *analyzer.AnalysisResult
	var attributes map[string]string
	var logEntry *tui.LogEntry

	if format == otlplog.FormatOTLP {
		// Handle OTLP format
		if m.formatDetector.IsOTLPBatch(line) {
			// Parse OTLP batch and extract ALL log entries
			logsData, err := m.formatDetector.ParseOTLPBatch(line)
			if err != nil {
				// Fallback to text analysis
				result = m.textAnalyzer.AnalyzeLine(line)
				attributes = make(map[string]string)
				logEntry = createFallbackLogEntry(line)
				
				// Process the single fallback entry
				m.processSingleLogEntry(result, attributes, logEntry)
			} else {
				// Extract ALL log entries from the batch
				logEntries := extractAllLogEntriesFromOTLPBatch(logsData)
				
				// Process each log entry individually
				for _, entry := range logEntries {
					// Analyze each log entry for frequency data
					entryResult := m.otlpAnalyzer.AnalyzeOTLPRecord(convertLogEntryToOTLPRecord(entry))
					entryAttributes := entry.Attributes // Already includes resource + record attributes
					
					m.processSingleLogEntry(entryResult, entryAttributes, entry)
				}
			}
			return // Important: return early for batch processing to avoid duplicate processing below
		} else {
			// Parse single OTLP record
			record, err := m.formatDetector.ParseSingleOTLPRecord(line)
			if err != nil {
				result = m.textAnalyzer.AnalyzeLine(line)
				attributes = make(map[string]string)
				logEntry = createFallbackLogEntry(line)
			} else {
				result = m.otlpAnalyzer.AnalyzeOTLPRecord(record)
				attributes = m.otlpAnalyzer.ExtractAttributesFromOTLPRecord(record)
				logEntry = extractLogEntryFromOTLPRecord(record)
			}
		}
	} else {
		// Convert non-OTLP format to OTLP
		otlpRecord, err := m.logConverter.ConvertToOTLP(line, format)
		if err != nil {
			result = m.textAnalyzer.AnalyzeLine(line)
			attributes = make(map[string]string)
			logEntry = createFallbackLogEntry(line)
		} else {
			result = m.otlpAnalyzer.AnalyzeOTLPRecord(otlpRecord)
			attributes = m.otlpAnalyzer.ExtractAttributesFromOTLPRecord(otlpRecord)
			logEntry = extractLogEntryFromOTLPRecord(otlpRecord)
		}
	}

	// Process single log entry (for non-batch OTLP and other formats)
	m.processSingleLogEntry(result, attributes, logEntry)
}

// processSingleLogEntry processes a single log entry for frequency analysis and dashboard updates
func (m *simpleTuiModel) processSingleLogEntry(result *analyzer.AnalysisResult, attributes map[string]string, logEntry *tui.LogEntry) {
	// Add results to frequency memory
	m.freqMemory.AddWords(result.Words)
	m.freqMemory.AddPhrases(result.Phrases)
	m.freqMemory.AddAttributes(attributes)

	// Track severity counts and send log entry to dashboard
	if logEntry != nil {
		// Count severity for this interval
		m.severityCounts.AddCount(logEntry.Severity)
		
		updateMsg := tui.UpdateMsg{NewLogEntry: logEntry}
		m.dashboard.Update(updateMsg)
	}
}

// isOTLPSignalLog detects if a log message is about OTLP trace/metric processing
// and should be filtered out from a log analyzer focused on application logs
func isOTLPSignalLog(message string) bool {
	// Convert to lowercase for case-insensitive matching
	msg := strings.ToLower(message)
	
	// Filter out OTLP collector logs that show up as separate columns with "Logs/Metrics/Traces"
	// These follow the pattern: "timestamp\tinfo\tLogs\t{json data}"
	// The tab-separated format creates the column appearance in the UI
	if (strings.Contains(msg, "\tlogs\t") || strings.Contains(msg, "\tmetrics\t") || strings.Contains(msg, "\ttraces\t")) &&
		strings.Contains(msg, "otelcol.component") {
		return true
	}
	
	// Skip OTLP collector logs about signal processing
	if strings.Contains(msg, "otelcol.signal") {
		// Check if it's about traces or metrics (keep actual log entries)
		if strings.Contains(msg, `"metrics"`) || strings.Contains(msg, `"traces"`) {
			return true
		}
	}
	
	// Filter out OTLP collector operational logs about signal processing  
	if strings.Contains(msg, "otelcol.component") {
		patterns := []string{
			"resource metrics",
			"resource traces", 
			"data points",
			"metrics exported",
			"traces exported",
			"spans exported",
		}
		for _, pattern := range patterns {
			if strings.Contains(msg, pattern) {
				return true
			}
		}
	}
	
	return false
}

// tryAccumulateJSON attempts to accumulate multi-line JSON and process when complete
func (m *simpleTuiModel) tryAccumulateJSON(line string) bool {
	// Check if this line could be part of a JSON object
	trimmed := strings.TrimSpace(line)
	
	// If we're not currently accumulating JSON, check if this line starts a JSON object
	if !m.inJsonObject {
		if trimmed == "{" || strings.HasPrefix(trimmed, "{") {
			// Start accumulating JSON
			m.inJsonObject = true
			m.jsonBuffer.Reset()
			m.jsonDepth = 0
			m.jsonBuffer.WriteString(line)
			m.jsonBuffer.WriteString("\n")
			
			// Count braces in this line
			m.jsonDepth += countJSONDepth(line)
			
			// If depth is already 0, we have a complete single-line JSON
			if m.jsonDepth <= 0 {
				completeJSON := strings.TrimSpace(m.jsonBuffer.String())
				m.resetJSONAccumulation()
				m.processCompleteJSON(completeJSON)
				return true
			}
			
			return true // Line was accumulated
		}
		// Not starting JSON, process normally
		return false
	}
	
	// We're already accumulating JSON, add this line
	m.jsonBuffer.WriteString(line)
	m.jsonBuffer.WriteString("\n")
	
	// Update depth count
	m.jsonDepth += countJSONDepth(line)
	
	// If depth reaches 0 or below, we have a complete JSON object
	if m.jsonDepth <= 0 {
		completeJSON := strings.TrimSpace(m.jsonBuffer.String())
		m.resetJSONAccumulation()
		m.processCompleteJSON(completeJSON)
		return true
	}
	
	return true // Line was accumulated, waiting for more
}

// countJSONDepth counts the net change in JSON nesting depth for a line
func countJSONDepth(line string) int {
	depth := 0
	inString := false
	escaped := false
	
	for _, char := range line {
		if escaped {
			escaped = false
			continue
		}
		
		switch char {
		case '\\':
			if inString {
				escaped = true
			}
		case '"':
			inString = !inString
		case '{', '[':
			if !inString {
				depth++
			}
		case '}', ']':
			if !inString {
				depth--
			}
		}
	}
	
	return depth
}

// resetJSONAccumulation resets the JSON accumulation state
func (m *simpleTuiModel) resetJSONAccumulation() {
	m.inJsonObject = false
	m.jsonDepth = 0
	m.jsonBuffer.Reset()
}

// processCompleteJSON processes a complete JSON object (single or multi-line)
func (m *simpleTuiModel) processCompleteJSON(jsonStr string) {
	// Count this as a log entry
	m.logCount++
	
	// Detect format of the complete JSON
	format := m.formatDetector.DetectFormat(jsonStr)
	
	var result *analyzer.AnalysisResult
	var attributes map[string]string
	var logEntry *tui.LogEntry

	if format == otlplog.FormatOTLP {
		// Handle OTLP format
		if m.formatDetector.IsOTLPBatch(jsonStr) {
			// Parse OTLP batch and extract ALL log entries
			logsData, err := m.formatDetector.ParseOTLPBatch(jsonStr)
			if err != nil {
				// Fallback to text analysis
				result = m.textAnalyzer.AnalyzeLine(jsonStr)
				attributes = make(map[string]string)
				logEntry = createFallbackLogEntry(jsonStr)
				
				// Process the single fallback entry
				m.processSingleLogEntry(result, attributes, logEntry)
			} else {
				// Extract ALL log entries from the batch
				logEntries := extractAllLogEntriesFromOTLPBatch(logsData)
				
				// Process each log entry individually
				for _, entry := range logEntries {
					// Analyze each log entry for frequency data
					entryResult := m.otlpAnalyzer.AnalyzeOTLPRecord(convertLogEntryToOTLPRecord(entry))
					entryAttributes := entry.Attributes // Already includes resource + record attributes
					
					m.processSingleLogEntry(entryResult, entryAttributes, entry)
				}
			}
			return // Important: return early for batch processing to avoid duplicate processing below
		} else {
			// Parse single OTLP record
			record, err := m.formatDetector.ParseSingleOTLPRecord(jsonStr)
			if err != nil {
				result = m.textAnalyzer.AnalyzeLine(jsonStr)
				attributes = make(map[string]string)
				logEntry = createFallbackLogEntry(jsonStr)
			} else {
				result = m.otlpAnalyzer.AnalyzeOTLPRecord(record)
				attributes = m.otlpAnalyzer.ExtractAttributesFromOTLPRecord(record)
				logEntry = extractLogEntryFromOTLPRecord(record)
			}
		}
	} else {
		// Convert non-OTLP format to OTLP
		otlpRecord, err := m.logConverter.ConvertToOTLP(jsonStr, format)
		if err != nil {
			result = m.textAnalyzer.AnalyzeLine(jsonStr)
			attributes = make(map[string]string)
			logEntry = createFallbackLogEntry(jsonStr)
		} else {
			result = m.otlpAnalyzer.AnalyzeOTLPRecord(otlpRecord)
			attributes = m.otlpAnalyzer.ExtractAttributesFromOTLPRecord(otlpRecord)
			logEntry = extractLogEntryFromOTLPRecord(otlpRecord)
		}
	}

	// Process single log entry (for non-batch OTLP and other formats)
	m.processSingleLogEntry(result, attributes, logEntry)
}