// Package docker provides Docker container log streaming.
package docker

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/control-theory/gonzo/internal/logger"
	"github.com/control-theory/gonzo/internal/plugin"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

// Pre-compiled regular expressions for better performance
var (
	// Date/time parsing patterns
	dateTimeRegexes = []*regexp.Regexp{
		regexp.MustCompile(`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d+Z`),           // RFC3339Nano
		regexp.MustCompile(`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z`),                 // RFC3339
		regexp.MustCompile(`\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}`),                  // 2006-01-02 15:04:05
		regexp.MustCompile(`\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2}`),                  // 2006/01/02 15:04:05
		regexp.MustCompile(`[A-Z][a-z]{2} \d{1,2} \d{2}:\d{2}:\d{2}`),              // Feb 21 21:37:48
	}

	dateTimeLayouts = []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006/01/02 15:04:05",
		time.Stamp,
	}

	// Log level detection patterns
	levelRegexes = []*regexp.Regexp{
		regexp.MustCompile(`level=([a-zA-Z]+)`),           // level=info, level=error, etc.
		regexp.MustCompile(`"level"\s*:\s*"([a-zA-Z]+)"`), // JSON: "level":"info"
		regexp.MustCompile(`'level'\s*:\s*'([a-zA-Z]+)'`), // JSON with single quotes
		regexp.MustCompile(`\blevel:\s*([a-zA-Z]+)`),      // YAML-style: level: info
	}

	// Severity keyword patterns with word boundaries
	severityKeywordRegexes = map[string][]*regexp.Regexp{
		"ERROR": {
			regexp.MustCompile(`\berror\b`),
			regexp.MustCompile(`\berr\b`),
			regexp.MustCompile(`\bfatal\b`),
			regexp.MustCompile(`\bpanic\b`),
			regexp.MustCompile(`\bfailed\b`),
			regexp.MustCompile(`\bcritical\b`),
		},
		"WARN": {
			regexp.MustCompile(`\bwarn\b`),
			regexp.MustCompile(`\bwarning\b`),
		},
		"INFO": {
			regexp.MustCompile(`\binfo\b`),
			regexp.MustCompile(`\bnotice\b`),
		},
		"DEBUG": {
			regexp.MustCompile(`\bdebug\b`),
			regexp.MustCompile(`\btrace\b`),
			regexp.MustCompile(`\bverbose\b`),
		},
	}
)

// Source implements the LogSource interface for Docker container logs.
type Source struct {
	client        *client.Client
	logChan       chan plugin.LogEntry
	ctx           context.Context
	cancel        context.CancelFunc
	metrics       plugin.Metrics
	metricsMux    sync.RWMutex

	endpoint      string
	interval      time.Duration
	followFilters []string                       // Container name patterns to follow
	followRegexes []*regexp.Regexp               // Pre-compiled regex patterns for wildcard filters
	tailers       map[string]context.CancelFunc // Track per-container goroutines
	tailersMux    sync.RWMutex

	// Docker metadata
	hostname      string
	dockerVersion string
}

// NewSource creates a new Docker log source.
func NewSource() plugin.LogSource {
	return &Source{
		logChan:  make(chan plugin.LogEntry, 1000),
		tailers:  make(map[string]context.CancelFunc),
		interval: 10 * time.Second,
		endpoint: "unix:///var/run/docker.sock",
	}
}

// Name returns the name of the Docker log source.
func (d *Source) Name() string {
	return "docker"
}

// Description returns a description of the Docker log source.
func (d *Source) Description() string {
	return "Stream logs from Docker containers"
}

// Configure configures the Docker log source with the provided configuration.
func (d *Source) Configure(config map[string]interface{}) error {
	if endpoint, ok := config["endpoint"].(string); ok {
		d.endpoint = endpoint
	}

	if interval, ok := config["interval"].(string); ok {
		dur, err := time.ParseDuration(interval)
		if err != nil {
			logger.Warnf("Invalid interval %s, using default 10s: %v", interval, err)
		} else if dur < 5*time.Second {
			logger.Warnf("Interval %s too short, using minimum 5s", interval)
			d.interval = 5 * time.Second
		} else {
			d.interval = dur
		}
	}

	// Parse follow filters (container names or patterns to follow)
	if follow, ok := config["follow"].([]interface{}); ok {
		for _, f := range follow {
			if pattern, ok := f.(string); ok {
				d.followFilters = append(d.followFilters, pattern)
			}
		}
	} else if follow, ok := config["follow"].(string); ok {
		// Support single string value
		d.followFilters = []string{follow}
	}

	// If no filters specified, follow all containers
	if len(d.followFilters) == 0 {
		d.followFilters = []string{"*"}
	}

	// Pre-compile regex patterns for wildcard filters
	d.followRegexes = make([]*regexp.Regexp, 0, len(d.followFilters))
	for _, filter := range d.followFilters {
		if strings.Contains(filter, "*") {
			// Convert wildcard pattern to regex
			pattern := "^" + strings.ReplaceAll(regexp.QuoteMeta(filter), "\\*", ".*") + "$"
			if re, err := regexp.Compile(pattern); err == nil {
				d.followRegexes = append(d.followRegexes, re)
			} else {
				logger.Warnf("Invalid wildcard pattern %s: %v", filter, err)
			}
		}
	}

	return nil
}

