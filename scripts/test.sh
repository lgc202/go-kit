#!/usr/bin/env bash
set -euo pipefail

# In restricted environments (e.g. sandbox/CI), Go default caches may be unwritable.
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

export GOCACHE="${GOCACHE:-${ROOT_DIR}/.gocache}"
export GOMODCACHE="${GOMODCACHE:-${ROOT_DIR}/.gomodcache}"

args=("./...")
if [[ "${RACE:-}" == "1" ]]; then
  args+=("-race")
fi

go test "${args[@]}"

