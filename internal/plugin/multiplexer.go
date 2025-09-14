package plugin

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// Multiplexer combines multiple log sources into a single stream
type Multiplexer struct {
	sources   map[string]*sourceWrapper
	outChan   chan LogEntry
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	mu        sync.RWMutex
	mergeMode MergeMode
}

// sourceWrapper wraps a source with its adapter and metadata
type sourceWrapper struct {
	source  LogSource
	adapter *Adapter
	name    string
	config  map[string]interface{}
}

// MergeMode defines how logs from multiple sources are combined
type MergeMode int

const (
	// MergeModeInterleaved interleaves logs from all sources (default)
	MergeModeInterleaved MergeMode = iota
	// MergeModeRoundRobin reads from sources in round-robin fashion
	MergeModeRoundRobin
	// MergeModePriority reads from sources based on priority
	MergeModePriority
)

// NewMultiplexer creates a new log source multiplexer
func NewMultiplexer() *Multiplexer {
	return &Multiplexer{
		sources:   make(map[string]*sourceWrapper),
		outChan:   make(chan LogEntry, 1000), // Larger buffer for multiple sources
		mergeMode: MergeModeInterleaved,
	}
}

// AddSource adds a new log source to the multiplexer
func (m *Multiplexer) AddSource(name string, source LogSource, config map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if source already exists
	if _, exists := m.sources[name]; exists {
		return fmt.Errorf("source %s already exists", name)
	}

	// Configure the source
	if err := source.Configure(config); err != nil {
		return fmt.Errorf("failed to configure %s: %w", name, err)
	}

	// Validate the source
	if err := source.Validate(); err != nil {
		return fmt.Errorf("validation failed for %s: %w", name, err)
	}

	// Store the source wrapper
	m.sources[name] = &sourceWrapper{
		source: source,
		name:   name,
		config: config,
	}

	log.Printf("Added source: %s (%s)", name, source.Description())
	return nil
}

// RemoveSource removes a log source from the multiplexer
func (m *Multiplexer) RemoveSource(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	wrapper, exists := m.sources[name]
	if !exists {
		return fmt.Errorf("source %s not found", name)
	}

	// Stop the source if it's running
	if wrapper.adapter != nil {
		wrapper.adapter.Stop()
	}

	delete(m.sources, name)
	log.Printf("Removed source: %s", name)
	return nil
}

// Start begins streaming from all configured sources
func (m *Multiplexer) Start(ctx context.Context) (<-chan LogEntry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.sources) == 0 {
		return nil, fmt.Errorf("no sources configured")
	}

	m.ctx, m.cancel = context.WithCancel(ctx)

	// Start all sources
	for name, wrapper := range m.sources {
		logChan, err := wrapper.source.Start(m.ctx)
		if err != nil {
			log.Printf("Failed to start source %s: %v", name, err)
			continue
		}

		// Start a goroutine to read from this source
		m.wg.Add(1)
		go m.readFromSource(name, wrapper.source, logChan)
	}

	// Start metrics reporter
	go m.reportMetrics()

	return m.outChan, nil
}

// Stop stops all sources
func (m *Multiplexer) Stop() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.cancel != nil {
		m.cancel()
	}

	// Stop all sources
	for name, wrapper := range m.sources {
		if err := wrapper.source.Stop(); err != nil {
			log.Printf("Error stopping source %s: %v", name, err)
		}
	}

	// Wait for all goroutines to finish
	m.wg.Wait()

	// Close output channel
	close(m.outChan)

	return nil
}

// readFromSource reads from a single source and forwards to output channel
func (m *Multiplexer) readFromSource(name string, _ LogSource, logChan <-chan LogEntry) {
	defer m.wg.Done()

	for {
		select {
		case <-m.ctx.Done():
			return
		case entry, ok := <-logChan:
			if !ok {
				log.Printf("Source %s closed", name)
				return
			}

			// Add source identifier to metadata
			if entry.Source.Metadata == nil {
				entry.Source.Metadata = make(map[string]string)
			}
			entry.Source.Metadata["multiplexer_source"] = name

			// Forward to output channel
			select {
			case m.outChan <- entry:
			case <-m.ctx.Done():
				return
			default:
				// Channel full, log and drop
				log.Printf("Warning: Multiplexer channel full, dropping log from %s", name)
			}
		}
	}
}

