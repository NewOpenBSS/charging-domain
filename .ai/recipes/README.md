# Recipes

Goose recipe files for the agentic development pipeline.
These are backup copies — the live copies that Goose runs are in
`~/.config/goose/recipes/`.

## Installing on a New Machine

Copy all recipe files to the Goose recipes directory:

```bash
cp .ai/recipes/*.yaml ~/.config/goose/recipes/
```

## Recipes

| Recipe | Stage | Purpose |
|---|---|---|
| `requirements-session.yaml` | Stage 1 — Requirements | Conversational capture of raw ideas into REQUIREMENTS.md |
| `feature-scoping.yaml` | Stage 2 — Scoping | Distils a requirement into one or more Features in FEATURES.md |
| `feature-design.yaml` | Stage 3 — Feature Design | Decomposes a Feature into branches + CURRENT.md |
| `dev-session.yaml` | Stage 4 — Development | Implements a feature autonomously to PR |
| `design-session.yaml` | Utility | Legacy simple worktree creation |

## Pipeline Flow

```
requirements-session  →  REQUIREMENTS.md  (R-NNN)
        ↓
feature-scoping       →  FEATURES.md      (F-NNN)
        ↓
feature-design        →  branch + CURRENT.md
        ↓
dev-session           →  PR
```
