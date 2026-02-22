# Jira REST API Reference

Target: Jira Server / Data Center (v2 REST API)

## Table of Contents

- [Core API (v2)](#core-api-v2)
- [Agile API](#agile-api)
- [Common JQL Queries](#common-jql-queries)
- [Field Reference](#field-reference)
- [Expand Parameters](#expand-parameters)
- [Pagination](#pagination)

## Core API (v2)

Base path: `/rest/api/2`

### Issues

| Endpoint | Method | Description |
|---|---|---|
| `/search` | GET | JQL search. Params: `jql`, `maxResults`, `startAt`, `fields`, `expand` |
| `/issue/{issueIdOrKey}` | GET | Get issue. Params: `fields`, `expand` |
| `/issue/{issueIdOrKey}` | PUT | Update issue fields |
| `/issue/{issueIdOrKey}/comment` | GET | List comments. Params: `maxResults`, `startAt`, `orderBy` |
| `/issue/{issueIdOrKey}/comment` | POST | Add comment. Body: `{"body": "text"}` |
| `/issue/{issueIdOrKey}/transitions` | GET | Available transitions |
| `/issue/{issueIdOrKey}/transitions` | POST | Execute transition. Body: `{"transition": {"id": "N"}}` |
| `/issue/{issueIdOrKey}/watchers` | GET | List watchers |
| `/issue/{issueIdOrKey}/watchers` | POST | Add watcher. Body: `"username"` |
| `/issue/{issueIdOrKey}/watchers` | DELETE | Remove watcher. Param: `username` |
| `/issue/{issueIdOrKey}/changelog` | GET | Change history (DC 8.x+). Params: `maxResults`, `startAt` |
| `/issue/{issueIdOrKey}/worklog` | GET | Work logs |

### Projects

| Endpoint | Method | Description |
|---|---|---|
| `/project` | GET | List projects. Params: `maxResults`, `startAt`, `expand` |
| `/project/{projectIdOrKey}` | GET | Project details. Expand: `description`, `lead`, `issueTypes`, `components`, `versions` |
| `/project/{projectIdOrKey}/statuses` | GET | Statuses by issue type |
| `/project/{projectIdOrKey}/components` | GET | Project components |
| `/project/{projectIdOrKey}/versions` | GET | Project versions |

### Users

| Endpoint | Method | Description |
|---|---|---|
| `/myself` | GET | Current authenticated user |
| `/user` | GET | Get user by `username` or `key` param |
| `/user/search` | GET | Search users. Params: `username`, `maxResults` |

### Filters

| Endpoint | Method | Description |
|---|---|---|
| `/filter/favourite` | GET | Current user's favourite filters |
| `/filter/{id}` | GET | Get filter by ID (includes JQL) |

### Statuses

| Endpoint | Method | Description |
|---|---|---|
| `/status` | GET | List all statuses |
| `/status/{idOrName}` | GET | Get specific status |

### Other

| Endpoint | Method | Description |
|---|---|---|
| `/priority` | GET | List all priorities |
| `/issuetype` | GET | List all issue types |
| `/field` | GET | List all fields (system + custom) |
| `/resolution` | GET | List all resolutions |
| `/serverInfo` | GET | Server version and info |

## Agile API

Base path: `/rest/agile/1.0`

| Endpoint | Method | Description |
|---|---|---|
| `/board` | GET | List boards. Params: `maxResults`, `startAt`, `type`, `projectKeyOrId` |
| `/board/{boardId}` | GET | Get board details |
| `/board/{boardId}/sprint` | GET | List sprints. Params: `state` (active, closed, future), `maxResults` |
| `/board/{boardId}/issue` | GET | Board backlog issues |
| `/sprint/{sprintId}` | GET | Sprint details |
| `/sprint/{sprintId}/issue` | GET | Sprint issues. Params: `maxResults`, `fields` |
| `/epic/{epicId}/issue` | GET | Issues in an epic |

## Common JQL Queries

```
# Recently updated (global)
ORDER BY updated DESC

# My open issues
assignee = currentUser() AND resolution = Unresolved ORDER BY updated DESC

# Issues I'm watching
watcher = currentUser() ORDER BY updated DESC

# Watched issues updated recently
watcher = currentUser() AND updated >= -7d ORDER BY updated DESC

# Issues in a project
project = "PROJ" ORDER BY updated DESC

# Issues by status
project = "PROJ" AND status = "In Progress" ORDER BY priority DESC

# Issues created recently
created >= -7d ORDER BY created DESC

# Unresolved issues by priority
resolution = Unresolved AND priority in (Blocker, Critical) ORDER BY priority ASC

# Text search in summary/description
summary ~ "search term" OR description ~ "search term"

# Issues with specific labels
labels = "backend" AND resolution = Unresolved

# Issues by component
component = "API" AND resolution = Unresolved ORDER BY priority ASC

# Sprint-scoped queries
sprint in openSprints() AND assignee = currentUser()

# Overdue issues
duedate < now() AND resolution = Unresolved ORDER BY duedate ASC

# Issues updated by a specific user
updatedBy = "username" AND updated >= -7d

# Issues transitioned recently
status CHANGED DURING (-7d, now()) ORDER BY updated DESC

# Epics
issuetype = Epic AND resolution = Unresolved ORDER BY priority ASC

# Sub-tasks of an issue
parent = "PROJ-123"

# Combined filters
project = "PROJ" AND status in ("To Do", "In Progress") AND assignee = currentUser() ORDER BY priority ASC, updated DESC
```

## Field Reference

### Standard Fields (for `fields` param)

Common field names for the `fields` query parameter:

- `summary` - Issue title
- `status` - Current status
- `assignee` - Assigned user
- `reporter` - Creator
- `priority` - Priority level
- `issuetype` - Bug, Story, Task, etc.
- `project` - Project key and name
- `description` - Full description
- `created` - Creation timestamp
- `updated` - Last update timestamp
- `resolution` - Resolution (null if unresolved)
- `labels` - Array of labels
- `components` - Array of components
- `fixVersions` - Target fix versions
- `duedate` - Due date
- `timetracking` - Original/remaining estimate
- `subtasks` - Child sub-tasks
- `issuelinks` - Linked issues
- `parent` - Parent issue (for sub-tasks)
- `comment` - Comments
- `worklog` - Work logs
- `attachment` - Attachments

Request specific fields: `fields=summary,status,assignee,priority`

Use `fields=*all` for everything, or `fields=*navigable` for default set.

## Expand Parameters

For `/issue/{key}`:

- `renderedFields` - HTML-rendered field values
- `names` - Human-readable field names
- `changelog` - Full change history (embedded, older Jira)
- `operations` - Available operations
- `transitions` - Available transitions

For `/search`:

- `changelog` - Include changelog per issue
- `renderedFields` - HTML versions of fields
- `names` - Field display names

For `/project/{key}`:

- `description` - Project description
- `lead` - Project lead details
- `issueTypes` - Available issue types
- `components` - Project components
- `versions` - Project versions

## Pagination

All list/search endpoints support:
- `maxResults` - Results per page (default varies, max usually 1000 for search, 50 for agile)
- `startAt` - Offset for pagination (0-based)

Search response includes:
```json
{
  "startAt": 0,
  "maxResults": 50,
  "total": 245,
  "issues": [...]
}
```

Agile endpoints use `values` instead of `issues` and include `isLast: true/false`.
