# Plugin Architecture for Gonzo

## Overview

Gonzo now supports a flexible plugin architecture that allows you to add custom log sources beyond the built-in options (stdin, files, OTLP, Victoria Logs). This document describes how to use and create plugins.

## Using the Loki Plugin

The Loki plugin is included as a built-in plugin and can stream logs from Grafana Loki instances.

### Basic Usage

```bash
# Stream all logs from Loki
gonzo --plugin-source=loki --plugin-config='{"url":"http://localhost:3100","query":"{}"}'

# Stream logs with a specific query
gonzo --plugin-source=loki --plugin-config='{"url":"http://localhost:3100","query":"{job=\"myapp\"}"}'

# With authentication
gonzo --plugin-source=loki --plugin-config='{
  "url":"http://loki.example.com",
  "user":"admin",
  "password":"secret",
  "query":"{env=\"prod\"} |= \"error\""
}'

# With label filters
gonzo --plugin-source=loki --plugin-config='{
  "url":"http://localhost:3100",
  "labels":{"job":"nginx","env":"prod"},
  "limit":5000,
  "since":"24h"
}'
```

### Loki Plugin Configuration

| Parameter | Type | Required | Description | Default |
|-----------|------|----------|-------------|---------|
| `url` | string | Yes | Loki server URL | - |
| `user` | string | No | Basic auth username | - |
| `password` | string | No | Basic auth password | - |
| `query` | string | No | LogQL query | `{}` |
| `labels` | map[string]string | No | Label selectors | - |
| `limit` | int | No | Max logs per request | 1000 |
| `since` | string | No | How far back to look (e.g., "1h", "24h") | "1h" |

## Plugin Interface

All plugins must implement the `LogSource` interface:

```go
type LogSource interface {
    // Core methods
    Name() string
    Description() string
    Configure(config map[string]interface{}) error
    Validate() error
    Start(ctx context.Context) (<-chan LogEntry, error)
    Stop() error
    GetMetrics() Metrics
}
```

### LogEntry Structure

Each log entry returned by a plugin must conform to this structure:

```go
type LogEntry struct {
    Timestamp  int64             // Unix nanoseconds
    Raw        string            // Original log line
    Message    string            // Parsed message
    Severity   string            // ERROR, WARN, INFO, DEBUG, etc.
    Attributes map[string]string // Key-value pairs
    Resource   map[string]string // Resource attributes
    TraceID    string            // Optional trace ID
    SpanID     string            // Optional span ID
    Source     SourceInfo        // Source metadata
}
```

## Creating a New Plugin

### Step 1: Implement the LogSource Interface

Create a new package under `internal/plugin/yourplugin/`:

```go
package yourplugin

import (
    "context"
    "github.com/control-theory/gonzo/internal/plugin"
)

type YourSource struct {
    // Your fields here
    url      string
    apiKey   string
    logChan  chan plugin.LogEntry
    // ...
}

func NewYourSource() plugin.LogSource {
    return &YourSource{
        logChan: make(chan plugin.LogEntry, 100),
    }
}

func (s *YourSource) Name() string {
    return "yourplugin"
}

func (s *YourSource) Configure(config map[string]interface{}) error {
    // Parse configuration
    if url, ok := config["url"].(string); ok {
        s.url = url
    }
    // ... parse other config
    return nil
}

func (s *YourSource) Start(ctx context.Context) (<-chan plugin.LogEntry, error) {
    // Start your log streaming logic
    go s.streamLogs(ctx)
    return s.logChan, nil
}

// ... implement other required methods
```

### Step 2: Register Your Plugin

Add your plugin to `cmd/gonzo/plugins.go`:

```go
import (
    "github.com/control-theory/gonzo/internal/plugin/yourplugin"
)

func registerBuiltinPlugins() {
    manager := plugin.GetManager()
    
    // ... existing plugins ...
    
    // Register your plugin
    manager.RegisterPlugin("yourplugin", func() plugin.LogSource {
        return yourplugin.NewYourSource()
    })
}
```

### Step 3: Use Your Plugin

