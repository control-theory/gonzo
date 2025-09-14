package timestamp

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	// timestampRegex matches various timestamp formats
	// Supports ISO 8601, RFC formats, syslog, and common log formats
	// Including support for comma decimal separators (international format)
	timestampRegex = regexp.MustCompile(`(\d{4}-\d{2}-\d{2}[T\s]\d{2}:\d{2}:\d{2}(?:[,.]\d{3,9})?(?:Z|[+-]\d{2}:?\d{2})?|\w{3}\s+\d{1,2}\s+\d{2}:\d{2}:\d{2}(?:[,.]\d{3,6})?|\[\d{4}-\d{2}-\d{2}\s+\d{2}:\d{2}:\d{2}(?:[,.]\d{3,6})?\]|\d{2}:\d{2}:\d{2}(?:[,.]\d{3,6})?)`)

	// commaRegex for comma to dot conversion in fractional seconds
	commaRegex = regexp.MustCompile(`,(\d+)`)

	// severityRegex for extracting severity levels from log lines
	severityRegex = regexp.MustCompile(`^(?:\s*\[)?(TRACE|DEBUG|INFO|WARN|WARNING|ERROR|FATAL|CRITICAL)(?:\])?\s*[:>-]?\s*(.*)$`)
)

// Parser handles timestamp detection and parsing from log lines
type Parser struct {
	// Compiled layouts for fast parsing
	layouts []string
}

// ParseResult contains the parsed timestamp and remaining text
type ParseResult struct {
	Timestamp time.Time
	Found     bool
	Remaining string // Text with timestamp removed (for log message extraction)
}

// NewParser creates a new timestamp parser
func NewParser() *Parser {
	return &Parser{
		// Ordered list of timestamp layouts for parsing
		// Most common formats first for better performance
		layouts: []string{
			// ISO 8601 and RFC3339 variants
			time.RFC3339Nano,
			time.RFC3339,
			"2006-01-02T15:04:05.000Z",
			"2006-01-02T15:04:05Z",
			"2006-01-02T15:04:05.000000Z",
			"2006-01-02T15:04:05.000000000Z",

			// Standard log formats with space separator
			"2006-01-02 15:04:05.000000000",
			"2006-01-02 15:04:05.000000",
			"2006-01-02 15:04:05.000",
			"2006-01-02 15:04:05",

			// International formats with comma decimal separator
			"2006-01-02 15:04:05,000000000",
			"2006-01-02 15:04:05,000000",
			"2006-01-02 15:04:05,000",
			"2006-01-02T15:04:05,000Z",
			"2006-01-02T15:04:05,000000Z",
			"2006-01-02T15:04:05,000000000Z",

			// Timezone aware formats
			"2006-01-02 15:04:05.000 -07:00",
			"2006-01-02 15:04:05.000000 -07:00",
			"2006-01-02 15:04:05 -07:00",
			"2006-01-02T15:04:05.000-07:00",
			"2006-01-02T15:04:05.000000-07:00",
			"2006-01-02T15:04:05-07:00",

			// Syslog format (month name)
			"Jan 02 15:04:05.000000",
			"Jan 02 15:04:05.000",
			"Jan 02 15:04:05",
			"Jan _2 15:04:05.000000", // Single digit day
			"Jan _2 15:04:05.000",
			"Jan _2 15:04:05",

			// Bracketed timestamps (common in some logs)
			"[2006-01-02 15:04:05.000000]",
			"[2006-01-02 15:04:05.000]",
			"[2006-01-02 15:04:05]",
			"[2006-01-02T15:04:05.000Z]",
			"[2006-01-02T15:04:05Z]",

			// Time-only formats (useful for within-day log analysis)
			"15:04:05.000000000",
			"15:04:05.000000",
			"15:04:05.000",
			"15:04:05",
			"15:04:05,000000000", // International comma format
			"15:04:05,000000",
			"15:04:05,000",
		},
	}
}

