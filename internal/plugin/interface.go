package plugin

import (
	"context"
	"fmt"
)

// LogSource represents a plugin interface for streaming logs from various sources
type LogSource interface {
	// Name returns the name of the log source (e.g., "loki", "elasticsearch", "cloudwatch")
	Name() string

	// Description returns a human-readable description of the log source
	Description() string

	// Configure sets up the log source with configuration parameters
	// Config is a map of key-value pairs specific to each source
	Configure(config map[string]interface{}) error

	// Validate checks if the configuration is valid and connection can be established
	Validate() error

	// Start begins streaming logs to the provided channel
	// The implementation should handle reconnection logic internally
	Start(ctx context.Context) (<-chan LogEntry, error)

	// Stop gracefully stops the log streaming
	Stop() error

	// GetMetrics returns current metrics about the log source
	GetMetrics() Metrics
}

// LogEntry represents a single log entry from any source
// This is a common format that all plugins must convert to
type LogEntry struct {
	// Timestamp in Unix nanoseconds
	Timestamp int64 `json:"timestamp"`

	// Original log line (raw format)
	Raw string `json:"raw"`

	// Parsed message body
	Message string `json:"message"`

	// Severity/Level (ERROR, WARN, INFO, DEBUG, etc.)
	Severity string `json:"severity"`

	// Attributes/Labels as key-value pairs
	Attributes map[string]string `json:"attributes"`

	// Resource attributes (e.g., service.name, host.name)
	Resource map[string]string `json:"resource"`

	// Trace context if available
	TraceID string `json:"trace_id,omitempty"`
	SpanID  string `json:"span_id,omitempty"`

	// Source metadata
	Source SourceInfo `json:"source"`
}

// SourceInfo contains metadata about where the log came from
type SourceInfo struct {
	// Type of source (e.g., "loki", "file", "otlp")
	Type string `json:"type"`

	// Specific identifier (e.g., filename, stream name, job name)
	Identifier string `json:"identifier"`

	// Additional metadata specific to the source
	Metadata map[string]string `json:"metadata,omitempty"`
}

// Metrics contains runtime metrics for a log source
type Metrics struct {
	// Total number of logs received
	TotalLogs int64 `json:"total_logs"`

	// Number of logs received in the last interval
	LogsPerSecond float64 `json:"logs_per_second"`

	// Number of errors encountered
	Errors int64 `json:"errors"`

	// Last error message if any
	LastError string `json:"last_error,omitempty"`

	// Connection status
	Connected bool `json:"connected"`

	// Lag/delay if applicable (in milliseconds)
	LagMs int64 `json:"lag_ms,omitempty"`
}

// Factory is a function that creates a new instance of a LogSource
type Factory func() LogSource

// Registry manages available log source plugins
type Registry struct {
	sources map[string]Factory
}

// NewRegistry creates a new plugin registry
func NewRegistry() *Registry {
	return &Registry{
		sources: make(map[string]Factory),
	}
}

// Register adds a new log source plugin to the registry
func (r *Registry) Register(name string, factory Factory) error {
	if _, exists := r.sources[name]; exists {
		return fmt.Errorf("plugin %s already registered", name)
	}
	r.sources[name] = factory
	return nil
}

// Get retrieves a log source plugin by name
func (r *Registry) Get(name string) (LogSource, error) {
	factory, exists := r.sources[name]
	if !exists {
		return nil, fmt.Errorf("plugin %s not found", name)
	}
	return factory(), nil
}

// List returns all available plugin names
func (r *Registry) List() []string {
	names := make([]string, 0, len(r.sources))
	for name := range r.sources {
		names = append(names, name)
	}
	return names
}

// ConfigValidator is an optional interface that plugins can implement
// to provide configuration schema validation
type ConfigValidator interface {
	// GetConfigSchema returns the expected configuration schema
	GetConfigSchema() ConfigSchema
}

// ConfigSchema describes the configuration parameters for a plugin
type ConfigSchema struct {
	// Required parameters
	Required []ConfigParam `json:"required"`

	// Optional parameters
	Optional []ConfigParam `json:"optional"`
}

// ConfigParam describes a single configuration parameter
type ConfigParam struct {
	// Name of the parameter
	Name string `json:"name"`

	// Type of the parameter (string, int, bool, etc.)
	Type string `json:"type"`

	// Description of what this parameter does
	Description string `json:"description"`

	// Default value if any
	Default interface{} `json:"default,omitempty"`

	// Example value
	Example interface{} `json:"example,omitempty"`
}

// Reconnectable is an optional interface for sources that support reconnection
type Reconnectable interface {
	// SetReconnectPolicy configures the reconnection behavior
	SetReconnectPolicy(policy ReconnectPolicy)
}

// ReconnectPolicy defines how a source should handle reconnections
type ReconnectPolicy struct {
	// Maximum number of reconnection attempts (0 = infinite)
	MaxAttempts int

	// Initial delay between reconnection attempts
	InitialDelay int

	// Maximum delay between reconnection attempts
	MaxDelay int

	// Backoff multiplier for exponential backoff
	BackoffMultiplier float64
}

// Filterable is an optional interface for sources that support server-side filtering
type Filterable interface {
	// SetFilter applies a filter expression to the log stream
	SetFilter(filter string) error

	// GetFilterSyntax returns documentation about the filter syntax
	GetFilterSyntax() string
}
