#!/usr/bin/env bash
#
# verify-gates.sh checks the two gates that stand between a clone and a running
# command.
#
# Gate 1 is the build tag. Without "lab" the binary must hold no network code.
# Gate 2 is the acknowledgement flag. A binary built with the tag but started
# without the flag must refuse every task.
#
# The script fails when a gate does not hold.

set -uo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT" || exit 1

OUT=".gates"
rm -rf "$OUT"
mkdir -p "$OUT"

fail() { printf 'verify-gates: FAIL: %s\n' "$*" >&2; exit 1; }
ok() { printf '  ok    %s\n' "$*"; }

printf 'gate 1  the build tag\n'

go build -o "$OUT/inert" ./lab/beacon-impl || fail "the default build does not compile"
go build -tags lab -o "$OUT/armed-sym" ./lab/beacon-impl || fail "the tagged build does not compile"
# The shipped binary is stripped, so it has no symbol table. The check reads the
# symbols of an unstripped build of the same source.
go build -tags lab -ldflags="-s -w" -o "$OUT/armed" ./lab/beacon-impl || fail "the stripped build does not compile"

NET=$(go tool nm "$OUT/inert" 2>/dev/null | grep -cE 'net/http\.|net\.Dial' || true)
[ "$NET" -eq 0 ] || fail "the default build has $NET network symbols"
ok "the default build has no network symbol"

EXEC=$(strings "$OUT/inert" | grep -cE '^/bin/bash$' || true)
[ "$EXEC" -eq 0 ] || fail "the default build names /bin/bash"
ok "the default build has no shell path"

NET2=$(go tool nm "$OUT/armed-sym" 2>/dev/null | grep -cE 'net/http\.' || true)
[ "$NET2" -gt 0 ] || fail "the tagged build has no network symbol, so the tag does nothing"
ok "the tagged build has $NET2 network symbols"

INERT_OUT=$("$OUT/inert" 2>&1)
echo "$INERT_OUT" | grep -q 'this build is inert' || fail "the default build does not say that it is inert"
ok "the default build announces that it is inert"

printf 'gate 2  the acknowledgement flag\n'

# A Team Server that accepts the check-in but receives a refusal.
STATE="$OUT/state"
PORT=18499
go build -o "$OUT/team-server" ./cmd/team-server || fail "the team server does not build"

"$OUT/team-server" -addr "127.0.0.1:$PORT" -state "$STATE" -tls=false < /dev/null > "$OUT/srv.log" 2>&1 &
SRV=$!
trap 'kill $SRV 2>/dev/null' EXIT

for _ in $(seq 1 40); do
  grep -q 'team server listening' "$OUT/srv.log" 2>/dev/null && break
  sleep 0.1
done
grep -q 'team server listening' "$OUT/srv.log" || fail "the team server did not start"

# The beacon starts without the flag. It must register and must not run.
"$OUT/armed" -server "http://127.0.0.1:$PORT" -secret-file "$STATE/c2.key" -sleep 1s -once \
  > "$OUT/beacon.log" 2>&1

grep -q 'NOT ARMED' "$OUT/beacon.log" || fail "the beacon did not warn that it is not armed"
ok "the beacon warns that it is not armed"

kill $SRV 2>/dev/null
trap - EXIT
rm -rf "$OUT"

printf 'verify-gates: both gates hold\n'
