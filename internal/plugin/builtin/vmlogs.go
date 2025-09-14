package builtin

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/control-theory/gonzo/internal/plugin"
	"github.com/control-theory/gonzo/internal/vmlogs"
)

// VmlogsSource implements the LogSource interface for Victoria Logs
type VmlogsSource struct {
	url        string
	user       string
	password   string
	query      string
	params     map[string]string
	receiver   *vmlogs.Receiver
	logChan    chan plugin.LogEntry
	ctx        context.Context
	cancel     context.CancelFunc
	metrics    plugin.Metrics
	metricsMux sync.RWMutex
}

// NewVmlogsSource creates a new Victoria Logs source
func NewVmlogsSource() plugin.LogSource {
	return &VmlogsSource{
		logChan: make(chan plugin.LogEntry, 100),
		query:   "*", // Default query
		params:  make(map[string]string),
	}
}

// Name returns the name of the log source
func (s *VmlogsSource) Name() string {
	return "vmlogs"
}

// Description returns a human-readable description
func (s *VmlogsSource) Description() string {
	return "Victoria Logs streaming receiver"
}

// Configure sets up the Victoria Logs source
func (s *VmlogsSource) Configure(config map[string]interface{}) error {
	// Required: URL
	if url, ok := config["url"].(string); ok {
		s.url = url
	} else {
		return fmt.Errorf("url is required")
	}

	// Optional: Authentication
	if user, ok := config["user"].(string); ok {
		s.user = user
	}
	if password, ok := config["password"].(string); ok {
		s.password = password
	}

	// Optional: Query
	if query, ok := config["query"].(string); ok {
		s.query = query
	}

	// Optional: Additional parameters
	if params, ok := config["params"].(map[string]string); ok {
		s.params = params
	} else if params, ok := config["params"].(map[string]interface{}); ok {
		s.params = make(map[string]string)
		for k, v := range params {
			s.params[k] = fmt.Sprintf("%v", v)
		}
	}

	return nil
}

// Validate checks if the configuration is valid
func (s *VmlogsSource) Validate() error {
	if s.url == "" {
		return fmt.Errorf("url is required")
	}

	// Could add a test query here to validate connection
	return nil
}

// Start begins streaming from Victoria Logs
func (s *VmlogsSource) Start(ctx context.Context) (<-chan plugin.LogEntry, error) {
	s.ctx, s.cancel = context.WithCancel(ctx)

	// Create and start Victoria Logs receiver
	s.receiver = vmlogs.NewReceiver(s.url, s.user, s.password, s.query, s.params)
	if err := s.receiver.Start(); err != nil {
		return nil, fmt.Errorf("failed to start Victoria Logs receiver: %w", err)
	}

	// Start metrics updater
	go s.updateMetrics()

	// Start reading from Victoria Logs
	go s.readVmlogs()

	s.setConnected(true)
	return s.logChan, nil
}

// Stop gracefully stops the receiver
func (s *VmlogsSource) Stop() error {
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
func (s *VmlogsSource) GetMetrics() plugin.Metrics {
	s.metricsMux.RLock()
	defer s.metricsMux.RUnlock()
	return s.metrics
}

// SetFilter applies a LogsQL filter
func (s *VmlogsSource) SetFilter(filter string) error {
	s.query = filter
	// Would need to restart the receiver with new query
	return nil
}

// GetFilterSyntax returns LogsQL syntax documentation
func (s *VmlogsSource) GetFilterSyntax() string {
	return `LogsQL Syntax (Victoria Logs):
	- *: Match all logs
	- word: Search for word
	- "exact phrase": Search for exact phrase
	- field:value: Match field with value
	- field:~"regex": Match field with regex
	- level:error: Match error level
	- _time:[1h, now]: Time range
	Example: service:"myapp" AND level:error`
}

// GetConfigSchema returns the configuration schema
func (s *VmlogsSource) GetConfigSchema() plugin.ConfigSchema {
	return plugin.ConfigSchema{
		Required: []plugin.ConfigParam{
			{
				Name:        "url",
				Type:        "string",
				Description: "Victoria Logs URL endpoint",
				Example:     "http://localhost:9428",
			},
		},
		Optional: []plugin.ConfigParam{
			{
				Name:        "user",
				Type:        "string",
				Description: "Basic auth username",
			},
			{
				Name:        "password",
				Type:        "string",
				Description: "Basic auth password",
			},
			{
				Name:        "query",
				Type:        "string",
				Description: "LogsQL query",
				Default:     "*",
				Example:     `service:"myapp" AND level:error`,
			},
			{
				Name:        "params",
				Type:        "map[string]string",
				Description: "Additional query parameters",
				Example:     map[string]string{"start_offset": "1h"},
			},
		},
	}
}

// readVmlogs reads logs from Victoria Logs
func (s *VmlogsSource) readVmlogs() {
	defer func() {
		s.setConnected(false)
		close(s.logChan)
	}()

	// Get the channel from Victoria Logs receiver
	vmlogsLineChan := s.receiver.GetLineChan()

	for {
		select {
		case <-s.ctx.Done():
			return
		case line, ok := <-vmlogsLineChan:
			if !ok {
				// Victoria Logs receiver finished
				return
			}
			if line != "" {
				// Victoria Logs lines are already in JSON format
				entry := plugin.LogEntry{
					Timestamp: time.Now().UnixNano(),
					Raw:       line,
					Message:   line, // Will be parsed by gonzo's JSON analyzer
					Severity:  "INFO",
					Source: plugin.SourceInfo{
						Type:       "vmlogs",
						Identifier: s.query,
						Metadata: map[string]string{
							"url": s.url,
						},
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

func (s *VmlogsSource) incrementMetrics() {
	s.metricsMux.Lock()
	defer s.metricsMux.Unlock()
	s.metrics.TotalLogs++
}

func (s *VmlogsSource) recordError(err error) {
	s.metricsMux.Lock()
	defer s.metricsMux.Unlock()
	s.metrics.Errors++
	s.metrics.LastError = err.Error()
}

func (s *VmlogsSource) setConnected(connected bool) {
	s.metricsMux.Lock()
	defer s.metricsMux.Unlock()
	s.metrics.Connected = connected
}

func (s *VmlogsSource) updateMetrics() {
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
