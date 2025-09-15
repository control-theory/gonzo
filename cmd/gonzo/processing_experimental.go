package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/control-theory/gonzo/internal/analyzer"
	"github.com/control-theory/gonzo/internal/logger"
	"github.com/control-theory/gonzo/internal/otlplog"
	"github.com/control-theory/gonzo/internal/tui"
)

// extractSourceInfo extracts source information from JSON line
func (m *experimentalTuiModel) extractSourceInfo(line string) (sourceType string, sourceID int) {
	// Default to stdin
	sourceType = "I"
	sourceID = 0

	// Try to extract _source field from JSON
	if strings.Contains(line, "_source") {
		var jsonData map[string]interface{}
		if err := json.Unmarshal([]byte(line), &jsonData); err == nil {
			if sourceInfo, ok := jsonData["_source"].(map[string]interface{}); ok {
				if typeStr, ok := sourceInfo["type"].(string); ok {
					// Map source type to single letter
					switch typeStr {
					case "loki":
						sourceType = "L"
					case "docker":
						sourceType = "D"
					case "vmlogs":
						sourceType = "V"
					case "otlp":
						sourceType = "O"
					case "file", "files":
						sourceType = "F"
					case "stdin":
						sourceType = "I"
					default:
						sourceType = "?"
					}

					// Track source instances
					// Use multiplexer_source if available (for multi-source setups)
					// Otherwise use the identifier field
					var sourceKey string
					if metadata, ok := sourceInfo["metadata"].(map[string]interface{}); ok {
						if muxSource, ok := metadata["multiplexer_source"].(string); ok {
							sourceKey = typeStr + ":" + muxSource
						}
					}

					// Fallback to identifier if no multiplexer_source
					if sourceKey == "" {
						if identifier, ok := sourceInfo["identifier"].(string); ok {
							sourceKey = typeStr + ":" + identifier
						} else {
							sourceKey = typeStr + ":default"
						}
					}

					if storedID, exists := m.sourceTypes[sourceKey]; !exists {
						// New source, assign an ID
						m.sourceCounters[typeStr]++
						m.sourceTypes[sourceKey] = fmt.Sprintf("%d", m.sourceCounters[typeStr])
						sourceID = m.sourceCounters[typeStr]
					} else {
						// Existing source, get its stored ID
						sourceID, _ = strconv.Atoi(storedID)
					}
				}
			}
		}
	}

	return sourceType, sourceID
}

// processLogLine processes a single log line for the experimental model
func (m *experimentalTuiModel) processLogLine(line string) {
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

	// Extract source information from the line
	sourceType, sourceID := m.extractSourceInfo(line)

	// Detect format
	format := m.formatDetector.DetectFormat(line)
	var result *analyzer.AnalysisResult
	var attributes map[string]string
	var logEntry *tui.LogEntry

	if format == otlplog.FormatOTLP {
		logger.Debug("[EXPERIMENTAL] Processing as OTLP format")
		// Handle OTLP format
		if m.formatDetector.IsOTLPBatch(line) {
			// Parse OTLP batch and extract ALL log entries
			logsData, err := m.formatDetector.ParseOTLPBatch(line)
			if err != nil {
				// Fallback to text analysis
				result = m.textAnalyzer.AnalyzeLine(line)
				attributes = make(map[string]string)
				logEntry = createFallbackLogEntry(line)
			logEntry.SourceType = sourceType
			logEntry.SourceID = sourceID

				// Process the single fallback entry
				m.processSingleLogEntry(result, attributes, logEntry)
			} else {
				// Extract ALL log entries from the batch
				logEntries := extractAllLogEntriesFromOTLPBatch(logsData)

				// Process each log entry individually
				for _, entry := range logEntries {
					// Set source info on the entry
					entry.SourceType = sourceType
					entry.SourceID = sourceID

					// Analyze each log entry for frequency data
					entryResult := m.otlpAnalyzer.AnalyzeOTLPRecord(convertLogEntryToOTLPRecord(entry))
					entryAttributes := entry.Attributes // Already includes resource + record attributes

					m.processSingleLogEntry(entryResult, entryAttributes, entry)
				}
			}
			return // Important: return early for batch processing
		}
		// Parse single OTLP record
		record, err := m.formatDetector.ParseSingleOTLPRecord(line)
		if err != nil {
			logger.Debugf("[EXPERIMENTAL] Failed to parse OTLP record: %v", err)
			result = m.textAnalyzer.AnalyzeLine(line)
			attributes = make(map[string]string)
			logEntry = createFallbackLogEntry(line)
			logEntry.SourceType = sourceType
			logEntry.SourceID = sourceID
		} else {
			logger.Debug("[EXPERIMENTAL] Successfully parsed OTLP record")
			result = m.otlpAnalyzer.AnalyzeOTLPRecord(record)
			attributes = m.otlpAnalyzer.ExtractAttributesFromOTLPRecord(record)
			logEntry = extractLogEntryFromOTLPRecord(record)
			if logEntry != nil {
				logEntry.SourceType = sourceType
				logEntry.SourceID = sourceID
			}
			if logEntry != nil {
				if len(logEntry.Attributes) > 0 {
					logger.Debugf("[EXPERIMENTAL] LogEntry has %d attributes", len(logEntry.Attributes))
					// Log first few attribute keys for debugging
					var keys []string
					for k := range logEntry.Attributes {
						keys = append(keys, k)
						if len(keys) >= 5 {
							break
						}
					}
					logger.Debugf("[EXPERIMENTAL] First attribute keys: %v", keys)
				} else {
					logger.Debug("[EXPERIMENTAL] LogEntry has NO attributes")
					// Debug: Check if OTLP record has attributes
					if record != nil && len(record.Attributes) > 0 {
						logger.Debugf("[EXPERIMENTAL] OTLP record has %d attributes but LogEntry has none!", len(record.Attributes))
					}
				}
			}
		}
	} else {
		logger.Debugf("[EXPERIMENTAL] Processing as non-OTLP format: %v", format)
		// Convert non-OTLP format to OTLP
		otlpRecord, err := m.logConverter.ConvertToOTLP(line, format)
		if err != nil {
			logger.Debugf("[EXPERIMENTAL] Failed to convert to OTLP: %v", err)
			result = m.textAnalyzer.AnalyzeLine(line)
			attributes = make(map[string]string)
			logEntry = createFallbackLogEntry(line)
			logEntry.SourceType = sourceType
			logEntry.SourceID = sourceID
		} else {
			logger.Debug("[EXPERIMENTAL] Successfully converted to OTLP")
			result = m.otlpAnalyzer.AnalyzeOTLPRecord(otlpRecord)
			attributes = m.otlpAnalyzer.ExtractAttributesFromOTLPRecord(otlpRecord)
			logEntry = extractLogEntryFromOTLPRecord(otlpRecord)
			if logEntry != nil {
				logEntry.SourceType = sourceType
				logEntry.SourceID = sourceID
			}
			// Debug: log.Printf("[UNIFIED-v2] Extracted logEntry: %+v", logEntry)
		}
	}

	// Process single log entry (for non-batch OTLP and other formats)
	m.processSingleLogEntry(result, attributes, logEntry)
}

