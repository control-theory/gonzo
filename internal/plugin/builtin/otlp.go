package builtin

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/control-theory/gonzo/internal/otlpreceiver"
	"github.com/control-theory/gonzo/internal/plugin"
)

// OTLPSource implements the LogSource interface for OTLP input
type OTLPSource struct {
	grpcPort   int
	httpPort   int
	receiver   *otlpreceiver.Receiver
	logChan    chan plugin.LogEntry
	ctx        context.Context
	cancel     context.CancelFunc
	metrics    plugin.Metrics
	metricsMux sync.RWMutex
}

// NewOTLPSource creates a new OTLP log source
func NewOTLPSource() plugin.LogSource {
	return &OTLPSource{
		logChan:  make(chan plugin.LogEntry, 100),
		grpcPort: 4317, // Default OTLP gRPC port
		httpPort: 4318, // Default OTLP HTTP port
	}
}

// Name returns the name of the log source
func (s *OTLPSource) Name() string {
	return "otlp"
}

// Description returns a human-readable description
func (s *OTLPSource) Description() string {
	return "OpenTelemetry Protocol receiver (gRPC and HTTP)"
}

// Configure sets up the OTLP source
func (s *OTLPSource) Configure(config map[string]interface{}) error {
	// Parse gRPC port
	if port, ok := config["grpc_port"].(int); ok {
		s.grpcPort = port
	} else if port, ok := config["grpc_port"].(float64); ok {
		s.grpcPort = int(port)
	}

	// Parse HTTP port
	if port, ok := config["http_port"].(int); ok {
		s.httpPort = port
	} else if port, ok := config["http_port"].(float64); ok {
		s.httpPort = int(port)
	}

	// Allow disabling specific protocols
	if enabled, ok := config["grpc_enabled"].(bool); ok && !enabled {
		s.grpcPort = 0
	}
	if enabled, ok := config["http_enabled"].(bool); ok && !enabled {
		s.httpPort = 0
	}

	return nil
}

// Validate checks if the configuration is valid
func (s *OTLPSource) Validate() error {
	if s.grpcPort == 0 && s.httpPort == 0 {
		return fmt.Errorf("at least one protocol (gRPC or HTTP) must be enabled")
	}

	if s.grpcPort < 0 || s.grpcPort > 65535 {
		return fmt.Errorf("invalid gRPC port: %d", s.grpcPort)
	}

	if s.httpPort < 0 || s.httpPort > 65535 {
		return fmt.Errorf("invalid HTTP port: %d", s.httpPort)
	}

	return nil
}

// Start begins the OTLP receiver
func (s *OTLPSource) Start(ctx context.Context) (<-chan plugin.LogEntry, error) {
	s.ctx, s.cancel = context.WithCancel(ctx)

	// Create and start OTLP receiver
	s.receiver = otlpreceiver.NewReceiver(s.grpcPort, s.httpPort)
	if err := s.receiver.Start(); err != nil {
		return nil, fmt.Errorf("failed to start OTLP receiver: %w", err)
	}

	// Start metrics updater
	go s.updateMetrics()

	// Start reading from OTLP receiver
	go s.readOTLP()

	s.setConnected(true)
	return s.logChan, nil
}

// Stop gracefully stops the receiver
func (s *OTLPSource) Stop() error {
	if s.cancel != nil {
		s.cancel()
	}

	if s.receiver != nil {
		s.receiver.Stop()
	}

	if s.logChan != nil {
		close(s.logChan)
	}

	return nil
}

// GetMetrics returns current metrics
func (s *OTLPSource) GetMetrics() plugin.Metrics {
	s.metricsMux.RLock()
	defer s.metricsMux.RUnlock()
	return s.metrics
}

// GetConfigSchema returns the configuration schema
func (s *OTLPSource) GetConfigSchema() plugin.ConfigSchema {
	return plugin.ConfigSchema{
		Optional: []plugin.ConfigParam{
			{
				Name:        "grpc_port",
				Type:        "int",
				Description: "Port for OTLP gRPC listener",
				Default:     4317,
			},
			{
				Name:        "http_port",
				Type:        "int",
				Description: "Port for OTLP HTTP listener",
				Default:     4318,
			},
			{
				Name:        "grpc_enabled",
				Type:        "bool",
				Description: "Enable gRPC protocol",
				Default:     true,
			},
			{
				Name:        "http_enabled",
				Type:        "bool",
				Description: "Enable HTTP protocol",
				Default:     true,
			},
		},
	}
}

// readOTLP reads logs from the OTLP receiver
func (s *OTLPSource) readOTLP() {
	defer func() {
		s.setConnected(false)
		close(s.logChan)
	}()

	// Get the channel from OTLP receiver
	otlpLineChan := s.receiver.GetLineChan()

	for {
		select {
		case <-s.ctx.Done():
			return
		case line, ok := <-otlpLineChan:
			if !ok {
				// OTLP receiver finished
				return
			}
			if line != "" {
				// OTLP lines are already in JSON format
				// Create a simple wrapper entry
				entry := plugin.LogEntry{
					Timestamp: time.Now().UnixNano(),
					Raw:       line,
					Message:   line, // Will be parsed by gonzo's OTLP analyzer
					Severity:  "INFO",
					Source: plugin.SourceInfo{
						Type:       "otlp",
						Identifier: fmt.Sprintf("grpc:%d,http:%d", s.grpcPort, s.httpPort),
					},
				}

				select {
				case s.logChan <- entry:
					s.incrementMetrics()
				case <-s.ctx.Done():
					return
				}
			}
		}
	}
}

// Helper functions for metrics

func (s *OTLPSource) incrementMetrics() {
	s.metricsMux.Lock()
	defer s.metricsMux.Unlock()
	s.metrics.TotalLogs++
}

func (s *OTLPSource) recordError(err error) {
	s.metricsMux.Lock()
	defer s.metricsMux.Unlock()
	s.metrics.Errors++
	s.metrics.LastError = err.Error()
}

func (s *OTLPSource) setConnected(connected bool) {
	s.metricsMux.Lock()
	defer s.metricsMux.Unlock()
	s.metrics.Connected = connected
}

func (s *OTLPSource) updateMetrics() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	var lastCount int64
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.metricsMux.Lock()
			currentCount := s.metrics.TotalLogs
			s.metrics.LogsPerSecond = float64(currentCount - lastCount)
			lastCount = currentCount
			s.metricsMux.Unlock()
		}
	}
}
