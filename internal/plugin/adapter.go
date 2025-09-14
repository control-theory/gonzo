// Package plugin provides the plugin system for log sources.
package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"
)

// Adapter wraps a LogSource plugin to work with gonzo's existing architecture
type Adapter struct {
	source      LogSource
	ctx         context.Context
	cancel      context.CancelFunc
	lineChan    chan string
	wg          sync.WaitGroup
	convertOTLP bool
}

// NewAdapter creates a new adapter for a LogSource plugin
func NewAdapter(source LogSource, convertToOTLP bool) *Adapter {
	return &Adapter{
		source:      source,
		lineChan:    make(chan string, 100),
		convertOTLP: convertToOTLP,
	}
}

// Start begins streaming logs from the plugin
func (a *Adapter) Start() error {
	a.ctx, a.cancel = context.WithCancel(context.Background())

	// Start the plugin source
	logEntryChan, err := a.source.Start(a.ctx)
	if err != nil {
		return fmt.Errorf("failed to start %s source: %w", a.source.Name(), err)
	}

	// Start converter goroutine
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		defer close(a.lineChan)

		for {
			select {
			case <-a.ctx.Done():
				return
			case entry, ok := <-logEntryChan:
				if !ok {
					return
				}

				// Convert LogEntry to string (JSON or OTLP format)
				var line string
				if a.convertOTLP {
					line = a.convertToOTLP(entry)
				} else {
					line = a.convertToJSON(entry)
				}

				select {
				case a.lineChan <- line:
				case <-a.ctx.Done():
					return
				}
			}
		}
	}()

	return nil
}

// Stop stops the adapter and underlying source
func (a *Adapter) Stop() {
	if a.cancel != nil {
		a.cancel()
	}

	if a.source != nil {
		_ = a.source.Stop()
	}

	a.wg.Wait()
}

// GetLineChan returns the channel for receiving log lines
func (a *Adapter) GetLineChan() <-chan string {
	return a.lineChan
}

// GetMetrics returns metrics from the underlying source
func (a *Adapter) GetMetrics() Metrics {
	return a.source.GetMetrics()
}

// convertToJSON converts a LogEntry to JSON format
func (a *Adapter) convertToJSON(entry LogEntry) string {
	// Create a JSON structure that gonzo's existing parsers can understand
	jsonMap := map[string]interface{}{
		"timestamp": entry.Timestamp,
		"message":   entry.Message,
		"severity":  entry.Severity,
		"raw":       entry.Raw,
	}

	// Add attributes
	if len(entry.Attributes) > 0 {
		jsonMap["attributes"] = entry.Attributes
	}

	// Add resource info
	if len(entry.Resource) > 0 {
		jsonMap["resource"] = entry.Resource
	}

	// Add trace context if available
	if entry.TraceID != "" {
		jsonMap["trace_id"] = entry.TraceID
	}
	if entry.SpanID != "" {
		jsonMap["span_id"] = entry.SpanID
	}

	// Add source metadata
	jsonMap["_source"] = map[string]interface{}{
		"type":       entry.Source.Type,
		"identifier": entry.Source.Identifier,
	}

	jsonBytes, _ := json.Marshal(jsonMap)
	return string(jsonBytes)
}

// convertToOTLP converts a LogEntry to OTLP JSON format
func (a *Adapter) convertToOTLP(entry LogEntry) string {
	// Build OTLP-compatible JSON structure
	otlpMap := map[string]interface{}{
		"timeUnixNano":   fmt.Sprintf("%d", entry.Timestamp),
		"severityText":   entry.Severity,
		"severityNumber": a.severityToNumber(entry.Severity),
		"body": map[string]interface{}{
			"stringValue": entry.Message,
		},
	}

	// Convert attributes to OTLP format
	if len(entry.Attributes) > 0 || len(entry.Resource) > 0 {
		attributes := make([]map[string]interface{}, 0)

		// Add resource attributes
		for k, v := range entry.Resource {
			attributes = append(attributes, map[string]interface{}{
				"key": k,
				"value": map[string]interface{}{
					"stringValue": v,
				},
			})
		}

		// Add regular attributes
		for k, v := range entry.Attributes {
			attributes = append(attributes, map[string]interface{}{
				"key": k,
				"value": map[string]interface{}{
					"stringValue": v,
				},
			})
		}

		otlpMap["attributes"] = attributes
	}

	// Add trace context if available
	if entry.TraceID != "" {
		otlpMap["traceId"] = entry.TraceID
	}
	if entry.SpanID != "" {
		otlpMap["spanId"] = entry.SpanID
	}

	jsonBytes, _ := json.Marshal(otlpMap)
	return string(jsonBytes)
}

