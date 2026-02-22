---
name: harbor-navigator
description: Navigate and query self-hosted Harbor container registry instances via REST API. Use when the user asks about Harbor projects, container images, repositories, artifacts, tags, vulnerability scans, replication policies, robot accounts, storage quotas, or audit logs. Triggers on mentions of Harbor, container registry, image repositories, docker images, artifact scanning, CVEs in images, replication, or requests to check what images exist or have been recently pushed. Supports multiple Harbor instances.
---

# Harbor Navigator

Query self-hosted Harbor v2.x container registry instances via REST API v2.0 using the bundled `scripts/harbor.sh` CLI wrapper. Reads credentials from `~/.netrc`. Uses unauthenticated access for read operations (works with OIDC instances where all projects are public). See [references/api_endpoints.md](references/api_endpoints.md) for full endpoint reference.

## Finding Hosts

Scan `~/.netrc` for Harbor hostnames:
```bash
scripts/harbor.sh discover
scripts/harbor.sh discover myorg   # custom substring
```

Test a connection (use hostname or substring):
```bash
scripts/harbor.sh lsre test
```

## Commands

All commands: `scripts/harbor.sh <host> <command> [args...]`

`<host>` is a hostname or substring matching a `~/.netrc` entry. The script auto-filters for harbor hosts.

### Projects and Repositories

1. **List projects:** `scripts/harbor.sh lsre projects 25`
2. **Project details:** `scripts/harbor.sh lsre project-info my-project`
3. **List repositories in a project:** `scripts/harbor.sh lsre repos my-project 25`
4. **List artifacts (images) in a repository:**
   ```bash
   scripts/harbor.sh lsre artifacts my-project/my-repo 25
   ```
5. **List tags:** `scripts/harbor.sh lsre tags my-project/my-repo`
6. **Search projects and repos:** `scripts/harbor.sh lsre search "nginx"`
7. **Recently pushed repos:**
   ```bash
   scripts/harbor.sh lsre recent-pushes my-project 20
   scripts/harbor.sh lsre recent-pushes          # all projects
   ```

### Vulnerability Scanning

8. **View vulnerability report:**
   ```bash
   scripts/harbor.sh lsre vulns my-project/my-repo latest
   ```
9. **Trigger a scan** (requires authenticated access): `scripts/harbor.sh lsre scan my-project/my-repo latest`

### Replication and Registries

10. **Replication policies:** `scripts/harbor.sh lsre replication-policies`
11. **Replication execution history:** `scripts/harbor.sh lsre replication-runs 5 10`
12. **Connected registries:** `scripts/harbor.sh lsre registries`

### Administration

13. **System info:** `scripts/harbor.sh lsre system-info`
14. **Component health:** `scripts/harbor.sh lsre health`
15. **Labels:** `scripts/harbor.sh lsre labels g` (g=global, p=project)
16. **Garbage collection:** `scripts/harbor.sh lsre gc`
17. **Storage quotas:** `scripts/harbor.sh lsre quotas`
18. **Robot accounts:** `scripts/harbor.sh lsre robot-accounts`
19. **Audit log:** `scripts/harbor.sh lsre audit-log 25`

Note: Some admin endpoints (gc, quotas, robot-accounts, audit-log) require authenticated access and may fail with unauthenticated mode.

### Utility

20. **Current user:** `scripts/harbor.sh lsre whoami`
21. **Test connection:** `scripts/harbor.sh lsre test`

## Workflow: Image Audit

1. Run `projects` to see all projects and repo counts
2. Run `repos <project>` to browse repositories
3. Run `artifacts <project/repo>` to see images with scan status
4. For interesting images, run `tags <project/repo>` to see available tags
5. Run `vulns <project/repo> <tag>` to view CVEs on specific images
6. Run `search <term>` to find specific images across all projects
7. Run `recent-pushes` to see what was updated recently
