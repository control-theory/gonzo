# Custom Format Quick Reference

## Basic Format Structure

```yaml
name: format-name
description: Description
type: text|json|structured

pattern:  # For text/structured
  use_regex: true
  main: "regex-pattern-with-(?P<name>...)-groups"

json:     # For json
  fields:
    internal_name: json_field_path

batch:    # For multi-entry logs (optional)
  enabled: true
  expand_path: "streams[].values[]"    # Array expansion pattern
  context_paths: ["streams[].stream"]  # Metadata to preserve

mapping:
  timestamp:
    field: field_name
    time_format: rfc3339|unix|unix_ms|custom
  severity:
    field: field_name
    transform: uppercase|lowercase|trim|status_to_severity
    default: INFO
  body:
    field: field_name
    # OR template: "{{.field1}} {{.field2}}"
  attributes:
    attr_name:
      field: source_field
      pattern: "regex"  # Extract from field
      default: value
```

## Common Regex Patterns

| Pattern | Matches | Example |
|---------|---------|---------|
| `(?P<timestamp>[\d\-T:\.Z]+)` | ISO timestamps | 2024-01-15T10:30:45.123Z |
| `(?P<level>\w+)` | Log levels | ERROR, INFO, DEBUG |
| `(?P<ip>[\d\.]+)` | IP addresses | 192.168.1.1 |
| `\[(?P<component>[^\]]+)\]` | Bracketed text | [Service] |
| `"(?P<method>\w+)\s+(?P<path>[^\s"]+)"` | HTTP requests | "GET /api/users" |
| `(?P<message>.*)$` | Rest of line | Any remaining text |

## Time Formats

| Format | Example | Description |
|--------|---------|-------------|
| `rfc3339` | 2024-01-15T10:30:45Z | ISO 8601 |
| `unix` | 1705316445 | Unix seconds |
| `unix_ms` | 1705316445123 | Unix milliseconds |
| `"2006-01-02 15:04:05"` | 2024-01-15 10:30:45 | Custom Go format |
| `"02/Jan/2006:15:04:05 -0700"` | 15/Jan/2024:10:30:45 +0000 | Apache format |

## Usage

```bash
# Use custom format
gonzo --format=myformat -f app.log

# List available formats
ls ~/.config/gonzo/formats/

# Test format without TUI
gonzo --format=myformat -f app.log --test-mode
```

## Format Types

- **`text`**: Plain text with regex patterns
- **`json`**: JSON logs with field extraction
- **`structured`**: Fixed-position logs (like Apache)

## Batch Processing

For logs where one line contains multiple entries:

```yaml
batch:
  enabled: true
  expand_path: "streams[].values[]"    # Which arrays to expand
  context_paths: ["streams[].stream"]  # Metadata to preserve
```

**Common patterns:**
- `logs[]` - Expand top-level array
- `streams[].values[]` - Expand nested arrays (Loki format)
- `events[].entries[]` - Multi-level expansion

## Transforms

- **`uppercase`**: Convert to uppercase (info → INFO)
- **`lowercase`**: Convert to lowercase (ERROR → error)
- **`trim`**: Remove whitespace (" text " → "text")
- **`status_to_severity`**: HTTP status to severity (200→INFO, 404→WARN, 500→ERROR)

## Common Issues

1. **Pattern not matching**: Test regex with online tools
2. **Wrong timestamp**: Check time format specification
3. **Missing attributes**: Verify field names and paths
4. **Performance**: Use specific patterns instead of `.*`