# Harbor REST API Reference

Target: Harbor v2.x (API v2.0)

## Table of Contents

- [Core API](#core-api)
- [Project API](#project-api)
- [Repository & Artifact API](#repository--artifact-api)
- [Vulnerability Scanning](#vulnerability-scanning)
- [Replication](#replication)
- [System & Admin](#system--admin)
- [Pagination](#pagination)

## Core API

Base path: `/api/v2.0`

### System

| Endpoint | Method | Description |
|---|---|---|
| `/systeminfo` | GET | Harbor version, auth mode, storage provider |
| `/health` | GET | Component health checks |
| `/statistics` | GET | Totals for projects, repos, public/private |
| `/users/current` | GET | Current authenticated user |
| `/users/current/permissions` | GET | Current user's permissions |

### Search

| Endpoint | Method | Description |
|---|---|---|
| `/search` | GET | Global search. Param: `q` (query string). Returns matching projects and repositories |

## Project API

| Endpoint | Method | Description |
|---|---|---|
| `/projects` | GET | List projects. Params: `page`, `page_size`, `name`, `public`, `owner`, `sort` |
| `/projects` | POST | Create project |
| `/projects/{project_name_or_id}` | GET | Project details |
| `/projects/{project_name_or_id}` | PUT | Update project |
| `/projects/{project_name_or_id}` | DELETE | Delete project |
| `/projects/{project_name_or_id}/summary` | GET | Project summary (quota, repo count) |
| `/projects/{project_name_or_id}/members` | GET | Project members |
| `/projects/{project_name_or_id}/logs` | GET | Project audit logs |

## Repository & Artifact API

| Endpoint | Method | Description |
|---|---|---|
| `/projects/{project}/repositories` | GET | List repos. Params: `page_size`, `sort` (e.g., `-update_time`) |
| `/projects/{project}/repositories/{repo}` | GET | Repo details (repo name URL-encoded) |
| `/projects/{project}/repositories/{repo}` | DELETE | Delete repo |
| `/projects/{project}/repositories/{repo}/artifacts` | GET | List artifacts. Params: `page_size`, `with_tag`, `with_scan_overview`, `with_label`, `with_signature` |
| `/projects/{project}/repositories/{repo}/artifacts/{reference}` | GET | Artifact by tag or digest |
| `/projects/{project}/repositories/{repo}/artifacts/{reference}` | DELETE | Delete artifact |
| `/projects/{project}/repositories/{repo}/artifacts/{reference}/tags` | GET | Tags on an artifact |
| `/projects/{project}/repositories/{repo}/artifacts/{reference}/tags` | POST | Create tag |
| `/projects/{project}/repositories/{repo}/artifacts/{reference}/tags/{tag}` | DELETE | Delete tag |
| `/projects/{project}/repositories/{repo}/artifacts/{reference}/labels` | GET | Labels on artifact |
| `/projects/{project}/repositories/{repo}/artifacts/{reference}/labels` | POST | Add label to artifact |

**Note:** Repository names containing `/` must be URL-encoded (e.g., `library%2Fnginx`).

## Vulnerability Scanning

| Endpoint | Method | Description |
|---|---|---|
| `/projects/{project}/repositories/{repo}/artifacts/{reference}/scan` | POST | Trigger scan |
| `/projects/{project}/repositories/{repo}/artifacts/{reference}/scan/{report_id}/log` | GET | Scan log |
| `/projects/{project}/repositories/{repo}/artifacts/{reference}/additions/vulnerabilities` | GET | Vulnerability report |

### Vulnerability Report Structure

```json
{
  "application/vnd.security.vulnerability.report; version=1.1": {
    "generated_at": "2024-01-15T10:30:00Z",
    "severity": "High",
    "scanner": { "name": "Trivy", "vendor": "Aqua Security", "version": "0.x" },
    "summary": { "Critical": 0, "High": 2, "Medium": 5, "Low": 8, "None": 0 },
    "vulnerabilities": [
      {
        "id": "CVE-2024-xxxx",
        "package": "openssl",
        "version": "1.1.1k",
        "fix_version": "1.1.1l",
        "severity": "High",
        "description": "...",
        "links": ["https://nvd.nist.gov/..."]
      }
    ]
  }
}
```

## Replication

| Endpoint | Method | Description |
|---|---|---|
| `/replication/policies` | GET | List policies. Params: `page_size`, `name` |
| `/replication/policies` | POST | Create policy |
| `/replication/policies/{id}` | GET | Policy details |
| `/replication/executions` | GET | Execution history. Params: `policy_id`, `status`, `page_size`, `sort` |
| `/replication/executions` | POST | Trigger replication. Body: `{"policy_id": N}` |
| `/replication/executions/{id}` | GET | Execution details |
| `/replication/executions/{id}/tasks` | GET | Tasks within execution |
| `/registries` | GET | Connected registries |

## System & Admin

| Endpoint | Method | Description |
|---|---|---|
| `/labels` | GET | Labels. Params: `scope` (g=global, p=project), `project_id`, `page_size` |
| `/robots` | GET | Robot accounts. Params: `page_size`, `sort` |
| `/robots/{id}` | GET | Robot account details |
| `/system/gc/schedule` | GET | GC schedule |
| `/system/gc` | GET | GC history. Params: `page_size`, `sort` |
| `/system/gc/{id}` | GET | GC job details |
| `/system/gc/{id}/log` | GET | GC job log |
| `/quotas` | GET | Storage quotas. Params: `page_size`, `sort`, `reference` |
| `/audit-logs` | GET | Audit logs. Params: `page_size`, `sort`, `q` (query filter) |
| `/users` | GET | List users (admin only). Params: `page_size`, `username` |

### Audit Log Query Syntax

The `q` parameter on `/audit-logs` supports filter expressions:

```
operation=create,resource_type=repository
operation=delete,username=admin
resource_type=artifact,operation=create
```

## Pagination

All list endpoints support:
- `page` - Page number (1-based, default: 1)
- `page_size` - Results per page (default: 10, max varies)

Response headers include:
- `X-Total-Count` - Total number of results
- `Link` - Pagination links (next, prev)

### Sort Parameter

Many endpoints accept `sort` with field name prefixed by `-` for descending:
- `-update_time` - Most recently updated first
- `-creation_time` - Most recently created first
- `-name` - Reverse alphabetical
- `name` - Alphabetical
