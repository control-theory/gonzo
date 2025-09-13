# Gonzo with Stern Usage

Use [Stern](https://github.com/stern/stern) to stream Kubernetes logs to Gonzo for analysis.

## Basic Examples

```bash
# Get logs from all pods, all namespaces
stern . --all-namespaces --output json | gonzo

# Monitor specific namespace
stern . -n kube-system --output json | gonzo

# Get last 100 logs from namespace
stern . -n kube-system --tail 100 --output json | gonzo

# Monitor pods matching pattern
stern "api-*" -n production --output json | gonzo

# Get logs from last hour
stern . -n production --since 1h --output json | gonzo
```

## Key Options

- `--output json` - Preferred for Gonzo compatibility (populating attributes)
- `--tail N` - Limit historical logs
- `--since TIME` - Time-based filtering (1h, 30m, etc.)
- `-n NAMESPACE` - Target specific namespace
- `--all-namespaces` - Monitor all namespaces

## Tips

- Use specific pod patterns instead of `.` for better performance
- Always include `--output json` for proper Gonzo (attribute) processing
- Consider `--tail` to limit log volume for large clusters
