# Claude Code Telegram Permission Relay

Relay Claude Code permission prompts to Telegram for remote approval/denial from your phone.

## How It Works

1. A Claude Code `PermissionRequest` hook fires when a permission dialog would appear
2. The hook calls a local Go HTTP server
3. The server forwards requests to Telegram with inline Allow/Deny buttons
4. In **telegram mode**, the hook blocks until you respond (or times out)
5. In **local mode** (default), the local prompt appears normally and Telegram gets a notification

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

Only `telegram_bot_token` and `allowed_telegram_user_id` are required. The rest have sensible defaults.

### 4. Start the Relay

```bash
./claude-relay
```

You should see:
```
HTTP server listening on 127.0.0.1:7654
Telegram bot started
Claude Code Relay running (mode: local, timeout: 300s)
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
- **telegram**: Prompts are sent to Telegram with Allow/Deny buttons. Terminal blocks until you respond or timeout (default 300s). On timeout, falls back to local prompt.

### Configuration Options

| Field | Default | Description |
|-------|---------|-------------|
| `telegram_bot_token` | required | Bot token from @BotFather |
| `allowed_telegram_user_id` | required | Your Telegram user ID |
| `port` | 7654 | Local HTTP server port |
| `default_mode` | local | Startup mode |
| `telegram_mode_timeout_seconds` | 300 | How long to wait for Telegram response |
| `notify_in_local_mode` | true | Send Telegram notifications in local mode |

## Architecture

```
Claude Code → hook script → HTTP server → Telegram bot
                                  ↑              ↓
                              handler ←── inline buttons
```

Single Go binary: HTTP server + Telegram bot in one process. All state in-memory.

## Uninstall

Remove the `PermissionRequest` entry from `~/.claude/settings.json`.
