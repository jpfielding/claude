---
name: jira-navigator
description: Navigate and query self-hosted Jira instances via REST API. Use when the user asks about Jira issues, tickets, sprints, boards, projects, recent activity, watched issues, or searching their Jira instance. Triggers on mentions of Jira, tickets, issues, sprints, boards, backlogs, epics, or requests to check what changed in their tracked work. Supports multiple Jira instances (Server/Data Center).
---

# Jira Navigator

Query self-hosted Jira Server/Data Center instances via REST API v2 and Agile REST API using the bundled `scripts/jira.sh` CLI wrapper. Credentials are read from `~/.netrc` (Bearer token auth). See [references/api_endpoints.md](references/api_endpoints.md) for full endpoint reference and JQL syntax.

## Finding Hosts

Scan `~/.netrc` for Jira hostnames:
```bash
scripts/jira.sh discover
scripts/jira.sh discover myorg   # custom substring
```

Test a connection (use hostname or substring):
```bash
scripts/jira.sh lsre test
```

## Commands

All commands: `scripts/jira.sh <host> <command> [args...]`

`<host>` is a hostname or substring matching a `~/.netrc` entry. The script auto-filters for jira hosts.

### Checking What Changed

1. **Recently updated issues across the instance:**
   ```bash
   scripts/jira.sh lsre recent 20
   ```

2. **Changes to issues you are watching (primary use case):**
   ```bash
   scripts/jira.sh lsre watch-changes 7
   ```

3. **Unresolved watched issues:** `scripts/jira.sh lsre watched 25`
4. **Your open issues:** `scripts/jira.sh lsre my-issues 25`

### Searching and Looking Up Issues

5. **JQL search** (most flexible):
   ```bash
   scripts/jira.sh lsre search 'project = "PROJ" AND status = "In Progress"' 10
   ```

6. **Full issue details:** `scripts/jira.sh lsre issue PROJ-123`
7. **Compact issue metadata (JSON):** `scripts/jira.sh lsre issue-info PROJ-123`
8. **Issue comments:** `scripts/jira.sh lsre comments PROJ-123`
9. **Issue changelog:** `scripts/jira.sh lsre changelog PROJ-123 10`
10. **Available status transitions:** `scripts/jira.sh lsre transitions PROJ-123`

### Projects and Structure

11. **List projects:** `scripts/jira.sh lsre projects`
12. **Project details:** `scripts/jira.sh lsre project-info PROJ`
13. **Statuses for a project:** `scripts/jira.sh lsre statuses PROJ`
14. **Favourite/saved filters:** `scripts/jira.sh lsre filters`

### Agile (Boards & Sprints)

15. **List boards:** `scripts/jira.sh lsre boards`
16. **Sprints on a board:** `scripts/jira.sh lsre sprints 42 active`
    State: `active`, `closed`, or `future`.
17. **Issues in a sprint:** `scripts/jira.sh lsre sprint-issues 100`

### Utility

18. **Current user:** `scripts/jira.sh lsre whoami`
19. **Test connection:** `scripts/jira.sh lsre test`

## JQL Reference

See [references/api_endpoints.md](references/api_endpoints.md) for full JQL syntax. Key examples:

- `watcher = currentUser() AND updated >= -7d ORDER BY updated DESC`
- `assignee = currentUser() AND resolution = Unresolved ORDER BY updated DESC`
- `project = "KEY" AND status = "In Progress" ORDER BY priority ASC`
- `summary ~ "search term" OR description ~ "search term"`
- `sprint in openSprints() AND assignee = currentUser()`

## Workflow: Daily Catch-Up

1. Run `watch-changes 7` to see updates to watched issues
2. Run `my-issues` to check your assigned work
3. Run `recent 15` for broader recent activity
4. For interesting issues, use `issue <key>` for full details or `changelog <key>` to see what changed
5. Use `comments <key>` to read discussion on specific issues
