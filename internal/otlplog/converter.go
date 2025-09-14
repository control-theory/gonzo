package otlplog

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/control-theory/gonzo/internal/timestamp"
	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	logspb "go.opentelemetry.io/proto/otlp/logs/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

// LogConverter converts various log formats to OTLP format
type LogConverter struct {
	timestampParser *timestamp.Parser
	levelRegex      *regexp.Regexp
	jsonRegex       *regexp.Regexp
	jsonMarshaler   protojson.MarshalOptions
	jsonUnmarshaler protojson.UnmarshalOptions
	customFormatName string
	customParser    interface{} // Will hold *formats.Parser when needed
}

// NewLogConverter creates a new log converter
func NewLogConverter() *LogConverter {
	return NewLogConverterWithFormat("", nil)
}

// NewLogConverterWithFormat creates a new log converter with optional custom format
func NewLogConverterWithFormat(formatName string, parser interface{}) *LogConverter {
	return &LogConverter{
		timestampParser: timestamp.NewParser(),
		// Common log level patterns
		levelRegex: regexp.MustCompile(`(?i)\b(TRACE|DEBUG|INFO|WARN|WARNING|ERROR|FATAL|CRITICAL)\b`),
		// JSON detection
		jsonRegex: regexp.MustCompile(`^\s*\{.*\}\s*$`),
		jsonMarshaler: protojson.MarshalOptions{
			UseProtoNames:   true,
			EmitUnpopulated: false,
		},
		jsonUnmarshaler: protojson.UnmarshalOptions{
			DiscardUnknown: true,
		},
		customFormatName: formatName,
		customParser:    parser,
	}
}

// ConvertToOTLP converts a log line to OTLP format
func (lc *LogConverter) ConvertToOTLP(line string, format LogFormat) (*logspb.LogRecord, error) {
	switch format {
	case FormatOTLP:
		return lc.parseExistingOTLP(line)
	case FormatJSON:
		return lc.convertJSONToOTLP(line)
	case FormatText:
		return lc.convertTextToOTLP(line)
	case FormatCustom:
		return lc.convertCustomToOTLP(line)
	default:
		return lc.convertTextToOTLP(line) // Default to text parsing
	}
}

// parseExistingOTLP parses an existing OTLP log record
func (lc *LogConverter) parseExistingOTLP(line string) (*logspb.LogRecord, error) {
	var record logspb.LogRecord
	err := lc.jsonUnmarshaler.Unmarshal([]byte(line), &record)
	if err != nil {
		return nil, fmt.Errorf("failed to parse OTLP record: %w", err)
	}
	return &record, nil
}

// convertJSONToOTLP converts JSON logs to OTLP format
func (lc *LogConverter) convertJSONToOTLP(line string) (*logspb.LogRecord, error) {
	var jsonData map[string]any
	if err := json.Unmarshal([]byte(line), &jsonData); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Check if this is Victoria Logs format (has _msg, _stream, _time fields)
	if lc.isVictoriaLogsFormat(jsonData) {
		return lc.convertVictoriaLogsToOTLP(jsonData)
	}

	record := &logspb.LogRecord{
		TimeUnixNano:   lc.extractTimestamp(jsonData),
		SeverityNumber: lc.extractSeverityNumber(jsonData),
		SeverityText:   lc.extractSeverityText(jsonData),
		Body:           lc.extractBody(jsonData),
		Attributes:     lc.extractAttributes(jsonData),
	}

	return record, nil
}

// convertTextToOTLP converts plain text logs to OTLP format
func (lc *LogConverter) convertTextToOTLP(line string) (*logspb.LogRecord, error) {
	record := &logspb.LogRecord{
		TimeUnixNano:   lc.extractTimestampFromText(line),
		SeverityNumber: lc.extractSeverityNumberFromText(line),
		SeverityText:   lc.extractSeverityTextFromText(line),
		Body: &commonpb.AnyValue{
			Value: &commonpb.AnyValue_StringValue{
				StringValue: strings.ReplaceAll(line, "\t", " "),
			},
		},
		Attributes: []*commonpb.KeyValue{},
	}

	return record, nil
}

// extractTimestamp extracts timestamp from JSON data
func (lc *LogConverter) extractTimestamp(data map[string]any) uint64 {
	// Try common timestamp field names
	timestampFields := []string{"timestamp", "time", "@timestamp", "ts", "date"}

	for _, field := range timestampFields {
		if value, exists := data[field]; exists {
			if nanos, ok := lc.timestampParser.ParseTimestampToNano(value); ok {
				return nanos
			}
		}
	}

	// If no timestamp found, use current time
	return uint64(time.Now().UnixNano())
}

// extractTimestampFromText extracts timestamp from text
func (lc *LogConverter) extractTimestampFromText(line string) uint64 {
	result := lc.timestampParser.ParseFromText(line)
	if result.Found {
		return uint64(result.Timestamp.UnixNano())
	}

	// If no timestamp found, use current time
	return uint64(time.Now().UnixNano())
}


