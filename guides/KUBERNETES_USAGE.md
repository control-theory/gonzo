# Kubernetes Integration Guide

Gonzo provides native Kubernetes integration for streaming logs directly from your clusters. This guide covers installation, configuration, and usage patterns for Kubernetes log analysis.

## Table of Contents

- [Overview](#overview)
- [Prerequisites](#prerequisites)
- [Quick Start](#quick-start)
- [Configuration Options](#configuration-options)
- [Interactive Filtering](#interactive-filtering)
- [Display Modes](#display-modes)
- [Common Use Cases](#common-use-cases)
- [Troubleshooting](#troubleshooting)

## Overview

Gonzo's Kubernetes integration provides:

- **Direct cluster access** - No need to pipe kubectl output
- **Multi-namespace support** - Watch multiple namespaces simultaneously
- **Label selectors** - Filter pods by Kubernetes labels
- **Interactive filtering** - Dynamic namespace and pod filtering with `Ctrl+k`
- **Auto-detection** - Automatically displays namespace and pod columns for k8s logs
- **Real-time streaming** - Live tail of pod logs with automatic reconnection

## Prerequisites

Before using Gonzo with Kubernetes, ensure you have:

1. **Kubernetes cluster access** - Valid kubeconfig file
2. **Gonzo installed** - See main [README](../README.md) for installation

### Required Kubernetes Permissions

Your Kubernetes user/service account needs these permissions:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: gonzo-log-reader
rules:
- apiGroups: [""]
  resources: ["pods", "pods/log"]
  verbs: ["get", "list", "watch"]
- apiGroups: [""]
  resources: ["namespaces"]
  verbs: ["get", "list"]
```

## Quick Start

### Watch All Pods in All Namespaces

```bash
# Stream logs from all pods in all namespaces
gonzo --k8s-enabled=true

# Show last 50 lines from each pod
gonzo --k8s-enabled=true --k8s-tail=50
```

### Watch Specific Namespaces

```bash
# Single namespace
gonzo --k8s-enabled=true --k8s-namespace=production

# Multiple namespaces
gonzo --k8s-enabled=true --k8s-namespace=production --k8s-namespace=staging
```

### Filter by Labels

```bash
# Watch pods with specific label
gonzo --k8s-enabled=true --k8s-selector="app=nginx"

# Complex label selector
gonzo --k8s-enabled=true --k8s-selector="app=nginx,tier=frontend"
gonzo --k8s-enabled=true --k8s-selector="environment in (production,staging)"
```

### Combine Filters

```bash
# Specific namespace with label selector
gonzo --k8s-enabled=true \
  --k8s-namespace=production \
  --k8s-selector="app=api"

# Multiple namespaces with label selector
gonzo --k8s-enabled=true \
  --k8s-namespace=production \
  --k8s-namespace=staging \
  --k8s-selector="tier=backend"
```

## Configuration Options

### Command Line Flags

```bash
--k8s-enabled=true          # Enable Kubernetes mode
--k8s-namespace NAMESPACE   # Target namespace (can specify multiple times)
--k8s-selector SELECTOR     # Kubernetes label selector
--k8s-tail N                # Number of previous log lines per pod (default: 10)
--k8s-since SECONDS         # Only logs newer than N seconds
--k8s-kubeconfig PATH       # Path to kubeconfig (default: ~/.kube/config)
--k8s-context CONTEXT       # Kubernetes context to use
```

### Configuration File

Add to `~/.config/gonzo/config.yml`:

```yaml
# Enable Kubernetes mode
k8s:
  enabled: true

  # Target namespaces (empty = all namespaces)
  namespaces:
    - production
    - staging

  # Label selector for filtering pods
  selector: "app=nginx,tier=frontend"

  # Number of historical log lines per pod
  tail: 50

  # Only logs newer than N seconds
  since: 3600  # Last hour

  # Path to kubeconfig file
  kubeconfig: ~/.kube/config

  # Kubernetes context to use
  context: my-cluster
```

See [examples/k8s_config.yml](../examples/k8s_config.yml) for a complete example.

### Environment Variables

```bash
# Use specific kubeconfig
export KUBECONFIG=/path/to/custom/kubeconfig

# Set default Kubernetes context
export KUBE_CONTEXT=production-cluster
```

## Interactive Filtering

Gonzo provides an interactive filtering modal for Kubernetes logs accessible with `Ctrl+k`.

### Features

- **Namespace tab** - Select which namespaces to monitor
- **Pod tab** - Select specific pods to watch
- **Live updates** - Applies filters in real-time
- **Select all/none** - Quick bulk operations
- **Persistent** - Selections persist across modal opens

### Usage

1. Press `Ctrl+k` to open the Kubernetes filter modal
2. Use `Tab` to switch between Namespaces and Pods views
3. Navigate with arrow keys (`↑`/`↓` or `j`/`k`)
4. Press `Space` to toggle selection
5. Press `Enter` to apply filters
6. Press `ESC` to cancel changes

### Keyboard Shortcuts

| Key                | Action                         |
| ------------------ | ------------------------------ |
| `Ctrl+k`           | Open Kubernetes filter modal   |
| `Tab`              | Switch between tabs            |
| `↑`/`↓` or `j`/`k` | Navigate items                 |
| `Space`            | Toggle selection               |
| `Enter`            | Apply filter and close         |
| `ESC`              | Cancel and close               |

## Display Modes

### K8s Mode (Auto-Detected)

When Gonzo detects Kubernetes attributes (`k8s.namespace`, `k8s.pod`), it automatically switches to K8s display mode:

```
Time     Level Namespace            Pod                  Message
15:04:05 INFO  production           nginx-7d9c-xkr2p     Request handled successfully
15:04:06 ERROR production           api-server-5c4f-m89x Failed to connect to database
15:04:07 WARN  staging              worker-2b3a-qz8l     High memory usage detected
```

**Column Layout:**
- **Time** - Log timestamp (8 chars)
- **Level** - Severity level (5 chars)
- **Namespace** - K8s namespace (20 chars, truncated with "...")
- **Pod** - Pod name (20 chars, truncated with "...")
- **Message** - Log message (remaining width)

### Toggle Columns

Press `c` to toggle column display on/off:

```
# With columns (default)
15:04:05 INFO  production           nginx-7d9c-xkr2p     Request handled

# Without columns
15:04:05 INFO  Request handled successfully
```

### Standard Mode

For non-Kubernetes logs, Gonzo displays host and service columns:

```
Time     Level Host         Service          Message
15:04:05 INFO  server01     api-gateway      Request handled
```

## Common Use Cases

### Development Workflow

```bash
# Watch your development namespace
gonzo --k8s-enabled=true --k8s-namespace=dev --k8s-selector="app=myapp"

# Quick check of specific pod
gonzo --k8s-enabled=true --k8s-namespace=dev --k8s-selector="app=myapp,version=v1.2.3"
```

### Production Monitoring

```bash
# Monitor production with error focus (using severity filter)
gonzo --k8s-enabled=true --k8s-namespace=production

# Then press Ctrl+f and select only ERROR and FATAL levels
```

### Multi-Environment Monitoring

```bash
# Watch both production and staging
gonzo --k8s-enabled=true \
  --k8s-namespace=production \
  --k8s-namespace=staging \
  --k8s-selector="tier=backend"
```

### Troubleshooting Deployments

```bash
# Check recent deployment logs
gonzo --k8s-enabled=true \
  --k8s-namespace=production \
  --k8s-selector="app=nginx,version=v2.0.0" \
  --k8s-since=300  # Last 5 minutes
```

### CI/CD Pipeline Integration

```bash
# Monitor deployment in CI/CD
#!/bin/bash
NAMESPACE="production"
APP="myapp"
VERSION="v1.2.3"

# Start monitoring
gonzo --k8s-enabled=true \
  --k8s-namespace=$NAMESPACE \
  --k8s-selector="app=$APP,version=$VERSION" \
  --k8s-tail=100 &

GONZO_PID=$!

# Run deployment
kubectl apply -f deployment.yaml

# Wait for rollout
kubectl rollout status deployment/$APP -n $NAMESPACE

# Stop monitoring
kill $GONZO_PID
```

### Label Selector Examples

```bash
# Single label
gonzo --k8s-enabled=true --k8s-selector="app=nginx"

# Multiple labels (AND)
gonzo --k8s-enabled=true --k8s-selector="app=nginx,tier=frontend"

# Set-based requirements
gonzo --k8s-enabled=true --k8s-selector="environment in (production,staging)"
gonzo --k8s-enabled=true --k8s-selector="tier notin (test,dev)"

# Existence check
gonzo --k8s-enabled=true --k8s-selector="critical"
gonzo --k8s-enabled=true --k8s-selector="!experimental"

# Complex combinations
gonzo --k8s-enabled=true --k8s-selector="app=nginx,environment in (prod,stage),!experimental"
```

## Troubleshooting

### No Logs Appearing

**Check cluster access:**
```bash
# Verify kubectl works
kubectl get pods --all-namespaces

# Check specific namespace
kubectl get pods -n production
```

**Check permissions:**
```bash
# Verify you can read logs
kubectl logs <pod-name> -n <namespace>
```

**Check pod status:**
```bash
# Ensure pods are running
kubectl get pods -n <namespace> --selector=<your-selector>
```

### Connection Issues

**Verify kubeconfig:**
```bash
# Check current context
kubectl config current-context

# List available contexts
kubectl config get-contexts

# Use specific context
gonzo --k8s-enabled=true --k8s-context=my-cluster
```

**Check network connectivity:**
```bash
# Test cluster API access
kubectl cluster-info

# Check pod network
kubectl get pods -A
```

### Filter Not Working

**Verify label selector syntax:**
```bash
# Test selector with kubectl first
kubectl get pods --selector="app=nginx" -A

# Then use same selector with Gonzo
gonzo --k8s-enabled=true --k8s-selector="app=nginx"
```

**Check namespace exists:**
```bash
# List all namespaces
kubectl get namespaces

# Verify specific namespace
kubectl get namespace production
```

### Performance Issues

**Reduce log volume:**
```bash
# Use more specific selectors
gonzo --k8s-enabled=true \
  --k8s-namespace=production \
  --k8s-selector="app=api,critical=true"

# Limit to recent logs
gonzo --k8s-enabled=true --k8s-tail=10 --k8s-since=300
```

**Adjust buffer size:**
```bash
# Increase buffer for high-volume logs
gonzo --k8s-enabled=true --log-buffer=5000
```

### Debug Mode

```bash
# Run with verbose output
GONZO_DEBUG=1 gonzo --k8s-enabled=true --k8s-namespace=default
```

## Best Practices

1. **Start specific** - Use namespace and selector filters to reduce noise
2. **Use interactive filters** - Press `Ctrl+k` to dynamically adjust filters
3. **Leverage severity filtering** - Press `Ctrl+f` to focus on errors
4. **Monitor resources** - Watch `--log-buffer` usage for high-volume clusters
5. **Use contexts** - Switch between clusters with `--k8s-context`
6. **Save configs** - Store common configurations in `~/.config/gonzo/config.yml`

## Next Steps

- [Main README](../README.md) - General Gonzo usage
- [USAGE_GUIDE.md](../USAGE_GUIDE.md) - Detailed feature guide
- [Examples](../examples/k8s_config.yml) - Sample configurations

## Getting Help

- [GitHub Issues](https://github.com/control-theory/gonzo/issues)
- [Slack Community](https://ctrltheorycommunity.slack.com)
- [Documentation](https://docs.controltheory.com/)
