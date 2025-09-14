package builtin

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/control-theory/gonzo/internal/plugin"
)

// StdinSource implements the LogSource interface for stdin input
type StdinSource struct {
	logChan    chan plugin.LogEntry
	ctx        context.Context
	cancel     context.CancelFunc
	metrics    plugin.Metrics
	metricsMux sync.RWMutex
}

// NewStdinSource creates a new stdin log source
func NewStdinSource() plugin.LogSource {
	return &StdinSource{
		logChan: make(chan plugin.LogEntry, 100),
	}
}

// Name returns the name of the log source
func (s *StdinSource) Name() string {
	return "stdin"
}

// Description returns a human-readable description
func (s *StdinSource) Description() string {
	return "Read logs from standard input (pipe or redirect)"
}

// Configure sets up the stdin source (no configuration needed)
func (s *StdinSource) Configure(_ map[string]interface{}) error {
	// Stdin doesn't need configuration
	return nil
}

// Validate checks if stdin is available
func (s *StdinSource) Validate() error {
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) != 0 {
		return fmt.Errorf("stdin is not available (no pipe or redirect detected)")
	}
	return nil
}

// Start begins reading from stdin
func (s *StdinSource) Start(ctx context.Context) (<-chan plugin.LogEntry, error) {
	s.ctx, s.cancel = context.WithCancel(ctx)

	// Start metrics updater
	go s.updateMetrics()

	// Start reading stdin
	go s.readStdin()

	s.setConnected(true)
	return s.logChan, nil
}

// Stop gracefully stops reading
func (s *StdinSource) Stop() error {
	if s.cancel != nil {
		s.cancel()
	}

	// logChan is closed in readStdin() goroutine
	// Don't close it here to avoid double-close panic

	return nil
}

// GetMetrics returns current metrics
func (s *StdinSource) GetMetrics() plugin.Metrics {
	s.metricsMux.RLock()
	defer s.metricsMux.RUnlock()
	return s.metrics
}

// readStdin reads lines from standard input
func (s *StdinSource) readStdin() {
	defer func() {
		s.setConnected(false)
		close(s.logChan)
	}()

	scanner := bufio.NewScanner(os.Stdin)

	// Set larger buffer size (1MB) to handle long lines
	const maxScanTokenSize = 1024 * 1024 // 1MB
	buf := make([]byte, maxScanTokenSize)
	scanner.Buffer(buf, maxScanTokenSize)

	for scanner.Scan() {
		select {
		case <-s.ctx.Done():
			return
		default:
			line := scanner.Text()
			if line != "" {
				entry := plugin.LogEntry{
					Timestamp: time.Now().UnixNano(),
					Raw:       line,
					Message:   line,
					Severity:  "INFO", // Default severity
					Source: plugin.SourceInfo{
						Type:       "stdin",
						Identifier: "stdin",
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

	if err := scanner.Err(); err != nil {
		s.recordError(err)
	}
}

// Helper functions for metrics

func (s *StdinSource) incrementMetrics() {
	s.metricsMux.Lock()
	defer s.metricsMux.Unlock()
	s.metrics.TotalLogs++
}

func (s *StdinSource) recordError(err error) {
	s.metricsMux.Lock()
	defer s.metricsMux.Unlock()
	s.metrics.Errors++
	s.metrics.LastError = err.Error()
}

func (s *StdinSource) setConnected(connected bool) {
	s.metricsMux.Lock()
	defer s.metricsMux.Unlock()
	s.metrics.Connected = connected
}

func (s *StdinSource) updateMetrics() {
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