// isVictoriaLogsFormat checks if JSON has Victoria Logs specific fields
func (lc *LogConverter) isVictoriaLogsFormat(data map[string]any) bool {
	// Victoria Logs has specific fields: _msg, _stream, _stream_id, _time
	_, hasMsg := data["_msg"]
	_, hasStream := data["_stream"]
	_, hasTime := data["_time"]

	// If it has at least _msg or (_stream and _time), consider it Victoria Logs format
	return hasMsg || (hasStream && hasTime)
}

// convertVictoriaLogsToOTLP converts Victoria Logs JSON to OTLP format
func (lc *LogConverter) convertVictoriaLogsToOTLP(data map[string]any) (*logspb.LogRecord, error) {
	record := &logspb.LogRecord{
		TimeUnixNano:   lc.extractVictoriaLogsTimestamp(data),
		SeverityNumber: lc.extractSeverityNumber(data),
		SeverityText:   lc.extractSeverityText(data),
		Body:           lc.extractVictoriaLogsBody(data),
		Attributes:     lc.extractVictoriaLogsAttributes(data),
	}

	return record, nil
}

// extractVictoriaLogsTimestamp extracts timestamp from Victoria Logs _time field
func (lc *LogConverter) extractVictoriaLogsTimestamp(data map[string]any) uint64 {
	if value, exists := data["_time"]; exists {
		if nanos, ok := lc.timestampParser.ParseTimestampToNano(value); ok {
			return nanos
		}
	}

	// Fallback to standard timestamp extraction
	return lc.extractTimestamp(data)
}

// extractVictoriaLogsBody extracts the _msg field as the log body
func (lc *LogConverter) extractVictoriaLogsBody(data map[string]any) *commonpb.AnyValue {
	// Victoria Logs uses _msg for the log message
	if msg, exists := data["_msg"]; exists {
		return &commonpb.AnyValue{
			Value: &commonpb.AnyValue_StringValue{
				StringValue: fmt.Sprintf("%v", msg),
			},
		}
	}

	// Fallback to standard body extraction
	return lc.extractBody(data)
}

// extractVictoriaLogsAttributes extracts all fields as attributes with special handling
func (lc *LogConverter) extractVictoriaLogsAttributes(data map[string]any) []*commonpb.KeyValue {
	var attributes []*commonpb.KeyValue

	// Map k8s.node.name or kubernetes.pod_node_name to host
	hostValue := ""
	if nodeNameValue, exists := data["k8s.node.name"]; exists {
		hostValue = fmt.Sprintf("%v", nodeNameValue)
	} else if nodeNameValue, exists := data["kubernetes.pod_node_name"]; exists {
		hostValue = fmt.Sprintf("%v", nodeNameValue)
	} else if nodeNameValue, exists := data["kubernetes_pod_node_name"]; exists {
		hostValue = fmt.Sprintf("%v", nodeNameValue)
	}

	if hostValue != "" {
		attributes = append(attributes, &commonpb.KeyValue{
			Key: "host",
			Value: &commonpb.AnyValue{
				Value: &commonpb.AnyValue_StringValue{StringValue: hostValue},
			},
		})
	}

	// Define fields to exclude from attributes (they're handled separately)
	excludeFields := map[string]bool{
		"_msg":    true,                     // Used for body
		"_time":   true,                     // Used for timestamp
		"_stream": true, "_stream_id": true, // Victoria Logs metadata, not needed as attributes
		"level": true, "severity": true, "log.level": true,
		"log_level": true, "loglevel": true, // Used for severity
		"k8s.node.name": true, "kubernetes.pod_node_name": true,
		"kubernetes_pod_node_name": true, // Mapped to host
	}

	// Add all other fields as attributes
	for key, value := range data {
		if !excludeFields[key] {
			attributes = append(attributes, &commonpb.KeyValue{
				Key:   key,
				Value: lc.convertToAnyValue(value),
			})
		}
	}

	return attributes
}

// extractSeverityNumber extracts severity number from JSON
func (lc *LogConverter) extractSeverityNumber(data map[string]any) logspb.SeverityNumber {
	levelFields := []string{"level", "severity", "log_level", "loglevel"}

	for _, field := range levelFields {
		if value, exists := data[field]; exists {
			return lc.severityToNumber(value)
		}
	}

	return logspb.SeverityNumber_SEVERITY_NUMBER_UNSPECIFIED
}

// extractSeverityNumberFromText extracts severity number from text
func (lc *LogConverter) extractSeverityNumberFromText(line string) logspb.SeverityNumber {
	matches := lc.levelRegex.FindStringSubmatch(line)
	if len(matches) > 1 {
		return lc.severityToNumber(matches[1])
	}
	return logspb.SeverityNumber_SEVERITY_NUMBER_UNSPECIFIED
}

