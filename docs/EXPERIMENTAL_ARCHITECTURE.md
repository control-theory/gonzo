# Unified Plugin Architecture

## Overview

Gonzo now supports a unified plugin architecture that standardizes all log input sources (stdin, files, OTLP, Victoria Logs, Loki, etc.) under a single interface. This provides consistency, better maintainability, and easier extensibility.

## Benefits of Unified Architecture

1. **Consistency**: All log sources use the same `LogSource` interface
2. **Unified Metrics**: Consistent metrics across all sources
3. **Simplified Codebase**: Single code path for all input sources
4. **Easy Extension**: Adding new sources only requires implementing the interface
5. **Better Testing**: Each source can be tested independently
6. **Backward Compatibility**: Existing CLI flags still work

## Architecture Components

### Core Interface (`LogSource`)

```go
type LogSource interface {
    Name() string
    Description() string
    Configure(config map[string]interface{}) error
    Validate() error
    Start(ctx context.Context) (<-chan LogEntry, error)
    Stop() error
    GetMetrics() Metrics
}
```

### Built-in Sources

All existing input methods are now available as plugins:

| Source | Plugin Name | Description |
|--------|------------|-------------|
| Standard Input | `stdin` | Read from pipe or redirect |
| Files | `files` | Read from local files with glob support |
| OTLP | `otlp` | OpenTelemetry Protocol (gRPC/HTTP) |
| Victoria Logs | `vmlogs` | Stream from Victoria Logs |
| Loki | `loki` | Stream from Grafana Loki |

## Usage

### Using the Unified Architecture

Enable the unified architecture with the `--use-unified` flag:

```bash
# Read from stdin with experimental architecture
cat logs.txt | gonzo --use-experimental

# Read from files
gonzo --use-experimental --plugin-source=files --plugin-config='{"files":["/var/log/*.log"],"follow":true}'

# Stream from Loki
gonzo --use-experimental --plugin-source=loki --plugin-config='{"url":"http://localhost:3100","query":"{job=\"myapp\"}"}'
```

### Backward Compatibility

The existing CLI flags continue to work and are automatically mapped to the plugin system:

```bash
# These commands work the same as before:
gonzo -f application.log --follow
gonzo --otlp-enabled
gonzo --vmlogs-url="http://localhost:9428"

# Behind the scenes, they're converted to:
# --plugin-source=files --plugin-config='{"files":["application.log"],"follow":true}'
# --plugin-source=otlp --plugin-config='{"grpc_port":4317,"http_port":4318}'
# --plugin-source=vmlogs --plugin-config='{"url":"http://localhost:9428","query":"*"}'
```

## Migration Path

### Phase 1: Current State (Completed)
- ✅ Plugin interface defined
- ✅ Loki plugin implemented as example
- ✅ Wrappers created for existing sources
- ✅ Backward compatibility maintained
- ✅ Experimental architecture available via `--use-experimental` flag

### Phase 2: Testing & Stabilization
- Test experimental architecture with various workloads
- Gather feedback and fix issues
- Optimize performance
- Add more external source plugins

### Phase 3: Default to Unified
- Make experimental architecture the default (once stable)
- Keep legacy code for fallback
- Update documentation

### Phase 4: Full Migration
- Remove legacy code paths
- Simplify codebase
- All sources use plugin architecture

## Configuration Examples

### Stdin
```json
{}  // No configuration needed
```

### Files
```json
{
  "files": ["/var/log/*.log", "app.log"],
  "follow": true
}
```

### OTLP
```json
{
  "grpc_port": 4317,
  "http_port": 4318,
  "grpc_enabled": true,
  "http_enabled": true
}
```

### Victoria Logs
```json
{
  "url": "http://localhost:9428",
  "user": "admin",
  "password": "secret",
  "query": "service:myapp AND level:error",
  "params": {"start_offset": "1h"}
}
```

### Loki
```json
{
  "url": "http://localhost:3100",
  "user": "admin",
  "password": "secret",
  "query": "{job=\"nginx\"} |= \"error\"",
  "labels": {"env": "prod"},
  "limit": 5000,
  "since": "24h"
}
```

## Performance Considerations

The experimental architecture has minimal performance overhead:

- **Channel Buffering**: Each plugin uses buffered channels (100 entries)
- **Goroutine per Source**: Each source runs in its own goroutine
- **Metrics Collection**: Lightweight metrics updated every second
- **Format Conversion**: OTLP conversion happens once at the adapter level

## Extending with Custom Plugins

See [PLUGIN_ARCHITECTURE.md](PLUGIN_ARCHITECTURE.md) for detailed instructions on creating custom plugins.

## Troubleshooting

### Checking Plugin Status

When using the experimental architecture, Gonzo logs plugin status every 30 seconds:

```
[loki] Status: Connected | Logs: 15234 total, 125.3/sec
```

### Common Issues

1. **Plugin not found**: Ensure the plugin is registered in `cmd/gonzo/plugins.go`
2. **Configuration errors**: Check JSON syntax in `--plugin-config`
3. **Connection failures**: Verify network connectivity and authentication

## Future Improvements

1. **Dynamic Plugin Loading**: Load plugins from external binaries
2. **Plugin Marketplace**: Central repository for community plugins
3. **Configuration Files**: Support for YAML/TOML configuration files
4. **Hot Reload**: Change sources without restarting
5. **Multiple Sources**: Read from multiple sources simultaneously
6. **Source Routing**: Route different sources to different analyzers

## Conclusion

The experimental plugin architecture (currently in beta) makes Gonzo more maintainable, extensible, and consistent. It provides a clear path for adding new log sources while maintaining backward compatibility with existing workflows.