// severityToNumber converts severity text to OTLP severity number
func (a *Adapter) severityToNumber(severity string) int {
	switch severity {
	case "TRACE":
		return 1
	case "DEBUG":
		return 5
	case "INFO":
		return 9
	case "WARN", "WARNING":
		return 13
	case "ERROR":
		return 17
	case "FATAL":
		return 21
	default:
		return 0 // UNSPECIFIED
	}
}

// Manager manages multiple plugin sources
type Manager struct {
	registry *Registry
	adapters map[string]*Adapter
	mu       sync.RWMutex
}

// NewManager creates a new plugin manager
func NewManager() *Manager {
	return &Manager{
		registry: NewRegistry(),
		adapters: make(map[string]*Adapter),
	}
}

// RegisterPlugin registers a plugin factory
func (m *Manager) RegisterPlugin(name string, factory Factory) error {
	return m.registry.Register(name, factory)
}

// GetPlugin gets a plugin instance by name
func (m *Manager) GetPlugin(name string) (LogSource, error) {
	return m.registry.Get(name)
}

// StartPlugin starts a plugin with the given configuration
func (m *Manager) StartPlugin(name string, config map[string]interface{}, convertToOTLP bool) (*Adapter, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if already running
	if _, exists := m.adapters[name]; exists {
		return nil, fmt.Errorf("plugin %s is already running", name)
	}

	// Get the plugin from registry
	source, err := m.registry.Get(name)
	if err != nil {
		return nil, err
	}

	// Configure the source
	if err := source.Configure(config); err != nil {
		return nil, fmt.Errorf("failed to configure %s: %w", name, err)
	}

	// Validate the configuration
	if err := source.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed for %s: %w", name, err)
	}

	// Create and start adapter
	adapter := NewAdapter(source, convertToOTLP)
	if err := adapter.Start(); err != nil {
		return nil, err
	}

	m.adapters[name] = adapter
	return adapter, nil
}

// StopPlugin stops a running plugin
func (m *Manager) StopPlugin(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	adapter, exists := m.adapters[name]
	if !exists {
		return fmt.Errorf("plugin %s is not running", name)
	}

	adapter.Stop()
	delete(m.adapters, name)
	return nil
}

// GetAdapter returns a running adapter by name
func (m *Manager) GetAdapter(name string) (*Adapter, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	adapter, exists := m.adapters[name]
	return adapter, exists
}

// ListRunning returns names of all running plugins
func (m *Manager) ListRunning() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.adapters))
	for name := range m.adapters {
		names = append(names, name)
	}
	return names
}

// GetMetrics returns metrics for all running plugins
func (m *Manager) GetMetrics() map[string]Metrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	metrics := make(map[string]Metrics)
	for name, adapter := range m.adapters {
		metrics[name] = adapter.GetMetrics()
	}
	return metrics
}

// MonitorMetrics periodically logs metrics for all plugins
func (m *Manager) MonitorMetrics(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		metrics := m.GetMetrics()
		for name, m := range metrics {
			if m.Connected {
				log.Printf("[%s] Logs: %d total, %.1f/sec", name, m.TotalLogs, m.LogsPerSecond)
			} else {
				log.Printf("[%s] Disconnected - Errors: %d, Last: %s", name, m.Errors, m.LastError)
			}
		}
	}
}
