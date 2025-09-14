// Package loki provides a Grafana Loki log source implementation.
package loki

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/control-theory/gonzo/internal/logger"
	"github.com/control-theory/gonzo/internal/plugin"
	"github.com/gorilla/websocket"
)

// Source implements the LogSource interface for Grafana Loki.
type Source struct {
	// Configuration
	url      string
	user     string
	password string
	query    string
	labels   map[string]string
	limit    int
	since    time.Duration
	oneShot  bool   // For testing - fetch once and stop
	config   map[string]interface{} // Store full config for runtime checks

	// Runtime state
	client     *http.Client
	wsConn     *websocket.Conn
	ctx        context.Context
	cancel     context.CancelFunc
	logChan    chan plugin.LogEntry
	metrics    plugin.Metrics
	metricsMux sync.RWMutex
	lastError  error

	// Reconnection
	reconnectPolicy plugin.ReconnectPolicy
}

// NewSource creates a new Loki log source.
func NewSource() plugin.LogSource {
	return &Source{
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
		limit: 1000,
		since: 1 * time.Hour,
		reconnectPolicy: plugin.ReconnectPolicy{
			MaxAttempts:       0, // Infinite
			InitialDelay:      1000,
			MaxDelay:          30000,
			BackoffMultiplier: 2.0,
		},
	}
}

// Name returns the name of the log source
func (l *Source) Name() string {
	return "loki"
}

// Description returns a human-readable description
func (l *Source) Description() string {
	return "Grafana Loki log aggregation system"
}

// Configure sets up the Loki source with configuration parameters
func (l *Source) Configure(config map[string]interface{}) error {
	// Store full config for runtime checks
	l.config = config

	// Required: URL
	if v, ok := config["url"].(string); ok {
		l.url = strings.TrimSuffix(v, "/")
	} else {
		return fmt.Errorf("url is required")
	}

	// Optional: Authentication
	if v, ok := config["user"].(string); ok {
		l.user = v
	}
	if v, ok := config["password"].(string); ok {
		l.password = v
	}

	// Optional: Query (LogQL)
	if v, ok := config["query"].(string); ok {
		l.query = v
	} else {
		// Default query to match all logs with job label
		l.query = `{job=~".+"}`
	}

	// Optional: Label filters
	if v, ok := config["labels"].(map[string]string); ok {
		l.labels = v
	}

	// Optional: Limit
	if v, ok := config["limit"].(float64); ok {
		l.limit = int(v)
	} else if v, ok := config["limit"].(int); ok {
		l.limit = v
	}

	// Optional: Since duration (how far back to look)
	if v, ok := config["since"].(string); ok {
		duration, err := time.ParseDuration(v)
		if err != nil {
			return fmt.Errorf("invalid since duration: %w", err)
		}
		l.since = duration
	}

	// Optional: oneShot mode (for testing - fetch once and stop)
	if v, ok := config["oneShot"].(bool); ok {
		l.oneShot = v
	}

	return nil
}

// Validate checks if the configuration is valid
func (l *Source) Validate() error {
	
	if l.url == "" {
		return fmt.Errorf("url is required")
	}

	// Try to parse the URL
	_, err := url.Parse(l.url)
	if err != nil {
		return fmt.Errorf("invalid url: %w", err)
	}

	// Build the query if labels are provided
	if len(l.labels) > 0 && (l.query == "{}" || l.query == `{job=~".+"}`) {
		l.query = l.buildQuery()
	}
	

	// Don't test connection during validation - let it fail at runtime
	// This allows users to configure the source even if the endpoint
	// is temporarily unavailable or doesn't exist yet

	return nil
}

// Start begins streaming logs
func (l *Source) Start(ctx context.Context) (<-chan plugin.LogEntry, error) {
	l.ctx, l.cancel = context.WithCancel(ctx)
	l.logChan = make(chan plugin.LogEntry, 100)

	// Start metrics updater
	go l.updateMetrics()

	// Determine which API to use
	if l.supportsWebSocket() {
		// Use WebSocket for real-time streaming
		go l.streamViaWebSocket()
	} else {
		// Fall back to polling
		go l.streamViaPolling()
	}

	return l.logChan, nil
}

// Stop gracefully stops the log streaming
func (l *Source) Stop() error {
	if l.cancel != nil {
		l.cancel()
	}

	if l.wsConn != nil {
		l.wsConn.Close()
	}

	// Don't close logChan here - the streaming goroutines handle it with defer

	return nil
}

// GetMetrics returns current metrics
func (l *Source) GetMetrics() plugin.Metrics {
	l.metricsMux.RLock()
	defer l.metricsMux.RUnlock()
	return l.metrics
}

// SetReconnectPolicy configures reconnection behavior
func (l *Source) SetReconnectPolicy(policy plugin.ReconnectPolicy) {
	l.reconnectPolicy = policy
}

