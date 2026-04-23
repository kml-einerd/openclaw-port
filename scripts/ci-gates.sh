#!/usr/bin/env bash
set -euo pipefail

# Adapted from openclaw/scripts/pr-lib/gates.sh.

echo "Running PM-OS CI Gates..."

# 1. Validation
if [ -d "recipes" ]; then
  go run ./cmd/validate-recipes/... recipes/*.v2.json || exit 1
fi

# 2. Cycles check
go run tools/pm-audit/cycle.go ./... || exit 1

# 3. Startup benchmark budget check
go run cmd/pm-bench/main.go --budget 200ms || exit 1

echo "All gates: PASS"
