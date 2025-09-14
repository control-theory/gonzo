#!/bin/bash

echo "Testing Loki in TUI mode with simulated TTY..."

# Run gonzo with script command to simulate TTY, timeout after 5 seconds
# Redirect stderr to a log file to capture debug output
timeout 5 script -q /dev/null ./gonzo --source='loki:{"url":"http://localhost:3100","query":"{job=~\".+\"}","oneShot":true}' 2>tui_debug.log || true

echo ""
echo "Checking debug log for dashboard updates..."
if [ -f tui_debug.log ]; then
    echo "Dashboard total logs updates:"
    grep "\[Dashboard\]" tui_debug.log | tail -5

    FINAL_COUNT=$(grep "\[Dashboard\]" tui_debug.log | tail -1 | grep -o "[0-9]*" | tail -1)
    echo ""
    echo "Final dashboard count: $FINAL_COUNT"

    # Clean up
    rm -f tui_debug.log typescript
else
    echo "No debug log found"
fi

echo "Test complete."