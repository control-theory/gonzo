# Custom Log Formats Guide

This guide explains how to create and use custom log formats with Gonzo, allowing you to parse and analyze logs from any application or service.

## Table of Contents

1. [Overview](#overview)
2. [Quick Start](#quick-start)
3. [Format Types](#format-types)
4. [Format Configuration Structure](#format-configuration-structure)
5. [Pattern Matching](#pattern-matching)
6. [Field Mapping](#field-mapping)
7. [Examples](#examples)
8. [Advanced Features](#advanced-features)
9. [Troubleshooting](#troubleshooting)

## Overview

Gonzo supports custom log formats through YAML configuration files. These formats define how to parse log lines and map extracted fields to OpenTelemetry (OTLP) attributes that Gonzo uses for analysis and visualization.

### Why Custom Formats?

- **Flexibility**: Parse logs from any application without modifying the source
- **Consistency**: Convert various log formats to a unified OTLP structure
- **Reusability**: Share format definitions across teams and projects
- **Performance**: Optimized parsing for specific log structures

## Quick Start

### 1. Create a Format File

Create a YAML file in `~/.config/gonzo/formats/` directory:

```bash
mkdir -p ~/.config/gonzo/formats
vim ~/.config/gonzo/formats/myapp.yaml
```

### 2. Define Your Format

```yaml
name: myapp
description: My Application Log Format
type: text

pattern:
  use_regex: true
  main: '^(?P<timestamp>[\d\-T:\.]+)\s+\[(?P<level>\w+)\]\s+(?P<message>.*)$'

mapping:
  timestamp:
    field: timestamp
    time_format: rfc3339
  severity:
    field: level
  body:
    field: message
```

### 3. Use the Format

```bash
gonzo --format=myapp -f application.log
```

## Format Types

Gonzo supports three format types:

### 1. `text` - Plain Text Logs

For unstructured or semi-structured text logs:

```yaml
type: text
pattern:
  use_regex: true
  main: 'your-regex-pattern-here'
```

### 2. `json` - JSON Structured Logs

For logs in JSON format:

```yaml
type: json
json:
  fields:
    timestamp: $.timestamp
    message: $.msg
```

### 3. `structured` - Fixed Position Logs

For logs with consistent field positions (like Apache logs):

```yaml
type: structured
pattern:
  use_regex: true
  main: 'pattern-with-named-groups'
```

## Format Configuration Structure

### Basic Structure

```yaml
# Metadata
name: format-name           # Required: Unique identifier
description: Description    # Optional: Human-readable description
author: Your Name           # Optional: Format author
type: text|json|structured  # Required: Format type

# Pattern Configuration (for text/structured types)
pattern:
  use_regex: true|false     # Use regex or template matching
  main: "pattern"           # Main pattern for parsing
  fields:                   # Additional field patterns
    field_name: "pattern"

# JSON Configuration (for json type)
json:
  fields:                   # Field mappings
    internal_name: json_path
  array_path: "path"        # For nested arrays
  root_is_array: true|false # If root is an array

# Field Mapping
mapping:
  timestamp:                # Timestamp extraction
    field: field_name
    time_format: format
    default: value

  severity:                 # Log level/severity
    field: field_name
    transform: operation
    default: value

  body:                     # Main log message
    field: field_name
    template: "{{.field}}"

  attributes:               # Additional attributes
    attr_name:
      field: source_field
      pattern: "regex"
      transform: operation
      default: value
```

## Pattern Matching

### Using Regular Expressions

Use named capture groups to extract fields:

```yaml
pattern:
  use_regex: true
  # Named groups: (?P<name>pattern)
  main: '^(?P<timestamp>[\d\-T:]+)\s+(?P<level>\w+)\s+(?P<msg>.*)$'
```

#### Common Regex Patterns

| Pattern | Description | Example |
|---------|-------------|---------|
| `[\d\-T:\.]+` | ISO timestamp | 2024-01-15T10:30:45.123 |
| `\w+` | Word characters | ERROR, INFO |
| `\d+` | Digits | 12345 |
| `[^\]]+` | Everything except ] | Content inside brackets |
| `.*` | Any characters | Rest of line |
| `\S+` | Non-whitespace | Token or word |

### Template Syntax

For combining multiple fields:

```yaml
body:
  template: "{{.method}} {{.path}} - Status: {{.status}}"
```

## Field Mapping

### Core Fields

#### Timestamp

```yaml
timestamp:
  field: timestamp_field    # Source field name
  time_format: rfc3339      # Format specification
  default: ""               # Empty = use current time
```

**Supported time formats:**
- `rfc3339`: 2006-01-02T15:04:05Z07:00
- `unix`: Unix seconds (1234567890)
- `unix_ms`: Unix milliseconds
- `unix_ns`: Unix nanoseconds
- `auto`: Auto-detect format
- Custom Go format: "2006-01-02 15:04:05"

#### Severity

```yaml
severity:
  field: level
  transform: uppercase      # Normalize to uppercase
  default: INFO            # Default severity
```

**Standard severity levels:**
- TRACE
- DEBUG
- INFO
- WARN/WARNING
- ERROR
- FATAL/CRITICAL

#### Body

```yaml
body:
  field: message           # Single field
  # OR
  template: "{{.field1}} - {{.field2}}"  # Multiple fields
```

### Attributes

Additional metadata fields:

```yaml
attributes:
  hostname:
    field: host

  request_id:
    field: headers.request_id    # Nested field access

  duration_ms:
    field: duration
    pattern: '(\d+)ms'           # Extract from field
    default: "0"
```

### Transformations

Available transformations:

| Transform | Description | Example |
|-----------|-------------|---------|
| `uppercase` | Convert to uppercase | info → INFO |
| `lowercase` | Convert to lowercase | ERROR → error |
| `trim` | Remove whitespace | " text " → "text" |
| `status_to_severity` | Convert HTTP status to severity | 200→INFO, 404→WARN, 500→ERROR |

## Examples

### Example 1: Node.js Application Logs

```yaml
# Format for: [Backend] 5300 LOG [Module] Message +6ms
name: nodejs
type: text

pattern:
  use_regex: true
  main: '^\[(?P<project>[^\]]+)\]\s+(?P<pid>\d+)\s+(?P<level>\w+)\s+\[(?P<module>[^\]]+)\]\s+(?P<message>[^+]+?)(?:\s+\+(?P<duration>\d+)ms)?$'

mapping:
  severity:
    field: level
    transform: uppercase
  body:
    field: message
  attributes:
    project:
      field: project
    pid:
      field: pid
    module:
      field: module
    duration_ms:
      field: duration
      default: "0"
```

### Example 2: Kubernetes/Docker JSON Logs

```yaml
name: k8s-json
type: json

json:
  fields:
    timestamp: time
    message: log
    stream: stream

mapping:
  timestamp:
    field: timestamp
    time_format: rfc3339
  body:
    field: message
  attributes:
    stream:
      field: stream
    container_name:
      field: kubernetes.container_name
    pod_name:
      field: kubernetes.pod_name
    namespace:
      field: kubernetes.namespace_name
```

### Example 3: Apache Access Logs

```yaml
name: apache-access
type: structured

pattern:
  use_regex: true
  main: '^(?P<ip>[\d\.]+).*?\[(?P<timestamp>[^\]]+)\]\s+"(?P<method>\w+)\s+(?P<path>[^\s]+).*?"\s+(?P<status>\d+)\s+(?P<bytes>\d+)'

mapping:
  timestamp:
    field: timestamp
    time_format: "02/Jan/2006:15:04:05 -0700"
  body:
    template: "{{.method}} {{.path}} - {{.status}}"
  attributes:
    client_ip:
      field: ip
    http_method:
      field: method
    http_path:
      field: path
    http_status:
      field: status
    response_bytes:
      field: bytes
```

## Advanced Features

### Batch Processing

For log formats where a single line contains multiple log entries (like Loki's batch format), use the batch processing configuration:

```yaml
# Enable batch processing
batch:
  # Enable batch processing for this format
  enabled: true

  # Path pattern for array expansion - tells the system which arrays to expand
  # "streams[].values[]" means: expand the 'streams' array, then expand the 'values' array within each stream
  # Each combination creates a separate log entry (e.g., 2 streams × 3 values = 6 individual log entries)
  expand_path: "streams[].values[]"

  # Context paths - data to preserve/copy for each expanded entry
  # "streams[].stream" means: copy the 'stream' metadata from each stream to its expanded entries
  # This ensures each individual log entry retains its associated metadata
  context_paths: ["streams[].stream"]
```

**How batch processing works:**

1. **Original batch line:**
   ```json
   {"streams":[{"stream":{"service":"app","level":"ERROR"},"values":[["1234567890","Message 1"],["1234567891","Message 2"]]}]}
   ```

2. **Gets expanded to individual entries:**
   ```json
   {"streams":[{"stream":{"service":"app","level":"ERROR"},"values":[["1234567890","Message 1"]]}]}
   {"streams":[{"stream":{"service":"app","level":"ERROR"},"values":[["1234567891","Message 2"]]}]}
   ```

3. **Each entry is then processed normally** using the format's mapping configuration

**Batch configuration fields:**

- `enabled`: Set to `true` to enable batch processing
- `expand_path`: Specifies which arrays to expand (uses `[]` notation for arrays)
- `context_paths`: Metadata to preserve for each expanded entry
- `entry_template`: (Optional) Custom template for expanded entries

**Common batch patterns:**

| Pattern | Description | Example Use Case |
|---------|-------------|------------------|
| `streams[].values[]` | Expand streams, then values within each | Loki batch format |
| `logs[]` | Expand top-level logs array | Simple batch logs |
| `events[].entries[]` | Expand events, then entries within each | Event batch format |

### Nested JSON Fields

Access nested fields using dot notation:

```yaml
attributes:
  user_id:
    field: user.id
  user_name:
    field: user.profile.name
```

### Pattern Extraction

Extract values from within a field:

```yaml
attributes:
  error_code:
    field: message
    pattern: 'ERROR\[(\d+)\]'  # Extracts code from "ERROR[404]: Not found"
```

### Conditional Defaults

Use defaults when fields are missing:

```yaml
attributes:
  environment:
    field: env
    default: "production"
```

### HTTP Status Code to Severity Mapping

For web server logs, use the `status_to_severity` transform:

```yaml
severity:
  field: http_status
  transform: status_to_severity
```

**Status code mapping:**
- 1xx (100-199): DEBUG (Informational)
- 2xx (200-299): INFO (Success)
- 3xx (300-399): INFO (Redirection)
- 4xx (400-499): WARN (Client Error)
- 5xx (500-599): ERROR (Server Error)

### Multiple Pattern Matching

Define additional patterns for specific fields:

```yaml
pattern:
  use_regex: true
  main: '^(?P<base>.*)'
  fields:
    request_id: 'RequestID:\s*(\w+)'
    user_id: 'UserID:\s*(\d+)'
```

## Troubleshooting

### Common Issues

#### 1. Pattern Not Matching

**Problem**: Logs aren't being parsed correctly

**Solution**: Test your regex pattern:
- Use online regex testers (regex101.com)
- Check for special characters that need escaping
- Ensure named groups are correctly formatted: `(?P<name>...)`

#### 2. Timestamp Parsing Errors

**Problem**: Timestamps showing current time instead of log time

**Solution**: Verify time format:
```yaml
# For "2024-01-15 10:30:45"
time_format: "2006-01-02 15:04:05"

# For "15/Jan/2024:10:30:45 +0000"
time_format: "02/Jan/2006:15:04:05 -0700"
```

#### 3. Missing Fields

**Problem**: Expected fields aren't appearing in attributes

**Solution**: Check field paths and mappings:
- For JSON: Ensure correct path notation
- For regex: Verify capture groups are named
- Add defaults for optional fields

### Testing Your Format

1. **Create a test log file** with sample lines
2. **Run with test mode** to verify parsing:
   ```bash
   gonzo --format=yourformat -f test.log --test-mode
   ```
3. **Check the output** for correct field extraction

### Debug Tips

- Start with simple patterns and gradually add complexity
- Use the `--test-mode` flag to see parsing results without full TUI
- Check Gonzo's log output for parsing errors
- Test regex patterns separately before adding to format file

## Best Practices

1. **Document your format**: Add comments explaining patterns and fields
2. **Use meaningful names**: Choose descriptive names for captured fields
3. **Handle edge cases**: Use defaults for optional fields
4. **Test thoroughly**: Verify with various log samples
5. **Version control**: Keep format files in version control for team sharing
6. **Optimize patterns**: More specific patterns perform better than generic ones

## Sharing Formats

Format files can be shared across teams:

1. **Central repository**: Store formats in a shared Git repository
2. **Naming conventions**: Use consistent naming (e.g., `company-service.yaml`)
3. **Documentation**: Include example log lines in comments
4. **Validation**: Test formats before sharing

## Format Library

Gonzo includes several built-in formats in the `formats/` directory:

- `nodejs.yaml` - Node.js application logs
- `common-log.yaml` - Generic timestamp/level/message format
- `loki-stream.yaml` - Loki streaming format (individual entries)
- `loki-batch.yaml` - Loki batch format with automatic expansion
- `apache-combined.yaml` - Apache/Nginx access logs

Copy and modify these as starting points for your custom formats.

**Note**: The `loki-batch.yaml` format demonstrates the batch processing system for handling multi-entry log lines. Use it as a reference for creating other batch formats.

## Contributing

To contribute a format to the Gonzo project:

1. Create a well-documented format file
2. Include example log lines
3. Test with various inputs
4. Submit a pull request with the format file in the `formats/` directory

---

For more information, see the [Gonzo documentation](https://github.com/control-theory/gonzo) or file an issue if you need help with a specific log format.