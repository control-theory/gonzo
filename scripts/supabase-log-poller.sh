#!/usr/bin/env bash
#
# supabase-log-poller.sh
# Polls all Supabase log sources and normalizes them into JSONL.
#
# Usage:
#   # Pipe straight into Gonzo (default):
#   ./supabase-log-poller.sh | gonzo
#
#   # Write to file instead (for Gonzo --follow or later analysis):
#   ./supabase-log-poller.sh -o /tmp/supabase-logs/all.jsonl
#   gonzo -f /tmp/supabase-logs/all.jsonl --follow
#
# Environment:
#   SUPABASE_ACCESS_TOKEN   — Required
#   SUPABASE_PROJECT_REF    — Required
#   POLL_INTERVAL           — Seconds between polls (default: 30)
#   MAX_FILE_SIZE           — Rotate file at this size in bytes (default: 100MB, file mode only)

set -euo pipefail

: "${SUPABASE_ACCESS_TOKEN:?Set SUPABASE_ACCESS_TOKEN}"
: "${SUPABASE_PROJECT_REF:?Set SUPABASE_PROJECT_REF}"

POLL_INTERVAL="${POLL_INTERVAL:-30}"
MAX_FILE_SIZE="${MAX_FILE_SIZE:-104857600}"
API_BASE="https://api.supabase.com/v1/projects/${SUPABASE_PROJECT_REF}/analytics/endpoints/logs.all"

# Parse flags
OUTPUT_FILE=""
while getopts "o:" opt; do
  case $opt in
    o) OUTPUT_FILE="$OPTARG" ;;
    *) echo "Usage: $0 [-o output_file] | gonzo" >&2; exit 1 ;;
  esac
done

# If writing to file, set up the directory
if [ -n "$OUTPUT_FILE" ]; then
  mkdir -p "$(dirname "$OUTPUT_FILE")"
  touch "$OUTPUT_FILE"
fi

# Status messages go to stderr so they don't pollute the JSONL on stdout
log_status() {
  echo "$@" >&2
}

SOURCES=(
  "edge_logs"
  "postgres_logs"
  "postgrest_logs"
  "auth_logs"
  "storage_logs"
  "realtime_logs"
  "function_logs"
  "function_edge_logs"
  "supavisor_logs"
)

if [ -n "$OUTPUT_FILE" ]; then
  log_status "╔══════════════════════════════════════════╗"
  log_status "║     Supabase → Gonzo Log Poller          ║"
  log_status "║     Mode: file ($OUTPUT_FILE)"
  log_status "╚══════════════════════════════════════════╝"
  log_status ""
  log_status "  In another terminal run:"
  log_status "    gonzo -f $OUTPUT_FILE --follow"
else
  log_status "╔══════════════════════════════════════════╗"
  log_status "║     Supabase → Gonzo Log Poller          ║"
  log_status "║     Mode: stdout (pipe into gonzo)       ║"
  log_status "╚══════════════════════════════════════════╝"
fi

log_status ""
log_status "  Project:   $SUPABASE_PROJECT_REF"
log_status "  Interval:  ${POLL_INTERVAL}s"
log_status "  Sources:   ${#SOURCES[@]}"
log_status ""
log_status "  Polling started at $(date). Ctrl+C to stop."
log_status "──────────────────────────────────────────────"

# Cross-platform date
get_iso_date() {
  local offset_seconds="$1"
  if date --version >/dev/null 2>&1; then
    date -u -d "${offset_seconds} seconds ago" +"%Y-%m-%dT%H:%M:%SZ"
  else
    date -u -v-"${offset_seconds}"S +"%Y-%m-%dT%H:%M:%SZ"
  fi
}

# Cross-platform file size
get_file_size() {
  if stat --version >/dev/null 2>&1; then
    stat -c%s "$1" 2>/dev/null || echo 0
  else
    stat -f%z "$1" 2>/dev/null || echo 0
  fi
}

# Rotate if file exceeds MAX_FILE_SIZE
rotate_if_needed() {
  [ -z "$OUTPUT_FILE" ] && return
  [ "$MAX_FILE_SIZE" -eq 0 ] && return
  [ ! -f "$OUTPUT_FILE" ] && return
  local size
  size=$(get_file_size "$OUTPUT_FILE")
  if [ "$size" -gt "$MAX_FILE_SIZE" ]; then
    mv "$OUTPUT_FILE" "${OUTPUT_FILE}.old"
    touch "$OUTPUT_FILE"
    log_status "  ↻ Rotated $(basename "$OUTPUT_FILE") (was $(( size / 1048576 ))MB)"
  fi
}

