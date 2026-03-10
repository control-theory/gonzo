# Supabase Usage Guide

Stream logs from all Supabase services into Gonzo for real-time analysis, filtering, and AI-powered insights — all from your terminal.

## Overview

Supabase doesn't offer a streaming log API, but the [Management API](https://supabase.com/docs/reference/api/introduction) exposes a SQL-based query endpoint for log data across all services. The included poller script (`scripts/supabase-log-poller.sh`) polls this endpoint, normalizes the deeply nested metadata from each service into flat JSONL, and pipes the result into Gonzo.

A single poller covers 9 log sources:

| Source | Service | What it captures |
|---|---|---|
| `edge_logs` | `api-gateway` | API Gateway / Cloudflare edge — every API request |
| `postgres_logs` | `postgres` | Database queries, connections, errors, pgAudit |
| `postgrest_logs` | `postgrest` | REST API server lifecycle (schema cache, config) |
| `auth_logs` | `gotrue` | Sign-ups, logins, token ops, auth errors |
| `storage_logs` | `storage` | Object uploads, downloads, bucket operations |
| `realtime_logs` | `realtime` | WebSocket connections, Phoenix server events |
| `function_logs` | `edge-function/<id>` | Edge Function console output |
| `function_edge_logs` | `edge-function/<id>` | Edge Function network request/response |
| `supavisor_logs` | `pooler` | Connection pooler activity |

## Prerequisites

