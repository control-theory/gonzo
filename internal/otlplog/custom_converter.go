package otlplog

import (
	"fmt"
	"time"

	"github.com/control-theory/gonzo/internal/formats"
	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	logspb "go.opentelemetry.io/proto/otlp/logs/v1"
)

// convertCustomToOTLP converts logs using a custom format parser
func (lc *LogConverter) convertCustomToOTLP(line string) (*logspb.LogRecord, error) {
	if lc.customParser == nil {
		return nil, fmt.Errorf("no custom parser configured")
	}

	parser, ok := lc.customParser.(*formats.Parser)
	if !ok {
		return nil, fmt.Errorf("invalid custom parser type")
	}

	// Parse the line using the custom format
	data, err := parser.ParseLogLine(line)
	if err != nil {
		// If parsing fails, treat as plain text
		return lc.convertTextToOTLP(line)
	}

	// Get the format definition from the parser
	format := parser.Format
	if format == nil {
		return nil, fmt.Errorf("no format definition in parser")
	}

	// Create the OTLP record
	record := &logspb.LogRecord{
		Attributes: []*commonpb.KeyValue{},
	}

	// Extract timestamp
	if format.Mapping.Timestamp.Field != "" || format.Mapping.Timestamp.Template != "" {
		if tsValue := parser.ExtractField(data, format.Mapping.Timestamp, "timestamp"); tsValue != nil {
			ts, err := parser.ParseTimestamp(tsValue, format.Mapping.Timestamp.TimeFormat)
			if err == nil {
				record.TimeUnixNano = uint64(ts.UnixNano())
			}
		}
	}

	// Default to current time if no timestamp
	if record.TimeUnixNano == 0 {
		record.TimeUnixNano = uint64(time.Now().UnixNano())
	}

	// Extract severity
	if format.Mapping.Severity.Field != "" || format.Mapping.Severity.Template != "" {
		if sevValue := parser.ExtractField(data, format.Mapping.Severity, "severity"); sevValue != nil {
			sevStr := fmt.Sprintf("%v", sevValue)
			record.SeverityText = sevStr
			record.SeverityNumber = lc.severityToNumber(sevStr)
		}
	}

	// Extract body
	if format.Mapping.Body.Field != "" || format.Mapping.Body.Template != "" {
		if bodyValue := parser.ExtractField(data, format.Mapping.Body, "body"); bodyValue != nil {
			record.Body = &commonpb.AnyValue{
				Value: &commonpb.AnyValue_StringValue{
					StringValue: fmt.Sprintf("%v", bodyValue),
				},
			}
		}
	} else {
		// Default body to the entire line if not specified
		record.Body = &commonpb.AnyValue{
			Value: &commonpb.AnyValue_StringValue{
				StringValue: line,
			},
		}
	}

	// Extract attributes
	for name, extractor := range format.Mapping.Attributes {
		if value := parser.ExtractField(data, extractor, "attr_"+name); value != nil {
			record.Attributes = append(record.Attributes, &commonpb.KeyValue{
				Key:   name,
				Value: lc.convertToAnyValue(value),
			})
		}
	}

	// Add any remaining fields from parsed data as attributes (if not already mapped)
	mappedFields := make(map[string]bool)
	if format.Mapping.Timestamp.Field != "" {
		mappedFields[format.Mapping.Timestamp.Field] = true
	}
	if format.Mapping.Severity.Field != "" {
		mappedFields[format.Mapping.Severity.Field] = true
	}
	if format.Mapping.Body.Field != "" {
		mappedFields[format.Mapping.Body.Field] = true
	}
	for _, extractor := range format.Mapping.Attributes {
		if extractor.Field != "" {
			mappedFields[extractor.Field] = true
		}
	}

	// Add unmapped fields as attributes
	for key, value := range data {
		if !mappedFields[key] && key != "raw" {
			record.Attributes = append(record.Attributes, &commonpb.KeyValue{
				Key:   key,
				Value: lc.convertToAnyValue(value),
			})
		}
	}

	return record, nil
}