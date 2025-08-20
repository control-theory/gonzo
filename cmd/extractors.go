package main

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/control-theory/gonzo/internal/tui"

	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	logspb "go.opentelemetry.io/proto/otlp/logs/v1"
)

// extractLogEntryFromOTLPBatch extracts a LogEntry from OTLP batch data (DEPRECATED - only gets first log)
// Use extractAllLogEntriesFromOTLPBatch instead for complete batch processing
func extractLogEntryFromOTLPBatch(logsData *logspb.LogsData) *tui.LogEntry {
	entries := extractAllLogEntriesFromOTLPBatch(logsData)
	if len(entries) > 0 {
		return entries[0]
	}
	return nil
}

// extractAllLogEntriesFromOTLPBatch extracts ALL LogEntries from OTLP batch data
func extractAllLogEntriesFromOTLPBatch(logsData *logspb.LogsData) []*tui.LogEntry {
	var allEntries []*tui.LogEntry

	// Process all resource logs
	for _, resourceLog := range logsData.ResourceLogs {
		// Extract resource attributes first
		resourceAttributes := make(map[string]string)
		if resourceLog.Resource != nil {
			for _, attr := range resourceLog.Resource.Attributes {
				if attr.Key != "" && attr.Value != nil {
					resourceAttributes[attr.Key] = extractStringFromAnyValue(attr.Value)
				}
			}
		}

		// Process all scope logs within each resource
		for _, scopeLog := range resourceLog.ScopeLogs {
			// Process all log records within each scope
			for _, logRecord := range scopeLog.LogRecords {
				entry := extractLogEntryFromOTLPRecordWithResource(logRecord, resourceAttributes)
				if entry != nil {
					allEntries = append(allEntries, entry)
				}
			}
		}
	}

	return allEntries
}

// extractLogEntryFromOTLPRecordWithResource extracts a LogEntry from OTLP record with resource attributes
func extractLogEntryFromOTLPRecordWithResource(record *logspb.LogRecord, resourceAttributes map[string]string) *tui.LogEntry {
	// Use receive time for all processing
	receiveTime := time.Now()
	
	// Extract original timestamp if available
	origTimestamp := time.Time{}
	if record.TimeUnixNano > 0 {
		origTimestamp = time.Unix(0, int64(record.TimeUnixNano))
	}

	// Extract severity with proper fallback priority:
	// 1. Use SeverityText if present
	// 2. Use SeverityNumber if specified (not UNSPECIFIED)
	// 3. Fall back to parsing message text for severity keywords
	var severity string
	
	if record.SeverityText != "" {
		// Priority 1: Use SeverityText if present
		severity = record.SeverityText
	} else if record.SeverityNumber != logspb.SeverityNumber_SEVERITY_NUMBER_UNSPECIFIED {
		// Priority 2: Use SeverityNumber if it's specified
		severity = severityNumberToString(record.SeverityNumber)
	} else {
		// Priority 3: Fall back to parsing message text
		message := extractMessageFromBody(record.Body)
		severity = extractSeverityFromText(message)
	}

	// Extract message from body
	message := extractMessageFromBody(record.Body)

	// Merge resource attributes with record attributes (record attributes take precedence)
	attributes := make(map[string]string)
	
	// First add resource attributes
	for key, value := range resourceAttributes {
		attributes[key] = value
	}
	
	// Then add/override with record attributes
	for _, attr := range record.Attributes {
		if attr.Key != "" && attr.Value != nil {
			attributes[attr.Key] = extractStringFromAnyValue(attr.Value)
		}
	}

	return &tui.LogEntry{
		Timestamp:     receiveTime,
		OrigTimestamp: origTimestamp,
		Severity:      normalizeSeverity(severity),
		Message:       message,
		RawLine:       message,
		Attributes:    attributes,
	}
}

// extractLogEntryFromOTLPRecord extracts a LogEntry from a single OTLP record
func extractLogEntryFromOTLPRecord(record *logspb.LogRecord) *tui.LogEntry {
	// Use the new function with empty resource attributes for backwards compatibility
	return extractLogEntryFromOTLPRecordWithResource(record, make(map[string]string))
}

// createFallbackLogEntry creates a basic LogEntry for unparseable lines
func createFallbackLogEntry(line string) *tui.LogEntry {
	// Replace tabs with spaces to prevent formatting issues
	cleanLine := strings.ReplaceAll(line, "\t", " ")
	
	// Extract severity from the message text instead of defaulting to INFO
	severity := extractSeverityFromText(cleanLine)
	
	// Use receive time for all processing
	receiveTime := time.Now()
	
	return &tui.LogEntry{
		Timestamp:     receiveTime,
		OrigTimestamp: time.Time{}, // No original timestamp available for fallback
		Severity:      normalizeSeverity(severity),
		Message:       cleanLine,
		RawLine:       cleanLine,
		Attributes:    make(map[string]string),
	}
}