// ParseFromText extracts and parses the first timestamp found in text
func (p *Parser) ParseFromText(text string) ParseResult {
	matches := timestampRegex.FindStringSubmatch(text)
	if len(matches) < 2 {
		return ParseResult{Found: false, Remaining: text}
	}

	timestampStr := matches[1]

	// Try parsing with each layout
	for _, layout := range p.layouts {
		// Handle international comma decimal separator
		normalizedTimestamp := p.normalizeDecimalSeparator(timestampStr, layout)

		if t, err := time.Parse(layout, normalizedTimestamp); err == nil {
			// Remove the timestamp from the original text
			remaining := strings.Replace(text, timestampStr, "", 1)
			remaining = strings.TrimSpace(remaining)

			return ParseResult{
				Timestamp: t,
				Found:     true,
				Remaining: remaining,
			}
		}
	}

	return ParseResult{Found: false, Remaining: text}
}

// ParseTimestamp attempts to parse a timestamp from various string and numeric formats
func (p *Parser) ParseTimestamp(value any) (time.Time, bool) {
	switch v := value.(type) {
	case string:
		if v == "" {
			return time.Time{}, false
		}

		// Try parsing with layouts
		for _, layout := range p.layouts {
			normalizedValue := p.normalizeDecimalSeparator(v, layout)
			if t, err := time.Parse(layout, normalizedValue); err == nil {
				return t, true
			}
		}

		// Try parsing as Unix timestamp (string format)
		if unixTime, err := strconv.ParseFloat(v, 64); err == nil {
			return p.parseUnixTimestamp(unixTime), true
		}

	case float64:
		return p.parseUnixTimestamp(v), true

	case int64:
		return p.parseUnixTimestamp(float64(v)), true

	case int:
		return p.parseUnixTimestamp(float64(v)), true
	}

	return time.Time{}, false
}

// ParseTimestampToNano parses timestamp and returns nanoseconds (for OTLP compatibility)
func (p *Parser) ParseTimestampToNano(value any) (uint64, bool) {
	if t, ok := p.ParseTimestamp(value); ok {
		return uint64(t.UnixNano()), true
	}
	return 0, false
}

// normalizeDecimalSeparator converts comma decimal separators to dots for Go time parsing
func (p *Parser) normalizeDecimalSeparator(timestamp, layout string) string {
	// Only normalize if the layout expects a dot but timestamp has a comma
	if strings.Contains(layout, ".") && strings.Contains(timestamp, ",") {
		// Find the position where fractional seconds would be
		// Look for comma followed by digits
		return commaRegex.ReplaceAllString(timestamp, ".$1")
	}
	return timestamp
}

// parseUnixTimestamp handles Unix timestamps in various scales
func (p *Parser) parseUnixTimestamp(unixTime float64) time.Time {
	// Determine the scale based on the magnitude
	if unixTime > 1e15 { // Nanoseconds (> year 2001 in nanoseconds)
		return time.Unix(0, int64(unixTime))
	} else if unixTime > 1e12 { // Microseconds (> year 2001 in microseconds)
		return time.Unix(0, int64(unixTime*1e3))
	} else if unixTime > 1e9 { // Milliseconds (> year 2001 in milliseconds)
		return time.Unix(0, int64(unixTime*1e6))
	} else { // Seconds
		return time.Unix(int64(unixTime), 0)
	}
}

// ExtractLogMessage extracts the log message from a full log line by removing timestamp and severity
func (p *Parser) ExtractLogMessage(line string) string {
	// First, try to remove timestamp using our unified parsing logic
	result := p.ParseFromText(line)
	workingLine := line
	if result.Found {
		workingLine = result.Remaining
	}

	// Now remove severity levels from the remaining text
	if matches := severityRegex.FindStringSubmatch(workingLine); len(matches) > 2 {
		message := strings.TrimSpace(matches[2])
		if message != "" {
			return message
		}
	}

	// If no severity pattern matches, return the working line (after timestamp removal)
	return strings.TrimSpace(workingLine)
}