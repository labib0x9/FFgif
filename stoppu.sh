#!/usr/bin/env bash

PORT="${1:-8080}"

echo "[*] Checking processes listening on TCP port $PORT..."
echo

OUTPUT=$(lsof -iTCP:$PORT -sTCP:LISTEN)

if [ -z "$OUTPUT" ]; then
    echo "[!] No process found listening on port $PORT"
    exit 1
fi

echo "$OUTPUT"
echo

# Extract PID(s), skipping header
PIDS=$(echo "$OUTPUT" | awk 'NR>1 {print $2}')

echo "[*] Found PID(s): $PIDS"
echo

read -p "Do you want to kill these process(es)? [y/N]: " CONFIRM

if [[ "$CONFIRM" =~ ^[Yy]$ ]]; then
    for PID in $PIDS; do
        kill -9 "$PID" && echo "[+] Killed PID $PID"
    done
else
    echo "[!] Aborted"
fi