// Validate validates the Docker configuration.
func (d *Source) Validate() error {
	cli, err := client.NewClientWithOpts(
		client.WithHost(d.endpoint),
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return fmt.Errorf("failed to create Docker client: %w", err)
	}

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = cli.Ping(ctx)
	if err != nil {
		return fmt.Errorf("failed to connect to Docker daemon at %s: %w", d.endpoint, err)
	}

	cli.Close()
	return nil
}

// Start starts streaming Docker container logs.
func (d *Source) Start(ctx context.Context) (<-chan plugin.LogEntry, error) {
	d.ctx, d.cancel = context.WithCancel(ctx)

	// Initialize Docker client
	cli, err := client.NewClientWithOpts(
		client.WithHost(d.endpoint),
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}
	d.client = cli

	// Get Docker info for metadata
	info, err := cli.Info(ctx)
	if err != nil {
		logger.Warnf("Failed to get Docker info: %v", err)
	} else {
		d.dockerVersion = info.ServerVersion
		d.hostname = info.Name
		logger.Debugf("Connected to Docker %s on %s", info.ServerVersion, info.Name)
	}

	// Start the container discovery loop
	go d.containerDiscoveryLoop()

	// Start metrics updater
	go d.updateMetrics()

	d.setConnected(true)
	return d.logChan, nil
}

// Stop stops streaming Docker container logs.
func (d *Source) Stop() error {
	if d.cancel != nil {
		d.cancel()
	}

	// Stop all tailers
	d.tailersMux.Lock()
	for _, cancelFunc := range d.tailers {
		cancelFunc()
	}
	d.tailersMux.Unlock()

	if d.client != nil {
		d.client.Close()
	}

	// Close channel after all tailers stop
	time.Sleep(100 * time.Millisecond)
	close(d.logChan)

	return nil
}

// GetMetrics returns the current metrics for the Docker log source.
func (d *Source) GetMetrics() plugin.Metrics {
	d.metricsMux.RLock()
	defer d.metricsMux.RUnlock()
	return d.metrics
}

func (d *Source) shouldFollowContainer(containerName string, containerImage string) bool {
	// Remove leading slash from container name if present
	containerName = strings.TrimPrefix(containerName, "/")

	for i, filter := range d.followFilters {
		// Support wildcards
		if filter == "*" || filter == "" {
			return true
		}

		// Check exact match
		if containerName == filter || containerImage == filter {
			return true
		}

		// Check pre-compiled wildcard patterns
		if strings.Contains(filter, "*") {
			// Find the corresponding pre-compiled regex
			regexIndex := 0
			for j := 0; j < i; j++ {
				if strings.Contains(d.followFilters[j], "*") {
					regexIndex++
				}
			}
			if regexIndex < len(d.followRegexes) {
				re := d.followRegexes[regexIndex]
				if re.MatchString(containerName) || re.MatchString(containerImage) {
					return true
				}
			}
		}

		// Support partial matches
		if strings.Contains(containerName, filter) || strings.Contains(containerImage, filter) {
			return true
		}
	}

	return false
}

func (d *Source) containerDiscoveryLoop() {
	ticker := time.NewTicker(d.interval)
	defer ticker.Stop()

	logger.Debugf("Starting container discovery loop (interval: %v, following: %v)", d.interval, d.followFilters)

	// Initial discovery
	d.discoverContainers()

	for {
		select {
		case <-d.ctx.Done():
			return
		case <-ticker.C:
			d.discoverContainers()
		}
	}
}

