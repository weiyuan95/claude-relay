# CLAUDE.md

## Project: Claude Code Telegram Permission Relay

A local Go service that relays Claude Code permission requests to Telegram, allowing remote approval/denial from a phone.

### Key Concepts
- **Modes**: `local` (default, non-blocking) vs `telegram` (blocking, waits for Telegram response)
- **Hook type**: Claude Code `PermissionRequest` hook — fires when a permission dialog would appear
- **Hook stdin JSON**: Claude Code sends `tool_name` and `tool_input` (with `command`/`description` fields), plus `session_id`. NOT `input` — the field is `tool_input`.
- **Timeout**: Configurable, default 300s. Hook script uses 60s curl timeout.
- **Multiple sessions**: Supported via session IDs in Telegram messages
- **CLI flags**: `-mode`, `-timeout`, `-notify` override config file defaults

### Architecture
- Single Go binary (`claude-relay`) — HTTP server + Telegram bot
- Shell hook script (`hook/claude-relay-hook`) — called by Claude Code, generates its own request ID, has EXIT trap for cleanup
- All state in-memory, no database
- Listens on localhost only
- Bot uses `HandlerInterface` to avoid circular dependency with concrete `hook.Handler`
- Context cancellation detects client disconnect (local response); EXIT trap in hook script sends `/cancel` on exit
- Graceful shutdown calls `ShutdownAll()` to cancel pending requests and update Telegram messages

### Commands
```bash
# Build
go build -o claude-relay .

# Run (uses config defaults)
./claude-relay

# Run with overrides
./claude-relay -mode telegram -timeout 60

# Run tests
go test ./... -v

# Install hook into ~/.claude/settings.json
./scripts/install-hook.sh
```

### Configuration
Config file at `.claude-relay.json` (next to the binary):
```json
{
  "telegram_bot_token": "...",
  "allowed_telegram_user_id": 123456,
  "port": 7654,
  "default_mode": "local",
  "telegram_mode_timeout_seconds": 300,
  "notify_in_local_mode": true
}
```

### API Endpoints
- `POST /permission` — Handle permission request (blocking in telegram mode)
- `POST /cancel` — Cancel a pending request (called by hook script EXIT trap)
- `GET /status` — Current mode and pending requests
- `POST /mode` — Change mode (local/telegram)

### File Structure
```
claude-code-relay/
├── CLAUDE.md                    # This file
├── README.md                    # User-facing docs
├── main.go                      # Entry point — wires server + bot, CLI flags, graceful shutdown
├── internal/
│   ├── config/                  # Config struct + loading from .claude-relay.json
│   ├── hook/                    # Handler: pending request state machine, mode, Notifier interface
│   ├── model/                   # Shared types
│   ├── server/                  # HTTP server: /permission, /cancel, /status, /mode
│   └── telegram/                # Telegram bot: HandlerInterface, inline keyboards, callbacks
├── hook/
│   └── claude-relay-hook        # Shell script — reads stdin, POSTs to daemon, EXIT trap cleanup
└── scripts/
    └── install-hook.sh          # Installs hook into ~/.claude/settings.json
```
