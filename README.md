# Claude Code Telegram Permission Relay

Relay Claude Code permission prompts to Telegram for remote approval/denial from your phone.

> **Disclaimer:** This project was fully vibe-coded with AI (Claude Code). It works, but you should review the code before relying on it for anything sensitive.
>
> **Status:** Work in progress. Core functionality is working but expect rough edges.

## Setup

### Prerequisites

- Go 1.22+
- `jq` and `curl` (for the hook script)
- A Telegram bot token and your user ID

### 1. Create a Telegram Bot

Message [@BotFather](https://t.me/BotFather) → `/newbot` → follow prompts. Copy the token.

Message [@userinfobot](https://t.me/userinfobot) to get your numeric user ID.

### 2. Build

```bash
go build -o claude-relay .
```

### 3. Configure

Create `.claude-relay.json` in the same directory as the binary:

```json
{
  "telegram_bot_token": "123456:ABC-DEF...",
  "allowed_telegram_user_id": 123456789,
  "port": 7654,
  "default_mode": "local",
  "telegram_mode_timeout_seconds": 300,
  "notify_in_local_mode": true
}
```

Only `telegram_bot_token` and `allowed_telegram_user_id` are required. The rest have sensible defaults. This file is gitignored.

### 4. Start the Relay

```bash
./claude-relay                          # use config defaults
./claude-relay -mode telegram           # start in telegram mode
./claude-relay -mode telegram -timeout 60  # telegram mode, 60s timeout
./claude-relay -notify=false            # disable local-mode notifications
```

You should see:
```
HTTP server listening on 127.0.0.1:7654
Telegram bot started
Claude Code Relay running (mode: local, timeout: 300s, notify: true)
```

### 5. Test in Telegram

Send `/start` to your bot. It should respond with the current mode and available commands.

### 6. Install the Hook

```bash
./scripts/install-hook.sh
```

This merges the hook config into `~/.claude/settings.json`. Open a new Claude Code session to activate it.

## Usage

### Telegram Commands

| Command | Description |
|---------|-------------|
| `/start` | Show status and available commands |
| `/mode` | Show current mode |
| `/mode local` | Switch to local mode (default prompts) |
| `/mode telegram` | Switch to telegram mode (block for response) |
| `/status` | Show pending requests |
| `/allow <id>` | Approve a pending request |
| `/deny <id>` | Deny a pending request |

### Modes

- **local** (default): Permission prompts appear in your terminal normally. Telegram gets a notification.
- **telegram**: Prompts are sent to Telegram with Allow/Deny buttons. The hook waits for a Telegram response up to the configured timeout (default 300s). If you respond on Telegram, that decision is used. If you respond locally or it times out, falls back to the local prompt.

> **Note:** Telegram mode does NOT disable local prompts. Both channels are always available — telegram mode just adds Telegram as an additional approval path. You can approve or deny from either your terminal or Telegram, whichever is faster.

### Responding locally in telegram mode

When in telegram mode, the local prompt also appears. If you respond locally instead of via Telegram:
- The Telegram message updates to show "handled locally" once the hook exits (up to the configured timeout)
- If you respond on Telegram first, that decision is used instead
- On relay shutdown (Ctrl+C), all pending messages update to "cancelled — relay shutting down"

### CLI Flags

Flags override config file defaults:

| Flag | Default | Description |
|------|---------|-------------|
| `-mode` | (from config) | Override startup mode: `local` or `telegram` |
| `-timeout` | (from config) | Telegram mode timeout in seconds |
| `-notify` | `true` | Send Telegram notifications in local mode |

### Configuration Options

| Field | Default | Description |
|-------|---------|-------------|
| `telegram_bot_token` | required | Bot token from @BotFather |
| `allowed_telegram_user_id` | required | Your Telegram user ID |
| `port` | 7654 | Local HTTP server port |
| `default_mode` | local | Startup mode |
| `telegram_mode_timeout_seconds` | 300 | How long to wait for Telegram response |
| `notify_in_local_mode` | true | Send Telegram notifications in local mode |

## Why this instead of Claude Code Remote Control?

Claude Code has a built-in [Remote Control](https://code.claude.com/docs/en/remote-control) feature that lets you continue a local session from your phone via claude.ai or the mobile app. It's the official solution and works well for full session control.

This relay is a narrower, lighter alternative:

| | Remote Control | This Relay |
|---|---|---|
| **What it does** | Full session handoff — chat, review, edit from mobile | Permission prompts only — approve/deny from mobile |
| **Requires** | claude.ai login + active Claude subscription | Claude Code + a Telegram bot (both free) |
| **Setup** | Built into Claude Code, enable via `/config` | Install a hook + run a local daemon |
| **Network** | Routes through Anthropic's servers | Localhost only, no external dependencies beyond Telegram |
| **Use case** | You want to actively code from your phone | You just want to tap approve/deny without opening a full session |

If you already use Remote Control and it covers your needs, you don't need this. This project exists for people who prefer a lightweight Telegram-based approval flow without the full mobile session overhead.

## How It Works

1. A Claude Code `PermissionRequest` hook fires when a permission dialog would appear
2. The hook script POSTs to a local Go HTTP server
3. The server forwards the request to Telegram with inline Allow/Deny buttons
4. In **telegram mode**, the hook blocks until you respond on Telegram (or times out, default 300s)
5. In **local mode** (default), the local prompt appears normally and Telegram gets a notification

## Architecture

```
Claude Code → hook script → HTTP server → Telegram bot
                                  ↑              ↓
                              handler ←── inline buttons
```

Single Go binary: HTTP server + Telegram bot in one process. All state in-memory. Listens on localhost only.

### Key design decisions

- **HandlerInterface**: The Telegram bot depends on a handler interface (not the concrete type), breaking the circular dependency between bot and handler
- **Context cancellation**: When the hook client disconnects (local response), the server detects it and updates the Telegram message
- **EXIT trap**: The hook script sends a cancel request on exit, ensuring cleanup for both local approvals and denials
- **Graceful shutdown**: Ctrl+C cancels all pending requests and updates their Telegram messages before stopping

## Uninstall

Remove the `PermissionRequest` entry from `~/.claude/settings.json`.
