# Vercel Usage Guide

Stream Vercel deployment logs into Gonzo for real-time analysis, pattern
detection, and AI-powered insights — directly from your terminal.

## Prerequisites

- **Gonzo** installed (`brew install gonzo` or [GitHub releases](https://github.com/control-theory/gonzo/releases))
- **Vercel CLI** installed and authenticated (`npm i -g vercel && vercel login`)
- **jq** installed (`brew install jq` or [stedolan.github.io/jq](https://stedolan.github.io/jq/download/))

## Setup

### Stream logs into Gonzo

Vercel's `-j` flag outputs logs as JSON, but the `message` field contains
double-encoded JSON — the actual app log is wrapped inside a stringified
object with a `[function-name] ` prefix (derived from the API route name,
e.g. `[orders]` for `/api/orders`).

The jq normalizer unwraps the inner JSON and merges it with the Vercel
envelope metadata:

```bash
vercel logs <deployment_url_or_id> -j | jq --unbuffered '
  (.message | sub("^\\[.*?\\] "; "") | fromjson) + {source, requestPath, domain, requestMethod, responseStatusCode}
' | gonzo
```

Replace `<deployment_url_or_id>` with your deployment URL or ID. You can find
this in the Vercel dashboard or via `vercel ls`.

When you specify a deployment URL, Vercel auto-follows (streams in real time)
by default. Use `--no-follow` if you only want a batch of recent logs.

> **⚠️ `--unbuffered` on jq is mandatory.** Without it, jq buffers output
> and the pipe appears to stall. This is the #1 setup issue.

## What you get

Once logs are flowing, Gonzo sees a flat, clean log line with both your
app-level fields and Vercel metadata:

**From your app (unwrapped):** `ts`, `level`, `service`, `msg`, `reqId`, `value`

**From the Vercel envelope:** `source`, `requestPath`, `domain`, `requestMethod`, `responseStatusCode`

This gives you:

- **Structured formatting** — all fields parsed and displayed cleanly
- **Severity filtering** — filter by the actual app log level, not Vercel's outer level
- **Pattern detection** — Gonzo extracts recurring patterns from the log stream
- **AI analysis** (optional) — trigger AI-powered log summaries and anomaly
  detection from within the TUI

## Notes

- **The `-j` flag is required.** Without it, Vercel outputs plain text and
  the normalizer has nothing to parse.
- **No log drain required.** This uses the Vercel CLI directly, which is
  currently free on Hobby and Pro tiers. No Vercel-side configuration needed.
- **The `[prefix]` is the function name.** Vercel prepends the API route
  name (e.g. `[noise]` for `/api/noise`). The normalizer strips any
  `[...] ` prefix automatically.

## Writing logs to file

To capture Vercel logs for analysis outside Gonzo (e.g. in Claude Code):

```bash
vercel logs <deployment_url_or_id> -j --no-follow | jq --unbuffered '
  (.message | sub("^\\[.*?\\] "; "") | fromjson) + {source, requestPath, domain, requestMethod, responseStatusCode}
' > /tmp/vercel-logs.jsonl
```

## Troubleshooting

| Problem | Fix |
|---------|-----|
| Pipe appears to stall | Make sure `--unbuffered` is on the jq call |
| No logs appearing | Check you're authenticated (`vercel whoami`) and the deployment is receiving traffic |
| `vercel: command not found` | Install the CLI: `npm i -g vercel` |
| `jq: error - Could not parse` | A log line may not have the expected `[prefix] {json}` format. Check for non-function logs in the stream |
| Logs appear but no inner fields | Make sure you're using the `-j` flag |
| Ctrl+C not killing the pipe | Use `ctrl+\` (SIGQUIT) or `kill $(pgrep -f "vercel logs")` from another terminal |
