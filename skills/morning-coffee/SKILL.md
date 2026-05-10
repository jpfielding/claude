---
name: morning-coffee
description: >-
  Daily project standup and planning workflow. Use when the user says
  "morning coffee", "daily review", "standup", "start my day", "what's the
  status", or asks for an executive briefing across ticketing, wiki, repo,
  and the local codebase. Also triggers on "/morning-coffee".
model: sonnet
thinking: high
---

# Morning Coffee

Automated daily project review: gather status from the project's ticketing
system, wiki, repo host, and local codebase, then synthesize an executive
briefing and build a prioritized day plan.

## Defaults

This skill is tuned for **Claude Sonnet 4.6 at high thinking effort**.
Sonnet is the right model for breadth-of-search synthesis across multiple
data sources; high effort lets it reason deliberately about prioritization
without burning Opus. If the session is on a different model, suggest
`/model claude-sonnet-4-6` before proceeding rather than running on Opus
or Haiku by default.

## Stacks

Two project stacks are first-class. Either is "normal" — do not treat one as
the default and the other as a fallback.

| Concern | GitHub-native stack | Enterprise stack |
|---|---|---|
| Ticketing | GitHub Issues | Jira |
| Wiki | GitHub Wiki | Confluence |
| Repo + reviews | GitHub PRs / Actions | GitLab MRs / Pipelines |
| Tooling | `gh` CLI via Bash | `jira-navigator`, `confluence-navigator`, `gitlab-navigator` |

Mix-and-match is fine (e.g. GitLab repo + Jira tickets + Confluence wiki, or
GitHub repo + Jira tickets). Resolve each concern independently.

## Discovery

Pick the stack for each concern in this order:

1. **README `## Project Management` section** if present:
   ```markdown
   ## Project Management

   | System | Tool | Location |
   |---|---|---|
   | Ticketing | GitHub Issues / Jira / Linear / ... | <url> |
   | Wiki | GitHub Wiki / Confluence / Notion / ... | <space or url> |
   | Repo | GitHub / GitLab / ... | <url> |
   ```
