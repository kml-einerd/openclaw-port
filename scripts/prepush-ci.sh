#!/usr/bin/env bash
set -euo pipefail

# Quick CI sanity check before git push
# Adapted from openclaw workflow scripts
echo "Running prepush sanity checks..."

go build ./... || exit 1
go test -short ./... || exit 1
go vet ./... || exit 1

echo "Prepush checks: PASS"
