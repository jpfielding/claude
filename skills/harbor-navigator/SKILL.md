---
name: harbor-navigator
description: Navigate and query self-hosted Harbor container registry instances via REST API. Use when the user asks about Harbor projects, container images, repositories, artifacts, tags, vulnerability scans, replication policies, robot accounts, storage quotas, or audit logs. Triggers on mentions of Harbor, container registry, image repositories, docker images, artifact scanning, CVEs in images, replication, or requests to check what images exist or have been recently pushed. Supports multiple Harbor instances.
---

# Harbor Navigator

Query self-hosted Harbor v2.x container registry instances via REST API v2.0 using the bundled `go run ~/.claude/scripts/harbor-navigator/main.go` CLI wrapper. Reads credentials from `~/.netrc`. Uses unauthenticated access for read operations (works with OIDC instances where all projects are public). See [references/api_endpoints.md](references/api_endpoints.md) for full endpoint reference.

## Finding Hosts

Scan `~/.netrc` for Harbor hostnames:
```bash
go run ~/.claude/scripts/harbor-navigator/main.go discover
go run ~/.claude/scripts/harbor-navigator/main.go discover myorg   # custom substring
```

Test a connection (use hostname or substring):
```bash
go run ~/.claude/scripts/harbor-navigator/main.go lsre test
```

## Commands

All commands: `go run ~/.claude/scripts/harbor-navigator/main.go <host> <command> [args...]`

`<host>` is a hostname or substring matching a `~/.netrc` entry. The script auto-filters for harbor hosts.

### Projects and Repositories

1. **List projects:** `go run ~/.claude/scripts/harbor-navigator/main.go lsre projects 25`
2. **Project details:** `go run ~/.claude/scripts/harbor-navigator/main.go lsre project-info my-project`
3. **List repositories in a project:** `go run ~/.claude/scripts/harbor-navigator/main.go lsre repos my-project 25`
4. **List artifacts (images) in a repository:**
   ```bash
   go run ~/.claude/scripts/harbor-navigator/main.go lsre artifacts my-project/my-repo 25
   ```
5. **List tags:** `go run ~/.claude/scripts/harbor-navigator/main.go lsre tags my-project/my-repo`
6. **Search projects and repos:** `go run ~/.claude/scripts/harbor-navigator/main.go lsre search "nginx"`
7. **Recently pushed repos:**
   ```bash
   go run ~/.claude/scripts/harbor-navigator/main.go lsre recent-pushes my-project 20
   go run ~/.claude/scripts/harbor-navigator/main.go lsre recent-pushes          # all projects
   ```

### Vulnerability Scanning

8. **View vulnerability report:**
   ```bash
   go run ~/.claude/scripts/harbor-navigator/main.go lsre vulns my-project/my-repo latest
   ```
9. **Trigger a scan** (requires authenticated access): `go run ~/.claude/scripts/harbor-navigator/main.go lsre scan my-project/my-repo latest`

### Replication and Registries

10. **Replication policies:** `go run ~/.claude/scripts/harbor-navigator/main.go lsre replication-policies`
11. **Replication execution history:** `go run ~/.claude/scripts/harbor-navigator/main.go lsre replication-runs 5 10`
12. **Connected registries:** `go run ~/.claude/scripts/harbor-navigator/main.go lsre registries`

### Administration

13. **System info:** `go run ~/.claude/scripts/harbor-navigator/main.go lsre system-info`
14. **Component health:** `go run ~/.claude/scripts/harbor-navigator/main.go lsre health`
15. **Labels:** `go run ~/.claude/scripts/harbor-navigator/main.go lsre labels g` (g=global, p=project)
16. **Garbage collection:** `go run ~/.claude/scripts/harbor-navigator/main.go lsre gc`
17. **Storage quotas:** `go run ~/.claude/scripts/harbor-navigator/main.go lsre quotas`
18. **Robot accounts:** `go run ~/.claude/scripts/harbor-navigator/main.go lsre robot-accounts`
19. **Audit log:** `go run ~/.claude/scripts/harbor-navigator/main.go lsre audit-log 25`

Note: Some admin endpoints (gc, quotas, robot-accounts, audit-log) require authenticated access and may fail with unauthenticated mode.

### Utility

20. **Current user:** `go run ~/.claude/scripts/harbor-navigator/main.go lsre whoami`
21. **Test connection:** `go run ~/.claude/scripts/harbor-navigator/main.go lsre test`

## Workflow: Image Audit

1. Run `projects` to see all projects and repo counts
2. Run `repos <project>` to browse repositories
3. Run `artifacts <project/repo>` to see images with scan status
4. For interesting images, run `tags <project/repo>` to see available tags
5. Run `vulns <project/repo> <tag>` to view CVEs on specific images
6. Run `search <term>` to find specific images across all projects
7. Run `recent-pushes` to see what was updated recently
