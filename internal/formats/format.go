package formats

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"

	"gopkg.in/yaml.v3"
)

// Format represents a custom log format definition
type Format struct {
	Name        string        `yaml:"name"`
	Description string        `yaml:"description,omitempty"`
	Author      string        `yaml:"author,omitempty"`
	Type        string        `yaml:"type"` // "text", "json", "structured"
	Pattern     PatternConfig `yaml:"pattern,omitempty"`
	JSON        JSONConfig    `yaml:"json,omitempty"`
	Mapping     FieldMapping  `yaml:"mapping"`
}

// PatternConfig defines patterns for text-based log parsing
type PatternConfig struct {
	// Main pattern - can use regex groups or template syntax
	// Example: "{{.timestamp}} [{{.level}}] {{.message}}"
	// Or regex: "^(?P<timestamp>[\d\-T:\.]+)\s+\[(?P<level>\w+)\]\s+(?P<message>.*)$"
	Main string `yaml:"main"`

	// Optional regex patterns for extracting specific fields
	// These are applied after the main pattern
	Fields map[string]string `yaml:"fields,omitempty"`

	// If true, uses regex with named groups. If false, uses simple template matching
	UseRegex bool `yaml:"use_regex,omitempty"`
}

// JSONConfig defines configuration for JSON-based log parsing
type JSONConfig struct {
	// JSONPath expressions or simple field names for extracting values
	// Example: "$.timestamp" or just "timestamp" for top-level fields
	Fields map[string]string `yaml:"fields,omitempty"`

	// For nested JSON like Loki's stream format
	// Example: "streams[*].values" to extract values from array
	ArrayPath string `yaml:"array_path,omitempty"`

	// If the JSON is wrapped in an array at the root
	RootIsArray bool `yaml:"root_is_array,omitempty"`
}

// FieldMapping maps extracted fields to OTLP attributes
type FieldMapping struct {
	// Core OTLP fields
	Timestamp    FieldExtractor `yaml:"timestamp,omitempty"`
	Severity     FieldExtractor `yaml:"severity,omitempty"`
	Body         FieldExtractor `yaml:"body,omitempty"`

	// Additional attributes to extract
	Attributes map[string]FieldExtractor `yaml:"attributes,omitempty"`
}

// FieldExtractor defines how to extract a field value
type FieldExtractor struct {
	// Source field name or index (for positional extraction)
	// Can be a field name, regex group name, or array index
	Field string `yaml:"field,omitempty"`

	// Alternative: use a template to combine multiple fields
	// Example: "{{.project}} - {{.module}}: {{.message}}"
	Template string `yaml:"template,omitempty"`

	// Optional regex to extract value from the field
	// Example: "duration:\\s*(\\d+)ms" to extract duration
	Pattern string `yaml:"pattern,omitempty"`

	// Optional transformation
	Transform string `yaml:"transform,omitempty"` // "uppercase", "lowercase", "trim", etc.

	// For timestamp fields, specify the format
	// Uses Go time format strings or common formats like "rfc3339", "unix", "unix_ms", "unix_ns"
	TimeFormat string `yaml:"time_format,omitempty"`

	// Default value if field is missing
	Default string `yaml:"default,omitempty"`
}

// Parser handles parsing logs using a format definition
type Parser struct {
	Format         *Format // Exported for access by converter
	mainRegex      *regexp.Regexp
	fieldRegexes   map[string]*regexp.Regexp
	extractorRegexes map[string]*regexp.Regexp
	templates      map[string]*template.Template
}