// processSingleLogEntry processes a single log entry for frequency analysis
func (m *experimentalTuiModel) processSingleLogEntry(result *analyzer.AnalysisResult, attributes map[string]string, logEntry *tui.LogEntry) {
	// Add results to frequency memory
	m.freqMemory.AddWords(result.Words)
	m.freqMemory.AddPhrases(result.Phrases)
	m.freqMemory.AddAttributes(attributes)

	// Track severity counts and send log entry to dashboard
	if logEntry != nil {
		// Count severity for this interval
		switch strings.ToUpper(logEntry.Severity) {
		case "FATAL", "CRITICAL":
			m.severityCounts.Fatal++
		case "ERROR":
			m.severityCounts.Error++
		case "WARN", "WARNING":
			m.severityCounts.Warn++
		case "INFO":
			m.severityCounts.Info++
		case "DEBUG", "TRACE":
			m.severityCounts.Debug++
		default:
			m.severityCounts.Info++ // Default to info if unknown
		}

		// Send log entry to dashboard via UpdateMsg
		updateMsg := tui.UpdateMsg{
			NewLogEntry: logEntry,
		}
		newDashboard, _ := m.dashboard.Update(updateMsg)
		m.dashboard = newDashboard.(*tui.DashboardModel)
		logger.Debugf("[EXPERIMENTAL] Sent log entry to dashboard: %s", logEntry.Message)
	}
}

// tryAccumulateJSON handles multi-line JSON accumulation for the experimental model
func (m *experimentalTuiModel) tryAccumulateJSON(line string) bool {
	// Check if this looks like the start of a JSON object
	trimmed := strings.TrimSpace(line)

	// If we're not in a JSON object and this line starts with {, begin accumulation
	if !m.inJSONObject && strings.HasPrefix(trimmed, "{") {
		m.jsonBuffer.Reset()
		m.jsonBuffer.WriteString(line)
		m.jsonDepth = 0

		// Count brackets to track nesting
		for _, ch := range line {
			switch ch {
			case '{':
				m.jsonDepth++
			case '}':
				m.jsonDepth--
			}
		}

		// If brackets are balanced, we have a complete JSON object
		if m.jsonDepth == 0 {
			// Complete single-line JSON, don't accumulate
			return false
		}

		// Start accumulating
		m.inJSONObject = true
		return true
	}

	// If we're accumulating, add this line
	if m.inJSONObject {
		m.jsonBuffer.WriteString("\n")
		m.jsonBuffer.WriteString(line)

		// Update bracket count
		for _, ch := range line {
			switch ch {
			case '{':
				m.jsonDepth++
			case '}':
				m.jsonDepth--
			}
		}

		// Check if JSON is complete
		if m.jsonDepth == 0 {
			// JSON is complete, process the accumulated buffer
			completeLine := m.jsonBuffer.String()
			m.inJSONObject = false
			m.jsonBuffer.Reset()

			// Process the complete JSON line by calling processLogLine again
			// but first set a flag to prevent re-accumulation
			wasInJSON := m.inJSONObject
			m.inJSONObject = false
			m.processLogLine(completeLine)
			m.inJSONObject = wasInJSON
			return true // This line was part of accumulation
		}

		return true // Still accumulating
	}

	return false // Not accumulating
}
