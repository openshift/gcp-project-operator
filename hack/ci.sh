#!/usr/bin/env bash
set -euo pipefail

# Change to repository root to ensure prek finds prek.toml
REPO_ROOT=$(git rev-parse --show-toplevel 2>/dev/null || dirname "$(dirname "$(readlink -f "$0")")")
cd "$REPO_ROOT"

if ! command -v prek &>/dev/null; then
  echo "Error: prek is not installed. See CONTRIBUTING.md for setup instructions." >&2
  exit 1
fi

prek run --all-files
