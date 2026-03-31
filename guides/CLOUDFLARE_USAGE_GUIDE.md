# Cloudflare Workers Usage Guide

Stream logs from your Cloudflare Workers into Gonzo for real-time analysis, filtering, and AI-powered insights, all from your terminal.

## Quick Start

```
wrangler tail --format json \
  | jq --unbuffered -c '
    . as $inv |
    (.logs[] | {
      timestamp: .timestamp,
      level: .level,
      message: (.message | map(if type == "string" then . else tostring end) | join(" ")),
      script: $inv.scriptName,
      outcome: $inv.outcome,
      url: $inv.event.request.url,
      method: $inv.event.request.method,
      status: ($inv.event.response.status // null),
      colo: $inv.event.request.cf.colo
    }),
    (.exceptions[]? | {
      timestamp: .timestamp,
      level: "error",
      message: (.name + ": " + .message),
      stack: .stack,
      script: $inv.scriptName,
      outcome: $inv.outcome,
      url: $inv.event.request.url,
      method: $inv.event.request.method,
      status: ($inv.event.response.status // null)
    })
  ' \
  | gonzo
```

Cloudflare's `wrangler tail` streams Worker invocation traces over WebSocket. Each trace is an envelope containing all `console.log` calls and exceptions from a single invocation. The `jq` transform flattens these into per-line JSONL that Gonzo can parse. The `--unbuffered` flag is required — without it, `jq` buffers output and logs stall in the pipe.

## Prerequisites

