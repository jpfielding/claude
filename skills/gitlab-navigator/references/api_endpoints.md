# GitLab REST API v4 Reference

## Authentication

All requests use `PRIVATE-TOKEN: <token>` header. GitLab PATs start with `glpat-`.

## Pagination

- `per_page` (max 100, default 20), `page` (default 1)
- Response headers: `X-Total`, `X-Total-Pages`, `X-Page`, `X-Per-Page`, `X-Next-Page`

## Core Endpoints

### Users
| Endpoint | Method | Description |
|---|---|---|
| `/user` | GET | Current authenticated user |
| `/users/:id` | GET | Single user |

### Projects
| Endpoint | Method | Key Params |
|---|---|---|
| `/projects` | GET | `membership=true`, `starred=true`, `order_by=updated_at`, `sort=desc`, `visibility`, `search` |
| `/projects/:id` | GET | `statistics=true` for repo stats |
| `/projects/:id/events` | GET | `per_page`, `sort` |
| `/projects/:id/starrers` | GET | Users who starred |

Project refs: numeric ID or URL-encoded path (`group%2Fsubgroup%2Fproject`).

### Merge Requests
| Endpoint | Method | Key Params |
|---|---|---|
| `/merge_requests` | GET | `state` (opened/closed/merged/all), `scope` (assigned_to_me/all/created_by_me), `reviewer_username`, `order_by`, `sort` |
| `/projects/:id/merge_requests` | GET | Same as above, project-scoped |
| `/projects/:id/merge_requests/:iid` | GET | Single MR details |
| `/projects/:id/merge_requests/:iid/changes` | GET | File diffs |
| `/projects/:id/merge_requests/:iid/notes` | GET | MR comments |
| `/projects/:id/merge_requests/:iid/approvals` | GET | Approval status |

### Issues
| Endpoint | Method | Key Params |
|---|---|---|
| `/issues` | GET | `state` (opened/closed/all), `scope` (assigned_to_me/all/created_by_me), `labels`, `milestone`, `search`, `order_by`, `sort` |
| `/projects/:id/issues` | GET | Project-scoped |
| `/projects/:id/issues/:iid` | GET | Single issue |
| `/projects/:id/issues/:iid/notes` | GET | Issue comments |

### Pipelines
| Endpoint | Method | Key Params |
|---|---|---|
| `/projects/:id/pipelines` | GET | `status` (running/pending/success/failed/canceled), `ref`, `order_by`, `sort` |
| `/projects/:id/pipelines/:pipeline_id` | GET | Pipeline details |
| `/projects/:id/pipelines/:pipeline_id/jobs` | GET | Jobs in pipeline |

### Repository
| Endpoint | Method | Key Params |
|---|---|---|
| `/projects/:id/repository/branches` | GET | `order_by` (name/updated), `sort`, `search` |
| `/projects/:id/repository/commits` | GET | `ref_name`, `since`, `until`, `path` |
| `/projects/:id/repository/tree` | GET | `path`, `ref`, `recursive`, `per_page` |
| `/projects/:id/repository/files/:file_path` | GET | `ref` - returns base64 content |
| `/projects/:id/repository/compare` | GET | `from`, `to` - branch/tag/SHA comparison |

### Groups
| Endpoint | Method | Key Params |
|---|---|---|
| `/groups` | GET | `min_access_level`, `order_by`, `sort`, `search` |
| `/groups/:id` | GET | Group details |
| `/groups/:id/projects` | GET | `include_subgroups=true`, `order_by`, `sort` |

### Search
| Endpoint | Method | Scopes |
|---|---|---|
| `/search` | GET | `projects`, `issues`, `merge_requests`, `milestones`, `blobs`, `commits`, `users` |
| `/projects/:id/search` | GET | `blobs`, `commits`, `issues`, `merge_requests`, `notes`, `wiki_blobs` |

Params: `search=<query>&scope=<scope>&per_page=<n>`

### Container Registry
| Endpoint | Method | Description |
|---|---|---|
| `/projects/:id/registry/repositories` | GET | List registry repos |
| `/projects/:id/registry/repositories/:repo_id/tags` | GET | List tags |

### Events (Activity)
| Endpoint | Method | Key Params |
|---|---|---|
| `/events` | GET | `action`, `target_type`, `after`, `before`, `sort` |
| `/projects/:id/events` | GET | Same params, project-scoped |

### System
| Endpoint | Method | Description |
|---|---|---|
| `/version` | GET | GitLab version info |

## Access Levels

| Value | Role |
|---|---|
| 10 | Guest |
| 20 | Reporter |
| 30 | Developer |
| 40 | Maintainer |
| 50 | Owner |

## Order By Options

- Projects: `id`, `name`, `path`, `created_at`, `updated_at`, `last_activity_at`, `similarity`
- MRs/Issues: `created_at`, `updated_at`, `priority`, `label_priority`
- Pipelines: `id`, `status`, `ref`, `updated_at`, `user_id`
- Branches: `name`, `updated`
