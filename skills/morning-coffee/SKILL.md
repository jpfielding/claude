---
name: morning-coffee
description: >-
  Daily project standup and planning workflow. Use when the user says
  "morning coffee", "daily review", "standup", "start my day", "what's the
  status", or asks for an executive briefing across ticketing, wiki, repo,
  the local codebase, and personal communications (Slack, Gmail, Google
  Drive, Notion). Also triggers on "/morning-coffee".
model: sonnet
thinking: high
---

# Morning Coffee

Automated daily review across two families: the **project stack** (ticketing,
wiki, repo host, local codebase) and the **personal communications layer**
(Slack, Gmail, Google Drive, Notion). Gather status from both, synthesize an
executive briefing, and build a prioritized day plan.

## Defaults

This skill is tuned for **Claude Sonnet 4.6 at high thinking effort**.
Sonnet is the right model for breadth-of-search synthesis across multiple
data sources; high effort lets it reason deliberately about prioritization
without burning Opus. If the session is on a different model, suggest
`/model claude-sonnet-4-6` before proceeding rather than running on Opus
or Haiku by default.

## Stacks

### Project stack (per-project)

Two project stacks are first-class. Either is "normal" — do not treat one as
the default and the other as a fallback.

| Concern | GitHub-native stack | Enterprise stack |
|---|---|---|
| Ticketing | GitHub Issues | Jira |
| Wiki | GitHub Wiki | Confluence / Notion |
| Repo + reviews | GitHub PRs / Actions | GitLab MRs / Pipelines |
| Tooling | `gh` CLI via Bash | `jira-navigator`, `confluence-navigator`, `gitlab-navigator` |

Mix-and-match is fine (e.g. GitLab repo + Jira tickets + Confluence wiki, or
GitHub repo + Jira tickets). Resolve each concern independently.

### Personal communications layer (user-level, always swept)

Independent of the project stack and **not tied to one repo** — these are
the user's account-wide claude.ai connectors. Sweep all that are connected
on every run:

| Source | Surface | Driver |
|---|---|---|
| Slack | @mentions, DMs, threads needing a reply | `mcp__claude_ai_Slack__*` |
| Gmail | unread/important threads addressed to you | `mcp__claude_ai_Gmail__*` |
| Google Drive | recently modified docs shared with you | `mcp__claude_ai_Google_Drive__*` |
| Notion | edited pages, comments/@mentions, your tasks | `notion-expert` agent |

These connectors are **deferred MCP tools** — load schemas with `ToolSearch`
before calling. If a connector exposes only its `authenticate` /
`complete_authentication` tools, it is not connected: note it in the briefing
and offer to connect it, but **never block the whole run** on one source.

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

The **personal communications layer** (Slack, Gmail, Drive, Notion) is
discovered separately: it is user-level, not declared per-project. Sweep
every connector that is authenticated, every run, regardless of which
project stack is active. A project's README may name specific Slack channels
or a Notion teamspace to prioritize — honor those hints, but the sweep runs
even when the README is silent.

## Workflow

### Phase 1 — Parallel Data Gathering

Launch the gatherers **concurrently** (single message, parallel tool calls)
across both families below. Skip any project system that isn't declared;
sweep every personal connector that is authenticated. If a source is
unreachable or unconnected, report it and continue — never block the whole
run on one source.

**Phase 1A — Project stack (steps 1–4).** For each step below, the
GitHub-native commands and the enterprise-stack commands are equally valid —
pick whichever matches the resolved stack.

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

**Notion (`notion-expert` agent):** If the project's wiki is Notion, delegate
the recent-pages + comments scan to the `notion-expert` agent. The personal
comms sweep (Phase 1B) already covers Notion workspace-wide; for a project
whose wiki *is* Notion, pass the relevant teamspace/page hints so the agent
focuses there rather than reporting the same pages twice.

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