- [Gonzo](https://github.com/control-theory/gonzo) installed
- `jq` installed (`brew install jq` on macOS)
- `curl`
- A Supabase project (free tier works) — [create one here](https://database.new)

## Setup

### 1. Get your credentials

You need two values from the Supabase dashboard:

**Project Ref** — Found in the URL of your project dashboard (`https://supabase.com/dashboard/project/<ref>`) or under **Settings → General**.

**Personal Access Token** — Go to [supabase.com/dashboard/account/tokens](https://supabase.com/dashboard/account/tokens), generate a new token. It starts with `sbp_`.

### 2. Set environment variables

```bash
export SUPABASE_ACCESS_TOKEN="sbp_your_token_here"
export SUPABASE_PROJECT_REF="your_project_ref"
```

Optionally save these to a file for convenience:

```bash
cat > ~/.supabase-env << EOF
export SUPABASE_ACCESS_TOKEN="sbp_your_token_here"
export SUPABASE_PROJECT_REF="your_project_ref"
EOF

# Then in any new terminal:
source ~/.supabase-env
```

### 3. Make the poller executable

```bash
chmod +x scripts/supabase-log-poller.sh
```

## Usage

### Pipe directly into Gonzo (recommended)

```bash
./scripts/supabase-log-poller.sh | gonzo
```

To suppress the poller's status messages:

```bash
./scripts/supabase-log-poller.sh 2>/dev/null | gonzo
```

### Write to file

```bash
./scripts/supabase-log-poller.sh -o /tmp/supabase-logs/all.jsonl

# In another terminal:
gonzo -f /tmp/supabase-logs/all.jsonl --follow
```

File mode supports automatic rotation at 100MB (configurable via `MAX_FILE_SIZE`).

### Faster polling

```bash
POLL_INTERVAL=10 ./scripts/supabase-log-poller.sh | gonzo
```

### With AI analysis

```bash
export OPENAI_API_KEY="sk-your-key"
./scripts/supabase-log-poller.sh 2>/dev/null | gonzo --ai-model="gpt-4"
```

Or with a local model via Ollama:

```bash
export OPENAI_API_KEY="ollama"
export OPENAI_API_BASE="http://localhost:11434"
./scripts/supabase-log-poller.sh 2>/dev/null | gonzo
```

## Filtering in Gonzo

Press `/` to open the filter. Useful patterns:

**By source:**

| Filter | What you'll see |
|---|---|
| `edge_logs` | API gateway requests |
| `postgres_logs` | Database activity |
| `auth_logs` | Authentication events |
| `storage_logs` | Storage operations |
| `function_logs` | Edge Function console output |
| `function_edge_logs` | Edge Function network layer |
| `realtime_logs` | Realtime server events |

**By service name:**

| Filter | What you'll see |
|---|---|
| `api-gateway` | All API gateway traffic |
| `gotrue` | Auth events |
| `postgres` | Database logs |
| `storage` | Storage operations |
| `edge-function` | All function invocations |
| `realtime` | Realtime events |

**By severity:**

| Filter | What you'll see |
|---|---|
| `ERROR` | Server errors (5xx) |
| `WARN` | Client errors (4xx) |
| `INFO` | Successful requests |

Press `Esc` to clear the filter.

## How It Works

The poller queries a single Supabase Management API endpoint with different SQL per source:

```
GET https://api.supabase.com/v1/projects/{ref}/analytics/endpoints/logs.all
  ?sql=SELECT id, timestamp, event_message, metadata FROM {source} ...
  &iso_timestamp_start=...
  &iso_timestamp_end=...
```

Each source returns deeply nested metadata arrays with different structures. The poller runs the response through a source-specific `jq` normalizer that flattens it into a common JSONL format:

```json
{
  "timestamp": "2026-03-05T14:52:18Z",
  "severity": "INFO",
  "body": "POST | 200 | https://example.supabase.co/functions/v1/my-func",
  "source": "edge_logs",
  "service": "api-gateway",
  "attributes": {
    "method": "POST",
    "path": "/functions/v1/my-func",
    "status": 200,
    "ip": "71.42.201.2",
    "cf_city": "Austin"
  }
}
```

Gonzo auto-detects this JSON format with no additional configuration needed.

Severity is derived from HTTP status codes for edge and function logs (5xx → ERROR, 4xx → WARN), or from the source's native level field for auth, postgres, storage, and realtime logs.

## Normalized Attributes per Source

Each source extracts the most operationally useful fields:

**edge_logs** — method, path, status, origin_time_ms, ip, country, user_agent, host, Cloudflare geo (city, region, continent, timezone), ASN/org, TLS version, HTTP protocol, bot score, cache status, Kong latency, request ID

**postgres_logs** — user, database, query, detail, hint, context, sql_state, backend_type, command_tag, application_name, connection_from, session_id, process_id, query_id

**auth_logs** — status, method, path, component, remote_addr, grant_type, user_id, provider, action, login_method, error, duration, request_id, referer, mail_type, SSO provider, factor_id

**storage_logs** — method, url, status, response_time_ms, execution_time_ms, tenant_id, operation, object_path, owner, role, region, user_agent, remote_address, etag, trace_id

**realtime_logs** — project, region, cluster, request_id, external_id, tenant, OTel trace/span IDs, error code/string, Phoenix module/function/file/line/pid/node

**function_logs** — function_id, execution_id, deployment_id, event_type, region, served_by, version, boot_time, cpu_time_used, reason

**function_edge_logs** — function_id, execution_id, execution_time_ms, method, pathname, status, user_agent, host, JWT role/issuer, auth_user, edge_region, request ID, served_by, content_type, deployment_id

**postgrest_logs** — host

**supavisor_logs** — host

## API Rate Limits

> **Note:** The Supabase Management API has a rate limit of 120 requests per minute per project, with stricter limits on analytics endpoints. The poller makes 9 requests per cycle. At the default 30-second interval this is 18 requests/min — well within limits. Avoid setting `POLL_INTERVAL` below 10 seconds (which would be 54 requests/min) to stay safely under the limit. If you receive 429 responses, increase your poll interval.

| POLL_INTERVAL | Requests/min | Risk |
|---|---|---|
| 30s (default) | 18 | ✅ Safe |
| 15s | 36 | ✅ Safe |
| 10s | 54 | ⚠️ Use with caution |
| 5s | 108 | ❌ Likely to hit limits |

## Log Propagation Delay

Supabase logs are ingested via Logflare and typically take 30–90 seconds to become available through the Management API. This means there's an inherent delay between an event occurring and it appearing in Gonzo. The poller's time window overlaps by 5 seconds to avoid gaps at poll boundaries.

## Generating Test Traffic

To exercise all log sources and verify your setup:

```bash
export SUPABASE_URL="https://<your-ref>.supabase.co"
export SUPABASE_ANON_KEY="<your-anon-key>"

# Edge + PostgREST + Postgres
curl -s -o /dev/null -w "edge/rest:    %{http_code}\n" "$SUPABASE_URL/rest/v1/" -H "apikey: $SUPABASE_ANON_KEY" -H "Authorization: Bearer $SUPABASE_ANON_KEY"
curl -s -o /dev/null -w "postgres/rpc: %{http_code}\n" "$SUPABASE_URL/rest/v1/rpc/nonexistent" -H "apikey: $SUPABASE_ANON_KEY" -H "Authorization: Bearer $SUPABASE_ANON_KEY" -H "Content-Type: application/json" -d '{}'

# Auth
curl -s -o /dev/null -w "auth/login:   %{http_code}\n" "$SUPABASE_URL/auth/v1/token?grant_type=password" -H "apikey: $SUPABASE_ANON_KEY" -H "Content-Type: application/json" -d '{"email":"x@x.com","password":"wrong"}'
curl -s -o /dev/null -w "auth/user:    %{http_code}\n" "$SUPABASE_URL/auth/v1/user" -H "apikey: $SUPABASE_ANON_KEY"

# Storage
curl -s -o /dev/null -w "storage:      %{http_code}\n" "$SUPABASE_URL/storage/v1/bucket" -H "apikey: $SUPABASE_ANON_KEY" -H "Authorization: Bearer $SUPABASE_ANON_KEY"

# Edge Functions (requires a deployed function)
curl -s -o /dev/null -w "function:     %{http_code}\n" "$SUPABASE_URL/functions/v1/hello-world" -H "Authorization: Bearer $SUPABASE_ANON_KEY" -H "Content-Type: application/json" -d '{"test":true}'

# Realtime
curl -s -o /dev/null -w "realtime:     %{http_code}\n" "$SUPABASE_URL/realtime/v1" -H "apikey: $SUPABASE_ANON_KEY"
```

Wait ~60 seconds for logs to propagate, then check Gonzo.

## Environment Variables

| Variable | Default | Description |
|---|---|---|
| `SUPABASE_ACCESS_TOKEN` | *required* | Personal access token (starts with `sbp_`) |
| `SUPABASE_PROJECT_REF` | *required* | Project reference ID |
| `POLL_INTERVAL` | `30` | Seconds between poll cycles |
| `MAX_FILE_SIZE` | `104857600` | Rotate file at 100MB (file mode only, set to `0` to disable) |

## Troubleshooting

**"Format is Authorization: Bearer [token]"** — Your access token isn't set or expired. Run `source ~/.supabase-env` or re-export `SUPABASE_ACCESS_TOKEN`.

**No logs appearing** — Logs take 30–90 seconds to propagate. Generate some traffic (see above) and wait for the next poll cycle.

**"restricted wildcard (*) in a result column"** — The logs API doesn't allow `SELECT *`. The poller already uses explicit column names; this shouldn't occur in normal use.

**429 Too Many Requests** — You're hitting the rate limit. Increase `POLL_INTERVAL` or wait for the limit to reset (resets each minute).

**Empty results for postgrest_logs / supavisor_logs** — These sources emit logs infrequently. PostgREST only logs server lifecycle events (schema cache reloads, config changes). Supavisor requires direct pooler connections.
