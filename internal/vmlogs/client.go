package vmlogs

import (
	"bufio"
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// Client represents a Victoria Logs client for streaming logs
type Client struct {
	BaseURL  string
	User     string
	Password string
	Client   *http.Client
}

// NewClient creates a new Victoria Logs client
func NewClient(baseURL, user, password string) *Client {
	return &Client{
		BaseURL:  strings.TrimSuffix(baseURL, "/"),
		User:     user,
		Password: password,
		Client: &http.Client{
			Timeout: 0, // No timeout for streaming
		},
	}
}

// Tail live-tails logs matching LogsQL. Optionally pass params like
// start_offset=1h to replay recent history before live mode kicks in.
func (c *Client) Tail(ctx context.Context, logsQL string, params map[string]string, onLine func(string) error) error {
	endpoint := c.BaseURL + "/select/logsql/tail"
	form := url.Values{}
	form.Set("query", logsQL)
	for k, v := range params {
		form.Set(k, v)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if c.User != "" {
		req.SetBasicAuth(c.User, c.Password)
	}

	// Important for streaming: no request timeout; rely on ctx cancellation.
	if c.Client.Transport == nil {
		c.Client.Transport = http.DefaultTransport
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))
		return errors.New("tail failed: " + resp.Status + " - " + string(b))
	}

	return scanNDJSON(resp.Body, onLine)
}

// scanNDJSON scans newline-delimited JSON from the response body
func scanNDJSON(r io.Reader, onLine func(string) error) error {
	scanner := bufio.NewScanner(r)

	// Set larger buffer size to handle long JSON lines
	const maxScanTokenSize = 1024 * 1024 // 1MB
	buf := make([]byte, maxScanTokenSize)
	scanner.Buffer(buf, maxScanTokenSize)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		// Pass through the raw Victoria Logs JSON
		// The existing log converter will handle it as JSON format
		if err := onLine(line); err != nil {
			return err
		}
	}

	return scanner.Err()
}

// Receiver implements a Victoria Logs receiver that streams logs to a channel
type Receiver struct {
	client     *Client
	query      string
	params     map[string]string
	lineChan   chan string
	ctx        context.Context
	cancelFunc context.CancelFunc
}

// NewReceiver creates a new Victoria Logs receiver
func NewReceiver(baseURL, user, password, query string, params map[string]string) *Receiver {
	client := NewClient(baseURL, user, password)
	ctx, cancel := context.WithCancel(context.Background())

	return &Receiver{
		client:     client,
		query:      query,
		params:     params,
		lineChan:   make(chan string, 100),
		ctx:        ctx,
		cancelFunc: cancel,
	}
}

// Start begins streaming logs from Victoria Logs
func (r *Receiver) Start() error {
	go func() {
		defer close(r.lineChan)

		onLine := func(line string) error {
			select {
			case r.lineChan <- line:
			case <-r.ctx.Done():
				return r.ctx.Err()
			}
			return nil
		}

		if err := r.client.Tail(r.ctx, r.query, r.params, onLine); err != nil && !errors.Is(err, context.Canceled) {
			// Silently ignore streaming errors to avoid UI interference
			// The receiver will stop gracefully
		}
	}()

	return nil
}

// Stop stops the Victoria Logs receiver
func (r *Receiver) Stop() {
	if r.cancelFunc != nil {
		r.cancelFunc()
	}
}

// GetLineChan returns the channel for receiving log lines
func (r *Receiver) GetLineChan() <-chan string {
	return r.lineChan
}
