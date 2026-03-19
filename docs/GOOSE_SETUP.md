# Goose Setup Guide

Step-by-step instructions for setting up the Goose agentic development environment.
Follow this guide to reproduce the setup on any machine.

---

## Prerequisites

- macOS (or Linux — commands are equivalent)
- Claude Code subscription (Pro or Max)
- Goose installed (desktop app or CLI)
- Node.js (for Claude Code)

---

## Step 1 — Install Claude Code

Claude Code must be installed and authenticated before Goose can use it.

```bash
# Install Claude Code globally
npm install -g @anthropic-ai/claude-code

# Verify installation
claude --version

# Authenticate (opens browser for OAuth)
claude setup-token
```

Confirm authentication works by running `claude --version` without errors.

---

## Step 2 — Install Goose

### macOS desktop app
Download from: https://block.github.io/goose/

The CLI is installed to `~/.local/bin/goose` automatically.

Verify `~/.local/bin` is in your PATH. Add to `~/.zshrc` if needed:
```bash
export PATH=~/.local/bin:$PATH
```

### Linux (CLI only)
```bash
curl -fsSL https://github.com/block/goose/releases/download/stable/download_cli.sh | bash
```

Verify installation:
```bash
goose --version
```

---

## Step 3 — Configure Goose to use Claude Code via ACP

Goose defaults to calling the Anthropic API directly (per-token cost).
Switching to Claude Code uses your subscription instead — no extra API cost.

Edit `~/.config/goose/config.yaml`:

```yaml
# Change from:
GOOSE_PROVIDER: anthropic
GOOSE_MODEL: claude-sonnet-4-6

# To:
GOOSE_PROVIDER: claude-code
GOOSE_MODEL: default
```

The rest of the config file (extensions, MCP servers, etc.) remains unchanged.

---

## Step 4 — Verify the Configuration

Open a terminal and start a Goose session:

```bash
goose session --debug
```

Confirm the startup line shows:
```
starting session | provider: claude-code model: default
```

Type a simple test message, then `/exit` to end the session.

---

## Notes

- Goose 1.28.0+ required for stable ACP support with Claude Code
- Claude Code 2.1.77+ required
- The `--debug` flag shows full tool responses and provider info
- MCP extensions configured in Goose are passed through to Claude via ACP
- Rate limits are governed by your Claude subscription, not the Anthropic API

---

## Troubleshooting

**`goose: command not found`**
Add `~/.local/bin` to PATH in `~/.zshrc` and restart your terminal.

**Claude Code not authenticated**
Run `claude setup-token` and complete the browser OAuth flow.

**Session starts but fails immediately**
Confirm Claude Code works independently: `claude --version`
If authenticated, restart Goose desktop app to pick up config changes.