**Phase 1B — Personal communications sweep (steps 5–8).**
User-level, cross-project, always-on. Load each connector's tools with
`ToolSearch` before calling. The goal is **signal directed at the user**
since the last working day — not a full inbox/channel dump. Resolve "since
last working day" to a concrete date and reuse it across sources.

#### 5. Slack (`mcp__claude_ai_Slack__*`)

- `slack_search_public_and_private` with `to:me after:YYYY-MM-DD` — DMs and
  direct messages.
- Search the user's `@`-handle / mentions since the cutoff for threads that
  tag them.
- For threads that surface, `slack_read_thread` to capture the ask.
- Flag: unanswered questions, review/decision requests, threads where the
  user was the last to be addressed.

#### 6. Gmail (`mcp__claude_ai_Gmail__*`)

- `search_threads` with `is:unread newer_than:2d -in:draft` — fresh unread.
- `search_threads` with `is:important is:unread` — priority inbox.
- `search_threads` with `to:me is:unread newer_than:2d` — addressed to you.
- Report sender · subject · one-line gist · whether a reply is expected.
  Do not open drafts; do not send anything.

#### 7. Google Drive (`mcp__claude_ai_Google_Drive__*`)

- `list_recent_files` ordered by `lastModified` — docs touched recently.
- `search_files` with `sharedWithMe = true and modifiedTime > 'YYYY-MM-DDT00:00:00Z'`
  — docs others changed that are shared with you.
- Surface title · owner/last editor · why it might need attention. Comment
  @-mentions aren't exposed by these tools — note that gap rather than
  implying full coverage.

#### 8. Notion (`notion-expert` agent)

Delegate to the `notion-expert` agent's **Daily Notion catch-up** workflow.
It returns: recently edited pages, comments/@mentions directed at the user,
and the user's open tasks from task databases. If Notion is not connected,
the agent runs the OAuth handshake; surface that in the briefing rather than
silently skipping it.

If any external system is unreachable or a connector is unauthenticated,
report what failed (and offer to connect it) and continue with available
data. Never block the whole workflow on one source.

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
- Recently updated wiki pages (GitHub Wiki / Confluence / Notion)
- Recently edited Notion pages and shared Google Drive docs worth a look
- Gaps between code state and documentation

## Communications & Mentions
Everything directed at the user, aggregated across project and personal
sources. Group by "needs a response" vs. "FYI".
- Ticketing: comments tagging you, newly assigned, watched/involved activity
- Repo: PR/MR review requests, comments tagging you
- Wiki: pages mentioning you, comments on your pages
- Slack: DMs and @mentions, threads where you owe a reply
- Gmail: unread/important threads addressed to you
- Notion: comments/@mentions on pages, open tasks assigned to you
- Drive: docs shared with you that changed (note: comment @mentions not visible)
- Flag items that need a response (questions, review requests, blockers others raised)
- Note any connector that was unreachable or not yet authenticated
- For solo projects with quiet channels, expect this to be light — say so explicitly

## Risks & Blockers
- Blocked tickets and why
- Failed pipelines (or "no CI configured" if that's the case)
- Stale items (no activity > 3 days)
```

### Phase 3 — Day Plan

Propose a prioritized work list:

1. **Responses owed** — quick replies others are waiting on (Slack threads,
   emails, PR/ticket/Notion comments addressed to you). Often cheap, unblocks others.
2. **Blockers** — anything blocking others or awaiting external input
3. **In-progress tickets** — continue momentum
4. **Ready to start** — unblocked, can begin today
5. **Housekeeping** — doc updates, CI fixes, code review, branch/worktree cleanup

Format (use the ticket id when there is one; otherwise a short source tag like
`slack:` / `email:` / `notion:`):
```
- [ ] <ticket-id | source-tag>: <one-line description> — <effort: small/medium/large>
```

Present the briefing and day plan, then ask the user if they want to
adjust priorities or dive into a specific item.
