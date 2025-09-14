#!/bin/bash

# Script to send test logs directly to Loki

LOKI_URL="${LOKI_URL:-http://localhost:3100}"

# Function to send a log entry to Loki
send_log() {
    local message="$1"
    local level="${2:-info}"
    local timestamp=$(date +%s%N)
    
    curl -s -X POST "${LOKI_URL}/loki/api/v1/push" \
        -H "Content-Type: application/json" \
        -d "{
            \"streams\": [
                {
                    \"stream\": {
                        \"job\": \"test-app\",
                        \"level\": \"${level}\",
                        \"source\": \"test-script\"
                    },
                    \"values\": [
                        [\"${timestamp}\", \"${message}\"]
                    ]
                }
            ]
        }"
}

# Send some test logs
echo "Sending test logs to Loki at ${LOKI_URL}..."

send_log "Application started successfully" "info"
sleep 0.1

send_log "Processing user request id=12345" "info"
sleep 0.1

send_log "Warning: Cache miss for key user:12345" "warn"
sleep 0.1

send_log "Error: Database connection timeout" "error"
sleep 0.1

send_log "Retrying database connection..." "info"
sleep 0.1

send_log "Database connection restored" "info"
sleep 0.1

send_log "Processing completed for request id=12345" "info"

echo "Done! Sent 7 test log entries to Loki"
echo ""
echo "You can now test gonzo with:"
echo "  ./gonzo --plugin-source=loki --plugin-config='{\"url\":\"${LOKI_URL}\",\"query\":\"{job=\\\"test-app\\\"}\"}'\"