```bash
gonzo --plugin-source=yourplugin --plugin-config='{"url":"http://example.com","apiKey":"secret"}'
```

## Advanced Features

### Optional Interfaces

Plugins can implement additional interfaces for extra functionality:

#### ConfigValidator

Provide configuration schema:

```go
type ConfigValidator interface {
    GetConfigSchema() ConfigSchema
}
```

#### Reconnectable

Support custom reconnection policies:

```go
type Reconnectable interface {
    SetReconnectPolicy(policy ReconnectPolicy)
}
```

#### Filterable

Support server-side filtering:

```go
type Filterable interface {
    SetFilter(filter string) error
    GetFilterSyntax() string
}
```

### Plugin Adapter

The plugin adapter automatically converts plugin log entries to the format expected by Gonzo's processing pipeline. It can output either JSON or OTLP format:

- **JSON Format**: Default format, preserves all fields
- **OTLP Format**: Converts to OpenTelemetry format for consistency with OTLP receiver

The adapter handles:
- Channel management
- Format conversion
- Graceful shutdown
- Error handling

## Example Plugin Implementations

### Simple File Watcher Plugin

```go
package filewatcher

import (
    "bufio"
    "context"
    "os"
    "time"
    "github.com/control-theory/gonzo/internal/plugin"
)

type FileWatcherSource struct {
    filePath string
    logChan  chan plugin.LogEntry
}

func (f *FileWatcherSource) Start(ctx context.Context) (<-chan plugin.LogEntry, error) {
    go func() {
        defer close(f.logChan)
        
        file, err := os.Open(f.filePath)
        if err != nil {
            return
        }
        defer file.Close()
        
        scanner := bufio.NewScanner(file)
        for scanner.Scan() {
            select {
            case <-ctx.Done():
                return
            case f.logChan <- plugin.LogEntry{
                Timestamp: time.Now().UnixNano(),
                Raw:       scanner.Text(),
                Message:   scanner.Text(),
                Severity:  "INFO",
                Source: plugin.SourceInfo{
                    Type:       "filewatcher",
                    Identifier: f.filePath,
                },
            }:
            }
        }
    }()
    
    return f.logChan, nil
}
```

## Best Practices

1. **Error Handling**: Always handle errors gracefully and report them through metrics
2. **Context Cancellation**: Respect context cancellation for clean shutdown
3. **Channel Buffering**: Use buffered channels to prevent blocking
4. **Reconnection**: Implement automatic reconnection for network-based sources
5. **Metrics**: Track important metrics like logs/second, errors, and connection status
6. **Configuration Validation**: Validate configuration early in the `Validate()` method
7. **Resource Cleanup**: Clean up resources properly in the `Stop()` method

## Testing Your Plugin

```go
func TestYourPlugin(t *testing.T) {
    source := NewYourSource()
    
    // Configure
    err := source.Configure(map[string]interface{}{
        "url": "http://test.example.com",
    })
    if err != nil {
        t.Fatal(err)
    }
    
    // Validate
    err = source.Validate()
    if err != nil {
        t.Fatal(err)
    }
    
    // Start
    ctx := context.Background()
    logChan, err := source.Start(ctx)
    if err != nil {
        t.Fatal(err)
    }
    
    // Read logs
    select {
    case log := <-logChan:
        // Verify log entry
        if log.Message == "" {
            t.Error("Empty message")
        }
    case <-time.After(5 * time.Second):
        t.Error("Timeout waiting for logs")
    }
    
    // Stop
    source.Stop()
}
```

## Future Enhancements

Planned plugin sources:
- Elasticsearch
- AWS CloudWatch Logs
- Splunk
- Kafka
- Google Cloud Logging
- Azure Monitor Logs
- Datadog Logs

## Contributing

To contribute a new plugin:

1. Fork the repository
2. Create your plugin under `internal/plugin/yourplugin/`
3. Add tests for your plugin
4. Register it in `cmd/gonzo/plugins.go`
5. Update this documentation
6. Submit a pull request

Make sure your plugin:
- Follows Go best practices
- Has comprehensive tests
- Includes documentation
- Handles errors gracefully
- Respects context cancellation