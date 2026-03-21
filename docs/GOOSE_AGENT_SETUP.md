# Goose Agent Setup

This document describes how to set up the `goose-agent` GitHub account for a new
repository. The goose-agent account is the identity used by all AI-driven workflows.
All commits, PRs, and issue comments made by the pipeline are attributed to this account.

---

## Why a Dedicated Agent Account

- Every AI-authored commit is clearly attributed in `git log` and `git blame`
- PRs opened by the pipeline are visually distinct from human PRs in the GitHub UI
- Issue assignment to `goose-agent` is the explicit human authorisation gate
- The account can be suspended as an instant kill switch for all AI workflows
- The audit trail is clean — humans and AI activity are never mixed

---

## Step 1 — Create the GitHub Account

1. Create a new GitHub account at https://github.com/signup
2. Use a dedicated email address — an iCloud alias works well:
   - Go to https://appleid.apple.com
   - Sign In → Sign-In and Security → iCloud Email Addresses → Add
   - Create an alias (e.g. `goose-agent@icloud.com`)
   - All emails forward to your main inbox
3. Suggested username: `goose-agent` (or `newopenbss-bot` for org-level agents)
4. Verify the email address

**Note your GitHub user ID** — needed for the noreply email format:
```bash
gh api users/goose-agent --jq '.id'
```
The noreply email format is: `{ID}+goose-agent@users.noreply.github.com`

---

## Step 2 — Invite as Repository Collaborator

In the delivery repository (e.g. `go-ocs`):

1. Go to: `Settings → Access → Collaborators → Add people`
2. Search for `goose-agent`
3. Grant **Write** access
   - Write is sufficient: push branches, open PRs, be assigned to issues
   - Do NOT grant Admin or Maintain access

If using a GitHub Organisation:
1. Add `goose-agent` as an org member with Member role
2. Grant Write access to the specific repositories it needs

---

## Step 3 — Create a Personal Access Token (PAT)

Log in to GitHub as `goose-agent` and:

1. Go to: `Settings → Developer Settings → Personal Access Tokens → Tokens (classic)`
2. Click **Generate new token (classic)**
3. Name: `{repo-name}-pipeline` (e.g. `go-ocs-pipeline`)
4. Expiration: No expiration (or 1 year — set a reminder to rotate)
5. Select scopes:
   - ✅ `repo` — full repository access (push, PRs, issues)
   - ✅ `workflow` — trigger GitHub Actions workflows
6. Click **Generate token** and copy it immediately

---

## Step 4 — Add PAT as Repository Secret

Back on your main account, in the delivery repository:

1. Go to: `Settings → Secrets and variables → Actions → New repository secret`
2. Name: `GOOSE_AGENT_PAT`
3. Value: paste the PAT from Step 3
4. Click **Add secret**

---

## Step 5 — Update Workflow Files

All workflow files must use `GOOSE_AGENT_PAT` and `goose-agent` identity.

Replace in every workflow:
```yaml
# Before
token: ${{ secrets.GITHUB_TOKEN }}
git config user.name "github-actions[bot]"
git config user.email "github-actions[bot]@users.noreply.github.com"
GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}

# After
token: ${{ secrets.GOOSE_AGENT_PAT }}
git config user.name "goose-agent"
git config user.email "{ID}+goose-agent@users.noreply.github.com"
GH_TOKEN: ${{ secrets.GOOSE_AGENT_PAT }}
```

Where `{ID}` is the GitHub user ID from Step 1.

The workflows affected in this repo:
- `.github/workflows/feature-design.yml`
- `.github/workflows/dev-session.yml`
- `.github/workflows/pr-review-session.yml`
- `.github/workflows/issue-session.yml` (when created)

---

## Step 6 — Verify

After the first workflow run, check:

```bash
git log --oneline --format="%an %s" | head -10
```

Commits authored by the pipeline should show `goose-agent` as the author.

PRs opened by the pipeline should show `goose-agent` as the author in the GitHub UI.

---

## Security Notes

- The PAT gives `goose-agent` push access to the repo — treat it like a password
- Rotate the PAT annually (set a calendar reminder)
- If the PAT is compromised, revoke it immediately in GitHub and generate a new one
- To disable all AI workflows instantly: suspend the `goose-agent` GitHub account
- The PR gate is the trust boundary — no AI change reaches `main` without human approval
- Issue assignment to `goose-agent` requires collaborator permissions — the public cannot trigger workflows

---

## Current goose-agent Details (this repo)

| Field | Value |
|---|---|
| GitHub username | `goose-agent` |
| GitHub user ID | `270002424` |
| Noreply email | `270002424+goose-agent@users.noreply.github.com` |
| Repo secret name | `GOOSE_AGENT_PAT` |
| Collaborator access | Write |
