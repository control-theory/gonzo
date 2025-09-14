#!/bin/bash

echo "Testing Loki log counting..."

# Count logs in Loki
LOKI_COUNT=$(curl -s 'http://localhost:3100/loki/api/v1/query_range?query=%7Bjob%3D~%22.%2B%22%7D&limit=1000' | jq '[.data.result[].values | length] | add')
echo "Logs in Loki: $LOKI_COUNT"

# Run gonzo with oneShot mode for proper testing
echo "Running gonzo with Loki source (oneShot mode)..."
OUTPUT=$(timeout 10 ./gonzo --test-mode --source='loki:{"url":"http://localhost:3100","query":"{job=~\".+\"}","oneShot":true}' 2>&1)

# Check if we got test mode results (removing ANSI codes for display)
echo "Gonzo output:"
echo "$OUTPUT" | sed 's/\x1b\[[0-9;]*[a-zA-Z]//g' | grep -E "Total lines:|Unique|Test Mode" | head -5

# Extract the total lines count (removing ANSI codes first)
GONZO_COUNT=$(echo "$OUTPUT" | sed 's/\x1b\[[0-9;]*[a-zA-Z]//g' | grep "Total lines:" | head -1 | awk '{print $3}')

# Remove any whitespace and control characters
LOKI_COUNT=$(echo $LOKI_COUNT | tr -d '[:space:][:cntrl:]')
GONZO_COUNT=$(echo $GONZO_COUNT | tr -d '[:space:][:cntrl:]')

echo ""
echo "Comparison:"
echo "  Loki has: $LOKI_COUNT logs"
echo "  Gonzo processed: $GONZO_COUNT logs"

if [ "$LOKI_COUNT" = "$GONZO_COUNT" ]; then
    echo "✓ Counts match!"
else
    echo "✗ Counts don't match"
    echo "  Debug: Loki=['$LOKI_COUNT'] len=${#LOKI_COUNT}"
    echo "  Debug: Gonzo=['$GONZO_COUNT'] len=${#GONZO_COUNT}"
fi

echo ""
echo "Test complete."