// SetFilter applies a LogQL filter
func (l *Source) SetFilter(filter string) error {
	l.query = filter
	return nil
}

// GetFilterSyntax returns LogQL syntax documentation
func (l *Source) GetFilterSyntax() string {
	return `LogQL Syntax:
	- {label="value"}: Select streams by labels
	- | json: Parse log lines as JSON
	- |= "text": Line contains text
	- |~ "regex": Line matches regex
	- != "text": Line doesn't contain text
	- !~ "regex": Line doesn't match regex
	Example: {job="nginx"} | json |= "error" |~ "timeout|refused"`
}

// GetConfigSchema returns the configuration schema
func (l *Source) GetConfigSchema() plugin.ConfigSchema {
	return plugin.ConfigSchema{
		Required: []plugin.ConfigParam{
			{
				Name:        "url",
				Type:        "string",
				Description: "Loki server URL (use ws:// or wss:// for WebSocket streaming, http:// or https:// for polling)",
				Example:     "ws://localhost:3100 or http://localhost:3100",
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
				Description: "LogQL query",
				Default:     `{job=~".+"}`,
				Example:     `{job="myapp"} |= "error" | json`,
			},
			{
				Name:        "labels",
				Type:        "map[string]string",
				Description: "Label selectors",
				Example:     map[string]string{"job": "myapp", "env": "prod"},
			},
			{
				Name:        "limit",
				Type:        "int",
				Description: "Maximum number of log lines to fetch per request",
				Default:     1000,
			},
			{
				Name:        "since",
				Type:        "string",
				Description: "How far back to look for logs",
				Default:     "1h",
				Example:     "24h",
			},
			{
				Name:        "disableWebSocket",
				Type:        "bool",
				Description: "Force HTTP polling even with ws:// URL",
				Default:     false,
			},
		},
	}
}

// buildQuery constructs a LogQL query from labels
func (l *Source) buildQuery() string {
	if len(l.labels) == 0 {
		// Use a query that matches all logs with a job label
		// job is the most common label in Loki deployments
		return `{job=~".+"}`
	}

	parts := make([]string, 0, len(l.labels))
	for k, v := range l.labels {
		parts = append(parts, fmt.Sprintf(`%s="%s"`, k, v))
	}
	return "{" + strings.Join(parts, ",") + "}"
}

// supportsWebSocket checks if the Loki instance supports WebSocket streaming
func (l *Source) supportsWebSocket() bool {
	// Check if WebSocket is explicitly disabled
	if disabled, ok := l.config["disableWebSocket"].(bool); ok && disabled {
		logger.Info("[LOKI] WebSocket disabled by configuration, using polling")
		return false
	}

	// Check if URL scheme indicates WebSocket (ws:// or wss://)
	if strings.HasPrefix(l.url, "ws://") || strings.HasPrefix(l.url, "wss://") {
		logger.Info("[LOKI] WebSocket URL detected (ws:// or wss://), using WebSocket for real-time log tailing")
		return true
	}

	// For http/https URLs, use polling by default
	logger.Info("[LOKI] HTTP URL detected, using polling (use ws:// or wss:// for WebSocket streaming)")
	return false
}

// streamViaWebSocket uses the WebSocket API for real-time streaming
func (l *Source) streamViaWebSocket() {
	defer close(l.logChan)

	// First fetch historical logs via HTTP
	logger.Info("[LOKI] Fetching historical logs before starting WebSocket tail")
	lastTimestamp := time.Now().Add(-l.since).UnixNano()

	entries, maxTimestamp, err := l.queryRange(lastTimestamp)
	if err != nil {
		logger.Errorf("[LOKI] Failed to fetch historical logs: %v", err)
	} else {
		logger.Infof("[LOKI] Fetched %d historical logs", len(entries))
		// Send historical entries to channel
		for _, entry := range entries {
			select {
			case l.logChan <- entry:
				l.incrementMetrics()
			case <-l.ctx.Done():
				return
			}
		}
		if maxTimestamp > 0 {
			lastTimestamp = maxTimestamp
		}
	}

	// Now start WebSocket for real-time tailing
	for {
		select {
		case <-l.ctx.Done():
			return
		default:
			if err := l.connectWebSocket(); err != nil {
				l.recordError(err)
				logger.Warnf("[LOKI] WebSocket connection failed, falling back to polling: %v", err)
				// Fall back to polling if WebSocket fails
				l.streamViaPollingFromTimestamp(lastTimestamp)
				return
			}

			// Read from WebSocket
			for {
				select {
				case <-l.ctx.Done():
					return
				default:
					var msg tailResponse
					if err := l.wsConn.ReadJSON(&msg); err != nil {
						if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
							logger.Info("[LOKI] WebSocket closed normally")
						} else {
							logger.Errorf("[LOKI] WebSocket read error: %v", err)
						}
						l.recordError(err)
						l.wsConn.Close()
						break
					}

					// Process streams
					logger.Debugf("[LOKI] Received %d streams via WebSocket", len(msg.Streams))
					for _, stream := range msg.Streams {
						l.processStream(stream)
					}
				}
			}
		}
	}
}