# Write JSONL to stdout or file
emit() {
  if [ -n "$OUTPUT_FILE" ]; then
    cat >> "$OUTPUT_FILE"
  else
    cat
  fi
}

# ── Normalizers ──

normalize_edge_logs() {
  jq -c '.result[]? | {
    timestamp: (.timestamp / 1000000 | todate),
    severity: (if .metadata[0].response[0].status_code >= 500 then "ERROR"
               elif .metadata[0].response[0].status_code >= 400 then "WARN"
               else "INFO" end),
    body: .event_message,
    source: "edge_logs",
    service: "api-gateway",
    attributes: {
      method: .metadata[0].request[0].method,
      path: .metadata[0].request[0].path,
      status: .metadata[0].response[0].status_code,
      origin_time_ms: .metadata[0].response[0].origin_time,
      ip: .metadata[0].request[0].headers[0].cf_connecting_ip,
      country: .metadata[0].request[0].headers[0].cf_ipcountry,
      user_agent: .metadata[0].request[0].headers[0].user_agent,
      host: .metadata[0].request[0].host,
      sb_request_id: .metadata[0].response[0].headers[0].sb_request_id,
      sb_error_code: .metadata[0].response[0].headers[0].x_sb_error_code,
      kong_proxy_latency: .metadata[0].response[0].headers[0].x_kong_proxy_latency,
      kong_upstream_latency: .metadata[0].response[0].headers[0].x_kong_upstream_latency,
      cf_colo: .metadata[0].request[0].cf[0].colo,
      cf_city: .metadata[0].request[0].cf[0].city,
      cf_region: .metadata[0].request[0].cf[0].region,
      cf_continent: .metadata[0].request[0].cf[0].continent,
      cf_timezone: .metadata[0].request[0].cf[0].timezone,
      cf_asn: .metadata[0].request[0].cf[0].asn,
      cf_as_org: .metadata[0].request[0].cf[0].asOrganization,
      cf_http_protocol: .metadata[0].request[0].cf[0].httpProtocol,
      cf_tls_version: .metadata[0].request[0].cf[0].tlsVersion,
      cf_bot_score: .metadata[0].request[0].cf[0].botManagement[0].score,
      cf_verified_bot: .metadata[0].request[0].cf[0].botManagement[0].verifiedBot,
      cf_cache_status: .metadata[0].response[0].headers[0].cf_cache_status,
      lb_redirect_id: .metadata[0].load_balancer_redirect_identifier
    }
  }' 2>/dev/null
}

normalize_postgres_logs() {
  jq -c '.result[]? | {
    timestamp: (.timestamp / 1000000 | todate),
    severity: (.metadata[0].parsed[0].error_severity // "INFO" | ascii_upcase),
    body: .event_message,
    source: "postgres_logs",
    service: "postgres",
    attributes: {
      user: .metadata[0].parsed[0].user_name,
      database: .metadata[0].parsed[0].database_name,
      query: .metadata[0].parsed[0].query,
      detail: .metadata[0].parsed[0].detail,
      hint: .metadata[0].parsed[0].hint,
      context: .metadata[0].parsed[0].context,
      sql_state: .metadata[0].parsed[0].sql_state_code,
      backend_type: .metadata[0].parsed[0].backend_type,
      command_tag: .metadata[0].parsed[0].command_tag,
      application_name: .metadata[0].parsed[0].application_name,
      connection_from: .metadata[0].parsed[0].connection_from,
      session_id: .metadata[0].parsed[0].session_id,
      process_id: .metadata[0].parsed[0].process_id,
      query_id: .metadata[0].parsed[0].query_id,
      host: .metadata[0].host
    }
  }' 2>/dev/null
}

normalize_postgrest_logs() {
  jq -c '.result[]? | {
    timestamp: (.timestamp / 1000000 | todate),
    severity: "INFO",
    body: .event_message,
    source: "postgrest_logs",
    service: "postgrest",
    attributes: {
      host: .metadata[0].host
    }
  }' 2>/dev/null
}

