// Package analyzer provides log analysis capabilities.
package analyzer

import (
	"fmt"
	"strings"

	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	logspb "go.opentelemetry.io/proto/otlp/logs/v1"
)

// OTLPAnalyzer analyzes OTLP log records and extracts text for frequency analysis
type OTLPAnalyzer struct {
	textAnalyzer *TextAnalyzer
}

// AttributeAnalysisResult contains extracted attributes
type AttributeAnalysisResult struct {
	Attributes map[string]string
}

// NewOTLPAnalyzer creates a new OTLP analyzer
func NewOTLPAnalyzer() *OTLPAnalyzer {
	return &OTLPAnalyzer{
		textAnalyzer: NewTextAnalyzer(),
	}
}

// AnalyzeOTLPRecord analyzes an OTLP log record and extracts words and phrases
func (oa *OTLPAnalyzer) AnalyzeOTLPRecord(record *logspb.LogRecord) *AnalysisResult {
	// Extract text content from various parts of the OTLP record
	textContent := oa.extractTextContent(record)
	
	// Use the existing text analyzer to process the extracted content
	return oa.textAnalyzer.AnalyzeLine(textContent)
}

// AnalyzeOTLPLogsData analyzes an entire OTLP LogsData structure
// Only analyzes the actual log message bodies, not metadata
func (oa *OTLPAnalyzer) AnalyzeOTLPLogsData(logsData *logspb.LogsData) *AnalysisResult {
	combinedResult := &AnalysisResult{
		Words:   []string{},
		Phrases: []string{},
	}
	
	// Process all resource logs
	for _, resourceLog := range logsData.ResourceLogs {
		// Process all scope logs within each resource
		for _, scopeLog := range resourceLog.ScopeLogs {
			// Process all log records within each scope - only analyze the message body
			for _, logRecord := range scopeLog.LogRecords {
				recordResult := oa.AnalyzeOTLPRecord(logRecord)
				combinedResult.Words = append(combinedResult.Words, recordResult.Words...)
				combinedResult.Phrases = append(combinedResult.Phrases, recordResult.Phrases...)
			}
		}
	}
	
	return combinedResult
}

// ExtractAttributesFromOTLPRecord extracts attribute key-value pairs from an OTLP log record
func (oa *OTLPAnalyzer) ExtractAttributesFromOTLPRecord(record *logspb.LogRecord) map[string]string {
	attributes := make(map[string]string)
	
	// Extract from log record attributes
	for _, attr := range record.Attributes {
		if attr.Key != "" && attr.Value != nil {
			value := oa.extractStringFromAnyValue(attr.Value)
			if value != "" {
				attributes[attr.Key] = value
			}
		}
	}
	
	return attributes
}

// ExtractAttributesFromOTLPLogsData extracts all attribute key-value pairs from OTLP LogsData
func (oa *OTLPAnalyzer) ExtractAttributesFromOTLPLogsData(logsData *logspb.LogsData) map[string]string {
	allAttributes := make(map[string]string)
	
	// Process all resource logs
	for _, resourceLog := range logsData.ResourceLogs {
		// Extract from resource attributes
		if resourceLog.Resource != nil {
			for _, attr := range resourceLog.Resource.Attributes {
				if attr.Key != "" && attr.Value != nil {
					value := oa.extractStringFromAnyValue(attr.Value)
					if value != "" {
						allAttributes[attr.Key] = value
					}
				}
			}
		}
		
		// Process all scope logs within each resource
		for _, scopeLog := range resourceLog.ScopeLogs {
			// Process all log records within each scope
			for _, logRecord := range scopeLog.LogRecords {
				recordAttrs := oa.ExtractAttributesFromOTLPRecord(logRecord)
				for key, value := range recordAttrs {
					allAttributes[key] = value
				}
			}
		}
	}
	
	return allAttributes
}

// extractTextContent extracts only the log message body from an OTLP log record
// This excludes metadata like severity text and attributes to focus on actual log content
func (oa *OTLPAnalyzer) extractTextContent(record *logspb.LogRecord) string {
	// Only extract from body - the actual log message content
	if bodyText := oa.extractFromBody(record.Body); bodyText != "" {
		return bodyText
	}
	
	return ""
}

// extractFromBody extracts text from the log record body
func (oa *OTLPAnalyzer) extractFromBody(body *commonpb.AnyValue) string {
	if body == nil {
		return ""
	}
	
	return oa.extractFromAnyValue(body)
}

// extractFromAttributes extracts text from log record attributes

// extractFromAnyValue extracts text from an AnyValue protobuf structure
func (oa *OTLPAnalyzer) extractFromAnyValue(value *commonpb.AnyValue) string {
	if value == nil {
		return ""
	}
	
	switch v := value.Value.(type) {
	case *commonpb.AnyValue_StringValue:
		return v.StringValue
	case *commonpb.AnyValue_IntValue:
		return fmt.Sprintf("%d", v.IntValue)
	case *commonpb.AnyValue_DoubleValue:
		return fmt.Sprintf("%f", v.DoubleValue)
	case *commonpb.AnyValue_BoolValue:
		return fmt.Sprintf("%t", v.BoolValue)
	case *commonpb.AnyValue_ArrayValue:
		return oa.extractFromArrayValue(v.ArrayValue)
	case *commonpb.AnyValue_KvlistValue:
		return oa.extractFromKVListValue(v.KvlistValue)
	case *commonpb.AnyValue_BytesValue:
		// Convert bytes to string if it's readable text
		return string(v.BytesValue)
	default:
		return ""
	}
}

// extractFromArrayValue extracts text from an ArrayValue
func (oa *OTLPAnalyzer) extractFromArrayValue(arrayValue *commonpb.ArrayValue) string {
	if arrayValue == nil || len(arrayValue.Values) == 0 {
		return ""
	}
	
	var textParts []string
	for _, value := range arrayValue.Values {
		if text := oa.extractFromAnyValue(value); text != "" {
			textParts = append(textParts, text)
		}
	}
	
	return strings.Join(textParts, " ")
}

// extractFromKVListValue extracts text from a KeyValueList
func (oa *OTLPAnalyzer) extractFromKVListValue(kvList *commonpb.KeyValueList) string {
	if kvList == nil || len(kvList.Values) == 0 {
		return ""
	}
	
	var textParts []string
	for _, kv := range kvList.Values {
		if kv.Key != "" {
			textParts = append(textParts, kv.Key)
		}
		if valueText := oa.extractFromAnyValue(kv.Value); valueText != "" {
			textParts = append(textParts, valueText)
		}
	}
	
	return strings.Join(textParts, " ")
}

// extractStringFromAnyValue extracts a string representation from an AnyValue for attribute tracking
func (oa *OTLPAnalyzer) extractStringFromAnyValue(value *commonpb.AnyValue) string {
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
	case *commonpb.AnyValue_BytesValue:
		// For bytes, only return if it's readable text (simple heuristic)
		str := string(v.BytesValue)
		if len(str) > 0 && len(str) < 100 {
			return str
		}
		return ""
	default:
		// For complex types like arrays and kvlists, we skip them for attribute tracking
		// to keep the attribute values simple and meaningful
		return ""
	}
}