// streamViaPollingFromTimestamp continues polling from a specific timestamp
func (l *Source) streamViaPollingFromTimestamp(lastTimestamp int64) {
	for {
		select {
		case <-l.ctx.Done():
			return
		default:
			// Query for new logs
			entries, maxTimestamp, err := l.queryRange(lastTimestamp)
			if err != nil {
				l.recordError(err)
				if !l.shouldReconnect() {
					return
				}
				time.Sleep(l.getReconnectDelay())
				continue
			}

			// Send entries to channel
			for _, entry := range entries {
				select {
				case l.logChan <- entry:
					l.incrementMetrics()
				case <-l.ctx.Done():
					return
				}
			}

			// Update timestamp
			if maxTimestamp > lastTimestamp {
				lastTimestamp = maxTimestamp
			}

			if l.oneShot {
				// Give a moment for any remaining data
				time.Sleep(500 * time.Millisecond)
				return
			}

			// Poll every 750ms (faster than 1s dashboard update)
			time.Sleep(750 * time.Millisecond)
		}
	}
}

// streamViaPolling uses the query_range API with polling
func (l *Source) streamViaPolling() {
	defer close(l.logChan)

	lastTimestamp := time.Now().Add(-l.since).UnixNano()
	l.streamViaPollingFromTimestamp(lastTimestamp)
}


// connectWebSocket establishes a WebSocket connection
func (l *Source) connectWebSocket() error {
	// URL is already ws:// or wss:// if we got here
	wsURL := fmt.Sprintf("%s/loki/api/v1/tail?query=%s", l.url, url.QueryEscape(l.query))

	logger.Infof("[LOKI] Connecting to WebSocket: %s", wsURL)

	header := http.Header{}
	if l.user != "" {
		header.Set("Authorization", "Basic "+basicAuth(l.user, l.password))
	}

	conn, resp, err := websocket.DefaultDialer.DialContext(l.ctx, wsURL, header)
	if err != nil {
		if resp != nil {
			logger.Errorf("[LOKI] WebSocket connection failed with status %d", resp.StatusCode)
		}
		return fmt.Errorf("WebSocket connection failed: %w", err)
	}

	l.wsConn = conn
	l.setConnected(true)
	logger.Info("[LOKI] WebSocket connection established successfully")
	return nil
}

// queryRange queries logs in a time range
func (l *Source) queryRange(startNano int64) ([]plugin.LogEntry, int64, error) {
	endNano := time.Now().UnixNano()

	// Convert ws/wss URLs to http/https for the REST API
	httpURL := l.url
	httpURL = strings.Replace(httpURL, "ws://", "http://", 1)
	httpURL = strings.Replace(httpURL, "wss://", "https://", 1)

	// Build the full URL with query parameters
	queryURL := fmt.Sprintf("%s/loki/api/v1/query_range?query=%s&start=%d&end=%d&limit=%d&direction=forward",
		httpURL,
		url.QueryEscape(l.query),
		startNano,
		endNano,
		l.limit,
	)
	
	
	// Try simple http.Get first to debug
	resp, err := http.Get(queryURL)
	if err != nil {
		return nil, startNano, err
	}
	defer resp.Body.Close()
	

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		// Provide more context for common errors
		if resp.StatusCode == http.StatusBadRequest {
			return nil, startNano, fmt.Errorf("bad query syntax (status %d): %s", resp.StatusCode, string(body))
		}
		return nil, startNano, fmt.Errorf("query failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result queryRangeResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, startNano, err
	}
	

	entries := []plugin.LogEntry{}
	maxTimestamp := startNano

	for _, stream := range result.Data.Result {
		for _, value := range stream.Values {
			if len(value) < 2 {
				continue
			}

			// Parse timestamp (first element)
			tsStr, ok := value[0].(string)
			if !ok {
				continue
			}
			timestamp, err := strconv.ParseInt(tsStr, 10, 64)
			if err != nil {
				// Log timestamp parse error but continue processing
				continue
			}

			// Parse log line (second element)
			logLine, ok := value[1].(string)
			if !ok {
				continue
			}

			entry := l.createLogEntry(timestamp, logLine, stream.Stream)
			entries = append(entries, entry)

			if timestamp > maxTimestamp {
				maxTimestamp = timestamp
			}
		}
	}

	l.setConnected(true)
	return entries, maxTimestamp, nil
}

