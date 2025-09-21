#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

PORT="${PORT:-18081}"
ADDR=":${PORT}"

echo "[k8s_webhook_poc/run.sh] starting server on ${ADDR}"
if [[ "${QUIET:-0}" == "1" ]]; then
  KWEB_ADDR="$ADDR" go run ./examples/k8s_webhook_poc >/dev/null 2>&1 &
else
  KWEB_ADDR="$ADDR" go run ./examples/k8s_webhook_poc &
fi
PID=$!
trap 'kill "$PID" >/dev/null 2>&1 || true' EXIT

# Wait for readiness
for i in $(seq 1 40); do
  if curl -sS "http://127.0.0.1:${PORT}/healthz" >/dev/null 2>&1; then
    echo "[k8s_webhook_poc/run.sh] server is up"
    break
  fi
  sleep 0.25
  if ! kill -0 "$PID" >/dev/null 2>&1; then
    echo "[k8s_webhook_poc/run.sh] server process exited unexpectedly" >&2
    exit 1
  fi
  if [[ "$i" == "40" ]]; then
    echo "[k8s_webhook_poc/run.sh] timeout waiting for server" >&2
    exit 1
  fi
done

pretty() {
  if command -v jq >/dev/null 2>&1; then
    echo "$1" | jq -M .
  else
    echo "$1"
  fi
}

post_review() {
  local body="$1"
  curl -sS --fail-with-body -H 'Content-Type: application/json' -X POST "http://127.0.0.1:${PORT}/validate" --data "$body"
}

wrap_ar() {
  local obj="$1"
  cat <<JSON
{
  "apiVersion":"admission.k8s.io/v1",
  "kind":"AdmissionReview",
  "request":{
    "uid":"00000000-0000-0000-0000-000000000001",
    "kind":{"group":"demo.example.com","version":"v1","kind":"Widget"},
    "resource":{"group":"demo.example.com","version":"v1","resource":"widgets"},
    "object": ${obj}
  }
}
JSON
}

# Same as wrap_ar but allows overriding GVR
wrap_ar_gvr() {
  local obj="$1"
  local group="$2"
  local version="$3"
  local resource="$4"
  cat <<JSON
{
  "apiVersion":"admission.k8s.io/v1",
  "kind":"AdmissionReview",
  "request":{
    "uid":"00000000-0000-0000-0000-000000000001",
    "kind":{"group":"demo.example.com","version":"v1","kind":"Widget"},
    "resource":{"group":"${group}","version":"${version}","resource":"${resource}"},
    "object": ${obj}
  }
}
JSON
}

fail() { echo "[FAIL] $1" >&2; exit 1; }
pass() { echo "[PASS] $1"; }

# 1) Unknown field under spec (should be denied) due to additionalProperties: false
DESC="unknown field rejected (UnknownStrict)"
OBJ='{"apiVersion":"demo.example.com/v1","kind":"Widget","metadata":{"name":"w1"},"spec":{"name":"n","oops":"x"}}'
REQ=$(wrap_ar "$OBJ")
RESP=$(post_review "$REQ")
echo "== $DESC"; pretty "$RESP"
echo "$RESP" | grep -q '"allowed":false' || fail "$DESC: allowed should be false"

# 2) Duplicate key in items element (JSON layer) — craft via raw string (some curl versions allow)
DESC="duplicate key in items triggers denial"
OBJ='{"apiVersion":"demo.example.com/v1","kind":"Widget","metadata":{"name":"w1"},"spec":{"name":"n","items":[{"id":"a","id":"b"}]}}'
REQ=$(wrap_ar "$OBJ")
RESP=$(post_review "$REQ")
echo "== $DESC"; pretty "$RESP"
echo "$RESP" | grep -q '"allowed":false' || fail "$DESC: allowed should be false"

# 3) Success object
DESC="valid object allowed"
OBJ='{"apiVersion":"demo.example.com/v1","kind":"Widget","metadata":{"name":"w1"},"spec":{"name":"n","labels":{"a":"b"},"items":[{"id":"a","port":80}]}}'
REQ=$(wrap_ar "$OBJ")
RESP=$(post_review "$REQ")
echo "== $DESC"; pretty "$RESP"
echo "$RESP" | grep -q '"allowed":true' || fail "$DESC: allowed should be true"

if command -v jq >/dev/null 2>&1; then
  PRES=$(echo "$RESP" | jq -r '.response.auditAnnotations["goskema/presence"]')
  if [[ -n "$PRES" && "$PRES" != "null" ]]; then
    PRES_JSON=$(echo "$PRES" | jq -r 'fromjson?')
    if [[ -n "$PRES_JSON" && "$PRES_JSON" != "null" ]]; then
      DEF=$(echo "$PRES_JSON" | jq -r '.defaultApplied // 0')
      if [[ "$DEF" -lt 1 ]]; then
        fail "presence: expected defaultApplied >= 1 (replicas default)"
      fi
    fi
  fi
else
  echo "[k8s_webhook_poc/run.sh] jq not found; skipping presence assertions" >&2
fi

# 4) Nullable field produces PresenceWasNull count
DESC="nullable note counted as null in presence"
OBJ='{"apiVersion":"demo.example.com/v1","kind":"Widget","metadata":{"name":"w1"},"spec":{"name":"n","note":null,"labels":{"a":"b"},"items":[{"id":"a","port":80}]}}'
REQ=$(wrap_ar "$OBJ")
RESP=$(post_review "$REQ")
echo "== $DESC"; pretty "$RESP"
echo "$RESP" | grep -q '"allowed":true' || fail "$DESC: allowed should be true"
if command -v jq >/dev/null 2>&1; then
  PRES=$(echo "$RESP" | jq -r '.response.auditAnnotations["goskema/presence"]')
  if [[ -n "$PRES" && "$PRES" != "null" ]]; then
    PRES_JSON=$(echo "$PRES" | jq -r 'fromjson?')
    if [[ -n "$PRES_JSON" && "$PRES_JSON" != "null" ]]; then
      NULLS=$(echo "$PRES_JSON" | jq -r '.null // 0')
      if [[ "$NULLS" -lt 1 ]]; then
        fail "presence: expected null >= 1 (note:null)"
      fi
    fi
  fi
fi

# 5) GVR mismatch (resource name) should be denied
DESC="GVR mismatch denied (resource name mismatch)"
OBJ='{"apiVersion":"demo.example.com/v1","kind":"Widget","metadata":{"name":"w1"},"spec":{"name":"n","labels":{"a":"b"},"items":[{"id":"a","port":80}]}}'
REQ=$(wrap_ar_gvr "$OBJ" "demo.example.com" "v1" "gadgets")
RESP=$(post_review "$REQ")
echo "== $DESC"; pretty "$RESP"
echo "$RESP" | grep -q '"allowed":false' || fail "$DESC: allowed should be false"

echo "[k8s_webhook_poc/run.sh] all tests passed"


