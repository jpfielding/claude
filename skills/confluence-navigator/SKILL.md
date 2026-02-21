---
name: confluence-navigator
description: Navigate and query self-hosted Confluence instances via REST API. Use when the user asks about Confluence content, recent changes, watched pages, space navigation, page lookups, or searching their Confluence wiki. Triggers on mentions of Confluence, wiki pages, spaces, or requests to check what changed in their documentation. Supports multiple Confluence instances.
---

# Confluence Navigator

Query self-hosted Confluence instances via REST API using the bundled `scripts/confluence.sh` CLI wrapper. Primary target: Confluence Data Center 10.x (v1 and v2 APIs available). The script uses v1 API for broad compatibility; see [references/api_endpoints.md](references/api_endpoints.md) for v2 endpoints when needed.

## First-Time Setup

If `~/.confluence-navigator/instances.json` does not exist, help the user configure their instance.

### Preferred: netrc auth (credentials in ~/.netrc)

The user's `~/.netrc` file stores credentials. The script reads them at runtime -- no secrets in the config.

1. Ensure `~/.netrc` has an entry for the Confluence hostname:
   ```
   machine confluence.psdo.lsre.launchpad-leidos.com
   login <username>
   password <pat_token>
   ```

2. Register the instance (validates the netrc entry exists):
   ```bash
   scripts/confluence.sh setup psdo https://confluence.psdo.lsre.launchpad-leidos.com netrc
   ```

Use `netrc-basic` instead of `netrc` if the instance requires Basic auth (login:password) rather than Bearer token auth.

### Alternative: inline credentials

For instances where netrc is not practical:
```bash
scripts/confluence.sh setup psdo https://confluence.psdo.lsre.launchpad-leidos.com pat <TOKEN>
```

### Test the connection

```bash
scripts/confluence.sh psdo test
```

## Commands

All commands follow the pattern: `scripts/confluence.sh <instance> <command> [args...]`

Use `default` as the instance name to use the configured default.

### Checking What Changed

1. **Recent changes across the instance:**
   ```bash
   scripts/confluence.sh default recent 20
   ```

2. **Changes to content you are watching (primary use case):**
   ```bash
   scripts/confluence.sh default watch-changes 7
   ```
   Argument is number of days to look back. Default: 7.

3. **List all watched content:**
   ```bash
   scripts/confluence.sh default watched
   ```

### Searching and Looking Up Pages

4. **CQL search** (most flexible):
   ```bash
   scripts/confluence.sh default search 'title ~ "deployment guide"' 10
   ```

5. **Full-text search:**
   ```bash
   scripts/confluence.sh default search 'text ~ "kubernetes" AND type = page' 15
   ```

6. **Get page content by ID:**
   ```bash
   scripts/confluence.sh default page 12345 view
   ```
   Format: `view` (rendered) or `storage` (raw XHTML).

7. **Get page metadata:**
   ```bash
   scripts/confluence.sh default page-info 12345
   ```

### Navigating Spaces and Structure

8. **List spaces:**
   ```bash
   scripts/confluence.sh default spaces
   ```

9. **Pages in a specific space:**
   ```bash
   scripts/confluence.sh default space-pages SPACEKEY 25
   ```

10. **Child pages of a page:**
    ```bash
    scripts/confluence.sh default children 12345
    ```

11. **Page version history:**
    ```bash
    scripts/confluence.sh default history 12345
    ```

12. **Page labels:**
    ```bash
    scripts/confluence.sh default labels 12345
    ```

### Utility

13. **Current user:** `scripts/confluence.sh default whoami`
14. **Test connection:** `scripts/confluence.sh default test`

## CQL Reference

For advanced searches, see [references/api_endpoints.md](references/api_endpoints.md) for common CQL patterns. Key examples:

- `watcher = currentUser() AND lastModified >= now("-7d")` - watched content, last 7 days
- `space = "KEY" AND type = page ORDER BY lastModified DESC` - space activity
- `label = "important" ORDER BY lastModified DESC` - content by label
- `text ~ "search term" AND type = page` - full-text search

## Multi-Instance Support

Add additional instances (add a ~/.netrc entry for the host first):
```bash
scripts/confluence.sh setup other-instance https://other-confluence.example.com netrc
```

Switch default:
```bash
scripts/confluence.sh other-instance set-default
```

List all:
```bash
scripts/confluence.sh list-instances
```

## Workflow: Daily Catch-Up

When the user wants to know what changed:

1. Run `watch-changes 7` to see updates to watched content
2. Run `recent 15` for broader recent activity
3. For interesting pages, use `page <id>` to read content or `history <id>` to see what changed