// processStream processes a stream from tail API
func (l *Source) processStream(stream lokiStream) {
	for _, value := range stream.Values {
		if len(value) < 2 {
			continue
		}

		// Parse timestamp
		tsStr, ok := value[0].(string)
		if !ok {
			continue
		}
		timestamp, err := strconv.ParseInt(tsStr, 10, 64)
		if err != nil {
			// Log timestamp parse error but continue processing
			continue
		}

		// Parse log line
		logLine, ok := value[1].(string)
		if !ok {
			continue
		}

		entry := l.createLogEntry(timestamp, logLine, stream.Stream)

		select {
		case l.logChan <- entry:
			l.incrementMetrics()
		case <-l.ctx.Done():
			return
		}
	}
}

// createLogEntry creates a LogEntry from Loki data
func (l *Source) createLogEntry(timestamp int64, logLine string, labels map[string]string) plugin.LogEntry {
	// Try to parse the log line as JSON to extract structured data
	var jsonData map[string]interface{}
	message := logLine
	severity := "INFO"
	attributes := make(map[string]string)

	// First check Loki labels for severity (these are more reliable)
	if lvl, ok := labels["severity_text"]; ok {
		severity = strings.ToUpper(lvl)
	} else if lvl, ok := labels["severity"]; ok {
		severity = strings.ToUpper(lvl)
	} else if lvl, ok := labels["level"]; ok {
		severity = strings.ToUpper(lvl)
	} else if lvl, ok := labels["detected_level"]; ok {
		severity = strings.ToUpper(lvl)
	}

	if err := json.Unmarshal([]byte(logLine), &jsonData); err == nil {
		// Successfully parsed as JSON
		if msg, ok := jsonData["message"].(string); ok {
			message = msg
		} else if msg, ok := jsonData["msg"].(string); ok {
			message = msg
		}

		// Extract severity/level from JSON if not already set from labels
		if severity == "INFO" {
			if lvl, ok := jsonData["level"].(string); ok {
				severity = strings.ToUpper(lvl)
			} else if lvl, ok := jsonData["severity"].(string); ok {
				severity = strings.ToUpper(lvl)
			}
		}

		// Extract other fields as attributes
		for k, v := range jsonData {
			if k != "message" && k != "msg" && k != "level" && k != "severity" && k != "timestamp" {
				attributes[k] = fmt.Sprintf("%v", v)
			}
		}
	}

	// Add Loki labels as attributes
	for k, v := range labels {
		attributes["loki."+k] = v
	}

	return plugin.LogEntry{
		Timestamp:  timestamp,
		Raw:        logLine,
		Message:    message,
		Severity:   severity,
		Attributes: attributes,
		Resource:   labels,
		Source: plugin.SourceInfo{
			Type:       "loki",
			Identifier: l.query,
		},
	}
}

// Helper functions for metrics and reconnection

func (l *Source) incrementMetrics() {
	l.metricsMux.Lock()
	defer l.metricsMux.Unlock()
	l.metrics.TotalLogs++
}

func (l *Source) recordError(err error) {
	l.metricsMux.Lock()
	defer l.metricsMux.Unlock()
	l.metrics.Errors++
	l.metrics.LastError = err.Error()
	l.lastError = err
}

func (l *Source) setConnected(connected bool) {
	l.metricsMux.Lock()
	defer l.metricsMux.Unlock()
	l.metrics.Connected = connected
}

func (l *Source) updateMetrics() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	var lastCount int64
	for {
		select {
		case <-l.ctx.Done():
			return
		case <-ticker.C:
			l.metricsMux.Lock()
			currentCount := l.metrics.TotalLogs
			l.metrics.LogsPerSecond = float64(currentCount - lastCount)
			lastCount = currentCount
			l.metricsMux.Unlock()
		}
	}
}

func (l *Source) shouldReconnect() bool {
	// Implement reconnection logic based on policy
	return l.reconnectPolicy.MaxAttempts == 0 || l.metrics.Errors < int64(l.reconnectPolicy.MaxAttempts)
}

func (l *Source) getReconnectDelay() time.Duration {
	// Implement exponential backoff
	delay := l.reconnectPolicy.InitialDelay
	for i := 0; i < int(l.metrics.Errors); i++ {
		delay = int(float64(delay) * l.reconnectPolicy.BackoffMultiplier)
		if delay > l.reconnectPolicy.MaxDelay {
			delay = l.reconnectPolicy.MaxDelay
			break
		}
	}
	return time.Duration(delay) * time.Millisecond
}

func basicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

// Response types for Loki API

type tailResponse struct {
	Streams []lokiStream `json:"streams"`
}

type lokiStream struct {
	Stream map[string]string `json:"stream"`
	Values [][]interface{}   `json:"values"`
}

type queryRangeResponse struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string       `json:"resultType"`
		Result     []lokiStream `json:"result"`
	} `json:"data"`
}
