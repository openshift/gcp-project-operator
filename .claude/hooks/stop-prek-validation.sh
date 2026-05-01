#!/usr/bin/env bash
set -uo pipefail

HOOK_INPUT=$(cat)

# Check for retry loop WITHOUT requiring jq (prevents infinite loop)
if echo "$HOOK_INPUT" | grep -q '"stop_hook_active"[[:space:]]*:[[:space:]]*true'; then
  exit 0
fi

# Check for jq (will block FIRST time, but loop protection above prevents infinite loop)
if ! command -v jq &>/dev/null; then
  echo '{"decision": "block", "reason": "jq is not installed. Install jq to use prek validation hooks. See CONTRIBUTING.md for setup instructions."}'
  exit 0
fi

# jq is available - use it to check loop protection properly
STOP_HOOK_ACTIVE=$(echo "$HOOK_INPUT" | jq -r '.stop_hook_active // false')
if [[ "$STOP_HOOK_ACTIVE" == "true" ]]; then
  exit 0
fi

# Check for prek
if ! command -v prek &>/dev/null; then
  jq -n --arg reason "prek is not installed. Install prek to use validation hooks. See CONTRIBUTING.md for setup instructions." \
    '{"decision": "block", "reason": $reason}'
  exit 0
fi

# Run prek validation
PREK_OUTPUT=$(prek run --all-files 2>&1)
PREK_EXIT=$?

if [[ $PREK_EXIT -eq 0 ]]; then
  exit 0
fi

# Block stop and tell Claude what to fix
jq -n \
  --arg reason "prek run --all-files failed. Fix the issues below, then try again:

$PREK_OUTPUT" \
  '{"decision": "block", "reason": $reason}'
