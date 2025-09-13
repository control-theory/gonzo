# Victoria Logs Usage Guide

## Overview
Gonzo now supports streaming logs directly from Victoria Logs with custom field parsing that properly maps Victoria Logs format to the internal OTLP format.

## Features

### Custom Field Parsing
- **`_msg`**: Mapped to the log body (displayed as the main message instead of raw JSON) 
- **`_time`**: Parsed as the log timestamp
- **All other fields**: Automatically mapped to attributes

### Special Field Mappings

#### Host Mapping
The following fields are checked in order and mapped to the `host` attribute:
1. `k8s.node.name`
2. `kubernetes.pod_node_name`
3. `kubernetes_pod_node_name`

#### Severity Detection
The following fields are checked for severity levels:
- `level`
- `severity`
- `log.level`
- `log_level`
- `loglevel`
- `levelname`

## Configuration

### Command Line Options
```bash
--vmlogs-url        Victoria Logs URL endpoint (e.g., http://localhost:9428)
--vmlogs-user       Basic auth username
--vmlogs-password   Basic auth password
--vmlogs-query      LogsQL query (default: "*")
```

### Environment Variables
```bash
export GONZO_VMLOGS_URL="http://localhost:9428"
export GONZO_VMLOGS_USER="myuser"
export GONZO_VMLOGS_PASSWORD="mypass"
export GONZO_VMLOGS_QUERY="service:'my-app'"
```

### Config File (config.yml)
```yaml
vmlogs-url: "http://localhost:9428"
vmlogs-user: "myuser"
vmlogs-password: "mypass"
vmlogs-query: "*"
```

## Usage Examples

### Basic Streaming
```bash
gonzo --vmlogs-url="http://localhost:9428" --vmlogs-query="*"
```

### With Authentication
```bash
gonzo --vmlogs-url="https://vmlogs.example.com" \
      --vmlogs-user="myuser" \
      --vmlogs-password="mypass" \
      --vmlogs-query='level:error'
```

### Filter by Service
```bash
gonzo --vmlogs-url="http://localhost:9428" \
      --vmlogs-query='service:"payment-processor" AND level:error'
```

### Using Environment Variables
```bash
export GONZO_VMLOGS_USER="myuser"
export GONZO_VMLOGS_PASSWORD="mypass"
gonzo --vmlogs-url="https://vmlogs.example.com" \
      --vmlogs-query='k8s.node.name:"node-01"'
```

## Victoria Logs JSON Format Example

Input from Victoria Logs:
```json
{
  "_msg": "Database connection established",
  "_stream": "app-logs",
  "_stream_id": "stream-001",
  "_time": "2024-01-15T10:30:46.456Z",
  "level": "INFO",
  "k8s.node.name": "node-02",
  "service": "database",
  "pool_size": 10
}
```

This will be displayed in Gonzo as:
- **Message**: "Database connection established" (not the raw JSON)
- **Severity**: INFO (with appropriate color coding)
- **Host**: node-02 (mapped from k8s.node.name)
- **Attributes**: service, pool_size, etc..

## Implementation Details

The Victoria Logs client (`internal/vmlogs/client.go`) includes:
1. **Streaming client** that connects to Victoria Logs `/select/logsql/tail` endpoint
2. **Format converter** that transforms Victoria Logs JSON to OTLP format
3. **Field mapper** that handles special fields and attributes
4. **Severity detector** that identifies log levels from various field names