normalize_auth_logs() {
  jq -c '.result[]? | {
    timestamp: (.timestamp / 1000000 | todate),
    severity: (.metadata[0].level // "info" | ascii_upcase),
    body: (.metadata[0].msg // .event_message),
    source: "auth_logs",
    service: "gotrue",
    attributes: {
      status: .metadata[0].status,
      method: .metadata[0].method,
      path: .metadata[0].path,
      component: .metadata[0].component,
      remote_addr: .metadata[0].remote_addr,
      grant_type: .metadata[0].grant_type,
      user_id: .metadata[0].user_id,
      provider: .metadata[0].provider,
      action: .metadata[0].action,
      login_method: .metadata[0].login_method,
      error: .metadata[0].error,
      duration: .metadata[0].duration,
      request_id: .metadata[0].request_id,
      referer: .metadata[0].referer,
      mail_type: .metadata[0].mail_type,
      mail_to: .metadata[0].mail_to,
      sso_provider_id: .metadata[0].sso_provider_id,
      factor_id: .metadata[0].factor_id,
      event: .metadata[0].event,
      host: .metadata[0].host
    }
  }' 2>/dev/null
}

normalize_storage_logs() {
  jq -c '.result[]? | {
    timestamp: (.timestamp / 1000000 | todate),
    severity: (.metadata[0].level // "info" | ascii_upcase),
    body: .event_message,
    source: "storage_logs",
    service: "storage",
    attributes: {
      method: .metadata[0].req[0].method,
      url: .metadata[0].req[0].url,
      status: .metadata[0].res[0].statusCode,
      response_time_ms: .metadata[0].responseTime,
      execution_time_ms: .metadata[0].executionTime,
      tenant_id: .metadata[0].tenantId,
      operation: .metadata[0].operation,
      object_path: .metadata[0].objectPath,
      object_version: .metadata[0].objectVersion,
      owner: .metadata[0].owner,
      role: .metadata[0].role,
      host: .metadata[0].context[0].host,
      req_id: .metadata[0].reqId,
      region: .metadata[0].region,
      user_agent: .metadata[0].req[0].headers[0].user_agent,
      remote_address: .metadata[0].req[0].remoteAddress,
      content_type: .metadata[0].res[0].headers[0].content_type,
      etag: .metadata[0].res[0].headers[0].etag,
      trace_id: .metadata[0].trace_id
    }
  }' 2>/dev/null
}

normalize_realtime_logs() {
  jq -c '.result[]? | {
    timestamp: (.timestamp / 1000000 | todate),
    severity: (.metadata[0].level // "info" | ascii_upcase),
    body: .event_message,
    source: "realtime_logs",
    service: "realtime",
    attributes: {
      project: .metadata[0].project,
      region: .metadata[0].region,
      cluster: .metadata[0].cluster,
      request_id: .metadata[0].request_id,
      external_id: .metadata[0].external_id,
      tenant: .metadata[0].tenant,
      tenant_id: .metadata[0].tenant_id,
      trace_id: .metadata[0].otel_trace_id,
      span_id: .metadata[0].otel_span_id,
      error_code: .metadata[0].error_code,
      error_string: .metadata[0].error_string,
      module: .metadata[0].context[0].module,
      function: .metadata[0].context[0].function,
      file: .metadata[0].context[0].file,
      line: .metadata[0].context[0].line,
      pid: .metadata[0].context[0].pid,
      node: .metadata[0].context[0].vm[0].node
    }
  }' 2>/dev/null
}

normalize_function_logs() {
  jq -c '.result[]? | {
    timestamp: (.timestamp / 1000000 | todate),
    severity: (.metadata[0].level // "info" | ascii_upcase),
    body: .event_message,
    source: "function_logs",
    service: ("edge-function/" + (.metadata[0].function_id // "unknown")),
    attributes: {
      function_id: .metadata[0].function_id,
      execution_id: .metadata[0].execution_id,
      deployment_id: .metadata[0].deployment_id,
      event_type: .metadata[0].event_type,
      region: .metadata[0].region,
      served_by: .metadata[0].served_by,
      version: .metadata[0].version,
      boot_time: .metadata[0].boot_time,
      cpu_time_used: .metadata[0].cpu_time_used,
      reason: .metadata[0].reason,
      log_timestamp: .metadata[0].timestamp
    }
  }' 2>/dev/null
}

normalize_function_edge_logs() {
  jq -c '.result[]? | {
    timestamp: (.timestamp / 1000000 | todate),
    severity: (if .metadata[0].response[0].status_code >= 500 then "ERROR"
               elif .metadata[0].response[0].status_code >= 400 then "WARN"
               else "INFO" end),
    body: .event_message,
    source: "function_edge_logs",
    service: ("edge-function/" + (.metadata[0].function_id // "unknown")),
    attributes: {
      function_id: .metadata[0].function_id,
      execution_id: .metadata[0].execution_id,
      execution_time_ms: .metadata[0].execution_time_ms,
      method: .metadata[0].request[0].method,
      pathname: .metadata[0].request[0].pathname,
      status: .metadata[0].response[0].status_code,
      user_agent: .metadata[0].request[0].headers[0].user_agent,
      host: .metadata[0].request[0].host,
      jwt_role: .metadata[0].request[0].sb[0].jwt[0].authorization[0].payload[0].role,
      jwt_issuer: .metadata[0].request[0].sb[0].jwt[0].authorization[0].payload[0].issuer,
      auth_user: .metadata[0].request[0].sb[0].auth_user,
      edge_region: .metadata[0].response[0].headers[0].x_sb_edge_region,
      sb_request_id: .metadata[0].response[0].headers[0].sb_request_id,
      served_by: .metadata[0].response[0].headers[0].x_served_by,
      content_type: .metadata[0].response[0].headers[0].content_type,
      deployment_id: .metadata[0].deployment_id,
      version: .metadata[0].version
    }
  }' 2>/dev/null
}

normalize_supavisor_logs() {
  jq -c '.result[]? | {
    timestamp: (.timestamp / 1000000 | todate),
    severity: "INFO",
    body: .event_message,
    source: "supavisor_logs",
    service: "pooler",
    attributes: {
      host: .metadata[0].host
    }
  }' 2>/dev/null
}

# Map source → normalizer
normalize() {
  local src="$1"
  case "$src" in
    edge_logs)          normalize_edge_logs ;;
    postgres_logs)      normalize_postgres_logs ;;
    postgrest_logs)     normalize_postgrest_logs ;;
    auth_logs)          normalize_auth_logs ;;
    storage_logs)       normalize_storage_logs ;;
    realtime_logs)      normalize_realtime_logs ;;
    function_logs)      normalize_function_logs ;;
    function_edge_logs) normalize_function_edge_logs ;;
    supavisor_logs)     normalize_supavisor_logs ;;
    *)                  jq -c '.result[]? | {
                          timestamp: (.timestamp / 1000000 | todate),
                          severity: "INFO",
                          body: .event_message,
                          source: "'"$src"'",
                          service: "'"$src"'"
                        }' 2>/dev/null ;;
  esac
}