2. **MEMORY.md** project entries that pin the stack.
3. **Auto-detect from `git remote -v`**:
   - `github.com/...` → default to GitHub Issues + GitHub Wiki + GitHub PRs.
   - `gitlab.*` → default to GitLab MRs/issues; ask whether wiki is GitLab
     wiki or Confluence (don't assume).
4. **Ask the user** only if the above produced nothing.

When auto-detecting, surface the inferred stack in one line ("Detected
GitHub-native stack from `git remote -v`") so the user can correct it before
data gathering starts.

## Workflow

### Phase 1 — Parallel Data Gathering

Launch up to four agents **concurrently** (single message, parallel tool
calls). Skip any system that isn't declared for this project. For each step
below, the GitHub-native commands and the enterprise-stack commands are
equally valid — pick whichever matches the resolved stack.

#### 1. Ticketing

Fetch the active sprint/iteration (or backlog, for projects that don't run
sprints). List tickets with status, assignee, priority. Flag status changes
since the last working day. Check notifications directed at the user:
@mentions, newly assigned tickets, watched/subscribed tickets with activity.

**GitHub Issues (`gh` CLI):**
- `gh issue list --repo <owner/repo> --state open --limit 50 --json number,title,state,assignees,labels,updatedAt,author`
- `gh search issues --repo <owner/repo> --mentions @me --updated '>YYYY-MM-DD'` — comments/issues mentioning you
- `gh search issues --repo <owner/repo> --assignee @me --state open` — assigned to you
- `gh search issues --repo <owner/repo> --involves @me --updated '>YYYY-MM-DD'` — anything you're subscribed to with recent activity
- For projects using GitHub Projects (v2): `gh project item-list <number> --owner <owner>` to see board state

**Jira (`jira-navigator`):**
- `GET /rest/api/2/search?jql=comment ~ currentUser() AND updated >= -1d` — mentions
- `GET /rest/api/2/search?jql=assignee changed TO currentUser() AND updated >= -1d` — newly assigned
- `GET /rest/api/2/search?jql=watcher = currentUser() AND updated >= -1d` — watched activity

#### 2. Wiki

Search for pages updated in the last 7 days relevant to current work.
Surface mentions of the user and inline comments directed at them.

**GitHub Wiki — use `gh` and browse on demand:**
The wiki is a separate git repo at `<repo-url>.wiki.git`. GitHub's REST
API doesn't expose wiki page content, so the practical path is `gh` plus
on-demand page browsing. The user does **not** need a pre-existing local
clone of the wiki — the skill clones to a scratch path each run.

- **Confirm wiki is enabled**: `gh api repos/<owner>/<repo> --jq '.has_wiki'`.
- **Recent changes** (clone-on-demand to a scratch path):
  ```
  gh repo clone <owner>/<repo>.wiki /tmp/<repo>-wiki 2>/dev/null || \
    git -C /tmp/<repo>-wiki pull --quiet
  git -C /tmp/<repo>-wiki log --since='7 days ago' --pretty='%h %ad %s' --date=short --stat
  ```
  Treat `/tmp/<repo>-wiki` as ephemeral. Do not clone into the user's
  workspace (`~/projects/...`); the scratch path keeps the wiki's git
  history available for further drill-down without polluting the project.
- **Read a specific page on demand**: `Read /tmp/<repo>-wiki/<Page-Title>.md`
  for any page surfaced by the recency scan or referenced from open
  tickets/PRs. Browse just what's needed for the briefing — don't bulk-read.
- **Read without cloning** (rare): `WebFetch` against
  `https://github.com/<owner>/<repo>/wiki/<Page-Title>`.
- Wiki pages don't notify on @-mentions, so a recency scan is sufficient
  for the "what's new" portion of the briefing. Discussion @-mentions are
  already covered by the GitHub ticketing search above.

**Confluence (`confluence-navigator`):**
- `GET /rest/api/content/search?cql=mention = currentUser() AND lastModified >= now("-7d")`
- Inline comments on pages the user authored or is watching

#### 3. Repo host

For the project's remote repo, fetch:
- Recent commits on the main branch (last 5 working days)
- Open MRs/PRs and review requests
- CI/pipeline status — passing, failing, queued
- Failed pipelines/checks in the last 24 hours
- Review requests assigned to or mentioning the user
- Comments tagging the user on open MRs/PRs

**GitHub (`gh` CLI):**
- `gh pr list --repo <owner/repo> --state open --json number,title,author,reviewRequests,updatedAt,isDraft,statusCheckRollup,headRefName`
- `gh pr list --search "review-requested:@me"` — PRs awaiting your review
- `gh run list --repo <owner/repo> --limit 10 --json status,conclusion,name,headBranch,workflowName,createdAt` — Actions runs
- `gh run list --status failure --created '>YYYY-MM-DD'` — recent failures
- If `gh run list` returns `[]`, that means no Actions are configured —
  call this out in the briefing as a CI gap, not as "no failures."

**GitLab (`gitlab-navigator`):** equivalent MR/pipeline queries via REST.

#### 4. Codebase (`Explore` agent or direct Bash)

Read the project README, run `git log --oneline -20`, check `git status`,
list local branches/worktrees. Identify recently changed packages and
current test suite status. For Go projects a quick `go test -short ./...`
is cheap and confirms the suite is healthy.

Pass project context (sprint name, board ID, ticket prefixes, known
blockers, repo path) from README and MEMORY.md into each agent prompt.

If any external system is unreachable, report what failed and continue
with available data. Never block the whole workflow on one source.

### Phase 2 — Synthesize Executive Briefing

Combine results into a scannable briefing. Tables over prose, bullets over
paragraphs.

```
## Sprint / Backlog Status
- Sprint name + dates (or "no sprints — backlog only")
- Ticket table: id | summary | status | assignee | changed since last working day?

## Code Status
- Recent commits grouped by ticket/theme
- Open MRs/PRs with review state and CI/pipeline health
- Test suite: passing/failing count
- Stale state worth cleaning (abandoned worktrees, branches well behind main)

## Documentation
- Recently updated wiki pages
- Gaps between code state and documentation

## Notifications & Mentions
- Ticketing: comments tagging you, newly assigned, watched/involved activity
- Wiki: pages mentioning you, comments on your pages
- Repo: PR/MR review requests, comments tagging you
- Flag items that need a response (questions, review requests, blockers others raised)
- For solo projects, expect this section to be empty — say so explicitly

## Risks & Blockers
- Blocked tickets and why
- Failed pipelines (or "no CI configured" if that's the case)
- Stale items (no activity > 3 days)
```

### Phase 3 — Day Plan

Propose a prioritized work list:

1. **Blockers first** — anything blocking others or awaiting external input
2. **In-progress tickets** — continue momentum
3. **Ready to start** — unblocked, can begin today
4. **Housekeeping** — doc updates, CI fixes, code review, branch/worktree cleanup

Format:
```
- [ ] <ticket-id>: <one-line description> — <effort: small/medium/large>
```

Present the briefing and day plan, then ask the user if they want to
adjust priorities or dive into a specific item.