// NewParser creates a parser for the given format
func NewParser(format *Format) (*Parser, error) {
	p := &Parser{
		Format:         format,
		fieldRegexes:   make(map[string]*regexp.Regexp),
		extractorRegexes: make(map[string]*regexp.Regexp),
		templates:      make(map[string]*template.Template),
	}

	// Compile main pattern if using regex
	if format.Type == "text" && format.Pattern.UseRegex && format.Pattern.Main != "" {
		regex, err := regexp.Compile(format.Pattern.Main)
		if err != nil {
			return nil, fmt.Errorf("invalid main pattern regex: %w", err)
		}
		p.mainRegex = regex
	}

	// Compile field patterns
	for name, pattern := range format.Pattern.Fields {
		regex, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid field pattern for %s: %w", name, err)
		}
		p.fieldRegexes[name] = regex
	}

	// Compile extractor patterns
	compileExtractorPatterns := func(e FieldExtractor, key string) error {
		if e.Pattern != "" {
			regex, err := regexp.Compile(e.Pattern)
			if err != nil {
				return fmt.Errorf("invalid extractor pattern for %s: %w", key, err)
			}
			p.extractorRegexes[key] = regex
		}
		if e.Template != "" {
			tmpl, err := template.New(key).Parse(e.Template)
			if err != nil {
				return fmt.Errorf("invalid template for %s: %w", key, err)
			}
			p.templates[key] = tmpl
		}
		return nil
	}

	// Compile patterns for core fields
	if err := compileExtractorPatterns(format.Mapping.Timestamp, "timestamp"); err != nil {
		return nil, err
	}
	if err := compileExtractorPatterns(format.Mapping.Severity, "severity"); err != nil {
		return nil, err
	}
	if err := compileExtractorPatterns(format.Mapping.Body, "body"); err != nil {
		return nil, err
	}

	// Compile patterns for attributes
	for name, extractor := range format.Mapping.Attributes {
		if err := compileExtractorPatterns(extractor, "attr_"+name); err != nil {
			return nil, err
		}
	}

	return p, nil
}

// ParseLogLine parses a log line according to the format
func (p *Parser) ParseLogLine(line string) (map[string]interface{}, error) {
	switch p.Format.Type {
	case "json":
		return p.parseJSON(line)
	case "text", "structured":
		return p.parseText(line)
	default:
		return nil, fmt.Errorf("unsupported format type: %s", p.Format.Type)
	}
}

// parseJSON parses JSON-formatted logs
func (p *Parser) parseJSON(line string) (map[string]interface{}, error) {
	var data interface{}
	if err := json.Unmarshal([]byte(line), &data); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	result := make(map[string]interface{})

	// Handle different JSON structures
	switch v := data.(type) {
	case map[string]interface{}:
		// Handle nested paths if specified
		if p.Format.JSON.ArrayPath != "" {
			// TODO: Implement JSONPath or simple path extraction
			// For now, just use the map directly
			result = v
		} else {
			result = v
		}
	case []interface{}:
		if p.Format.JSON.RootIsArray && len(v) > 0 {
			// Take first element if it's an object
			if obj, ok := v[0].(map[string]interface{}); ok {
				result = obj
			}
		}
	default:
		return nil, fmt.Errorf("unexpected JSON structure")
	}

	// Apply field mappings
	mapped := make(map[string]interface{})
	for key, path := range p.Format.JSON.Fields {
		if value := extractJSONField(result, path); value != nil {
			mapped[key] = value
		}
	}

	if len(mapped) > 0 {
		return mapped, nil
	}

	return result, nil
}

// parseText parses text-formatted logs
func (p *Parser) parseText(line string) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	if p.Format.Pattern.UseRegex && p.mainRegex != nil {
		// Use regex parsing
		matches := p.mainRegex.FindStringSubmatch(line)
		if matches == nil {
			return nil, fmt.Errorf("line does not match pattern")
		}

		// Extract named groups
		for i, name := range p.mainRegex.SubexpNames() {
			if i > 0 && i < len(matches) && name != "" {
				result[name] = matches[i]
			}
		}
	} else if p.Format.Pattern.Main != "" {
		// Use simple pattern matching
		result = p.parseSimplePattern(line, p.Format.Pattern.Main)
	} else {
		// No pattern specified, treat as raw text
		result["raw"] = line
	}

	// Apply additional field patterns
	for name, regex := range p.fieldRegexes {
		if matches := regex.FindStringSubmatch(line); len(matches) > 1 {
			result[name] = matches[1]
		}
	}

	return result, nil
}

