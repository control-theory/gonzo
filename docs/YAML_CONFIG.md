# YAML Configuration Files for Gonzo

## Overview

Gonzo supports YAML configuration files for defining multiple input sources and global settings. This provides a cleaner, more maintainable way to configure complex multi-source setups compared to command-line arguments.

## Usage

### Basic Usage

```bash
# Use a specific config file
gonzo --config config.yml

# Or place in default location
cp config.yml ~/.config/gonzo/config.yml
gonzo

# Config file with overrides
gonzo --config production.yml --update-interval=500ms
```

### Default Locations

Gonzo automatically looks for config files in these locations (in order):
1. `gonzo.yml` or `gonzo.yaml` (current directory)
2. `.gonzo.yml` or `.gonzo.yaml` (current directory)
3. `~/.config/gonzo/config.yml` or `config.yaml`

## Configuration Structure

### Complete Example

```yaml
# Global settings
memory-size: 20000           # Max entries in memory
update-interval: 1s           # Dashboard refresh rate
log-buffer: 2000             # Log buffer size
test-mode: false             # Test mode flag
ai-model: gpt-4              # AI model for analysis
skin: dracula                # UI color scheme

# Multiple input sources
sources:
  - name: production_logs     # Unique name for this source
    type: loki                # Source type (loki, files, otlp, vmlogs, stdin)
    config:                   # Source-specific configuration
      url: http://loki.prod:3100
      query: '{env="prod"}'
      since: 1h
      
  - name: local_files
    type: files
    config:
      files:
        - "/var/log/*.log"
      follow: true
```

## Source Types Reference

### 1. Files Source

```yaml
sources:
  - name: application_logs
    type: files
    config:
      files:                  # List of files or glob patterns
        - "/var/log/myapp/*.log"
        - "/var/log/nginx/access.log"
        - "./logs/**/*.json"  # Recursive glob
      follow: true            # Like tail -f (optional, default: false)
```

### 2. Loki Source

```yaml
sources:
  - name: loki_production
    type: loki
    config:
      url: https://loki.example.com:3100
      user: ${LOKI_USER}      # Environment variable
      password: ${LOKI_PASSWORD}
      query: |                # LogQL query (can be multiline)
        {job="myapp", env="production"}
        |= "error"
        |~ "timeout|refused"
      labels:                 # Additional label filters (optional)
        team: backend
        region: us-east
      since: 24h              # How far back to look (optional)
      limit: 10000            # Max logs to fetch (optional)
```

### 3. Victoria Logs Source

```yaml
sources:
  - name: vmlogs_staging
    type: vmlogs
    config:
      url: https://vmlogs.example.com:9428
      user: ${VMLOGS_USER}
      password: ${VMLOGS_PASSWORD}
      query: 'service:"api" AND level:error'  # LogsQL query
      params:                 # Additional parameters (optional)
        start_offset: 2h
        limit: 5000
```

### 4. OTLP Source

```yaml
sources:
  - name: otlp_receiver
    type: otlp
    config:
      grpc_port: 4317         # gRPC port (optional, default: 4317)
      http_port: 4318         # HTTP port (optional, default: 4318)
      grpc_enabled: true      # Enable gRPC (optional, default: true)
      http_enabled: true      # Enable HTTP (optional, default: true)
```

### 5. Stdin Source

```yaml
sources:
  - name: pipe_input
    type: stdin
    config: {}                # No configuration needed
```

## Environment Variables

Use environment variables for sensitive data:

```yaml
sources:
  - name: secure_source
    type: loki
    config:
      url: ${LOKI_URL}
      user: ${LOKI_USER}
      password: ${LOKI_PASSWORD}
      query: ${LOKI_QUERY:-{job="default"}}  # With default value
```

Set environment variables:
```bash
export LOKI_URL="https://loki.prod:3100"
export LOKI_USER="admin"
export LOKI_PASSWORD="secret"
gonzo --config config.yml
```

## Real-World Examples

### Example 1: Development Environment

```yaml
# dev.yml - Local development setup
memory-size: 5000
update-interval: 500ms
skin: monokai

sources:
  # Local application logs
  - name: app
    type: files
    config:
      files: ["./logs/app.log"]
      follow: true
      
  # Local Docker containers
  - name: docker
    type: files
    config:
      files: ["/var/lib/docker/containers/*/*.log"]
      follow: true
      
  # Staging environment
  - name: staging
    type: loki
    config:
      url: http://staging.loki:3100
      query: '{env="staging", app="myapp"}'
      since: 1h
```

### Example 2: Production Multi-Region

```yaml
# production.yml - Multi-region production monitoring
memory-size: 30000
update-interval: 2s
ai-model: gpt-4

sources:
  # US East Region
  - name: us_east_main
    type: loki
    config:
      url: https://loki-us-east.example.com
      user: ${US_EAST_USER}
      password: ${US_EAST_PASSWORD}
      query: '{region="us-east", env="prod"}'
      since: 2h
      
  - name: us_east_errors
    type: loki
    config:
      url: https://loki-us-east.example.com
      user: ${US_EAST_USER}
      password: ${US_EAST_PASSWORD}
      query: '{region="us-east", level=~"error|critical"}'
      since: 6h
      
  # EU Region
  - name: eu_central
    type: vmlogs
    config:
      url: https://vmlogs-eu.example.com:9428
      user: ${EU_USER}
      password: ${EU_PASSWORD}
      query: 'region:"eu-central" AND env:"prod"'
      
  # Asia Pacific
  - name: ap_south
    type: loki
    config:
      url: https://loki-ap.example.com
      user: ${AP_USER}
      password: ${AP_PASSWORD}
      query: '{region="ap-south"}'
      
  # Global OTLP collector
  - name: traces
    type: otlp
    config:
      grpc_port: 4317
```

