#!/bin/bash
# install-hook.sh — Installs the PermissionRequest hook into Claude Code settings
# Usage: ./scripts/install-hook.sh [/path/to/claude-relay-hook]

set -euo pipefail

HOOK_SCRIPT="${1:-$(dirname "$0")/../hook/claude-relay-hook}"
HOOK_SCRIPT="$(cd "$(dirname "$HOOK_SCRIPT")" && pwd)/$(basename "$HOOK_SCRIPT")"

if [ ! -f "$HOOK_SCRIPT" ]; then
  echo "Error: Hook script not found at $HOOK_SCRIPT"
  exit 1
fi

SETTINGS_FILE="$HOME/.claude/settings.json"

if [ ! -f "$SETTINGS_FILE" ]; then
  echo "Error: Claude Code settings not found at $SETTINGS_FILE"
  exit 1
fi

# Use jq to merge the hook config into existing settings
# This adds the PermissionRequest hook without removing other settings
TMP_FILE=$(mktemp)
jq --arg hook "$HOOK_SCRIPT" '
  .hooks = (.hooks // {}) |
  .hooks.PermissionRequest = [
    {
      "matcher": "",
      "hooks": [
        {
          "type": "command",
          "command": $hook,
          "timeout": 300
        }
      ]
    }
  ]
' "$SETTINGS_FILE" > "$TMP_FILE"

mv "$TMP_FILE" "$SETTINGS_FILE"
echo "Hook installed successfully."
echo "Hook command: $HOOK_SCRIPT"
echo ""
echo "To uninstall, remove the PermissionRequest entry from $SETTINGS_FILE"