// normalizeSeverity converts various severity level formats to consistent all caps short forms
func normalizeSeverity(severity string) string {
	// Convert to uppercase for consistent matching
	normalized := strings.ToUpper(strings.TrimSpace(severity))
	
	// Map various formats to standard all caps short forms
	switch normalized {
	case "TRACE", "TRAC", "TRC":
		return "TRACE"
	case "DEBUG", "DEBU", "DBG", "DEB":
		return "DEBUG"
	case "INFO", "INFORMATION", "INF":
		return "INFO"
	case "WARN", "WARNING", "WRNG", "WRN":
		return "WARN"
	case "ERROR", "ERR", "ERRO":
		return "ERROR"
	case "FATAL", "FATL", "FTL", "CRITICAL", "CRIT", "CRT":
		return "FATAL"
	case "PANIC", "PNC":
		return "FATAL"
	default:
		// If unrecognized, try to extract first few characters
		if len(normalized) >= 4 {
			prefix := normalized[:4]
			switch prefix {
			case "INFO":
				return "INFO"
			case "WARN":
				return "WARN"
			case "ERRO":
				return "ERROR"
			case "DEBU":
				return "DEBUG"
			case "TRAC":
				return "TRACE"
			case "FATA", "CRIT":
				return "FATAL"
			}
		}
		// Default to INFO if we can't determine the severity
		return "INFO"
	}
}

// severityNumberToString converts OTLP severity number to string
func severityNumberToString(severityNumber logspb.SeverityNumber) string {
	switch severityNumber {
	case logspb.SeverityNumber_SEVERITY_NUMBER_TRACE:
		return "TRACE"
	case logspb.SeverityNumber_SEVERITY_NUMBER_DEBUG:
		return "DEBUG"
	case logspb.SeverityNumber_SEVERITY_NUMBER_INFO:
		return "INFO"
	case logspb.SeverityNumber_SEVERITY_NUMBER_WARN:
		return "WARN"
	case logspb.SeverityNumber_SEVERITY_NUMBER_ERROR:
		return "ERROR"
	case logspb.SeverityNumber_SEVERITY_NUMBER_FATAL:
		return "FATAL"
	default:
		return "INFO"
	}
}

// extractMessageFromBody extracts string message from OTLP AnyValue body
func extractMessageFromBody(body *commonpb.AnyValue) string {
	if body == nil {
		return ""
	}

	switch v := body.Value.(type) {
	case *commonpb.AnyValue_StringValue:
		// Replace tabs with spaces to prevent formatting issues
		return strings.ReplaceAll(v.StringValue, "\t", " ")
	case *commonpb.AnyValue_IntValue:
		return fmt.Sprintf("%d", v.IntValue)
	case *commonpb.AnyValue_DoubleValue:
		return fmt.Sprintf("%.2f", v.DoubleValue)
	case *commonpb.AnyValue_BoolValue:
		return fmt.Sprintf("%t", v.BoolValue)
	default:
		return fmt.Sprintf("%v", body)
	}
}

// extractStringFromAnyValue extracts string representation from OTLP AnyValue
func extractStringFromAnyValue(value *commonpb.AnyValue) string {
	if value == nil {
		return ""
	}

	switch v := value.Value.(type) {
	case *commonpb.AnyValue_StringValue:
		return v.StringValue
	case *commonpb.AnyValue_IntValue:
		return fmt.Sprintf("%d", v.IntValue)
	case *commonpb.AnyValue_DoubleValue:
		return fmt.Sprintf("%.2f", v.DoubleValue)
	case *commonpb.AnyValue_BoolValue:
		return fmt.Sprintf("%t", v.BoolValue)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// convertLogEntryToOTLPRecord converts a LogEntry back to OTLP LogRecord for analysis
func convertLogEntryToOTLPRecord(entry *tui.LogEntry) *logspb.LogRecord {
	// Convert severity string back to OTLP severity number
	var severityNumber logspb.SeverityNumber
	switch strings.ToUpper(entry.Severity) {
	case "TRACE":
		severityNumber = logspb.SeverityNumber_SEVERITY_NUMBER_TRACE
	case "DEBUG":
		severityNumber = logspb.SeverityNumber_SEVERITY_NUMBER_DEBUG
	case "INFO":
		severityNumber = logspb.SeverityNumber_SEVERITY_NUMBER_INFO
	case "WARN":
		severityNumber = logspb.SeverityNumber_SEVERITY_NUMBER_WARN
	case "ERROR":
		severityNumber = logspb.SeverityNumber_SEVERITY_NUMBER_ERROR
	case "FATAL":
		severityNumber = logspb.SeverityNumber_SEVERITY_NUMBER_FATAL
	default:
		severityNumber = logspb.SeverityNumber_SEVERITY_NUMBER_INFO
	}

	// Convert attributes back to OTLP KeyValue format
	var attributes []*commonpb.KeyValue
	for key, value := range entry.Attributes {
		attributes = append(attributes, &commonpb.KeyValue{
			Key: key,
			Value: &commonpb.AnyValue{
				Value: &commonpb.AnyValue_StringValue{StringValue: value},
			},
		})
	}

	return &logspb.LogRecord{
		TimeUnixNano:   uint64(entry.Timestamp.UnixNano()),
		SeverityNumber: severityNumber,
		SeverityText:   entry.Severity,
		Body: &commonpb.AnyValue{
			Value: &commonpb.AnyValue_StringValue{StringValue: entry.Message},
		},
		Attributes: attributes,
	}
}

// Regular expression for extracting severity levels from text
var severityRegex = regexp.MustCompile(`(?i)\b(TRACE|DEBUG|INFO|WARN|WARNING|ERROR|FATAL|CRITICAL)\b`)

// extractSeverityFromText extracts severity level from log message text
func extractSeverityFromText(message string) string {
	matches := severityRegex.FindStringSubmatch(message)
	if len(matches) > 1 {
		severity := strings.ToUpper(matches[1])
		// Normalize some variants
		switch severity {
		case "WARNING":
			return "WARN"
		case "CRITICAL":
			return "FATAL"
		default:
			return severity
		}
	}
	// Default to INFO if no severity found in text
	return "INFO"
}