// parseSimplePattern does simple pattern matching for formats like "[Backend] 5300 LOG [InstanceLoader] Message"
func (p *Parser) parseSimplePattern(line, pattern string) map[string]interface{} {
	result := make(map[string]interface{})

	// Simple implementation: split by common delimiters
	// This is a basic implementation that can be enhanced
	parts := strings.Fields(line)

	// Try to identify common patterns
	// Example: [Backend] 5300 LOG [InstanceLoader] TypeOrmModule dependencies initialized +6ms
	if len(parts) >= 3 {
		idx := 0

		// Check for bracketed values
		for _, part := range parts {
			if strings.HasPrefix(part, "[") && strings.HasSuffix(part, "]") {
				key := fmt.Sprintf("field%d", idx)
				result[key] = strings.Trim(part, "[]")
				idx++
			} else if regexp.MustCompile(`^\d+$`).MatchString(part) {
				result["pid"] = part
			} else if regexp.MustCompile(`(?i)^(TRACE|DEBUG|INFO|WARN|ERROR|FATAL)$`).MatchString(part) {
				result["level"] = part
			}
		}

		// Extract duration if present (e.g., +6ms)
		if matches := regexp.MustCompile(`\+(\d+)ms`).FindStringSubmatch(line); len(matches) > 1 {
			result["duration"] = matches[1]
		}

		// The rest is the message
		// This is simplified - a real implementation would be more sophisticated
		result["message"] = line
	}

	return result
}

// ExtractField extracts a field value using the configured extractor
func (p *Parser) ExtractField(data map[string]interface{}, extractor FieldExtractor, key string) interface{} {
	var value interface{}

	// Get the base value
	if extractor.Field != "" {
		value = data[extractor.Field]
	} else if extractor.Template != "" {
		// Apply template
		if tmpl, ok := p.templates[key]; ok {
			var buf strings.Builder
			if err := tmpl.Execute(&buf, data); err == nil {
				value = buf.String()
			}
		}
	}

	// Apply pattern extraction if specified
	if extractor.Pattern != "" && value != nil {
		if regex, ok := p.extractorRegexes[key]; ok {
			if matches := regex.FindStringSubmatch(fmt.Sprintf("%v", value)); len(matches) > 1 {
				value = matches[1]
			}
		}
	}

	// Apply transformation
	if extractor.Transform != "" && value != nil {
		valueStr := fmt.Sprintf("%v", value)
		switch extractor.Transform {
		case "uppercase":
			value = strings.ToUpper(valueStr)
		case "lowercase":
			value = strings.ToLower(valueStr)
		case "trim":
			value = strings.TrimSpace(valueStr)
		case "status_to_severity":
			value = httpStatusToSeverity(valueStr)
		}
	}

	// Use default if no value
	if value == nil && extractor.Default != "" {
		value = extractor.Default
	}

	return value
}

// ParseTimestamp parses a timestamp field according to the format
func (p *Parser) ParseTimestamp(value interface{}, format string) (time.Time, error) {
	if value == nil {
		return time.Now(), nil
	}

	valueStr := fmt.Sprintf("%v", value)

	switch format {
	case "", "auto":
		// Try to auto-detect format
		return parseTimestampAuto(valueStr)
	case "unix":
		if i, err := strconv.ParseInt(valueStr, 10, 64); err == nil {
			return time.Unix(i, 0), nil
		}
	case "unix_ms":
		if i, err := strconv.ParseInt(valueStr, 10, 64); err == nil {
			return time.Unix(i/1000, (i%1000)*1e6), nil
		}
	case "unix_ns":
		if i, err := strconv.ParseInt(valueStr, 10, 64); err == nil {
			return time.Unix(0, i), nil
		}
	case "rfc3339":
		return time.Parse(time.RFC3339, valueStr)
	default:
		// Use custom format
		return time.Parse(format, valueStr)
	}

	return time.Time{}, fmt.Errorf("failed to parse timestamp: %s", valueStr)
}

