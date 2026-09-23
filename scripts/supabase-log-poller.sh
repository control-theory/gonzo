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
API_BASE="https://api.supabase.com/v1/projects/${SUPABASE_PROJECT_REF}/analytics/endpoints/logs"

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

# Convert Supabase's unified ClickHouse log rows into Gonzo JSONL.
#
# The new Management API exposes all sources through one `logs` table.
# Source-specific fields are carried in `log_attributes`.
normalize() {
  jq -c '
    def status_code:
      ((.log_attributes["response.status_code"]
        // .log_attributes["res.statusCode"]
        // "") | tonumber? // null);

    def service_name:
      if .source == "edge_logs" then "api-gateway"
      elif .source == "postgres_logs" then "postgres"
      elif .source == "postgrest_logs" then "postgrest"
      elif .source == "auth_logs" then "gotrue"
      elif .source == "storage_logs" then "storage"
      elif .source == "realtime_logs" then "realtime"
      elif .source == "supavisor_logs" then "pooler"
      elif .source == "function_logs" or .source == "function_edge_logs"
        then ("edge-function/" + (.log_attributes["function_id"] // "unknown"))
      else .source
      end;

    .result[]? |
    . as $row |
    ($row | status_code) as $status |
    {
      timestamp: (if ($row.timestamp | type) == "string"
                    and ($row.timestamp | test("T[0-9:.]+$"))
                  then $row.timestamp + "Z" else $row.timestamp end),
      severity: (
        if (
          ($row.source == "edge_logs" or $row.source == "function_edge_logs")
          and $status != null
        ) then
          if $status >= 500 then "ERROR"
          elif $status >= 400 then "WARN"
          else "INFO"
          end
        else
          (($row.severity_text // $row.log_attributes["level"] // "INFO") | ascii_upcase)
        end
      ),
      body: (if $row.source == "auth_logs"
        then ($row.event_message // $row.log_attributes["msg"])
        else $row.event_message
        end),
      source: $row.source,
      service: ($row | service_name),
      attributes: ($row.log_attributes // {})
    }
  ' 2>/dev/null
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
      --data-urlencode "sql=SELECT id, timestamp, event_message, source, severity_text, log_attributes FROM logs WHERE source = '${src}' ORDER BY timestamp DESC LIMIT 200" \
      --data-urlencode "iso_timestamp_start=${START}" \
      --data-urlencode "iso_timestamp_end=${END}" \
      "$API_BASE" 2>/dev/null || echo '{"result":[]}')

    if echo "$response" | jq -e '.error' >/dev/null 2>&1; then
      continue
    fi

    result_count=$(echo "$response" | jq '.result | length' 2>/dev/null || echo "0")

    if [ "$result_count" -gt 0 ]; then
      echo "$response" | normalize | emit
      POLL_SUMMARY="${POLL_SUMMARY}  ✓ ${src}: +${result_count}\n"
    fi
  done

  if [ -n "$POLL_SUMMARY" ]; then
    log_status "[Poll #${POLL_COUNT} @ $(date +%H:%M:%S)]"
    log_status -e "$POLL_SUMMARY"
  fi

  sleep "$POLL_INTERVAL"
done
