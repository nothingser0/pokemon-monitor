#!/bin/bash

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LOG_FILE="$SCRIPT_DIR/monitor.log"
PID_FILE="$SCRIPT_DIR/.monitor.pid"

if [ -f "$PID_FILE" ]; then
    OLD_PID=$(cat "$PID_FILE")
    if kill -0 "$OLD_PID" 2>/dev/null; then
        echo "Already running (PID: $OLD_PID)"
        exit 0
    fi
fi

nohup python3 "$SCRIPT_DIR/monitor.py" >> "$LOG_FILE" 2>&1 &

MONITOR_PID=$!
echo "$MONITOR_PID" > "$PID_FILE"

echo "Pokemon monitor started (PID: $MONITOR_PID)"
echo "Log: tail -f $LOG_FILE"
echo "Stop: kill $MONITOR_PID"