// reportMetrics periodically reports metrics for all sources
func (m *Multiplexer) reportMetrics() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.mu.RLock()
			for name, wrapper := range m.sources {
				metrics := wrapper.source.GetMetrics()
				if metrics.Connected {
					log.Printf("[Multiplexer/%s] Logs: %d total, %.1f/sec",
						name, metrics.TotalLogs, metrics.LogsPerSecond)
				} else if metrics.LastError != "" {
					log.Printf("[Multiplexer/%s] Error: %s",
						name, metrics.LastError)
				}
			}
			m.mu.RUnlock()
		}
	}
}

// GetSources returns the list of configured sources
func (m *Multiplexer) GetSources() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sources := make([]string, 0, len(m.sources))
	for name := range m.sources {
		sources = append(sources, name)
	}
	return sources
}

// GetSourceMetrics returns metrics for a specific source
func (m *Multiplexer) GetSourceMetrics(name string) (Metrics, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	wrapper, exists := m.sources[name]
	if !exists {
		return Metrics{}, fmt.Errorf("source %s not found", name)
	}

	return wrapper.source.GetMetrics(), nil
}

// GetAllMetrics returns metrics for all sources
func (m *Multiplexer) GetAllMetrics() map[string]Metrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	metrics := make(map[string]Metrics)
	for name, wrapper := range m.sources {
		metrics[name] = wrapper.source.GetMetrics()
	}
	return metrics
}

// SetMergeMode sets how logs from multiple sources are merged
func (m *Multiplexer) SetMergeMode(mode MergeMode) {
	m.mergeMode = mode
}

// MultiplexerAdapter wraps a Multiplexer to work with the existing adapter interface
type MultiplexerAdapter struct {
	multiplexer *Multiplexer
	lineChan    chan string
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	convertOTLP bool
}

// NewMultiplexerAdapter creates an adapter for the multiplexer
func NewMultiplexerAdapter(multiplexer *Multiplexer, convertToOTLP bool) *MultiplexerAdapter {
	return &MultiplexerAdapter{
		multiplexer: multiplexer,
		lineChan:    make(chan string, 1000),
		convertOTLP: convertToOTLP,
	}
}

// Start begins streaming from all sources
func (a *MultiplexerAdapter) Start() error {
	a.ctx, a.cancel = context.WithCancel(context.Background())

	// Start the multiplexer
	logEntryChan, err := a.multiplexer.Start(a.ctx)
	if err != nil {
		return fmt.Errorf("failed to start multiplexer: %w", err)
	}

	// Start converter goroutine
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		defer close(a.lineChan)

		adapter := &Adapter{convertOTLP: a.convertOTLP}

		for {
			select {
			case <-a.ctx.Done():
				return
			case entry, ok := <-logEntryChan:
				if !ok {
					return
				}

				// Convert LogEntry to string using the same logic as regular adapter
				var line string
				if a.convertOTLP {
					line = adapter.convertToOTLP(entry)
				} else {
					line = adapter.convertToJSON(entry)
				}

				select {
				case a.lineChan <- line:
				case <-a.ctx.Done():
					return
				}
			}
		}
	}()

	sourcesCount := len(a.multiplexer.GetSources())
	log.Printf("Started multiplexer with %d sources", sourcesCount)
	return nil
}

// Stop stops the multiplexer adapter
func (a *MultiplexerAdapter) Stop() {
	if a.cancel != nil {
		a.cancel()
	}

	if a.multiplexer != nil {
		_ = a.multiplexer.Stop()
	}

	a.wg.Wait()
}

// GetLineChan returns the channel for receiving log lines
func (a *MultiplexerAdapter) GetLineChan() <-chan string {
	return a.lineChan
}

// GetMetrics returns combined metrics from all sources
func (a *MultiplexerAdapter) GetMetrics() Metrics {
	allMetrics := a.multiplexer.GetAllMetrics()

	// Combine metrics from all sources
	combined := Metrics{
		Connected: false,
	}

	for _, m := range allMetrics {
		combined.TotalLogs += m.TotalLogs
		combined.LogsPerSecond += m.LogsPerSecond
		combined.Errors += m.Errors
		if m.Connected {
			combined.Connected = true
		}
		if m.LastError != "" {
			combined.LastError = m.LastError
		}
	}

	return combined
}