func (d *Source) discoverContainers() {
	containers, err := d.client.ContainerList(d.ctx, container.ListOptions{All: false}) // Only running containers
	if err != nil {
		logger.Errorf("Failed to list containers: %v", err)
		d.recordError(err)
		return
	}

	logger.Debugf("Found %d running containers", len(containers))

	for _, c := range containers {
		// Check if we should follow this container
		containerName := strings.TrimPrefix(c.Names[0], "/")
		if !d.shouldFollowContainer(containerName, c.Image) {
			logger.Debugf("Skipping container %s (image: %s) - not in follow list", containerName, c.Image)
			continue
		}

		// Check if already tailing
		d.tailersMux.RLock()
		_, exists := d.tailers[c.ID]
		d.tailersMux.RUnlock()

		if !exists {
			logger.Debugf("Starting to follow container: %s (image: %s, id: %s)", containerName, c.Image, c.ID[:12])

			// Create context for this tailer
			tailerCtx, tailerCancel := context.WithCancel(d.ctx)

			// Store the cancel function
			d.tailersMux.Lock()
			d.tailers[c.ID] = tailerCancel
			d.tailersMux.Unlock()

			// Start tailing in a new goroutine
			go d.tailContainer(tailerCtx, c)
		}
	}

	// Clean up tailers for stopped containers
	d.cleanupStoppedContainers(containers)
}

func (d *Source) cleanupStoppedContainers(runningContainers []container.Summary) {
	runningIDs := make(map[string]bool)
	for _, c := range runningContainers {
		runningIDs[c.ID] = true
	}

	d.tailersMux.Lock()
	defer d.tailersMux.Unlock()

	for id, cancelFunc := range d.tailers {
		if !runningIDs[id] {
			logger.Debugf("Container %s is no longer running, stopping tailer", id[:12])
			cancelFunc()
			delete(d.tailers, id)
		}
	}
}

func (d *Source) tailContainer(ctx context.Context, c container.Summary) {
	containerName := strings.TrimPrefix(c.Names[0], "/")

	defer func() {
		d.tailersMux.Lock()
		delete(d.tailers, c.ID)
		d.tailersMux.Unlock()
		logger.Debugf("Stopped following container: %s", containerName)
	}()

	// First, check if the container uses a TTY
	inspect, err := d.client.ContainerInspect(ctx, c.ID)
	if err != nil {
		logger.Errorf("Failed to inspect container %s: %v", containerName, err)
		d.recordError(err)
		return
	}

	options := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
		Timestamps: true,
		Tail:       "10", // Start with last 10 lines
	}

	reader, err := d.client.ContainerLogs(ctx, c.ID, options)
	if err != nil {
		logger.Errorf("Failed to get logs for container %s: %v", containerName, err)
		d.recordError(err)
		return
	}
	defer reader.Close()

	// If container uses TTY, logs are not multiplexed
	if inspect.Config.Tty {
		// Simple line-by-line reading for TTY containers
		scanner := bufio.NewScanner(reader)
		const maxScanTokenSize = 1024 * 1024 // 1MB
		buf := make([]byte, maxScanTokenSize)
		scanner.Buffer(buf, maxScanTokenSize)

		for scanner.Scan() {
			select {
			case <-ctx.Done():
				return
			default:
				line := scanner.Text()
				if line == "" {
					continue
				}

				// Clean the line (remove timestamps if present)
				line = d.cleanLogLine(line)
				if line == "" {
					continue
				}

				entry := d.createLogEntry(c, line)

				select {
				case d.logChan <- entry:
					d.incrementMetrics()
				case <-ctx.Done():
					return
				}
			}
		}

		if err := scanner.Err(); err != nil && err != io.EOF {
			logger.Errorf("Error reading logs for container %s: %v", containerName, err)
			d.recordError(err)
		}
	} else {
		// Use stdcopy to demultiplex stdout/stderr for non-TTY containers
		// Create pipes to capture demultiplexed output
		stdoutReader, stdoutWriter := io.Pipe()
		stderrReader, stderrWriter := io.Pipe()

		// Start goroutine to demultiplex
		go func() {
			_, err := stdcopy.StdCopy(stdoutWriter, stderrWriter, reader)
			stdoutWriter.Close()
			stderrWriter.Close()
			if err != nil && err != io.EOF {
				logger.Debugf("Demultiplexing ended for container %s: %v", containerName, err)
			}
		}()

		// Read from both stdout and stderr concurrently
		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			defer wg.Done()
			d.readFromReader(ctx, c, stdoutReader, "stdout")
		}()

		go func() {
			defer wg.Done()
			d.readFromReader(ctx, c, stderrReader, "stderr")
		}()

		wg.Wait()
	}
}

