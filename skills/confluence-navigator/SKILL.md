---
name: confluence-navigator
description: Navigate and query self-hosted Confluence instances via REST API. Use when the user asks about Confluence content, recent changes, watched pages, space navigation, page lookups, or searching their Confluence wiki. Triggers on mentions of Confluence, wiki pages, spaces, or requests to check what changed in their documentation. Supports multiple Confluence instances.
---

# Confluence Navigator

Query self-hosted Confluence instances via REST API using the bundled `go run ~/.claude/scripts/confluence-navigator/main.go` CLI wrapper. Credentials are read from `~/.netrc` (Bearer token auth). See [references/api_endpoints.md](references/api_endpoints.md) for full endpoint reference and CQL syntax.

## Finding Hosts

Scan `~/.netrc` for Confluence hostnames:
```bash
go run ~/.claude/scripts/confluence-navigator/main.go discover
go run ~/.claude/scripts/confluence-navigator/main.go discover myorg   # custom substring
```

Test a connection (use hostname or substring):
```bash
go run ~/.claude/scripts/confluence-navigator/main.go lsre test
```

## Commands

All commands: `go run ~/.claude/scripts/confluence-navigator/main.go <host> <command> [args...]`

`<host>` is a hostname or substring matching a `~/.netrc` entry. The script auto-filters for confluence hosts.

### Checking What Changed

1. **Recent changes across the instance:**
   ```bash
   go run ~/.claude/scripts/confluence-navigator/main.go lsre recent 20
   ```

2. **Changes to content you are watching (primary use case):**
   ```bash
   go run ~/.claude/scripts/confluence-navigator/main.go lsre watch-changes 7
   ```

3. **List all watched content:** `go run ~/.claude/scripts/confluence-navigator/main.go lsre watched`

### Searching and Looking Up Pages

4. **CQL search** (most flexible):
   ```bash
   go run ~/.claude/scripts/confluence-navigator/main.go lsre search 'title ~ "deployment guide"' 10
   ```

5. **Full-text search:**
   ```bash
   go run ~/.claude/scripts/confluence-navigator/main.go lsre search 'text ~ "kubernetes" AND type = page' 15
   ```

6. **Get page content by ID:** `go run ~/.claude/scripts/confluence-navigator/main.go lsre page 12345 view`
   Format: `view` (rendered) or `storage` (raw XHTML).

7. **Get page metadata:** `go run ~/.claude/scripts/confluence-navigator/main.go lsre page-info 12345`

### Navigating Spaces and Structure

8. **List spaces:** `go run ~/.claude/scripts/confluence-navigator/main.go lsre spaces`
9. **Pages in a specific space:** `go run ~/.claude/scripts/confluence-navigator/main.go lsre space-pages SPACEKEY 25`
10. **Child pages of a page:** `go run ~/.claude/scripts/confluence-navigator/main.go lsre children 12345`
11. **Page version history:** `go run ~/.claude/scripts/confluence-navigator/main.go lsre history 12345`
12. **Page labels:** `go run ~/.claude/scripts/confluence-navigator/main.go lsre labels 12345`

### Utility

13. **Current user:** `go run ~/.claude/scripts/confluence-navigator/main.go lsre whoami`
14. **Test connection:** `go run ~/.claude/scripts/confluence-navigator/main.go lsre test`

## CQL Reference

See [references/api_endpoints.md](references/api_endpoints.md) for common CQL patterns. Key examples:

- `watcher = currentUser() AND lastModified >= now("-7d")`
- `space = "KEY" AND type = page ORDER BY lastModified DESC`
- `label = "important" ORDER BY lastModified DESC`
- `text ~ "search term" AND type = page`

## Workflow: Daily Catch-Up

1. Run `watch-changes 7` to see updates to watched content
2. Run `recent 15` for broader recent activity
3. For interesting pages, use `page <id>` to read content or `history <id>` to see what changed
