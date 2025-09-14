// Package builtin provides built-in log source implementations.
package builtin

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/control-theory/gonzo/internal/filereader"
	"github.com/control-theory/gonzo/internal/plugin"
)

// FileSource implements the LogSource interface for file input
type FileSource struct {
	files      []string
	follow     bool
	reader     *filereader.FileReader
	logChan    chan plugin.LogEntry
	ctx        context.Context
	cancel     context.CancelFunc
	metrics    plugin.Metrics
	metricsMux sync.RWMutex
}

// NewFileSource creates a new file log source
func NewFileSource() plugin.LogSource {
	return &FileSource{
		logChan: make(chan plugin.LogEntry, 100),
	}
}

// Name returns the name of the log source
func (s *FileSource) Name() string {
	return "files"
}

// Description returns a human-readable description
func (s *FileSource) Description() string {
	return "Read logs from local files with glob support and tail -f capability"
}

// Configure sets up the file source
func (s *FileSource) Configure(config map[string]interface{}) error {
	// Parse file paths
	if files, ok := config["files"].([]string); ok {
		s.files = files
	} else if files, ok := config["files"].([]interface{}); ok {
		s.files = make([]string, len(files))
		for i, f := range files {
			if str, ok := f.(string); ok {
				s.files[i] = str
			} else {
				return fmt.Errorf("invalid file path at index %d", i)
			}
		}
	} else {
		return fmt.Errorf("files parameter is required")
	}

	// Parse follow flag
	if follow, ok := config["follow"].(bool); ok {
		s.follow = follow
	}

	return nil
}

// Validate checks if the configuration is valid
func (s *FileSource) Validate() error {
	if len(s.files) == 0 {
		return fmt.Errorf("at least one file path is required")
	}

	// Create file reader to validate paths
	var err error
	s.reader, err = filereader.New(s.files, s.follow)
	if err != nil {
		return fmt.Errorf("failed to initialize file reader: %w", err)
	}

	return nil
}

// Start begins reading from files
func (s *FileSource) Start(ctx context.Context) (<-chan plugin.LogEntry, error) {
	s.ctx, s.cancel = context.WithCancel(ctx)

	if s.reader == nil {
		// Create reader if not already created in Validate
		var err error
		s.reader, err = filereader.New(s.files, s.follow)
		if err != nil {
			return nil, fmt.Errorf("failed to create file reader: %w", err)
		}
	}

	// Start metrics updater
	go s.updateMetrics()

	// Start reading files
	go s.readFiles()

	s.setConnected(true)
	return s.logChan, nil
}

// Stop gracefully stops reading
func (s *FileSource) Stop() error {
	if s.cancel != nil {
		s.cancel()
	}

	if s.reader != nil {
		s.reader.Stop()
	}

	if s.logChan != nil {
		close(s.logChan)
	}

	return nil
}

// GetMetrics returns current metrics
func (s *FileSource) GetMetrics() plugin.Metrics {
	s.metricsMux.RLock()
	defer s.metricsMux.RUnlock()
	return s.metrics
}

// GetConfigSchema returns the configuration schema
func (s *FileSource) GetConfigSchema() plugin.ConfigSchema {
	return plugin.ConfigSchema{
		Required: []plugin.ConfigParam{
			{
				Name:        "files",
				Type:        "[]string",
				Description: "List of file paths or glob patterns",
				Example:     []string{"/var/log/*.log", "app.log"},
			},
		},
		Optional: []plugin.ConfigParam{
			{
				Name:        "follow",
				Type:        "bool",
				Description: "Follow files like 'tail -f'",
				Default:     false,
			},
		},
	}
}

// readFiles reads lines from files
func (s *FileSource) readFiles() {
	defer func() {
		s.setConnected(false)
		close(s.logChan)
	}()

	// Start the file reader and get the channel
	fileLineChan := s.reader.Start()

	for {
		select {
		case <-s.ctx.Done():
			return
		case line, ok := <-fileLineChan:
			if !ok {
				// File reader finished
				return
			}
			if line != "" {
				entry := plugin.LogEntry{
					Timestamp: time.Now().UnixNano(),
					Raw:       line,
					Message:   line,
					Severity:  "INFO", // Default severity
					Source: plugin.SourceInfo{
						Type:       "file",
						Identifier: "files", // Could be enhanced to include specific file name
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

func (s *FileSource) incrementMetrics() {
	s.metricsMux.Lock()
	defer s.metricsMux.Unlock()
	s.metrics.TotalLogs++
}

func (s *FileSource) recordError(err error) {
	s.metricsMux.Lock()
	defer s.metricsMux.Unlock()
	s.metrics.Errors++
	s.metrics.LastError = err.Error()
}

func (s *FileSource) setConnected(connected bool) {
	s.metricsMux.Lock()
	defer s.metricsMux.Unlock()
	s.metrics.Connected = connected
}

func (s *FileSource) updateMetrics() {
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
