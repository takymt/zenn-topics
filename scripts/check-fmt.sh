#!/bin/bash
set -euo pipefail

out="$(gofmt -l .)"

if [ -n "$out" ]; then
  echo "$out"
  echo "ERROR: Unformatted Go files found." >&2
  exit 1
fi
