# Render Usage Guide

Stream logs from your Render services into Gonzo for real-time analysis, filtering, and AI-powered insights, all from your terminal.

## Quick Start

```bash
render logs -r <service-id> -o json --tail \
  | jq --unbuffered -c '{timestamp: .timestamp, message: .message} + ([.labels[] | {(.name): .value}] | add)' \
  | sed -u 's/\\u001b\[[0-9;]*m//g; s/\\u001b(B//g' \
  | gonzo
```

Render's CLI streams logs over WebSocket. The `jq` transform flattens Render's label metadata into top-level fields. The `sed -u` strips ANSI color codes that Render injects into build and deploy messages (e.g. `\u001b[32m` around `==> Build successful 🎉`) which would otherwise render as garbage in Gonzo. The `-u` flag disables sed's output buffering - without it, logs can get stalled/delayed in the pipe.

## Prerequisites

- [Gonzo](https://github.com/control-theory/gonzo) installed
- [Render CLI](https://render.com/docs/cli) installed (`brew install render`)
- `jq` installed (`brew install jq`)

## Setup

```bash
# 1. Authenticate
render login

# 2. Find your service ID (starts with srv-)
render services -o json | jq '.[].service | {id, name}'

# 3. Stream into Gonzo
render logs -r srv-XXXXX -o json --tail \
  | jq --unbuffered -c '{timestamp: .timestamp, message: .message} + ([.labels[] | {(.name): .value}] | add)' \
  | sed -u 's/\\u001b\[[0-9;]*m//g; s/\\u001b(B//g' \
  | gonzo
```

## How the Pipe Works

Render emits each log line with metadata nested in a `labels` array:

```json
{
  "labels": [
    { "name": "resource", "value": "srv-abc123" },
    { "name": "instance", "value": "srv-abc123-54m4m" },
    { "name": "level",    "value": "info" },
    { "name": "type",     "value": "app" }
  ],
  "message": "{\"level\":\"info\",\"message\":\"heartbeat\"}",
  "timestamp": "2026-03-27T14:34:36.521Z"
}
```

The `jq` transform flattens all labels to top-level fields dynamically - if Render adds new labels (e.g. `method`, `path`, `status_code` on Professional plans), they appear automatically.

## Log Phases

Build, deploy, and runtime logs come through a single stream. You can identify the phase by which labels are present:

| Phase | Labels present | Example |
|---|---|---|
| **Build** | `resource`, `level`, `type` | `==> Running build command 'yarn'...` |
| **Deploy** | `resource`, `type` (no `level`, no `instance`) | `==> Your service is live 🎉` |
| **Runtime** | `resource`, `instance`, `level`, `type` | Your app's structured log output |

## Usage Patterns

**Stream from multiple services** (same region and workspace):

```bash
render logs -r srv-XXXXX,srv-YYYYY -o json --tail \
  | jq --unbuffered -c '{timestamp: .timestamp, message: .message} + ([.labels[] | {(.name): .value}] | add)' \
  | sed -u 's/\\u001b\[[0-9;]*m//g; s/\\u001b(B//g' \
  | gonzo
```

**Historical query:**

```bash
render logs -r srv-XXXXX -o json --limit 100 \
  --start 2026-03-27T14:00:00Z --end 2026-03-27T15:00:00Z \
  | jq -c '{timestamp: .timestamp, message: .message} + ([.labels[] | {(.name): .value}] | add)' \
  | sed -u 's/\\u001b\[[0-9;]*m//g; s/\\u001b(B//g' \
  | gonzo
```

**Filter by level or text:**

```bash
render logs -r srv-XXXXX -o json --tail --level error | ...
render logs -r srv-XXXXX -o json --tail --text "timeout" | ...
```

## Structured Logging Tips

Render parses your app's stdout and maps the `level` field to its internal severity. Emit structured JSON for the best Gonzo experience:

```javascript
console.log(JSON.stringify({
  message: "User signup completed",
  level: "info",
  userId: 123
}));
```

Render tags all **stderr** output as errors by default. Python's `logging` library defaults to stderr, so `logging.info()` calls appear as errors. Use structured JSON logging to override this.

## Rate Limits

**Streaming** (`--tail`) uses WebSocket and is **not rate limited**. This is the recommended mode.

**Fetch mode** (`--limit`, `--start`, `--end`) uses the REST API and is subject to plan-based limits.

**Platform cap:** 6,000 log lines per minute per instance. Excess lines are silently dropped.

## Troubleshooting

**"required flag(s) 'resources' not set"**: Add `-r <service-id>`. Required when using `-o json`.

**`level` shows as null**: Make sure `jq` extracts from `.labels[]`, not `.level` directly.

**ANSI garbage in messages**: Add the `sed` strip to your pipe.

**No build logs in `--tail`**: The `--tail` flag streams from the moment you connect. Use `--start`/`--end` for historical build logs.

## Contributing

Guide: `guides/RENDER_USAGE_GUIDE.md` in the [Gonzo repo](https://github.com/control-theory/gonzo). PRs and issues welcome.
