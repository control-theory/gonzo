package otlplog

import (
	"encoding/json"
	"strings"

	logspb "go.opentelemetry.io/proto/otlp/logs/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

// LogFormat represents the detected log format
type LogFormat int

const (
	FormatUnknown LogFormat = iota
	FormatOTLP
	FormatJSON
	FormatText
)

// FormatDetector detects the format of incoming log lines
type FormatDetector struct {
	otlpKeywords    []string
	jsonMarshaler   protojson.MarshalOptions
	jsonUnmarshaler protojson.UnmarshalOptions
}

// NewFormatDetector creates a new format detector
func NewFormatDetector() *FormatDetector {
	return &FormatDetector{
		otlpKeywords: []string{
			"timeUnixNano",
			"observedTimeUnixNano",
			"severityNumber",
			"severityText",
			"resourceLogs",
			"scopeLogs",
			"logRecords",
		},
		jsonMarshaler: protojson.MarshalOptions{
			UseProtoNames:   true,
			EmitUnpopulated: false,
		},
		jsonUnmarshaler: protojson.UnmarshalOptions{
			DiscardUnknown: true,
		},
	}
}

// DetectFormat analyzes a log line and returns the detected format
func (fd *FormatDetector) DetectFormat(line string) LogFormat {
	line = strings.TrimSpace(line)

	if line == "" {
		return FormatUnknown
	}

	// Try to parse as JSON first
	var jsonData map[string]interface{}
	if err := json.Unmarshal([]byte(line), &jsonData); err == nil {
		// Check if it contains OTLP-specific fields
		if fd.containsOTLPFields(jsonData) {
			return FormatOTLP
		}
		return FormatJSON
	}

	// If not JSON, assume it's plain text
	return FormatText
}

// containsOTLPFields checks if the JSON contains OTLP-specific fields
func (fd *FormatDetector) containsOTLPFields(data map[string]interface{}) bool {
	for _, keyword := range fd.otlpKeywords {
		if fd.hasKeyRecursive(data, keyword) {
			return true
		}
	}
	return false
}

// hasKeyRecursive recursively searches for a key in nested JSON
func (fd *FormatDetector) hasKeyRecursive(data interface{}, key string) bool {
	switch v := data.(type) {
	case map[string]interface{}:
		if _, exists := v[key]; exists {
			return true
		}
		for _, value := range v {
			if fd.hasKeyRecursive(value, key) {
				return true
			}
		}
	case []interface{}:
		for _, item := range v {
			if fd.hasKeyRecursive(item, key) {
				return true
			}
		}
	}
	return false
}

// IsOTLPBatch checks if the line represents an OTLP batch format
func (fd *FormatDetector) IsOTLPBatch(line string) bool {
	var logsData logspb.LogsData
	return fd.jsonUnmarshaler.Unmarshal([]byte(line), &logsData) == nil && len(logsData.ResourceLogs) > 0
}

// ParseOTLPBatch parses an OTLP batch and returns the LogsData structure
func (fd *FormatDetector) ParseOTLPBatch(line string) (*logspb.LogsData, error) {
	var logsData logspb.LogsData
	err := fd.jsonUnmarshaler.Unmarshal([]byte(line), &logsData)
	if err != nil {
		return nil, err
	}
	return &logsData, nil
}

// GetAllLogRecords extracts all log records from an OTLP LogsData structure
func (fd *FormatDetector) GetAllLogRecords(logsData *logspb.LogsData) []*logspb.LogRecord {
	var allRecords []*logspb.LogRecord

	for _, resourceLog := range logsData.ResourceLogs {
		for _, scopeLog := range resourceLog.ScopeLogs {
			allRecords = append(allRecords, scopeLog.LogRecords...)
		}
	}

	return allRecords
}

// ParseSingleOTLPRecord parses a single OTLP log record
func (fd *FormatDetector) ParseSingleOTLPRecord(line string) (*logspb.LogRecord, error) {
	var record logspb.LogRecord
	err := fd.jsonUnmarshaler.Unmarshal([]byte(line), &record)
	return &record, err
}
