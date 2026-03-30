# Netlify Usage Guide

Stream function logs from your Netlify projects into Gonzo for real-time analysis, filtering, and AI-powered insights, all from your terminal.

## Overview

Netlify's CLI streams serverless function logs to your terminal. A lightweight `jq` normalizer transforms the output into JSONL that Gonzo auto-detects. The full pipe is:

```bash
netlify logs:function <name> | jq --unbuffered -R '
  select(length > 0) |
  (index(" ")) as $i |
  if $i then
    {level: .[:$i] | ascii_downcase, message: .[($i+1):]}
  else
    {level: "info", message: .}
  end |
  (.message | fromjson? // null) as $json |
  if $json then . + $json else . end |
  select(.message | length > 0)
' | gonzo
```

Tested aginst the free tier. Covers serverless function logs (your primary runtime logs).

## Prerequisites

- [Gonzo](https://github.com/control-theory/gonzo) installed (`brew install gonzo` or `go install github.com/control-theory/gonzo/cmd/gonzo@latest`)
- [Netlify CLI](https://cli.netlify.com/) installed (`npm install -g netlify-cli`)
- [jq](https://jqlang.github.io/jq/) installed (`brew install jq`)
- A Netlify project with at least one deployed serverless function

## Setup

### 1. Authenticate the Netlify CLI

```bash
netlify login
```

### 2. Link to your project

```bash
netlify link
```

Select your project when prompted.

### 3. Stream function logs into Gonzo

```bash
netlify logs:function <name> | jq --unbuffered -R '
  select(length > 0) |
  (index(" ")) as $i |
  if $i then
    {level: .[:$i] | ascii_downcase, message: .[($i+1):]}
  else
    {level: "info", message: .}
  end |
  (.message | fromjson? // null) as $json |
  if $json then . + $json else . end |
  select(.message | length > 0)
' | gonzo
```

Replace `<name>` with your function name (e.g. `hello`, `submit-form`). The CLI will prompt you to select a function if you omit the name.

## Log Types

Netlify has several log types. Serverless function logs are the primary target for Gonzo:

| Log type | CLI access | Gonzo compatible | What it captures |
|---|---|---|---|
| **Function logs** | `logs:function <name>` | ✅ Yes (with normalizer) | Your serverless function's `console.log`, `console.warn`, `console.error` output plus platform invocation metrics |
| **Deploy/build logs** | `logs:deploy` | ⚠️ Limited | Build output during git-triggered deploys. Plain text only — no structured fields |
| **Edge Function logs** | Dashboard only | ❌ Not via CLI | `console.log` output from Edge Functions |
| **Traffic logs** | Enterprise Log Drains only | ❌ Not via CLI | CDN request logs: method, path, status, duration |

### Function logs (primary)

Every `console.log()`, `console.warn()`, and `console.error()` call in your serverless functions is captured. The CLI output format is:

```
LEVEL MESSAGE
```

Where `LEVEL` is one of `INFO`, `WARN`, or `ERROR`, followed by a space and the raw message content. If your function emits structured JSON via `console.log(JSON.stringify({...}))`, the JSON survives intact as the message.

After each function invocation, Netlify injects a platform metrics line:

```
INFO Duration: 59.54 ms	Memory Usage: 101 MB
```

The `jq` normalizer transforms all of this into flat JSONL that Gonzo auto-detects.

### Deploy logs

Available only during active remote builds (triggered via git push or the Netlify UI). Not available for CLI-initiated deploys (`netlify deploy --prod`).

```bash
netlify logs:deploy | gonzo
```

These are plain text build output — dependency installs, framework detection, bundling. Gonzo will display each line as-is without structured fields. Mainly useful for watching a deploy in real time.

### Edge Function logs and traffic logs

Edge Function logs (`console.log` from Deno-based edge functions) are only visible in the Netlify dashboard under Logs & Metrics. Traffic logs (CDN request data) require Enterprise-tier Log Drains configured to export to an external provider.

## How the Normalizer Works

Netlify's CLI does not have a `--json` flag for function logs. The output is a text-based format: `LEVEL MESSAGE`, one per line, with blank lines between entries. The `jq` normalizer handles three things:

1. **Splits level from message.** Extracts the first token as the log level and everything after the first space as the message body.

2. **Promotes JSON fields.** If the message body is valid JSON, its fields are merged into the top-level object. A log line like `INFO {"event":"signup","userId":42,"ts":"..."}` becomes:

   ```json
   {"level":"info","message":"{...}","event":"signup","userId":42,"ts":"..."}
   ```

3. **Drops blank lines.** The WebSocket stream includes empty lines between log entries. The normalizer filters these out.

The `--unbuffered` flag on `jq` is critical — without it, `jq` buffers output and logs appear in delayed chunks rather than in real time.

## Usage Patterns

**Stream a specific function:**

```bash
netlify logs:function hello | jq --unbuffered -R '
  select(length > 0) |
  (index(" ")) as $i |
  if $i then
    {level: .[:$i] | ascii_downcase, message: .[($i+1):]}
  else
    {level: "info", message: .}
  end |
  (.message | fromjson? // null) as $json |
  if $json then . + $json else . end |
  select(.message | length > 0)
' | gonzo
```

**Filter by level at the source** (only WARN and ERROR — reduces noise):

```bash
netlify logs:function hello --level warn error | jq --unbuffered -R '
  select(length > 0) |
  (index(" ")) as $i |
  if $i then
    {level: .[:$i] | ascii_downcase, message: .[($i+1):]}
  else
    {level: "info", message: .}
  end |
  (.message | fromjson? // null) as $json |
  if $json then . + $json else . end |
  select(.message | length > 0)
' | gonzo
```

**Write to a file** so you can restart Gonzo without losing history:

```bash
netlify logs:function hello | jq --unbuffered -R '
  select(length > 0) |
  (index(" ")) as $i |
  if $i then
    {level: .[:$i] | ascii_downcase, message: .[($i+1):]}
  else
    {level: "info", message: .}
  end |
  (.message | fromjson? // null) as $json |
  if $json then . + $json else . end |
  select(.message | length > 0)
' > /tmp/netlify-logs.jsonl &
gonzo -f /tmp/netlify-logs.jsonl --follow
```

**Add AI analysis** with a local model (logs never leave your machine):

```bash
export OPENAI_API_KEY="ollama"
export OPENAI_API_BASE="http://localhost:11434"
netlify logs:function hello | jq --unbuffered -R '
  select(length > 0) |
  (index(" ")) as $i |
  if $i then
    {level: .[:$i] | ascii_downcase, message: .[($i+1):]}
  else
    {level: "info", message: .}
  end |
  (.message | fromjson? // null) as $json |
  if $json then . + $json else . end |
  select(.message | length > 0)
' | gonzo
```

## Structured Logging Tips

Netlify does not add timestamps to function log output. If your function only uses plain `console.log("some message")`, you'll get log lines with a level and message but no timestamp. For the best experience in Gonzo, emit structured JSON with at least `message`, `level`, and a timestamp:

```javascript
console.log(JSON.stringify({
  message: "User signup completed",
  level: "info",
  ts: new Date().toISOString(),
  userId: 123,
  provider: "github"
}));
```

The normalizer automatically promotes these fields to the top level, giving Gonzo access to timestamps and custom fields for filtering and display.

**Matching console methods to levels.** Netlify maps `console.log()` and `console.info()` to `INFO`, `console.warn()` to `WARN`, and `console.error()` to `ERROR`. If you include a `level` field in your JSON payload, the normalizer uses your field (it merges on top of the Netlify prefix). In practice the two always agree when you use the matching console method.

**Avoid pretty-printed JSON.** Multi-line `JSON.stringify(obj, null, 2)` output will span multiple lines in the log stream, each treated as a separate log entry. Always use minified JSON: `JSON.stringify(obj)`.

## Platform Latency

Function logs typically appear in the CLI stream **5–15 seconds** after execution (observed on the free tier). This delay is inherent to Netlify's log delivery infrastructure and is not affected by the Gonzo pipe or the `jq` normalizer.

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
| `Duration:` | Netlify platform invocation metrics only |
| `signup` | Logs matching a specific event |
| `userId` | Logs containing a user identifier |

## Multiple Functions

The Netlify CLI streams one function at a time. If you omit the function name, the CLI prompts you to select one — there is no "stream all" mode. To monitor multiple functions, run separate terminal sessions each streaming a different function into Gonzo.

## Troubleshooting

**"No site linked"**: Run `netlify link` to connect your terminal session to a project.

**Function returns 404**: Verify your function deployed with `netlify functions:list`. Check that the `[functions]` directory in `netlify.toml` matches your project structure.

**Logs appear empty**: Your function may not have been invoked recently. Function logs stream in real time — trigger the function endpoint and wait 5–15 seconds for delivery.

**`jq` hangs / no output**: Ensure you're using `--unbuffered` on the `jq` command. Without it, `jq` buffers output when writing to a pipe and logs arrive in delayed chunks.

**Blank JSON lines in output**: If you see `{"level":"","message":""}` entries, the normalizer's blank line filter isn't matching. Ensure you're using the full normalizer expression with `select(length > 0)` and `select(.message | length > 0)`.

**`logs:deploy` says "No active builds"**: Deploy logs only attach to remote builds triggered via git push or the Netlify UI. CLI-initiated deploys (`netlify deploy --prod`) build locally and upload the result — there is no remote build to stream.

## Contributing

This guide lives in the [Gonzo repo](https://github.com/control-theory/gonzo):

- Guide: `guides/NETLIFY_USAGE_GUIDE.md`

If you run into issues or find edge cases with Netlify's log format, PRs and issues are welcome. Gonzo is open source (MIT) and community-driven.
