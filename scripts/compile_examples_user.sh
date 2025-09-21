#!/usr/bin/env bash
set -euo pipefail

# Resolve repo root (scripts/..)
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

OUT="$ROOT_DIR/examples/user/gen_user.go"

echo "[compile-dsl] building CLI..."
go build ./cmd/goskema

echo "[compile-dsl] generating $OUT"
./goskema compile-dsl -v -pkgdir ./examples/user -symbol DSL -type User -o "$OUT"

echo "[compile-dsl] done: $OUT"
