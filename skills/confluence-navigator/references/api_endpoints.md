# Confluence REST API Reference

Target: Confluence Data Center 10.x (confirmed 10.2.1)

Both v1 (`/rest/api`) and v2 (`/api/v2`) APIs are available. The script uses v1 for broad compatibility. v2 endpoints are noted below where relevant.

## v1 API

Base path: `/rest/api`

## Content

| Endpoint | Method | Description |
|---|---|---|
| `/content` | GET | List/search content. Params: `type`, `spaceKey`, `title`, `orderby`, `expand`, `limit`, `start` |
| `/content/{id}` | GET | Get content by ID. Expand: `body.view`, `body.storage`, `space`, `version`, `ancestors`, `children.page`, `metadata.labels`, `history` |
| `/content/{id}/child/page` | GET | List child pages |
| `/content/{id}/version` | GET | List page versions (history) |
| `/content/{id}/label` | GET | List labels on content |
| `/content/search` | GET | CQL search. Params: `cql`, `limit`, `expand` |

## Spaces

| Endpoint | Method | Description |
|---|---|---|
| `/space` | GET | List spaces. Params: `type`, `limit`, `expand` |
| `/space/{key}` | GET | Get space by key |
| `/space/{key}/content/page` | GET | Pages in a space. Params: `limit`, `orderby`, `expand` |

## User

| Endpoint | Method | Description |
|---|---|---|
| `/user/current` | GET | Current authenticated user |
| `/user/watch` | GET | Content watched by current user |

## Common CQL Queries

```
# Recent changes across all content
type = page ORDER BY lastModified DESC

# Changes in watched content
watcher = currentUser() ORDER BY lastModified DESC

# Watched content changed in last N days
watcher = currentUser() AND lastModified >= now("-7d") ORDER BY lastModified DESC

# Search by title
title ~ "search term"

# Search in a specific space
space = "SPACEKEY" AND type = page ORDER BY lastModified DESC

# Full-text search
text ~ "search term" AND type = page

# Content by label
label = "my-label" ORDER BY lastModified DESC

# Content by specific author
contributor = "username" AND lastModified >= now("-30d")

# Combined filters
space = "DEV" AND label = "architecture" AND type = page ORDER BY title ASC
```

## Expand Parameters

The `expand` parameter controls what nested data is returned. Commonly used:

- `body.view` - Rendered HTML content
- `body.storage` - Raw storage format (XHTML)
- `space` - Space info (key, name)
- `version` - Version info (number, by, when, message)
- `ancestors` - Parent page chain
- `children.page` - Child pages
- `history` - Creation date and creator
- `history.lastUpdated` - Last modification info
- `metadata.labels` - Labels/tags on the page

Multiple expands: `expand=space,version,body.view`

## Pagination

All list endpoints support:
- `limit` - Results per page (default varies, max usually 200)
- `start` - Offset for pagination

Response includes `_links.next` when more results exist.

## Common orderby Values

- `history.lastUpdated desc` - Most recently modified first
- `title asc` - Alphabetical by title

## v2 API

Base path: `/api/v2` (available in Confluence Data Center 8.0+)

The v2 API uses cursor-based pagination and a flatter response structure. Useful when v1 endpoints are insufficient.

### Key v2 Endpoints

| Endpoint | Method | Description |
|---|---|---|
| `/api/v2/pages` | GET | List pages. Params: `space-id`, `title`, `sort`, `body-format`, `limit`, `cursor` |
| `/api/v2/pages/{id}` | GET | Get page by ID. Params: `body-format` (storage, atlas_doc_format, view) |
| `/api/v2/pages/{id}/children` | GET | Direct child pages |
| `/api/v2/pages/{id}/labels` | GET | Labels on a page |
| `/api/v2/spaces` | GET | List spaces. Params: `keys`, `type`, `sort`, `limit` |
| `/api/v2/spaces/{id}` | GET | Get space by ID |
| `/api/v2/labels/{id}/pages` | GET | Pages with a specific label |

### v2 Differences from v1

- Pagination uses `cursor` instead of `start` offset
- No `expand` parameter; use `body-format` for content
- Spaces identified by numeric ID instead of string key
- Responses are flatter (no nested `results` array for single items)
- Use `sort` instead of `orderby` (e.g., `sort=-modified-date`)

### Direct curl example (v2)

```bash
curl -H "Authorization: Bearer $TOKEN" \
  "https://confluence.example.com/api/v2/pages?space-id=12345&sort=-modified-date&limit=10"
```

If the v1 script commands are insufficient for a query, construct v2 API calls directly via curl.
