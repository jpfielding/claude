---
name: gitlab-navigator
description: Navigate and query self-hosted GitLab instances via REST API. Use when the user asks about GitLab projects, merge requests, issues, pipelines, branches, commits, starred projects, recent activity, code search, container registries, or groups. Triggers on mentions of GitLab, MRs, merge requests, CI/CD pipelines, branches, commits, starred projects, or requests to check what changed in their tracked projects. Supports multiple GitLab instances.
---

# GitLab Navigator

Query self-hosted GitLab instances via REST API v4 using the bundled `go run ~/.claude/scripts/gitlab-navigator/main.go` CLI wrapper. Credentials are read from `~/.netrc` (PRIVATE-TOKEN auth with `glpat-` PATs). See [references/api_endpoints.md](references/api_endpoints.md) for full endpoint reference.

## Finding Hosts

Scan `~/.netrc` for GitLab hostnames:
```bash
go run ~/.claude/scripts/gitlab-navigator/main.go discover
go run ~/.claude/scripts/gitlab-navigator/main.go discover myorg   # custom substring
```

Test a connection (use hostname or substring):
```bash
go run ~/.claude/scripts/gitlab-navigator/main.go lsre test
```

## Commands

All commands: `go run ~/.claude/scripts/gitlab-navigator/main.go <host> <command> [args...]`

`<host>` is a hostname or substring matching a `~/.netrc` entry. The script auto-filters for gitlab hosts.

### Checking What Changed

1. **Starred projects (primary watchlist):**
   ```bash
   go run ~/.claude/scripts/gitlab-navigator/main.go lsre starred 25
   ```

2. **Starred projects with recent activity:**
   ```bash
   go run ~/.claude/scripts/gitlab-navigator/main.go lsre starred-activity 7
   ```

3. **Your recent activity feed:**
   ```bash
   go run ~/.claude/scripts/gitlab-navigator/main.go lsre events 20
   ```

4. **Activity on a specific project:**
   ```bash
   go run ~/.claude/scripts/gitlab-navigator/main.go lsre project-events my-group/my-project 20
   ```

### Merge Requests

5. **MRs assigned to you:** `go run ~/.claude/scripts/gitlab-navigator/main.go lsre my-mrs opened 25`
6. **MRs awaiting your review:** `go run ~/.claude/scripts/gitlab-navigator/main.go lsre mr-review opened 25`
7. **MRs in a project:** `go run ~/.claude/scripts/gitlab-navigator/main.go lsre project-mrs my-group/my-project opened 25`
8. **MR details:** `go run ~/.claude/scripts/gitlab-navigator/main.go lsre mr my-group/my-project 42`
9. **MR changed files:** `go run ~/.claude/scripts/gitlab-navigator/main.go lsre mr-changes my-group/my-project 42`

### Issues

10. **Issues assigned to you:** `go run ~/.claude/scripts/gitlab-navigator/main.go lsre my-issues opened 25`
11. **Issues in a project:** `go run ~/.claude/scripts/gitlab-navigator/main.go lsre project-issues my-group/my-project opened 25`
12. **Issue details:** `go run ~/.claude/scripts/gitlab-navigator/main.go lsre issue my-group/my-project 10`

### Projects and Groups

13. **Your projects (by membership):** `go run ~/.claude/scripts/gitlab-navigator/main.go lsre projects 25`
14. **Project details + statistics:** `go run ~/.claude/scripts/gitlab-navigator/main.go lsre project-info my-group/my-project`
15. **Your groups:** `go run ~/.claude/scripts/gitlab-navigator/main.go lsre groups 25`
16. **Projects in a group:** `go run ~/.claude/scripts/gitlab-navigator/main.go lsre group-projects my-group 25`

### Pipelines (CI/CD)

17. **Recent pipelines:** `go run ~/.claude/scripts/gitlab-navigator/main.go lsre pipelines my-group/my-project 15`
18. **Pipeline details + jobs:** `go run ~/.claude/scripts/gitlab-navigator/main.go lsre pipeline my-group/my-project 12345`

### Code

19. **List branches:** `go run ~/.claude/scripts/gitlab-navigator/main.go lsre branches my-group/my-project 25`
20. **Recent commits:** `go run ~/.claude/scripts/gitlab-navigator/main.go lsre commits my-group/my-project main 15`
21. **Directory listing:** `go run ~/.claude/scripts/gitlab-navigator/main.go lsre tree my-group/my-project . main`
22. **Read file content:** `go run ~/.claude/scripts/gitlab-navigator/main.go lsre file my-group/my-project README.md main`

### Search

23. **Global search:**
    ```bash
    go run ~/.claude/scripts/gitlab-navigator/main.go lsre search "rke2" projects
    ```
    Scopes: `projects`, `issues`, `merge_requests`, `milestones`, `blobs`.

24. **Project-scoped search:**
    ```bash
    go run ~/.claude/scripts/gitlab-navigator/main.go lsre project-search my-group/my-project "function_name" blobs
    ```

### Container Registry

25. **Registry repositories:** `go run ~/.claude/scripts/gitlab-navigator/main.go lsre registries my-group/my-project`

### Utility

26. **Current user:** `go run ~/.claude/scripts/gitlab-navigator/main.go lsre whoami`
27. **Test connection:** `go run ~/.claude/scripts/gitlab-navigator/main.go lsre test`

## Workflow: Daily Catch-Up

1. Run `starred-activity 7` to see starred projects with recent activity
2. Run `my-mrs` to check merge requests assigned to you
3. Run `mr-review` to check MRs awaiting your review
4. Run `my-issues` to check your assigned issues
5. For interesting projects, use `project-events <project>` for detailed activity
6. Use `mr <project> <iid>` or `issue <project> <iid>` for full details
