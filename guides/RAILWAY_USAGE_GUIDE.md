# Railway Usage Guide

Stream logs from your Railway services into Gonzo for real-time analysis, filtering, and AI-powered insights, all from your terminal.

## Overview

Railway's CLI includes built-in log streaming over WebSocket with JSON output. Gonzo auto-detects Railway's log format with zero configuration. No poller script, no normalizer, no `jq` transforms needed.

```bash
railway logs --json | gonzo
```

Works on all Railway plans (Trial, Hobby, Pro). Covers deployment logs (your app's stdout/stderr) and build logs.

## Prerequisites

- [Gonzo](https://github.com/control-theory/gonzo) installed (`brew install gonzo` or `go install github.com/control-theory/gonzo/cmd/gonzo@latest`)
- [Railway CLI](https://docs.railway.com/guides/cli) installed (`brew install railway`)
- A Railway project with at least one deployed service

## Setup

### 1. Authenticate the Railway CLI

```bash
railway login
```

### 2. Link to your project

```bash
railway link
```

Select your project and environment (e.g. `production`) when prompted.

### 3. Stream logs into Gonzo

```bash
railway logs --json | gonzo
```

That's it. Logs appear immediately via WebSocket streaming.

## Log Types

Railway exposes three log types. Two are available via the CLI:

| Log type | CLI flag | Gonzo compatible | What it captures |
|---|---|---|---|
| **Deployment logs** | `--json` (default) | ✅ Yes | Your app's stdout/stderr: application logs, errors, structured JSON output |
| **Build logs** | `--build --json` | ✅ Yes | Railpack/Nixpacks build output: dependency installs, compilation, image creation |
| **HTTP logs** | Dashboard only | ❌ Not via CLI | Railway edge proxy: request method, path, status, duration, source IP, edge region |

### Deployment logs (default)

Your application's stdout and stderr. If your app emits structured JSON logs, Railway preserves them as-is. Plain text logs are normalized to `{"message":"...","level":"..."}`.

```bash
railway logs --json | gonzo
```

Example output:
```json
{"timestamp":"2026-03-23T22:50:39.377Z","message":"GET /api/users","level":"info","ts":"2026-03-23T22:50:39.377Z"}
{"timestamp":"2026-03-23T22:50:44.487Z","message":"heartbeat","level":"debug","uptime":465.19}
```

### Build logs

Emitted during the build/deploy phase. Useful for debugging failed deployments.

```bash
railway logs --build --json --lines 50
```

Build logs include additional fields like `source`, `type`, `vertex`, `digest`, and `cached` from the Railpack/BuildKit pipeline, but always include `message`, `level`, and `timestamp` which Gonzo auto-detects.

### HTTP logs

Railway's edge proxy logs (request method, path, HTTP status, total duration, source IP, edge region) are **only available in the Railway dashboard** under the Observability tab → Network Flow Logs. They are not exposed via the CLI.

## Usage Patterns

**Real-time streaming** (default — WebSocket, no rate limits):
```bash
railway logs --json | gonzo
```

**Fetch last N lines** (uses GraphQL API, subject to rate limits):
```bash
railway logs --json --lines 100 | gonzo
```

**Fetch logs from a time range:**
```bash
railway logs --json --since 1h | gonzo
railway logs --json --since 2026-03-23T20:00:00Z --until 2026-03-23T21:00:00Z | gonzo
```

**Build logs after a failed deploy:**
```bash
railway logs --build --json --lines 100 | gonzo
```

**Filter logs using Railway's query syntax:**
```bash
railway logs --json --filter "@level:error" | gonzo
```

**Write to a file** so you can restart Gonzo without losing history:
```bash
railway logs --json > /tmp/railway-logs.jsonl &
gonzo -f /tmp/railway-logs.jsonl --follow
```

**Add AI analysis** with a local model (logs never leave your machine):
```bash
export OPENAI_API_KEY="ollama"
export OPENAI_API_BASE="http://localhost:11434"
railway logs --json | gonzo
```

## Structured Logging Tips

Railway auto-normalizes all logs. If your app emits structured JSON to stdout, Railway preserves the structure. Non-JSON output is wrapped in `{"message":"...","level":"..."}`.

For the best experience in Gonzo, emit structured JSON with at least `message` and `level`:

```javascript
console.log(JSON.stringify({
  message: "User signup completed",
  level: "info",
  userId: 123,
  provider: "github"
}));
```

Logs emitted to **stderr** are automatically tagged as `level: "error"` by Railway. This is a common gotcha for Python apps where the standard `logging` library defaults to stderr — all `logging.info()` calls appear as errors. Use structured JSON logging to override this.

## Rate Limits

**Streaming mode** (`railway logs --json | gonzo`) uses a persistent WebSocket connection and is **not rate limited**. This is the recommended mode for Gonzo.

**Fetch mode** (`--lines`, `--since`, `--until`) uses Railway's GraphQL API:

| Plan | Requests per hour |
|---|---|
| Free/Trial | 100 |
| Hobby | 1,000 |
| Pro | 10,000 |

**Platform log rate limit:** Railway enforces a hard cap of 500 log lines per second per replica. Logs exceeding this rate are silently dropped *before* they reach the CLI stream. Gonzo cannot recover logs that Railway never emits. If your app emits high-volume logs, use structured logging with minimal formatting (minified JSON rather than pretty-printed) to stay under the limit. Pretty-printing a JSON object turns one log entry into many lines, each counting against the cap.

## Filtering in Gonzo

Once logs are streaming, use Gonzo's built-in filtering:

| Key | Action |
|---|---|
| `/` | Open filter — type text to search |
| `i` | AI analysis of selected log entry |
| `Tab` | Switch between panels |
| `q` | Quit |

Example filters:

| Filter | What you'll see |
|---|---|
| `error` | All error-level logs |
| `GET /api` | HTTP request logs matching a path |
| `heartbeat` | Background task logs |
| Service name (e.g. `checkout`) | Logs from a specific service (multi-service projects) |

## Multi-Service Projects

If your Railway project has multiple services, the CLI will prompt you to select one when you run `railway logs`. To stream logs from a specific service:

```bash
railway logs --json --service <service-id> | gonzo
```

## Troubleshooting

**"No linked project found"**: Run `railway link` to connect your terminal session to a project.

**Logs appear empty**: Your service may not have an active deployment. Check `railway status` or the Railway dashboard.

**Missing logs / gaps**: If your app emits more than 500 lines/second, Railway silently drops the excess. Reduce log volume or switch to minified JSON output.

**Build logs not showing**: Use `railway logs --build --json --lines 50` with the `--lines` flag, as build logs are only available in fetch mode after the build completes.

## Contributing

This guide lives in the [Gonzo repo](https://github.com/control-theory/gonzo):

- Guide: `guides/RAILWAY_USAGE_GUIDE.md`

If you run into issues or find edge cases with Railway's log format, PRs and issues are welcome. Gonzo is open source (MIT) and community-driven.