### Example 3: Kubernetes Cluster

```yaml
# k8s.yml - Kubernetes cluster monitoring
memory-size: 15000
update-interval: 1s

sources:
  # Pod logs
  - name: pod_logs
    type: files
    config:
      files:
        - "/var/log/pods/**/containers/*/*.log"
      follow: true
      
  # System components
  - name: kube_system
    type: loki
    config:
      url: http://loki.kube-system:3100
      query: '{namespace="kube-system"}'
      
  # Application namespaces
  - name: app_prod
    type: loki
    config:
      url: http://loki.kube-system:3100
      query: '{namespace="production"}'
      
  - name: app_staging
    type: loki
    config:
      url: http://loki.kube-system:3100
      query: '{namespace="staging"}'
      
  # OTLP from services
  - name: otlp
    type: otlp
    config:
      grpc_port: 4317
```

### Example 4: Microservices Debugging

```yaml
# microservices.yml - Debug distributed system
memory-size: 10000
update-interval: 500ms
log-buffer: 5000

sources:
  # API Gateway
  - name: gateway
    type: loki
    config:
      url: http://loki:3100
      query: '{service="api-gateway"}'
      since: 30m
      
  # Core Services
  - name: auth
    type: loki
    config:
      url: http://loki:3100
      query: '{service="auth-service"}'
      since: 30m
      
  - name: users
    type: loki
    config:
      url: http://loki:3100
      query: '{service="user-service"}'
      since: 30m
      
  - name: orders
    type: loki
    config:
      url: http://loki:3100
      query: '{service="order-service"}'
      since: 30m
      
  # Databases
  - name: postgres
    type: files
    config:
      files: ["/var/log/postgresql/*.log"]
      follow: true
      
  - name: redis
    type: files
    config:
      files: ["/var/log/redis/*.log"]
      follow: true
      
  # Message Queue
  - name: rabbitmq
    type: files
    config:
      files: ["/var/log/rabbitmq/*.log"]
      follow: true
```

## Advanced Features

### Conditional Sources

Use shell scripting for conditional sources:

```bash
# Generate config based on environment
if [ "$ENV" = "production" ]; then
  cat production-sources.yml > gonzo.yml
else
  cat dev-sources.yml > gonzo.yml
fi
gonzo
```

### Dynamic Configuration

Generate config programmatically:

```python
# generate_config.py
import yaml
import os

config = {
    'memory-size': 10000,
    'sources': []
}

# Add sources based on environment
if os.getenv('INCLUDE_LOKI'):
    config['sources'].append({
        'name': 'loki',
        'type': 'loki',
        'config': {
            'url': os.getenv('LOKI_URL'),
            'query': '{}'
        }
    })

with open('gonzo.yml', 'w') as f:
    yaml.dump(config, f)
```

### Config Templates

Use templates for different environments:

```yaml
# base.yml
memory-size: 10000
update-interval: 1s

# prod.yml
!include base.yml
sources:
  - name: production
    type: loki
    config:
      url: ${PROD_LOKI_URL}
```

## Best Practices

1. **Use Environment Variables**: Never commit passwords or secrets
2. **Organize by Environment**: Separate configs for dev/staging/prod
3. **Comment Your Config**: Document why sources are included
4. **Version Control**: Track config files in git (without secrets)
5. **Validate Before Deploy**: Test config files locally first

## Troubleshooting

### Config Not Loading

```bash
# Check if config is being found
gonzo --config ./config.yml --log-level=debug

# Validate YAML syntax
python -c "import yaml; yaml.safe_load(open('config.yml'))"
```

### Environment Variables Not Expanding

```bash
# Check variable is set
echo $LOKI_URL

# Export if needed
export LOKI_URL="http://localhost:3100"
```

### Sources Not Connecting

```yaml
# Add debug info to config
sources:
  - name: debug_source
    type: loki
    config:
      url: http://localhost:3100  # Check URL is accessible
      query: '{}'  # Start with simple query
      since: 5m    # Use short time range for testing
```

## Migration from CLI Flags

### Before (CLI flags):
```bash
gonzo \
  --vmlogs-url="http://localhost:9428" \
  --vmlogs-query="*" \
  --otlp-enabled \
  --files="/var/log/*.log" \
  --follow
```

### After (YAML config):
```yaml
# config.yml
sources:
  - name: vmlogs
    type: vmlogs
    config:
      url: http://localhost:9428
      query: "*"
      
  - name: otlp
    type: otlp
    config:
      grpc_port: 4317
      http_port: 4318
      
  - name: files
    type: files
    config:
      files: ["/var/log/*.log"]
      follow: true
```

Then simply run:
```bash
gonzo --config config.yml
```

## Conclusion

YAML configuration files make it easy to:
- Define complex multi-source setups
- Share configurations across teams
- Maintain different configs for different environments
- Keep sensitive data in environment variables
- Version control your monitoring setup

Start with the provided examples and customize for your needs!