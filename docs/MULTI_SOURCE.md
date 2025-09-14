# Multiple Simultaneous Input Sources

## Overview

Gonzo now supports reading from multiple log sources simultaneously using the `--multi-source` flag. This allows you to aggregate and analyze logs from various sources in a single unified view.

## Key Features

- **Simultaneous Streaming**: Read from multiple sources concurrently
- **Mixed Source Types**: Combine local files, remote services, and network protocols
- **Unified Processing**: All logs are processed through the same analysis pipeline
- **Per-Source Metrics**: Track metrics for each individual source
- **Flexible Configuration**: Simple or detailed JSON configuration

## Usage

### Simple Format

Use semicolon-separated source specifications:

```bash
# Combine stdin with files
cat live.log | gonzo --multi-source='stdin:;files:{"files":["/var/log/*.log"]}'

# Multiple file sources
gonzo --multi-source='files:{"files":["app.log"]};files:{"files":["error.log"]}'

# Mix local and remote
gonzo --multi-source='files:{"files":["local.log"],"follow":true};loki:{"url":"http://loki:3100"}'
```

### JSON Format

Use JSON for more control and named sources:

```bash
gonzo --multi-source='{
  "sources": [
    {
      "name": "application_logs",
      "type": "files",
      "config": {
        "files": ["/app/logs/*.log"],
        "follow": true
      }
    },
    {
      "name": "loki_errors",
      "type": "loki",
      "config": {
        "url": "http://localhost:3100",
        "query": "{job=\"app\"} |= \"error\""
      }
    },
    {
      "name": "otlp_traces",
      "type": "otlp",
      "config": {
        "grpc_port": 4317
      }
    }
  ]
}'
```

## Real-World Examples

### 1. Development Environment

Monitor local development logs alongside staging environment:

```bash
gonzo --multi-source='{
  "sources": [
    {
      "name": "local_app",
      "type": "files",
      "config": {"files": ["./logs/app.log"], "follow": true}
    },
    {
      "name": "local_docker",
      "type": "stdin",
      "config": {}
    },
    {
      "name": "staging_logs",
      "type": "loki",
      "config": {
        "url": "http://staging.loki:3100",
        "query": "{env=\"staging\",app=\"myapp\"}"
      }
    }
  ]
}'
```

### 2. Production Monitoring

Aggregate logs from multiple production sources:

```bash
gonzo --multi-source='{
  "sources": [
    {
      "name": "prod_loki",
      "type": "loki",
      "config": {
        "url": "http://loki.prod:3100",
        "query": "{env=\"prod\"}",
        "since": "1h"
      }
    },
    {
      "name": "prod_vmlogs",
      "type": "vmlogs",
      "config": {
        "url": "http://vmlogs.prod:9428",
        "query": "level:error OR level:warn"
      }
    },
    {
      "name": "prod_otlp",
      "type": "otlp",
      "config": {
        "grpc_port": 4317,
        "http_port": 4318
      }
    }
  ]
}'
```

### 3. Debugging Distributed Systems

Correlate logs from multiple microservices:

```bash
gonzo --multi-source='{
  "sources": [
    {
      "name": "api_gateway",
      "type": "loki",
      "config": {
        "url": "http://loki:3100",
        "query": "{service=\"api-gateway\"}"
      }
    },
    {
      "name": "auth_service",
      "type": "loki",
      "config": {
        "url": "http://loki:3100",
        "query": "{service=\"auth\"}"
      }
    },
    {
      "name": "database_logs",
      "type": "files",
      "config": {
        "files": ["/var/log/postgresql/*.log"],
        "follow": true
      }
    }
  ]
}'
```

### 4. Cross-Region Log Aggregation

Combine logs from multiple regions:

```bash
gonzo --multi-source='{
  "sources": [
    {
      "name": "us_east",
      "type": "loki",
      "config": {
        "url": "http://loki.us-east.example.com:3100",
        "query": "{region=\"us-east\"}"
      }
    },
    {
      "name": "eu_west",
      "type": "loki",
      "config": {
        "url": "http://loki.eu-west.example.com:3100",
        "query": "{region=\"eu-west\"}"
      }
    },
    {
      "name": "ap_south",
      "type": "vmlogs",
      "config": {
        "url": "http://vmlogs.ap-south.example.com:9428",
        "query": "*"
      }
    }
  ]
}'
```

## Source Types

All built-in sources are supported:

| Source Type | Description | Key Config Options |
|------------|-------------|-------------------|
| `stdin` | Standard input | None required |
| `files` | Local files | `files`, `follow` |
| `otlp` | OpenTelemetry | `grpc_port`, `http_port` |
| `vmlogs` | Victoria Logs | `url`, `query`, `user`, `password` |
| `loki` | Grafana Loki | `url`, `query`, `labels`, `since` |

## How It Works

### Architecture

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   Source 1  │     │   Source 2  │     │   Source N  │
│   (files)   │     │   (loki)    │     │   (otlp)    │
└──────┬──────┘     └──────┬──────┘     └──────┬──────┘
       │                   │                   │
       ▼                   ▼                   ▼
┌────────────────────────────────────────────────────┐
│                   Multiplexer                      │
│  - Concurrent reading from all sources             │
│  - Per-source buffering                            │
│  - Unified output channel                          │
└─────────────────────┬──────────────────────────────┘
                      │
                      ▼
┌────────────────────────────────────────────────────┐
│              Processing Pipeline                   │
│  - Format detection                                │
│  - OTLP conversion                                 │
│  - Analysis & pattern extraction                   │
└─────────────────────┬──────────────────────────────┘
                      │
                      ▼
┌────────────────────────────────────────────────────┐
│                  Dashboard UI                      │
│  - Unified visualization                           │
│  - Combined metrics                                │
│  - Source identification                           │
└────────────────────────────────────────────────────┘
```

### Processing

1. Each source runs in its own goroutine
2. Logs are tagged with source metadata
3. Multiplexer combines streams into single channel
4. Processing pipeline handles all logs uniformly
5. Dashboard shows aggregated view

## Monitoring

### Source Metrics

Gonzo reports metrics for each source every 30 seconds:

```
[Multiplexer/application_logs] Logs: 15234 total, 125.3/sec
[Multiplexer/loki_errors] Logs: 8421 total, 52.1/sec
[Multiplexer/otlp_traces] Logs: 29183 total, 243.2/sec
```

### Source Identification

Each log entry includes source metadata:
- `multiplexer_source`: Name of the source
- Original source metadata is preserved

## Performance Considerations

### Buffering
- Each source has its own buffer (100 entries)
- Multiplexer output buffer: 1000 entries
- Prevents slow sources from blocking fast ones

### Concurrency
- All sources read concurrently
- Independent goroutines per source
- Non-blocking channel operations

### Resource Usage
- Memory: ~10MB per source baseline
- CPU: Scales with log volume
- Network: Depends on remote sources

## Best Practices

1. **Name Your Sources**: Use descriptive names in JSON format for easier identification
2. **Balance Sources**: Avoid mixing very high and very low volume sources
3. **Use Filters**: Apply source-side filters to reduce data volume
4. **Monitor Metrics**: Watch per-source metrics to identify bottlenecks
5. **Test Incrementally**: Start with few sources, add more gradually

## Limitations

- Maximum recommended sources: 10-20 (depending on volume)
- All sources share the same processing pipeline
- No per-source filtering after ingestion (use source-side filters)
- Sources cannot be added/removed dynamically (requires restart)

## Troubleshooting

### Source Not Connecting
```
[Multiplexer/prod_loki] Error: connection refused
```
- Check network connectivity
- Verify authentication credentials
- Confirm source configuration

### Channel Full Warnings
```
Warning: Multiplexer channel full, dropping log from source_name
```
- Reduce log volume with source filters
- Increase buffer sizes (requires code change)
- Remove high-volume sources

### Uneven Source Distribution
- Use source-side filtering
- Adjust query time ranges
- Consider separate gonzo instances

## Future Enhancements

- [ ] Dynamic source management (add/remove without restart)
- [ ] Per-source processing pipelines
- [ ] Source priority levels
- [ ] Automatic load balancing
- [ ] Source health checks
- [ ] Configuration hot-reload

## Migration from Single Source

Existing single-source commands can be easily converted:

### Before (Single Source)
```bash
gonzo --plugin-source=loki --plugin-config='{"url":"http://loki:3100"}'
```

### After (Multi-Source)
```bash
gonzo --multi-source='loki:{"url":"http://loki:3100"}'
```

Or add more sources:
```bash
gonzo --multi-source='loki:{"url":"http://loki:3100"};files:{"files":["local.log"]}'
```

## Conclusion

The multi-source feature transforms Gonzo into a powerful log aggregation tool, capable of providing unified insights across your entire infrastructure. Whether you're debugging distributed systems, monitoring production environments, or analyzing logs across regions, multi-source support provides the flexibility and power you need.