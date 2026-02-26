---
name: confluence-navigator
description: Navigate and query self-hosted Confluence instances via REST API. Use when the user asks about Confluence content, recent changes, watched pages, space navigation, page lookups, comments, calendars, or searching their Confluence wiki. Triggers on mentions of Confluence, wiki pages, spaces, or requests to check what changed in their documentation. Supports multiple Confluence instances.
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
go run ~/.claude/scripts/confluence-navigator/main.go acme test
```

## Commands

All commands: `go run ~/.claude/scripts/confluence-navigator/main.go <host> <command> [args...]`

`<host>` is a hostname or substring matching a `~/.netrc` entry. The script auto-filters for confluence hosts.

### Checking What Changed

1. **Recent changes across the instance:**
   ```bash
   go run ~/.claude/scripts/confluence-navigator/main.go acme recent 20
   ```

2. **Changes to content you are watching (primary use case):**
   ```bash
   go run ~/.claude/scripts/confluence-navigator/main.go acme watch-changes 7
   ```

3. **List all watched content:** `go run ~/.claude/scripts/confluence-navigator/main.go acme watched`

### Searching and Looking Up Pages

4. **CQL search** (most flexible):
   ```bash
   go run ~/.claude/scripts/confluence-navigator/main.go acme search 'title ~ "deployment guide"' 10
   ```

5. **Full-text search:**
   ```bash
   go run ~/.claude/scripts/confluence-navigator/main.go acme search 'text ~ "kubernetes" AND type = page' 15
   ```

6. **Get page content by ID:** `go run ~/.claude/scripts/confluence-navigator/main.go acme page 12345 view`
   Format: `view` (rendered) or `storage` (raw XHTML).

7. **Get page metadata:** `go run ~/.claude/scripts/confluence-navigator/main.go acme page-info 12345`

### Navigating Spaces and Structure

8. **List spaces:** `go run ~/.claude/scripts/confluence-navigator/main.go acme spaces`
9. **Pages in a specific space:** `go run ~/.claude/scripts/confluence-navigator/main.go acme space-pages SPACEKEY 25`
10. **Child pages of a page:** `go run ~/.claude/scripts/confluence-navigator/main.go acme children 12345`
11. **Page version history:** `go run ~/.claude/scripts/confluence-navigator/main.go acme history 12345`
12. **Page labels:** `go run ~/.claude/scripts/confluence-navigator/main.go acme labels 12345`
13. **Tree view of space:** `go run ~/.claude/scripts/confluence-navigator/main.go acme tree SPACEKEY [root-page-id]`
    Shows hierarchical structure with ASCII tree formatting. Optionally specify a root page to show subtree only.

### Comments

14. **List comments on a page:** `go run ~/.claude/scripts/confluence-navigator/main.go acme comments 12345`
15. **Add a comment:** `go run ~/.claude/scripts/confluence-navigator/main.go acme comment-add 12345 "Your comment text"`
16. **Update a comment:** `go run ~/.claude/scripts/confluence-navigator/main.go acme comment-update 67890 "Updated comment text"`

### Watch/Unwatch

17. **Watch a page:** `go run ~/.claude/scripts/confluence-navigator/main.go acme watch 12345`
18. **Unwatch a page:** `go run ~/.claude/scripts/confluence-navigator/main.go acme unwatch 12345`

### Reading List (Save for Later)

19. **Add to reading list:** `go run ~/.claude/scripts/confluence-navigator/main.go acme read-later-add 12345`
20. **Remove from reading list:** `go run ~/.claude/scripts/confluence-navigator/main.go acme read-later-remove 12345`
21. **View reading list:** `go run ~/.claude/scripts/confluence-navigator/main.go acme read-later-list`

### Analytics

22. **Get page analytics:** `go run ~/.claude/scripts/confluence-navigator/main.go acme analytics 12345`
    Displays view counts, unique viewers, and recent activity in a formatted table.

### Calendar Management (Team Calendars Plugin)

23. **List available calendars:** `go run ~/.claude/scripts/confluence-navigator/main.go acme calendars`
24. **List events in a calendar:** `go run ~/.claude/scripts/confluence-navigator/main.go acme calendar-events CAL-123 [start-date] [end-date]`
    Date format: `YYYY-MM-DD` (defaults to current month if not specified)
25. **Get event details:** `go run ~/.claude/scripts/confluence-navigator/main.go acme calendar-event EVENT-456`
26. **Create a calendar event:**
    ```bash
    # With specific date and time
    go run ~/.claude/scripts/confluence-navigator/main.go acme calendar-event-add CAL-123 "Meeting Title" "2024-03-15 14:00" "2024-03-15 15:00" "Optional description"

    # All-day event (no time specified)
    go run ~/.claude/scripts/confluence-navigator/main.go acme calendar-event-add CAL-123 "All Day Event" "2024-03-15" "2024-03-15"
    ```
27. **Update a calendar event:**
    ```bash
    go run ~/.claude/scripts/confluence-navigator/main.go acme calendar-event-update EVENT-456 "Updated Title" "2024-03-15 15:00" "2024-03-15 16:00" "Updated description"
    ```
28. **Delete a calendar event:** `go run ~/.claude/scripts/confluence-navigator/main.go acme calendar-event-delete EVENT-456`

#### Calendar Date/Time Input Formats

The calendar commands support flexible date/time input:
- **Date only** (all-day event): `2024-03-15`
- **Date with time**: `2024-03-15 14:30` (24-hour format)
- **Unix timestamp** (milliseconds): `1710511800000`

All times are interpreted in the local system timezone.

### Utility

29. **Current user:** `go run ~/.claude/scripts/confluence-navigator/main.go acme whoami`
30. **Test connection:** `go run ~/.claude/scripts/confluence-navigator/main.go acme test`

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
4. Add important pages to your reading list with `read-later-add`
5. Review analytics for high-traffic pages with `analytics <id>`

## Plugin Availability Notes

Some features require specific Confluence plugins:
- **Calendar commands** require the Team Calendars plugin (bundled with Confluence Data Center)
- **Analytics** requires the Analytics API (available in Confluence 7.0+)
- **Reading list** may not be available in all Confluence versions

If a feature is unavailable, the script will provide a helpful error message.
