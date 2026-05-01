#!/usr/bin/env bash
set -euo pipefail

if ! command -v prek &>/dev/null; then
  echo "Error: prek is not installed. See CONTRIBUTING.md for setup instructions." >&2
  exit 1
fi

prek run --all-files
