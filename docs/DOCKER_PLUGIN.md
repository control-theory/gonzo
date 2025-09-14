# Docker Plugin for Gonzo

The Docker plugin allows Gonzo to stream logs directly from Docker containers in real-time.

## Features

- Real-time log streaming from running Docker containers
- Flexible container filtering to follow specific containers
- Automatic discovery of new containers
- Metadata enrichment (container name, image, ID, etc.)
- Automatic severity detection from log content
- Support for both local and remote Docker daemons

## Configuration

### Basic Usage

```bash
# Follow all containers
gonzo --source='docker:'

# Follow specific containers by name
gonzo --source='docker:{"follow":["app","database"]}'

# Connect to remote Docker daemon
gonzo --source='docker:{"endpoint":"tcp://192.168.1.100:2376"}'
```

### Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `endpoint` | string | `unix:///var/run/docker.sock` | Docker daemon endpoint |
| `interval` | string | `10s` | Container discovery interval (minimum 5s) |
| `follow` | []string | `["*"]` | Container names/patterns to follow |

### Container Filtering

The `follow` option supports several matching patterns:

- **Exact match**: `"nginx"` - matches container named "nginx"
- **Wildcard**: `"*"` - matches all containers (default)
- **Pattern**: `"app-*"` - matches containers starting with "app-"
- **Partial match**: `"db"` - matches containers containing "db" in the name
- **Image match**: `"redis:latest"` - matches containers running this image

## Examples

### Single Docker Source

```bash
# Follow all containers
gonzo --source='docker:'

# Follow specific containers
gonzo --source='docker:{"follow":["myapp","postgres","redis"]}'

# Follow containers matching a pattern
gonzo --source='docker:{"follow":["frontend-*","backend-*"]}'

# Custom discovery interval
gonzo --source='docker:{"interval":"30s","follow":["production-*"]}'
```

### Multiple Sources Including Docker

```bash
# Combine Docker with other sources
gonzo --source='[
  {
    "name": "containers",
    "type": "docker",
    "config": {
      "follow": ["app-*", "db-*"],
      "interval": "15s"
    }
  },
  {
    "name": "system_logs",
    "type": "files",
    "config": {
      "files": ["/var/log/syslog"],
      "follow": true
    }
  }
]'
```

### Remote Docker Daemon

```bash
# Connect to remote Docker daemon
gonzo --source='docker:{"endpoint":"tcp://docker.example.com:2376"}'

# With TLS (requires certificates)
gonzo --source='docker:{"endpoint":"tcp://docker.example.com:2376"}'
```

## Log Entry Format

The Docker plugin enriches each log entry with container metadata:

**Attributes:**
- `container.id`: Full container ID
- `container.name`: Container name (without leading /)
- `container.image`: Image name with tag
- `container.image.id`: Image ID
- `container.status`: Container status
- `container.command`: Container command

**Resource:**
- `host.name`: Docker host name
- `docker.version`: Docker daemon version
- `service.name`: Container name (for service identification)

**Source:**
- `type`: "docker"
- `identifier`: Container name
- `metadata.container_id`: Short container ID (12 chars)
- `metadata.image`: Image name

## Severity Detection

The plugin automatically detects log severity based on content:

- **ERROR**: Contains "error", "err", "fatal", "panic", "failed"
- **WARN**: Contains "warn", "warning"
- **INFO**: Contains "info", "notice"
- **DEBUG**: Contains "debug", "trace", "verbose"
- **Default**: INFO (if no keywords found)

## Performance Considerations

- The plugin maintains one goroutine per followed container
- Log entries are buffered in a channel (buffer size: 1000)
- Container discovery runs periodically (configurable interval)
- Stopped containers are automatically cleaned up

## Troubleshooting

### Permission Denied

If you get permission errors accessing the Docker socket:

```bash
# Add user to docker group
sudo usermod -aG docker $USER

# Or run gonzo with sudo
sudo gonzo --source='docker:'
```

### No Containers Found

Check that containers are running:

```bash
docker ps
```

Verify the follow filter matches your containers:

```bash
# List all container names
docker ps --format "table {{.Names}}"
```

### Connection Issues

Test Docker daemon connectivity:

```bash
docker version
```

For remote daemons, ensure the endpoint is accessible:

```bash
docker -H tcp://docker.example.com:2376 version
```

## Integration with ctdocker

This plugin is adapted from the ctdocker OTEL receiver, simplified for Gonzo's streaming architecture:

- Removed OTEL dependencies
- Simplified to channel-based streaming
- Changed from exclusion to inclusion filtering
- Adapted configuration to Gonzo's plugin interface

The core Docker interaction logic remains similar, ensuring reliable container log collection.