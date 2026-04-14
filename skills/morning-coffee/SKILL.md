---
name: morning-coffee
description: >-
  Daily project standup and planning workflow. Use when the user says
  "morning coffee", "daily review", "standup", "start my day", "what's the
  status", or asks for an executive briefing across ticketing, wiki, repo,
  and the local codebase. Also triggers on "/morning-coffee".
---

# Morning Coffee

Automated daily project review: gather status from the project's ticketing
system, wiki, repo host, and local codebase, then synthesize an executive
briefing and build a prioritized day plan.

## Discovery

Before launching agents, read the project's README.md and look for a
**Project Management** section that declares which tools this project uses.
Expected format:

```markdown
## Project Management

| System | Tool | Location |
|---|---|---|
| Ticketing | Jira / GitHub Issues / Linear / ... | <url or identifier> |
| Wiki | Confluence / Notion / GitHub Wiki / ... | <space or url> |
| Repo | GitLab / GitHub / ... | <url or identifier> |
```

Map each declared tool to the appropriate agent:

| Tool | Agent |
|---|---|
| Jira | `jira-navigator` |
| GitHub Issues | `gh` CLI via Bash |
| Confluence | `confluence-navigator` |
| GitHub Wiki | `gh` CLI or WebFetch |
| GitLab | `gitlab-navigator` |
| GitHub | `gh` CLI via Bash |

If the README has no Project Management section, check MEMORY.md for
context. If neither source identifies the stack, ask the user before
proceeding.

## Workflow

### Phase 1 — Parallel Data Gathering

Launch up to four agents **concurrently** (single message, parallel Agent
tool calls). Skip any system that isn't declared for this project.

1. **Ticketing**: Fetch the active sprint/iteration. List all tickets with
   status, assignee, and priority. Flag tickets that changed status since
   the last working day. **Also check for notifications directed at the
   user** — comments mentioning/tagging them, tickets newly assigned to
   them, and tickets where they are a watcher that had activity since the
   last working day. Use the Jira REST API:
   - `GET /rest/api/2/search?jql=comment ~ currentUser() AND updated >= -1d`
     (comments mentioning the user)
   - `GET /rest/api/2/search?jql=assignee changed TO currentUser() AND updated >= -1d`
     (newly assigned)
   - `GET /rest/api/2/search?jql=watcher = currentUser() AND updated >= -1d`
     (watched tickets with activity)

2. **Wiki**: Search for project-related pages updated in the last 7 days.
   Surface new or modified documentation relevant to current sprint tickets.
   **Also check for pages where the user was mentioned or comments directed
   at them** — use the Confluence REST API:
   - `GET /rest/api/content/search?cql=mention = currentUser() AND lastModified >= now("-7d")`
   - Check inline comments on pages the user authored or is watching

3. **Repo host**: For the project's remote repo:
   - Recent commits on main (last 5 working days)
   - Open merge/pull requests and CI pipeline status
   - Failed pipelines/checks in the last 24 hours
   - **MR/PR review requests** assigned to or mentioning the user
   - **Comments tagging the user** on open MRs or issues

4. **Codebase** (`Explore` agent): Read the project README, run
   `git log --oneline -20`, check `git status`. Identify recently changed
   packages/components and current test suite status.

Pass project context (sprint name, board ID, ticket prefixes, known
blockers, repo path) from README and MEMORY.md into each agent prompt.

If any external system is unreachable, report what failed and continue
with available data. Never block the whole workflow on one source.

### Phase 2 — Synthesize Executive Briefing

Combine agent results into a scannable briefing. Tables over prose, bullets
over paragraphs.

```
## Sprint Status
- Sprint name, dates, days remaining
- Ticket table: ticket | summary | status | assignee | changed since yesterday?

## Code Status
- Recent commits grouped by ticket/theme
- Open MRs/PRs and pipeline health
- Test suite: passing/failing count

## Documentation
- Recently updated wiki pages
- Gaps between code state and documentation

## Notifications & Mentions
- Jira: comments tagging you, newly assigned tickets, watched ticket activity
- Confluence: pages mentioning you, comments on your pages
- GitLab/GitHub: MR review requests, comments tagging you
- Flag items that need a response (questions, review requests, blockers others raised)

## Risks & Blockers
- Blocked tickets and why
- Failed pipelines
- Stale items (no activity > 3 days)
```

### Phase 3 — Day Plan

Propose a prioritized work list:

1. **Blockers first** — anything blocking others or awaiting external input
2. **In-progress tickets** — continue momentum
3. **Ready to start** — unblocked, can begin today
4. **Housekeeping** — doc updates, CI fixes, code review

Format:
```
- [ ] <ticket-id>: <one-line description> — <effort: small/medium/large>
```

Present the briefing and day plan, then ask the user if they want to
adjust priorities or dive into a specific item.
