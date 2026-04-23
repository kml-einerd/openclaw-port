#!/usr/bin/env bash
set -euo pipefail

# Adapted from openclaw/scripts/pr-lib/common.sh.
# Provides helper functions for CI validation scripts.

require_artifact() {
  local artifact_path="$1"
  if [ ! -f "$artifact_path" ]; then
    echo "ERROR: Required artifact not found: $artifact_path" >&2
    exit 1
  fi
}

check_recipe_artifacts() {
  local recipe_dir="$1"
  echo "Validating recipes in $recipe_dir..."
  
  # Uses the PM-OS validator
  if ! go run ./cmd/validate-recipes/... "$recipe_dir"/*.v2.json; then
    echo "ERROR: Recipe validation failed in $recipe_dir." >&2
    exit 1
  fi
  
  echo "All recipes in $recipe_dir are valid."
}
