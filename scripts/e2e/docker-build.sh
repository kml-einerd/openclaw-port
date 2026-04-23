#!/usr/bin/env bash
set -euo pipefail

# Adapted from openclaw/scripts/lib/docker-e2e-image.sh.
# Helper script to build docker images or skip if they already exist
# when SKIP_DOCKER_BUILD=1 is set. Useful for accelerating E2E test runs.

# Usage: docker_build_or_skip "image_name:tag" "path/to/Dockerfile"
docker_build_or_skip() {
  local image_name="$1"
  local dockerfile="$2"
  
  if [ "${SKIP_DOCKER_BUILD:-0}" = "1" ]; then
    if ! docker image inspect "$image_name" >/dev/null 2>&1; then
      echo "ERROR: image not found: $image_name" >&2
      exit 1
    fi
    echo "Reusing: $image_name"
    return
  fi
  
  echo "Building: $image_name"
  docker build -t "$image_name" -f "$dockerfile" . || exit 1
}

# Provide an entry point if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  if [ "$#" -lt 2 ]; then
    echo "Usage: $0 <image_name> <dockerfile>"
    exit 1
  fi
  docker_build_or_skip "$1" "$2"
fi
