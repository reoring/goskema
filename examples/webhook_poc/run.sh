#!/usr/bin/env bash
set -euo pipefail

# Resolve repo root (this file is at examples/webhook_poc/run.sh)
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

PORT="${PORT:-8081}"
ADDR=":${PORT}"

echo "[run.sh] starting PoC server on ${ADDR}"
POC_ADDR="$ADDR" go run ./examples/webhook_poc >/dev/null 2>&1 &
PID=$!
trap 'kill "$PID" >/dev/null 2>&1 || true' EXIT

# Wait for server readiness
for i in $(seq 1 40); do
  if curl -sS -X POST "http://127.0.0.1:${PORT}/items" -H 'Content-Type: application/json' --data '[]' >/dev/null 2>&1; then
    echo "[run.sh] server is up"
    break
  fi
  sleep 0.25
  if ! kill -0 "$PID" >/dev/null 2>&1; then
    echo "[run.sh] server process exited unexpectedly" >&2
    exit 1
  fi
  if [[ "$i" == "40" ]]; then
    echo "[run.sh] timeout waiting for server" >&2
    exit 1
  fi
done

fail() { echo "[FAIL] $1" >&2; exit 1; }
pass() { echo "[PASS] $1"; }

# pretty print JSON using jq when available
pretty() {
  if command -v jq >/dev/null 2>&1; then
    # Color policy:
    # - If PRETTY_COLOR=0, force no color
    # - Else if stdout is a TTY, enable color (-C)
    # - Otherwise (CI/pipes), disable color (-M)
    if [[ "${PRETTY_COLOR:-1}" == "0" ]]; then
      echo "$1" | jq -M .
    elif [ -t 1 ]; then
      echo "$1" | jq -C .
    else
      echo "$1" | jq -M .
    fi
  else
    echo "$1"
  fi
}

# HTTP helper: POST JSON and capture status/body
do_post() {
  local path="$1"; shift
  local data="$1"; shift
  local url="http://127.0.0.1:${PORT}${path}"
  local tmp
  tmp="$(mktemp)"
  local code
  code=$(curl -sS -o "$tmp" -w '%{http_code}' -H 'Content-Type: application/json' -X POST "$url" --data "$data" || true)
  BODY="$(cat "$tmp")"
  STATUS="$code"
  rm -f "$tmp"
}

print_scenario() {
  echo "== Scenario: $1"
  echo "   Given:  $2"
  echo "   When:   $3"
  echo "   Expect: $4"
}

# --- /items: error case ---
DESC="/items invalid element type triggers per-index error"
REQ='[{"id":"a"},{"id":1},{"id":"c"}]'
print_scenario "$DESC" "array with an invalid element at index 1 (id=number)" "POST /items" "HTTP 400 and Issues[0] Path=/1 Code=parse_error"
do_post "/items" "$REQ"
echo ">> Request:"; pretty "$REQ"
echo "<< Response (status=$STATUS):"; pretty "$BODY"
[[ "$STATUS" == "400" ]] || fail "$DESC: http status not 400 (got $STATUS)"
grep -q '"issues"' <<<"$BODY" || fail "$DESC: no issues key"
grep -q '"Path":"/1"' <<<"$BODY" || fail "$DESC: path not /1"
# Accept either invalid_type or parse_error depending on implementation detail
grep -Eq '"Code":"(invalid_type|parse_error)"' <<<"$BODY" || fail "$DESC: unexpected code"
pass "$DESC"

# --- /items: success case ---
DESC="/items success returns canonical array and presence"
REQ='[{"id":"a","note":"n"},{"id":"b"}]'
print_scenario "$DESC" "two valid items" "POST /items" "HTTP 200 and ok=true, presence present"
do_post "/items" "$REQ"
echo ">> Request:"; pretty "$REQ"
echo "<< Response (status=$STATUS):"; pretty "$BODY"
[[ "$STATUS" == "200" ]] || fail "$DESC: http status not 200 (got $STATUS)"
grep -q '"ok":true' <<<"$BODY" || fail "$DESC: ok not true"
grep -q '"presence"' <<<"$BODY" || fail "$DESC: presence missing"
pass "$DESC"

# --- /webhook: error case ---
DESC="/webhook strict object validates nested items and reports errors"
REQ='{ "source":"poc", "items":[{"id":"a"},{"id":1},{"id":"c"}], "meta": {"trace":"t-1"} }'
print_scenario "$DESC" "object with items[1].id invalid" "POST /webhook" "HTTP 400 and Issues present (path may be /items/1 or /)"
do_post "/webhook" "$REQ"
echo ">> Request:"; pretty "$REQ"
echo "<< Response (status=$STATUS):"; pretty "$BODY"
[[ "$STATUS" == "400" ]] || fail "$DESC: http status not 400 (got $STATUS)"
grep -q '"issues"' <<<"$BODY" || fail "$DESC: no issues key"
if grep -q '"Path":"/items/1"' <<<"$BODY"; then
  :
elif grep -q '"Path":"/"' <<<"$BODY"; then
  :
else
  fail "$DESC: unexpected path"
fi
pass "$DESC"

# --- /webhook: success case ---
DESC="/webhook success returns object canonical, presence and preserving"
REQ='{ "source":"poc", "items":[{"id":"a","note":"n"},{"id":"b"}], "meta": {"trace":"t-2"} }'
print_scenario "$DESC" "valid top-level object with items/meta" "POST /webhook" "HTTP 200 and ok=true, presence+preserving present"
do_post "/webhook" "$REQ"
echo ">> Request:"; pretty "$REQ"
echo "<< Response (status=$STATUS):"; pretty "$BODY"
[[ "$STATUS" == "200" ]] || fail "$DESC: http status not 200 (got $STATUS)"
grep -q '"ok":true' <<<"$BODY" || fail "$DESC: ok not true"
grep -q '"presence"' <<<"$BODY" || fail "$DESC: presence missing"
grep -q '"preserving"' <<<"$BODY" || fail "$DESC: preserving missing"
pass "$DESC"

echo "[run.sh] all tests passed"

