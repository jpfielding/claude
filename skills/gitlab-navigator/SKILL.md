---
name: gitlab-navigator
description: Navigate and query self-hosted GitLab instances via REST API. Use when the user asks about GitLab projects, merge requests, issues, pipelines, branches, commits, starred projects, recent activity, code search, container registries, or groups. Triggers on mentions of GitLab, MRs, merge requests, CI/CD pipelines, branches, commits, starred projects, or requests to check what changed in their tracked projects. Supports multiple GitLab instances.
---

# GitLab Navigator

Query self-hosted GitLab instances via REST API v4 using the bundled `scripts/gitlab.sh` CLI wrapper. Credentials are read from `~/.netrc` (PRIVATE-TOKEN auth with `glpat-` PATs). See [references/api_endpoints.md](references/api_endpoints.md) for full endpoint reference.

## Finding Hosts

Scan `~/.netrc` for GitLab hostnames:
```bash
scripts/gitlab.sh discover
scripts/gitlab.sh discover myorg   # custom substring
```

Test a connection (use hostname or substring):
```bash
scripts/gitlab.sh lsre test
```

## Commands

All commands: `scripts/gitlab.sh <host> <command> [args...]`

`<host>` is a hostname or substring matching a `~/.netrc` entry. The script auto-filters for gitlab hosts.

### Checking What Changed

1. **Starred projects (primary watchlist):**
   ```bash
   scripts/gitlab.sh lsre starred 25
   ```

2. **Starred projects with recent activity:**
   ```bash
   scripts/gitlab.sh lsre starred-activity 7
   ```

3. **Your recent activity feed:**
   ```bash
   scripts/gitlab.sh lsre events 20
   ```

4. **Activity on a specific project:**
   ```bash
   scripts/gitlab.sh lsre project-events my-group/my-project 20
   ```

### Merge Requests

5. **MRs assigned to you:** `scripts/gitlab.sh lsre my-mrs opened 25`
6. **MRs awaiting your review:** `scripts/gitlab.sh lsre mr-review opened 25`
7. **MRs in a project:** `scripts/gitlab.sh lsre project-mrs my-group/my-project opened 25`
8. **MR details:** `scripts/gitlab.sh lsre mr my-group/my-project 42`
9. **MR changed files:** `scripts/gitlab.sh lsre mr-changes my-group/my-project 42`

### Issues

10. **Issues assigned to you:** `scripts/gitlab.sh lsre my-issues opened 25`
11. **Issues in a project:** `scripts/gitlab.sh lsre project-issues my-group/my-project opened 25`
12. **Issue details:** `scripts/gitlab.sh lsre issue my-group/my-project 10`

### Projects and Groups

13. **Your projects (by membership):** `scripts/gitlab.sh lsre projects 25`
14. **Project details + statistics:** `scripts/gitlab.sh lsre project-info my-group/my-project`
15. **Your groups:** `scripts/gitlab.sh lsre groups 25`
16. **Projects in a group:** `scripts/gitlab.sh lsre group-projects my-group 25`

### Pipelines (CI/CD)

17. **Recent pipelines:** `scripts/gitlab.sh lsre pipelines my-group/my-project 15`
18. **Pipeline details + jobs:** `scripts/gitlab.sh lsre pipeline my-group/my-project 12345`

### Code

19. **List branches:** `scripts/gitlab.sh lsre branches my-group/my-project 25`
20. **Recent commits:** `scripts/gitlab.sh lsre commits my-group/my-project main 15`
21. **Directory listing:** `scripts/gitlab.sh lsre tree my-group/my-project . main`
22. **Read file content:** `scripts/gitlab.sh lsre file my-group/my-project README.md main`

### Search

23. **Global search:**
    ```bash
    scripts/gitlab.sh lsre search "rke2" projects
    ```
    Scopes: `projects`, `issues`, `merge_requests`, `milestones`, `blobs`.

24. **Project-scoped search:**
    ```bash
    scripts/gitlab.sh lsre project-search my-group/my-project "function_name" blobs
    ```

### Container Registry

25. **Registry repositories:** `scripts/gitlab.sh lsre registries my-group/my-project`

### Utility

26. **Current user:** `scripts/gitlab.sh lsre whoami`
27. **Test connection:** `scripts/gitlab.sh lsre test`

## Workflow: Daily Catch-Up

1. Run `starred-activity 7` to see starred projects with recent activity
2. Run `my-mrs` to check merge requests assigned to you
3. Run `mr-review` to check MRs awaiting your review
4. Run `my-issues` to check your assigned issues
5. For interesting projects, use `project-events <project>` for detailed activity
6. Use `mr <project> <iid>` or `issue <project> <iid>` for full details
