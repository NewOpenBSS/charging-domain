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

## Step 5 — Create the Workflow Recipes

Goose recipes automate the development workflow. Two recipes are needed — one for
design (interactive, Desktop app), one for implementation (autonomous, CLI).

Recipes live in `.goose/recipes/` inside the repo — versioned alongside the code.

### The Four Recipes

| Recipe | Stage | Tool | Purpose |
|---|---|---|---|
| `feature-scoping.yaml` | Stage 2 — Human Design | Goose Desktop | Distils a raw idea into a well-formed Feature in FEATURES.md |
| `feature-design.yaml` | Stage 3 — AI Design | Goose Desktop or CLI | Decomposes a Feature into features, creates worktrees + CURRENT.md |
| `dev-session.yaml` | Stage 4 — Implementation | CLI (autonomous) | Implements a feature from CURRENT.md to PR |
| `design-session.yaml` | Utility | Goose Desktop | Legacy — simple worktree creation without Feature process |

Run this in the **main repository directory** using **Goose Desktop**.
It creates the feature worktree and summarises what will be built.

Parameters it asks for:
- `feature_name` — short name for the branch e.g. `wholesaler-admin`
- `feature_description` — plain language description of the feature
- `repo_name` — repository directory name e.g. `go-ocs`

### dev-session.yaml

Run this **inside the feature worktree** from the **CLI**.
It reads all context files, confirms the requirement once, then implements
autonomously through to a PR with no further interruptions.

Parameters it asks for:
- `feature_description` — plain language description of what to build
- `repo_path` — absolute path to the worktree e.g.
  `/Users/eddiecarpenter/Development/goplay/branches/go-ocs-wholesaler-admin`

```bash
# Run the dev session from inside the feature worktree
goose run --recipe .goose/recipes/dev-session.yaml
```

### Full Workflow

```
1. Open Goose Desktop
   → Run design-session recipe
   → Fill in feature name and description
   → Worktree is created

2. Open the worktree in your editor of choice

3. Open terminal in the worktree directory
   → goose run --recipe .goose/recipes/dev-session.yaml
   → Fill in parameters
   → Confirm the summary
   → Walk away — runs autonomously to PR
```

---

## Notes

- Recipes are stored in `.goose/recipes/` inside the repo — always in sync
- Copy them to the same location on any new machine
- The `max_turns: 200` setting in dev-session.yaml gives enough headroom for
  complex multi-task features without hitting the default turn limit
- Design session is interactive by nature — Goose Desktop is better suited
- Dev session is autonomous — CLI is better suited


---

## Troubleshooting

**`goose: command not found`**
Add `~/.local/bin` to PATH in `~/.zshrc` and restart your terminal.

**Claude Code not authenticated**
Run `claude setup-token` and complete the browser OAuth flow.

**Session starts but fails immediately**
Confirm Claude Code works independently: `claude --version`
If authenticated, restart Goose desktop app to pick up config changes.