- [Gonzo](https://github.com/control-theory/gonzo) installed
- [Wrangler](https://developers.cloudflare.com/workers/wrangler/) installed (`npm install -g wrangler`)
- `jq` installed (`brew install jq`)
- A Cloudflare account with at least one deployed Worker

## Setup

```
# 1. Authenticate
wrangler login

# 2. Navigate to your Worker project directory
cd my-worker

# 3. Stream into Gonzo
wrangler tail --format json \
  | jq --unbuffered -c '<normalizer>' \
  | gonzo
```

You can also tail by Worker name from any directory:

```
wrangler tail my-worker --format json \
  | jq --unbuffered -c '<normalizer>' \
  | gonzo
```

## How the Pipe Works

`wrangler tail --format json` emits pretty-printed JSON envelopes — one per Worker invocation. Each envelope wraps the entire request lifecycle:

```json
{
  "outcome": "ok",
  "scriptName": "my-worker",
  "logs": [
    {
      "message": ["Root hit - healthy"],
      "level": "log",
      "timestamp": 1774657827795
    },
    {
      "message": [{"event": "db_write", "ts": 1774657827795}],
      "level": "log",
      "timestamp": 1774657827795
    }
  ],
  "exceptions": [],
  "eventTimestamp": 1774657827795,
  "event": {
    "request": {
      "url": "https://my-worker.example.workers.dev/",
      "method": "GET",
      "cf": { "colo": "ATL" }
    },
    "response": { "status": 200 }
  }
}
```

Key things the normalizer handles:

- **Envelope flattening**: Each `logs[]` entry becomes its own JSONL line, enriched with invocation context (`scriptName`, `outcome`, `url`, `status`, `colo`).
- **Message arrays**: `console.log("foo", "bar")` produces `["foo", "bar"]`. The normalizer joins these into a single string, stringifying objects.
- **Exceptions**: Uncaught exceptions land in `exceptions[]`, not `logs[]`. The normalizer emits these as `level: "error"` lines with stack traces.
- **Cron triggers**: Scheduled invocations have no HTTP request, so `url`, `method`, `status`, and `colo` come through as `null`.

## What's Visible (and What Isn't)

`wrangler tail` streams Worker execution traces — your `console.log`/`warn`/`error` calls, uncaught exceptions, and request/response metadata. This is what Gonzo receives.

What is **not** visible in tail output:

| Component | Visible in tail? | Notes |
|---|---|---|
| `console.log()` calls | ✅ Yes | All levels: log, warn, error, debug, info |
| Uncaught exceptions | ✅ Yes | With stack traces |
| Request/response metadata | ✅ Yes | URL, method, status, colo, headers |
| Cron trigger invocations | ✅ Yes | `url`/`method`/`status` will be null |
| D1 database queries | ❌ No | Only your explicit console.log around D1 calls |
| KV read/write operations | ❌ No | Same — instrument manually |
| R2 object storage operations | ❌ No | Same |
| Workers AI inference calls | ❌ No | Same |
| Vectorize queries | ❌ No | Same |

If you need visibility into D1, KV, or other binding operations, wrap them with `console.log` statements that capture timing and results. The normalizer will surface these like any other log line.

## Log Levels

Cloudflare maps `console` methods directly to levels in the tail output:

| Console method | Tail level |
|---|---|
| `console.log()` | `log` |
| `console.info()` | `info` |
| `console.debug()` | `debug` |
| `console.warn()` | `warn` |
| `console.error()` | `error` |

## Usage Patterns

**Real-time streaming** (default — WebSocket):

```
wrangler tail --format json | jq --unbuffered -c '<normalizer>' | gonzo
```

**Tail a specific Worker by name:**

```
wrangler tail my-worker --format json | jq --unbuffered -c '<normalizer>' | gonzo
```

**Filter by status in wrangler before piping:**

```
wrangler tail --format json --status error | jq --unbuffered -c '<normalizer>' | gonzo
```

**Filter by HTTP method:**

```
wrangler tail --format json --method GET | jq --unbuffered -c '<normalizer>' | gonzo
```

**Filter by search string:**

```
wrangler tail --format json --search "TypeError" | jq --unbuffered -c '<normalizer>' | gonzo
```

**Sample high-traffic Workers:**

```
wrangler tail --format json --sampling-rate 0.1 | jq --unbuffered -c '<normalizer>' | gonzo
```

**Capture to file for replay:**

```
wrangler tail --format json | jq --unbuffered -c '<normalizer>' > /tmp/cf-logs.jsonl &
gonzo -f /tmp/cf-logs.jsonl --follow
```

**Add AI analysis** with a local model:

```
export OPENAI_API_KEY="ollama"
export OPENAI_API_BASE="http://localhost:11434"
wrangler tail --format json | jq --unbuffered -c '<normalizer>' | gonzo
```

## Structured Logging Tips

For the best Gonzo experience, emit structured JSON via `console.log`:

```javascript
console.log(JSON.stringify({
  message: "User signup completed",
  level: "info",
  userId: 123,
  provider: "github",
  duration_ms: 45
}));
```

Objects passed directly to `console.log` also work — the normalizer stringifies them:

```javascript
console.log({ event: "db_write", table: "users", rows: 1 });
```

Both approaches surface in Gonzo. The `JSON.stringify` approach gives you a cleaner `message` field; the direct object approach is faster to write.

## Rate Limits and Sampling

**Streaming mode** (`wrangler tail`) uses a WebSocket connection. For high-traffic Workers, Cloudflare may enter **sampling mode**, dropping some invocation traces to stay within delivery limits. When this happens, a warning appears in the tail output.

**Tail client limit:** A maximum of 10 concurrent tail sessions per Worker.

**`--sampling-rate`:** You can proactively set a sampling rate (0.0 to 1.0) to reduce volume before Cloudflare auto-samples.

Unlike Railway and Render, there is no separate fetch mode with `--lines` or `--since` flags. `wrangler tail` is streaming only. For historical log queries, use [Workers Logs](https://developers.cloudflare.com/workers/observability/logs/workers-logs/) in the Cloudflare dashboard.

## Pages Functions

Cloudflare Pages Functions use the same underlying Worker runtime and produce identical tail output. The same normalizer works without modification.

The command is slightly different — you need a deployment ID, which you can find with:

```
wrangler pages deployment list --project-name my-site
```

Then tail with the deployment ID as a positional argument:

```
wrangler pages deployment tail <DEPLOYMENT_ID> --project-name my-site --format json \
  | jq --unbuffered -c '<normalizer>' \
  | gonzo
```

Or tail the latest production deployment by environment:

```
wrangler pages deployment tail --project-name my-site --environment production --format json \
  | jq --unbuffered -c '<normalizer>' \
  | gonzo
```

**Note:** When piping output (i.e. non-interactive mode), the deployment ID or `--environment` flag is required — `wrangler pages deployment tail` can't prompt you to select a deployment when its stdout is a pipe.

Two minor differences from Workers tail:

- **`scriptName`** shows as something like `pages-worker--12087785-production` rather than a user-defined Worker name.
- **Static asset requests** (HTML, CSS, JS files) do not trigger Functions and won't appear in the tail. Only requests that hit a Pages Function route produce tail events.

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
| `error` | All error-level logs and exceptions |
| `/api` | Logs from API route invocations |
| `exception` | Uncaught exceptions with stack traces |
| `cron` | Scheduled trigger invocations |

## Troubleshooting

**Logs not appearing in Gonzo**: Make sure `jq` has `--unbuffered`. Without it, `jq` buffers when its stdout is a pipe and logs stall.

**"Could not create tail"**: You may have hit the 10 concurrent tail session limit. Close other `wrangler tail` sessions or dashboard live log views.

**Cron logs show null fields**: Expected. Cron triggers have no HTTP request, so `url`, `method`, `status`, and `colo` are null.

**D1/KV operations missing from logs**: These don't produce tail events. Add `console.log` statements before and after binding calls to surface them.

**Pretty-printed JSON instead of JSONL**: `wrangler tail --format json` outputs pretty-printed envelopes. The `jq` normalizer compacts them — don't pipe raw tail output directly to Gonzo.

**Out-of-order logs under concurrency**: Cloudflare executes Workers at edge locations globally. Concurrent requests may arrive in the tail out of timestamp order. This is expected behavior.

## Contributing

Guide: `guides/CLOUDFLARE_USAGE_GUIDE.md` in the [Gonzo repo](https://github.com/control-theory/gonzo). PRs and issues welcome.
