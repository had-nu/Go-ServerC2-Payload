#!/usr/bin/env bash
#
# demo.sh runs the Team Server and the Lab Beacon, sends one command, shows the
# output, and stops both programs.
#
# The script uses a separate state directory and a separate port. It does not
# touch the state of a Team Server that the operator started.
#
# Usage: scripts/demo.sh <work-dir> <port>

set -uo pipefail

DIR="${1:-.demo}"
PORT="${2:-18443}"
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT" || exit 1

FIFO="$DIR/console.in"
SERVER_LOG="$DIR/server.out"
SERVER_ERR="$DIR/server.err"
BEACON_LOG="$DIR/beacon.out"
BEACON_ERR="$DIR/beacon.err"
PIDFILE="$DIR/pids"

say() { printf '%s\n' "$*"; }
step() { printf '\n\033[1m== %s\033[0m\n' "$*"; }
fail() { printf 'demo: %s\n' "$*" >&2; cleanup; exit 1; }

cleanup() {
  if [ -f "$PIDFILE" ]; then
    while read -r pid; do
      kill "$pid" 2>/dev/null
    done < "$PIDFILE"
    rm -f "$PIDFILE"
  fi
  exec 3>&- 2>/dev/null
  wait 2>/dev/null
}

trap cleanup EXIT INT TERM

mkdir -p "$DIR"
rm -f "$FIFO" "$PIDFILE"
: > "$PIDFILE"

# The cert and the secret live only in the demo directory.
STATE="$DIR/state"
rm -rf "$STATE"

step "1 of 6  start the Team Server"
mkfifo "$FIFO"
./bin/team-server -addr "127.0.0.1:$PORT" -state "$STATE" < "$FIFO" > "$SERVER_LOG" 2> "$SERVER_ERR" &
SRV=$!
echo "$SRV" >> "$PIDFILE"
exec 3> "$FIFO"

for _ in $(seq 1 50); do
  if grep -q 'team server listening' "$SERVER_LOG" 2>/dev/null; then break; fi
  if ! kill -0 "$SRV" 2>/dev/null; then
    cat "$SERVER_ERR" >&2
    fail "the Team Server stopped during start-up"
  fi
  sleep 0.1
done
grep -q 'team server listening' "$SERVER_LOG" || fail "the Team Server did not start"
say "listening on $(grep -o 'listening on .*' "$SERVER_LOG" | head -1 | sed 's/listening on //')"
say "secret: $STATE/c2.key"

step "2 of 6  start the Lab Beacon"
./bin/beacon \
  -server "https://127.0.0.1:$PORT" \
  -secret-file "$STATE/c2.key" \
  -ca "$STATE/team-server.crt" \
  -sleep 3s -jitter 20 \
  > "$BEACON_LOG" 2> "$BEACON_ERR" &
BN=$!
echo "$BN" >> "$PIDFILE"
say "beacon pid $BN"

step "3 of 6  wait for the check-in"
AGENT=""
for _ in $(seq 1 60); do
  AGENT=$(grep -oE 'agent registered: [0-9a-f]+' "$SERVER_LOG" 2>/dev/null | head -1 | awk '{print $3}')
  [ -n "$AGENT" ] && break
  kill -0 "$BN" 2>/dev/null || { cat "$BEACON_ERR" >&2; fail "the Lab Beacon stopped"; }
  sleep 0.1
done
[ -n "$AGENT" ] || { cat "$BEACON_ERR" >&2; fail "no agent checked in"; }
say "agent $AGENT checked in"
grep 'agent registered' "$SERVER_LOG" | tail -1

step "4 of 6  select the agent and queue a command"
echo "interact $AGENT" >&3
sleep 0.3
say "command: id"
echo "id" >&3

step "5 of 6  wait for the result"
RESULT=""
for _ in $(seq 1 80); do
  RESULT=$(grep -E '^uid=' "$SERVER_LOG" 2>/dev/null | head -1)
  [ -n "$RESULT" ] && break
  sleep 0.1
done
[ -n "$RESULT" ] || { cat "$SERVER_LOG" >&2; fail "no result arrived"; }
printf '\n%s\n' "$RESULT"

step "6 of 6  check the agent and stop"
echo "agents" >&3
sleep 0.4
echo "exit" >&3
sleep 0.5

say ""
say "the Lab Beacon process name:"
say "  $(grep 'beacon: id=' "$BEACON_ERR" | head -1)"
say ""
say "the Team Server wrote:"
sed -n '/agent registered/,$p' "$SERVER_LOG" | sed 's/^/  /'
say ""
say "demo finished. The state is in $DIR/state. Run 'make clean' to remove it."