func (d *Source) readFromReader(ctx context.Context, c container.Summary, reader io.Reader, _ string) {
	scanner := bufio.NewScanner(reader)
	const maxScanTokenSize = 1024 * 1024 // 1MB
	buf := make([]byte, maxScanTokenSize)
	scanner.Buffer(buf, maxScanTokenSize)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		default:
			line := scanner.Text()
			if line == "" {
				continue
			}

			// Clean the line (remove timestamps if present)
			line = d.cleanLogLine(line)
			if line == "" {
				continue
			}

			entry := d.createLogEntry(c, line)

			select {
			case d.logChan <- entry:
				d.incrementMetrics()
			case <-ctx.Done():
				return
			}
		}
	}
}

func (d *Source) cleanLogLine(line string) string {
	// Strip Docker log timestamps if present (when Timestamps: true)
	// Format: 2024-01-15T10:30:45.123456789Z <actual log>
	if len(line) > 30 && line[4] == '-' && line[7] == '-' && line[10] == 'T' {
		if spaceIdx := strings.Index(line, " "); spaceIdx > 0 && spaceIdx < 35 {
			line = line[spaceIdx+1:]
		}
	}

	// Trim any whitespace
	line = strings.TrimSpace(line)

	return line
}

func (d *Source) createLogEntry(c container.Summary, line string) plugin.LogEntry {
	containerName := strings.TrimPrefix(c.Names[0], "/")

	// Parse timestamp from log line if possible
	ts := time.Now()
	parsedTime, err := parseDateTime(line)
	if err == nil {
		ts = parsedTime
	}

	return plugin.LogEntry{
		Timestamp: ts.UnixNano(),
		Raw:       line,
		Message:   line,
		Severity:  detectSeverity(line),
		Attributes: map[string]string{
			"container.id":       c.ID,
			"container.name":     containerName,
			"container.image":    c.Image,
			"container.image.id": c.ImageID,
			"container.status":   c.State,
			"container.command":  c.Command,
		},
		Resource: map[string]string{
			"host.name":      d.hostname,
			"docker.version": d.dockerVersion,
			"service.name":   containerName,
		},
		Source: plugin.SourceInfo{
			Type:       "docker",
			Identifier: containerName,
			Metadata: map[string]string{
				"container_id": c.ID[:12],
				"image":        c.Image,
			},
		},
	}
}

// Helper functions

func parseDateTime(input string) (time.Time, error) {
	for i, re := range dateTimeRegexes {
		if match := re.FindString(input); match != "" {
			if tm, err := time.Parse(dateTimeLayouts[i], match); err == nil {
				if tm.Year() == 0 {
					tm = tm.AddDate(time.Now().Year(), 0, 0)
				}
				return tm, nil
			}
		}
	}

	return time.Time{}, fmt.Errorf("no valid date/time found")
}

func detectSeverity(line string) string {
	// First, check for explicit level fields in structured logs
	for _, re := range levelRegexes {
		if matches := re.FindStringSubmatch(line); len(matches) > 1 {
			level := strings.ToUpper(matches[1])
			// Normalize common level names
			switch level {
			case "ERROR", "ERR", "FATAL", "PANIC", "CRITICAL", "CRIT":
				return "ERROR"
			case "WARN", "WARNING":
				return "WARN"
			case "INFO", "INFORMATION", "NOTICE":
				return "INFO"
			case "DEBUG", "TRACE", "VERBOSE":
				return "DEBUG"
			default:
				// If we found an explicit level field but don't recognize it,
				// still use it as found (uppercase)
				if level != "" {
					return level
				}
			}
		}
	}

	// Fall back to searching for severity keywords in the text
	lower := strings.ToLower(line)
	for severity, regexes := range severityKeywordRegexes {
		for _, re := range regexes {
			if re.MatchString(lower) {
				return severity
			}
		}
	}

	return "INFO"
}

func (d *Source) incrementMetrics() {
	d.metricsMux.Lock()
	defer d.metricsMux.Unlock()
	d.metrics.TotalLogs++
}

func (d *Source) recordError(err error) {
	d.metricsMux.Lock()
	defer d.metricsMux.Unlock()
	d.metrics.Errors++
	d.metrics.LastError = err.Error()
}

func (d *Source) setConnected(connected bool) {
	d.metricsMux.Lock()
	defer d.metricsMux.Unlock()
	d.metrics.Connected = connected
}

func (d *Source) updateMetrics() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	var lastCount int64
	for {
		select {
		case <-d.ctx.Done():
			return
		case <-ticker.C:
			d.metricsMux.Lock()
			currentCount := d.metrics.TotalLogs
			d.metrics.LogsPerSecond = float64(currentCount - lastCount)
			lastCount = currentCount
			d.metricsMux.Unlock()
		}
	}
}