// parseTimestampAuto tries to automatically detect and parse timestamp format
func parseTimestampAuto(str string) (time.Time, error) {
	// Try common formats
	formats := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05.999999999Z07:00",
		"2006-01-02 15:04:05",
		"2006-01-02 15:04:05.999",
		"2006/01/02 15:04:05",
		"Jan 02 15:04:05",
		"Jan _2 15:04:05",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, str); err == nil {
			return t, nil
		}
	}

	// Try Unix timestamp
	if i, err := strconv.ParseInt(str, 10, 64); err == nil {
		// Determine if it's seconds, milliseconds, or nanoseconds
		if i < 1e10 { // Likely seconds
			return time.Unix(i, 0), nil
		} else if i < 1e13 { // Likely milliseconds
			return time.Unix(i/1000, (i%1000)*1e6), nil
		} else { // Likely nanoseconds
			return time.Unix(0, i), nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse timestamp: %s", str)
}

// extractJSONField extracts a field from JSON using a simple path
func extractJSONField(data map[string]interface{}, path string) interface{} {
	// Simple implementation - just handles direct fields and one level of nesting
	// Can be enhanced to support full JSONPath

	if strings.Contains(path, ".") {
		parts := strings.Split(path, ".")
		current := data
		for i, part := range parts {
			if i == len(parts)-1 {
				return current[part]
			}
			if next, ok := current[part].(map[string]interface{}); ok {
				current = next
			} else {
				return nil
			}
		}
	}

	return data[path]
}

// LoadFormat loads a format definition from a YAML file
func LoadFormat(path string) (*Format, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read format file: %w", err)
	}

	var format Format
	if err := yaml.Unmarshal(data, &format); err != nil {
		return nil, fmt.Errorf("failed to parse format file: %w", err)
	}

	return &format, nil
}

// LoadFormatByName loads a format by name from the formats directory
func LoadFormatByName(name string, configDir string) (*Format, error) {
	// Check for .yaml or .yml extension
	if filepath.Ext(name) == "" {
		// Try both extensions
		for _, ext := range []string{".yaml", ".yml"} {
			testName := name + ext
			if format, err := tryLoadFormat(testName, configDir); err == nil {
				return format, nil
			}
		}
		return nil, fmt.Errorf("format '%s' not found", name)
	}

	return tryLoadFormat(name, configDir)
}

// tryLoadFormat attempts to load a format from the formats directory
func tryLoadFormat(name string, configDir string) (*Format, error) {
	formatsDir := filepath.Join(configDir, "formats")
	formatPath := filepath.Join(formatsDir, name)

	// Check if file exists
	if _, err := os.Stat(formatPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("format file not found: %s", formatPath)
	}

	return LoadFormat(formatPath)
}

// ListAvailableFormats returns a list of available format names
func ListAvailableFormats(configDir string) ([]string, error) {
	formatsDir := filepath.Join(configDir, "formats")

	entries, err := os.ReadDir(formatsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}

	var formats []string
	for _, entry := range entries {
		if !entry.IsDir() {
			name := entry.Name()
			if strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml") {
				// Remove extension
				name = strings.TrimSuffix(name, filepath.Ext(name))
				formats = append(formats, name)
			}
		}
	}

	return formats, nil
}

// httpStatusToSeverity converts HTTP status codes to log severity levels
func httpStatusToSeverity(status string) string {
	// Convert string to integer for range checking
	if statusCode, err := strconv.Atoi(status); err == nil {
		switch {
		case statusCode >= 100 && statusCode < 200:
			return "DEBUG" // Informational responses
		case statusCode >= 200 && statusCode < 300:
			return "INFO" // Success responses
		case statusCode >= 300 && statusCode < 400:
			return "INFO" // Redirection responses
		case statusCode >= 400 && statusCode < 500:
			return "WARN" // Client error responses
		case statusCode >= 500 && statusCode < 600:
			return "ERROR" // Server error responses
		default:
			return "INFO" // Unknown status codes default to INFO
		}
	}

	// If we can't parse as integer, return as-is
	return status
}