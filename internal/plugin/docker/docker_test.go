package docker

import (
	"testing"
)

func TestDetectSeverity(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Explicit level fields - these should take priority
		{
			name:     "explicit level=info",
			input:    "2024-01-15 10:30:45 level=info message='Application started successfully'",
			expected: "INFO",
		},
		{
			name:     "explicit level=error",
			input:    "level=error msg='Database connection failed' timestamp=1234567890",
			expected: "ERROR",
		},
		{
			name:     "explicit level=warn",
			input:    "timestamp=2024-01-15T10:30:45Z level=warn message='High memory usage detected'",
			expected: "WARN",
		},
		{
			name:     "explicit level=debug",
			input:    "level=debug Processing request with parameters: foo=bar",
			expected: "DEBUG",
		},
		{
			name:     "JSON format with level",
			input:    `{"timestamp":"2024-01-15T10:30:45Z","level":"error","message":"Failed to process request"}`,
			expected: "ERROR",
		},
		{
			name:     "JSON format with spaces",
			input:    `{"timestamp": "2024-01-15T10:30:45Z", "level" : "warning", "message": "Deprecated API usage"}`,
			expected: "WARN",
		},
		{
			name:     "YAML-style level",
			input:    "level: info message: Starting application",
			expected: "INFO",
		},
		{
			name:     "explicit level=fatal",
			input:    "level=fatal System crash imminent",
			expected: "ERROR",
		},
		{
			name:     "explicit level=panic",
			input:    "level=panic Unrecoverable error occurred",
			expected: "ERROR",
		},
		{
			name:     "explicit level=trace",
			input:    "level=trace Entering function processRequest()",
			expected: "DEBUG",
		},
		{
			name:     "unknown explicit level",
			input:    "level=custom This is a custom level",
			expected: "CUSTOM",
		},
		// Fallback to keyword detection when no explicit level
		{
			name:     "error keyword without explicit level",
			input:    "Connection error: Unable to reach database",
			expected: "ERROR",
		},
		{
			name:     "warning keyword without explicit level",
			input:    "Warning: Disk space is running low",
			expected: "WARN",
		},
		{
			name:     "info keyword without explicit level",
			input:    "Info: Server started on port 8080",
			expected: "INFO",
		},
		{
			name:     "debug keyword without explicit level",
			input:    "Debug output: variable x = 42",
			expected: "DEBUG",
		},
		// Mixed cases - explicit level should win
		{
			name:     "explicit level wins over keyword",
			input:    "level=info ERROR: This is actually an info message about errors",
			expected: "INFO",
		},
		{
			name:     "explicit debug wins over error keyword",
			input:    "level=debug Checking for errors in configuration",
			expected: "DEBUG",
		},
		// Edge cases
		{
			name:     "no level or keywords",
			input:    "Application is running normally",
			expected: "INFO",
		},
		{
			name:     "partial word shouldn't match",
			input:    "The debugger is not available",
			expected: "INFO",
		},
		{
			name:     "word boundary test - error",
			input:    "An error occurred while processing",
			expected: "ERROR",
		},
		{
			name:     "word boundary test - terror shouldn't match",
			input:    "The terror of the situation was evident",
			expected: "INFO",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detectSeverity(tt.input)
			if result != tt.expected {
				t.Errorf("detectSeverity(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}