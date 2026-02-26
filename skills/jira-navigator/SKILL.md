---
name: jira-navigator
description: Navigate and query self-hosted Jira instances via REST API. Use when the user asks about Jira issues, tickets, sprints, boards, projects, recent activity, watched issues, or searching their Jira instance. Triggers on mentions of Jira, tickets, issues, sprints, boards, backlogs, epics, or requests to check what changed in their tracked work. Supports multiple Jira instances (Server/Data Center).
---

# Jira Navigator

Query self-hosted Jira Server/Data Center instances via REST API v2 and Agile REST API using the bundled `go run ~/.claude/scripts/jira-navigator/main.go` CLI wrapper. Credentials are read from `~/.netrc` (Bearer token auth). See [references/api_endpoints.md](references/api_endpoints.md) for full endpoint reference and JQL syntax.

## Finding Hosts

Scan `~/.netrc` for Jira hostnames:
```bash
go run ~/.claude/scripts/jira-navigator/main.go discover
go run ~/.claude/scripts/jira-navigator/main.go discover myorg   # custom substring
```

Test a connection (use hostname or substring):
```bash
go run ~/.claude/scripts/jira-navigator/main.go acme test
```

## Commands

All commands: `go run ~/.claude/scripts/jira-navigator/main.go <host> <command> [args...]`

`<host>` is a hostname or substring matching a `~/.netrc` entry. The script auto-filters for jira hosts.

### Checking What Changed

1. **Recently updated issues across the instance:**
   ```bash
   go run ~/.claude/scripts/jira-navigator/main.go acme recent 20
   ```

2. **Changes to issues you are watching (primary use case):**
   ```bash
   go run ~/.claude/scripts/jira-navigator/main.go acme watch-changes 7
   ```

3. **Unresolved watched issues:** `go run ~/.claude/scripts/jira-navigator/main.go acme watched 25`
4. **Your open issues:** `go run ~/.claude/scripts/jira-navigator/main.go acme my-issues 25`

### Searching and Looking Up Issues

5. **JQL search** (most flexible):
   ```bash
   go run ~/.claude/scripts/jira-navigator/main.go acme search 'project = "PROJ" AND status = "In Progress"' 10
   ```

6. **Full issue details:** `go run ~/.claude/scripts/jira-navigator/main.go acme issue PROJ-123`
7. **Compact issue metadata (JSON):** `go run ~/.claude/scripts/jira-navigator/main.go acme issue-info PROJ-123`
8. **Issue comments:** `go run ~/.claude/scripts/jira-navigator/main.go acme comments PROJ-123`
9. **Issue changelog:** `go run ~/.claude/scripts/jira-navigator/main.go acme changelog PROJ-123 10`
10. **Available status transitions:** `go run ~/.claude/scripts/jira-navigator/main.go acme transitions PROJ-123`

### Projects and Structure

11. **List projects:** `go run ~/.claude/scripts/jira-navigator/main.go acme projects`
12. **Project details:** `go run ~/.claude/scripts/jira-navigator/main.go acme project-info PROJ`
13. **Statuses for a project:** `go run ~/.claude/scripts/jira-navigator/main.go acme statuses PROJ`
14. **Favourite/saved filters:** `go run ~/.claude/scripts/jira-navigator/main.go acme filters`

### Agile (Boards & Sprints)

15. **List boards:** `go run ~/.claude/scripts/jira-navigator/main.go acme boards`
16. **Sprints on a board:** `go run ~/.claude/scripts/jira-navigator/main.go acme sprints 42 active`
    State: `active`, `closed`, or `future`.
17. **Issues in a sprint:** `go run ~/.claude/scripts/jira-navigator/main.go acme sprint-issues 100`

### Utility

18. **Current user:** `go run ~/.claude/scripts/jira-navigator/main.go acme whoami`
19. **Test connection:** `go run ~/.claude/scripts/jira-navigator/main.go acme test`

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