// extractSeverityText extracts severity text from JSON
func (lc *LogConverter) extractSeverityText(data map[string]any) string {
	levelFields := []string{"level", "severity", "log_level", "loglevel"}

	for _, field := range levelFields {
		if value, exists := data[field]; exists {
			if str, ok := value.(string); ok {
				return strings.ToUpper(str)
			}
		}
	}

	return ""
}

// extractSeverityTextFromText extracts severity text from text
func (lc *LogConverter) extractSeverityTextFromText(line string) string {
	matches := lc.levelRegex.FindStringSubmatch(line)
	if len(matches) > 1 {
		return strings.ToUpper(matches[1])
	}
	return ""
}

// severityToNumber converts severity text to OTLP severity number
func (lc *LogConverter) severityToNumber(value any) logspb.SeverityNumber {
	var severity string

	switch v := value.(type) {
	case string:
		severity = strings.ToUpper(v)
	case float64:
		return logspb.SeverityNumber(int32(v))
	case int:
		return logspb.SeverityNumber(int32(v))
	case int32:
		return logspb.SeverityNumber(v)
	default:
		return logspb.SeverityNumber_SEVERITY_NUMBER_UNSPECIFIED
	}

	switch severity {
	case "TRACE":
		return logspb.SeverityNumber_SEVERITY_NUMBER_TRACE
	case "DEBUG":
		return logspb.SeverityNumber_SEVERITY_NUMBER_DEBUG
	case "INFO":
		return logspb.SeverityNumber_SEVERITY_NUMBER_INFO
	case "WARN", "WARNING":
		return logspb.SeverityNumber_SEVERITY_NUMBER_WARN
	case "ERROR":
		return logspb.SeverityNumber_SEVERITY_NUMBER_ERROR
	case "FATAL", "CRITICAL":
		return logspb.SeverityNumber_SEVERITY_NUMBER_FATAL
	default:
		return logspb.SeverityNumber_SEVERITY_NUMBER_UNSPECIFIED
	}
}

// extractBody extracts the main message body from JSON
func (lc *LogConverter) extractBody(data map[string]any) *commonpb.AnyValue {
	bodyFields := []string{"message", "msg", "body", "text", "content"}

	for _, field := range bodyFields {
		if value, exists := data[field]; exists {
			return &commonpb.AnyValue{
				Value: &commonpb.AnyValue_StringValue{
					StringValue: fmt.Sprintf("%v", value),
				},
			}
		}
	}

	// If no specific body field, use the entire JSON as body
	jsonBytes, _ := json.Marshal(data)
	return &commonpb.AnyValue{
		Value: &commonpb.AnyValue_StringValue{
			StringValue: string(jsonBytes),
		},
	}
}

// extractAttributes extracts attributes from JSON, excluding body and standard fields
func (lc *LogConverter) extractAttributes(data map[string]any) []*commonpb.KeyValue {
	var attributes []*commonpb.KeyValue
	excludeFields := map[string]bool{
		"timestamp": true, "time": true, "@timestamp": true, "ts": true, "date": true,
		"level": true, "severity": true, "log_level": true, "loglevel": true,
		"message": true, "msg": true, "body": true, "text": true, "content": true,
		"attributes": true, // Exclude the attributes field itself as it gets special handling below
	}

	// First, handle nested attributes object if it exists
	if attrValue, exists := data["attributes"]; exists {
		if attrMap, ok := attrValue.(map[string]any); ok {
			for key, value := range attrMap {
				attributes = append(attributes, &commonpb.KeyValue{
					Key:   key,
					Value: lc.convertToAnyValue(value),
				})
			}
		}
	}

	// Then, handle other top-level fields as attributes
	for key, value := range data {
		if !excludeFields[key] {
			attributes = append(attributes, &commonpb.KeyValue{
				Key:   key,
				Value: lc.convertToAnyValue(value),
			})
		}
	}

	return attributes
}

// convertToAnyValue converts any to AnyValue
func (lc *LogConverter) convertToAnyValue(value any) *commonpb.AnyValue {
	switch v := value.(type) {
	case string:
		return &commonpb.AnyValue{
			Value: &commonpb.AnyValue_StringValue{StringValue: v},
		}
	case int:
		return &commonpb.AnyValue{
			Value: &commonpb.AnyValue_IntValue{IntValue: int64(v)},
		}
	case int64:
		return &commonpb.AnyValue{
			Value: &commonpb.AnyValue_IntValue{IntValue: v},
		}
	case float64:
		return &commonpb.AnyValue{
			Value: &commonpb.AnyValue_DoubleValue{DoubleValue: v},
		}
	case bool:
		return &commonpb.AnyValue{
			Value: &commonpb.AnyValue_BoolValue{BoolValue: v},
		}
	default:
		// For complex types, convert to JSON string
		if jsonBytes, err := json.Marshal(v); err == nil {
			return &commonpb.AnyValue{
				Value: &commonpb.AnyValue_StringValue{StringValue: string(jsonBytes)},
			}
		}
		return &commonpb.AnyValue{
			Value: &commonpb.AnyValue_StringValue{StringValue: fmt.Sprintf("%v", v)},
		}
	}
}
