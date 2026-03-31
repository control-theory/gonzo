# Fly.io Usage Guide

Stream logs from your Fly.io apps into Gonzo for real-time analysis, filtering, and AI-powered insights, all from your terminal.

## Quick Start

```bash
fly logs -a <app-name> -j \
  | jq --unbuffered -c '
    {timestamp, region, instance}
    + (try (.message | fromjson) // empty)' \
  | gonzo
```

Fly's CLI streams logs over NATS via an internal API. The `-j` flag gives you structured JSON, but your app's log output is double-encoded inside the `message` field as an escaped JSON string. The `jq` transform pulls `timestamp`, `region`, and `instance` from the envelope, then parses the inner message to extract your actual log fields (`level`, `message`, and any custom fields your app emits). Lines that aren't valid JSON (Fly init logs, Firecracker boot messages) are silently dropped.

## Prerequisites

- [Gonzo](https://github.com/control-theory/gonzo) installed
- [Fly CLI](https://fly.io/docs/flyctl/) installed (`brew install flyctl`)
- `jq` installed (`brew install jq`)

## Setup

```bash
# 1. Authenticate
fly auth login

# 2. Find your app name
fly apps list

# 3. Stream into Gonzo
fly logs -a my-app -j \
  | jq --unbuffered -c '
    {timestamp, region, instance}
    + (try (.message | fromjson) // empty)' \
  | gonzo
```

## How the Pipe Works

Fly emits each log line as a JSON envelope with your app's output double-encoded in the `message` field:

```json
{
  "level": "info",
  "instance": "6835d41f765168",
  "message": "{\"level\":\"error\",\"message\":\"connection timeout\",\"code\":\"ETIMEOUT\"}",
  "region": "dfw",
  "timestamp": "2026-03-28T16:43:33.005496849Z",
  "meta": {
    "Instance": "6835d41f765168",
    "Region": "dfw",
    "Event": { "Provider": "app" },
    "HTTP": { "Request": { "ID": "", "Method": "", "Version": "" }, "Response": { "status_code": 0 } },
    "Error": { "Code": 0, "Message": "" },
    "URL": { "Full": "" }
  }
}
```

Two things to notice:

**The envelope `level` is useless for app logs.** Fly sets it to `"info"` regardless of whether your app wrote to stdout or stderr. The real severity lives inside the double-encoded `message`. The `jq` transform parses the inner JSON so your app's `level` field becomes the top-level value Gonzo sees.

**The `meta` object is mostly empty on app logs.** `meta.HTTP`, `meta.Error`, and `meta.URL` are present but zero-valued — Fly doesn't enrich your app's log lines with request context. The only useful envelope fields are `timestamp`, `region`, and `instance`.

After the transform, Gonzo receives clean flat JSON:

```json
{"timestamp":"2026-03-28T16:43:33.005Z","region":"dfw","instance":"6835d41f765168","level":"error","message":"connection timeout","code":"ETIMEOUT"}
```

## Log Sources

Fly streams multiple log sources through a single pipe. You can identify them by the `meta.Event.Provider` field in the `-j` output, or by the prefix in the default (non-JSON) output:

| Source | Provider | Examples |
|---|---|---|
| **App** | `app` | Your app's stdout/stderr, plus Fly init messages (`INFO Starting init`, `INFO Starting clean up`) |
| **Runner** | `runner` | Image pulls, Firecracker config, machine start/stop |
| **Proxy** | `proxy` | Machine reachability, startup timing |

The `jq` transform with `// empty` drops everything that isn't valid JSON from your app. If you want to keep platform logs (useful during deploy debugging), swap `// empty` for `// {message: .message}`:

```bash
fly logs -a my-app -j \
  | jq --unbuffered -c '
    {timestamp, region, instance, provider: .meta.Event.Provider}
    + (try (.message | fromjson) // {message: .message})' \
  | gonzo
```

Note that Fly's init process (`INFO Starting init`, `INFO Starting clean up`) is tagged as `provider: "app"` — it runs inside the VM alongside your code. The JSON parse test is a more reliable filter than the provider field.

## Usage Patterns

**Filter by region:**

```bash
fly logs -a my-app -j --region lhr | ...
```

**Filter by machine:**

```bash
fly logs -a my-app -j --machine 6835d41f765168 | ...
```

**Buffer dump (no streaming):**

```bash
fly logs -a my-app -j --no-tail \
  | jq -c '{timestamp, region, instance}
    + (try (.message | fromjson) // empty)' \
  | gonzo
```

`--no-tail` dumps whatever is in the current log buffer and exits. There is no `--start`/`--end` flag on the CLI — for historical queries with date ranges, use Fly's [Grafana log search](https://fly.io/docs/monitoring/search-logs/) (30-day retention).

## Structured Logging Tips

Fly captures stdout from your app process and ships it through Vector → NATS. Emit structured JSON for the best Gonzo experience:

```javascript
console.log(JSON.stringify({
  message: "User signup completed",
  level: "info",
  userId: 123
}));
```

JSON must be single-line. Multi-line JSON objects are split into separate log entries by Fly's log pipeline — each line becomes its own log line.

Fly tags all **stderr** output with `"level": "info"` at the envelope level, same as stdout. If you rely on stderr for error output (e.g. Python's `logging` library defaults to stderr), the envelope level won't reflect the real severity. Use structured JSON with an explicit `level` field to ensure Gonzo sees the correct severity.

## Machines and Auto-Stop

Fly Machines [auto-stop](https://fly.io/docs/launch/autostop-autostart/) when there's no inbound traffic. This means:

- Log streams will end when machines stop. This is expected — not a Gonzo or pipe issue.
- Machines restart automatically on the next inbound request.
- To keep machines running for sustained log tailing, either send periodic requests or configure `auto_stop_machines = false` in `fly.toml`.

## Troubleshooting

**No logs appearing**: Check `fly status -a <app>`. If machines show `stopped`, hit your app's URL to wake them. If `Image` shows `-`, the app was never deployed — run `fly deploy`.

**Only build/boot logs, no app logs**: Your app process may not have started yet. Firecracker VM boot can take a few seconds. Wait 10-15 seconds after deploy.

**Envelope `level` is always "info"**: This is expected. Fly doesn't map your app's log level to the envelope. The `jq` transform parses the inner JSON to extract the real level.

**Logs stall in the pipe**: Make sure `jq` has the `--unbuffered` flag. Without it, jq buffers output and logs appear in delayed bursts.

**Init messages mixed with app logs**: Fly's init process (`INFO Starting init`, `INFO Preparing to run`) is tagged as `provider: "app"`. Use the `// empty` jq pattern to filter these out — they're not valid JSON and get dropped automatically.

**Trial account limits**: Without a payment method, Fly kills machines after 5 minutes. Add a credit card at https://fly.io/trial — the free allowance covers small apps without charges.

## Contributing

Guide: `guides/FLY_USAGE_GUIDE.md` in the [Gonzo repo](https://github.com/control-theory/gonzo). PRs and issues welcome.