# ── Main loop ──

POLL_COUNT=0

while true; do
  POLL_COUNT=$((POLL_COUNT + 1))
  END=$(get_iso_date 0)
  START=$(get_iso_date $((POLL_INTERVAL + 5)))

  rotate_if_needed

  POLL_SUMMARY=""

  for src in "${SOURCES[@]}"; do
    response=$(curl -s --get \
      --max-time 15 \
      -H "Authorization: Bearer $SUPABASE_ACCESS_TOKEN" \
      --data-urlencode "sql=SELECT id, timestamp, event_message, metadata FROM ${src} ORDER BY timestamp DESC LIMIT 200" \
      --data-urlencode "iso_timestamp_start=${START}" \
      --data-urlencode "iso_timestamp_end=${END}" \
      "$API_BASE" 2>/dev/null || echo '{"result":[]}')

    if echo "$response" | jq -e '.error' >/dev/null 2>&1; then
      continue
    fi

    result_count=$(echo "$response" | jq '.result | length' 2>/dev/null || echo "0")

    if [ "$result_count" -gt 0 ]; then
      echo "$response" | normalize "$src" | emit
      POLL_SUMMARY="${POLL_SUMMARY}  ✓ ${src}: +${result_count}\n"
    fi
  done

  if [ -n "$POLL_SUMMARY" ]; then
    log_status "[Poll #${POLL_COUNT} @ $(date +%H:%M:%S)]"
    log_status -e "$POLL_SUMMARY"
  fi

  sleep "$POLL_INTERVAL"
done
