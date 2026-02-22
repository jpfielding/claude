---
name: confluence-navigator
description: Navigate and query self-hosted Confluence instances via REST API. Use when the user asks about Confluence content, recent changes, watched pages, space navigation, page lookups, or searching their Confluence wiki. Triggers on mentions of Confluence, wiki pages, spaces, or requests to check what changed in their documentation. Supports multiple Confluence instances.
---

# Confluence Navigator

Query self-hosted Confluence instances via REST API using the bundled `scripts/confluence.sh` CLI wrapper. Credentials are read from `~/.netrc` (Bearer token auth). See [references/api_endpoints.md](references/api_endpoints.md) for full endpoint reference and CQL syntax.

## Finding Hosts

Scan `~/.netrc` for Confluence hostnames:
```bash
scripts/confluence.sh discover
scripts/confluence.sh discover myorg   # custom substring
```

Test a connection (use hostname or substring):
```bash
scripts/confluence.sh lsre test
```

## Commands

All commands: `scripts/confluence.sh <host> <command> [args...]`

`<host>` is a hostname or substring matching a `~/.netrc` entry. The script auto-filters for confluence hosts.

### Checking What Changed

1. **Recent changes across the instance:**
   ```bash
   scripts/confluence.sh lsre recent 20
   ```

2. **Changes to content you are watching (primary use case):**
   ```bash
   scripts/confluence.sh lsre watch-changes 7
   ```

3. **List all watched content:** `scripts/confluence.sh lsre watched`

### Searching and Looking Up Pages

4. **CQL search** (most flexible):
   ```bash
   scripts/confluence.sh lsre search 'title ~ "deployment guide"' 10
   ```

5. **Full-text search:**
   ```bash
   scripts/confluence.sh lsre search 'text ~ "kubernetes" AND type = page' 15
   ```

6. **Get page content by ID:** `scripts/confluence.sh lsre page 12345 view`
   Format: `view` (rendered) or `storage` (raw XHTML).

7. **Get page metadata:** `scripts/confluence.sh lsre page-info 12345`

### Navigating Spaces and Structure

8. **List spaces:** `scripts/confluence.sh lsre spaces`
9. **Pages in a specific space:** `scripts/confluence.sh lsre space-pages SPACEKEY 25`
10. **Child pages of a page:** `scripts/confluence.sh lsre children 12345`
11. **Page version history:** `scripts/confluence.sh lsre history 12345`
12. **Page labels:** `scripts/confluence.sh lsre labels 12345`

### Utility

13. **Current user:** `scripts/confluence.sh lsre whoami`
14. **Test connection:** `scripts/confluence.sh lsre test`

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
