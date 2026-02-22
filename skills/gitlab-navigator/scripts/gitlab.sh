#!/usr/bin/env bash
#
# gitlab.sh - GitLab REST API CLI wrapper
#
# Usage: gitlab.sh <hostname-or-substring> <command> [args...]
#
# Credentials from ~/.netrc, PRIVATE-TOKEN auth

set -euo pipefail

# --- helpers ---

die() { echo "ERROR: $*" >&2; exit 1; }

require_jq() {
  command -v jq >/dev/null 2>&1 || die "jq is required but not installed"
}

parse_netrc() {
  local machine="$1"
  local netrc_file="${HOME}/.netrc"
  [[ -f "$netrc_file" ]] || die "~/.netrc not found. Create it with:\n  machine $machine\n  login <username>\n  password <glpat-token>"

  NETRC_LOGIN=""
  NETRC_PASSWORD=""

  local tokens
  tokens=$(tr '\n' ' ' < "$netrc_file")
  local in_machine=false
  set -- $tokens
  while [[ $# -gt 0 ]]; do
    case "$1" in
      machine)
        shift
        if [[ "${1:-}" == "$machine" ]]; then
          in_machine=true
        else
          in_machine=false
        fi
        ;;
      login)
        shift
        $in_machine && NETRC_LOGIN="${1:-}"
        ;;
      password)
        shift
        $in_machine && NETRC_PASSWORD="${1:-}"
        ;;
    esac
    shift
  done

  [[ -n "$NETRC_PASSWORD" ]] || die "No entry for machine '$machine' in ~/.netrc"
}

resolve_host() {
  local input="$1"
  if [[ "$input" == *"."* ]]; then
    HOSTNAME="$input"
    BASE_URL="https://${HOSTNAME}"
    parse_netrc "$HOSTNAME"
    return
  fi

  # Substring match against machine entries in ~/.netrc
  local netrc_file="${HOME}/.netrc"
  [[ -f "$netrc_file" ]] || die "~/.netrc not found"

  local matches=()
  local tokens
  tokens=$(tr '\n' ' ' < "$netrc_file")
  set -- $tokens
  while [[ $# -gt 0 ]]; do
    case "$1" in
      machine)
        shift
        if [[ "${1:-}" == *"$input"* && "${1:-}" == *"gitlab"* ]]; then
          matches+=("${1:-}")
        fi
        ;;
    esac
    shift
  done

  if [[ ${#matches[@]} -eq 0 ]]; then
    die "No ~/.netrc machine entry matching '$input'"
  elif [[ ${#matches[@]} -gt 1 ]]; then
    echo "Multiple matches for '$input':" >&2
    for m in "${matches[@]}"; do
      echo "  $m" >&2
    done
    die "Ambiguous host substring '$input'. Use a more specific name or full hostname."
  fi

  HOSTNAME="${matches[0]}"
  BASE_URL="https://${HOSTNAME}"
  parse_netrc "$HOSTNAME"
}

# URL-encode a string
urlencode() {
  python3 -c "import urllib.parse,sys; print(urllib.parse.quote(sys.argv[1], safe=''))" "$1"
}

api_get() {
  local endpoint="$1"
  shift
  local url="${BASE_URL}/api/v4${endpoint}"

  if [[ $# -gt 0 && -n "${1:-}" ]]; then
    local params="$1"
    url="${url}?${params}"
  fi

  local response
  response=$(curl -sS -w "\n%{http_code}" \
    -H "PRIVATE-TOKEN: $NETRC_PASSWORD" \
    -H "Accept: application/json" \
    "$url" 2>&1) || die "curl failed: $response"

  local http_code body
  http_code=$(echo "$response" | tail -1)
  body=$(echo "$response" | sed '$d')

  if [[ "$http_code" -ge 400 ]]; then
    die "API returned HTTP $http_code: $body"
  fi

  echo "$body"
}

api_get_paged() {
  # Fetch with pagination awareness - returns body and total from headers
  local endpoint="$1"
  shift
  local url="${BASE_URL}/api/v4${endpoint}"

  if [[ $# -gt 0 && -n "${1:-}" ]]; then
    local params="$1"
    url="${url}?${params}"
  fi

  local header_file
  header_file=$(mktemp)

  local response
  response=$(curl -sS -w "\n%{http_code}" \
    -D "$header_file" \
    -H "PRIVATE-TOKEN: $NETRC_PASSWORD" \
    -H "Accept: application/json" \
    "$url" 2>&1) || { rm -f "$header_file"; die "curl failed: $response"; }

  local http_code body total
  http_code=$(echo "$response" | tail -1)
  body=$(echo "$response" | sed '$d')
  total=$(grep -i '^x-total:' "$header_file" | tr -d '\r' | awk '{print $2}' || echo "")
  rm -f "$header_file"

  if [[ "$http_code" -ge 400 ]]; then
    die "API returned HTTP $http_code: $body"
  fi

  if [[ -n "$total" ]]; then
    echo "TOTAL:${total}"
  fi
  echo "$body"
}

# --- commands ---

cmd_discover() {
  # Scan ~/.netrc for machine entries that look like GitLab hosts
  local filter="${1:-gitlab}"
  local netrc_file="${HOME}/.netrc"
  [[ -f "$netrc_file" ]] || die "~/.netrc not found"

  local machines=()
  local tokens
  tokens=$(tr '\n' ' ' < "$netrc_file")
  set -- $tokens
  while [[ $# -gt 0 ]]; do
    case "$1" in
      machine)
        shift
        if [[ "${1:-}" == *"$filter"* ]]; then
          machines+=("${1:-}")
        fi
        ;;
    esac
    shift
  done

  if [[ ${#machines[@]} -eq 0 ]]; then
    echo "No ~/.netrc entries matching '$filter' found."
    echo ""
    echo "To search with a different substring: gitlab.sh discover <substring>"
  else
    echo "Found ${#machines[@]} matching host(s) in ~/.netrc:"
    for m in "${machines[@]}"; do
      echo "  $m"
    done
    echo ""
    echo "Usage: gitlab.sh <hostname-or-substring> <command>"
  fi
}

cmd_whoami() {
  local result
  result=$(api_get "/user" "")
  echo "$result" | jq '{
    username: .username,
    name: .name,
    email: .email,
    state: .state,
    is_admin: .is_admin,
    id: .id
  }'
}

cmd_test() {
  echo "Testing connection..."
  local result
  result=$(api_get "/user" "")
  local name
  name=$(echo "$result" | jq -r '.name // .username // "unknown"')
  local admin
  admin=$(echo "$result" | jq -r '.is_admin // false')
  echo "Connected as: $name (admin: $admin)"
  echo ""
  echo "Testing project listing..."
  local projects
  projects=$(api_get "/projects" "per_page=3&order_by=updated_at&sort=desc&membership=true")
  echo "$projects" | jq -r '.[] | "  \(.path_with_namespace)"'
  echo ""
  local version
  version=$(api_get "/version" "" 2>/dev/null || echo '{}')
  local ver
  ver=$(echo "$version" | jq -r '.version // "unknown"')
  echo "GitLab version: $ver"
  echo "Connection OK."
}

# --- activity & starred ---

cmd_starred() {
  local limit="${1:-25}"
  local result
  result=$(api_get "/projects" "starred=true&per_page=${limit}&order_by=updated_at&sort=desc")
  echo "$result" | jq -r '
    .[] |
    "\(.path_with_namespace)\n  Updated: \(.last_activity_at)  Stars: \(.star_count // 0)  Forks: \(.forks_count // 0)\n  URL: \(.web_url)\n"
  '
}

cmd_starred_activity() {
  local days="${1:-7}"
  local limit="${2:-30}"
  local after
  after=$(python3 -c "from datetime import datetime,timedelta; print((datetime.utcnow()-timedelta(days=${days})).strftime('%Y-%m-%dT%H:%M:%SZ'))")
  local result
  result=$(api_get "/projects" "starred=true&per_page=100&order_by=updated_at&sort=desc")
  local count
  count=$(echo "$result" | jq --arg after "$after" '[.[] | select(.last_activity_at > $after)] | length')
  echo "Starred projects with activity in last ${days} days (${count} projects):"
  echo ""
  echo "$result" | jq -r --arg after "$after" '
    .[] | select(.last_activity_at > $after) |
    "\(.path_with_namespace)\n  Last activity: \(.last_activity_at)\n  Default branch: \(.default_branch // "main")\n"
  '
}

cmd_events() {
  local limit="${1:-20}"
  local result
  result=$(api_get "/events" "per_page=${limit}&sort=desc")
  echo "$result" | jq -r '
    .[] |
    "\(.created_at) [\(.action_name)] \(.target_type // .push_data.ref_type // "project"): \(.target_title // .push_data.commit_title // "N/A")\n  Project: \(.project_id)  Author: \(.author_username // "unknown")\n"
  '
}

cmd_project_events() {
  local project_ref="${1:-}"
  [[ -n "$project_ref" ]] || die "Usage: project-events <project-id-or-path> [limit]"
  local limit="${2:-20}"
  local encoded
  encoded=$(urlencode "$project_ref")
  local result
  result=$(api_get "/projects/${encoded}/events" "per_page=${limit}&sort=desc")
  echo "$result" | jq -r '
    .[] |
    "\(.created_at) [\(.action_name)] \(.target_type // .push_data.ref_type // "event"): \(.target_title // .push_data.commit_title // "N/A")\n  Author: \(.author_username // "unknown")\n"
  '
}

# --- projects ---

cmd_projects() {
  local limit="${1:-25}"
  local raw
  raw=$(api_get_paged "/projects" "per_page=${limit}&order_by=updated_at&sort=desc&membership=true")
  local total
  total=$(echo "$raw" | head -1 | grep -o 'TOTAL:.*' | sed 's/TOTAL://' || echo "")
  local body
  body=$(echo "$raw" | grep -v '^TOTAL:')
  if [[ -n "$total" ]]; then
    echo "Your projects (${total} total):"
  else
    echo "Your projects:"
  fi
  echo ""
  echo "$body" | jq -r '
    .[] |
    "\(.path_with_namespace)\n  Updated: \(.last_activity_at)  Visibility: \(.visibility)  Default: \(.default_branch // "main")\n"
  '
}

cmd_project_info() {
  local project_ref="${1:-}"
  [[ -n "$project_ref" ]] || die "Usage: project-info <project-id-or-path>"
  local encoded
  encoded=$(urlencode "$project_ref")
  local result
  result=$(api_get "/projects/${encoded}" "statistics=true")
  echo "$result" | jq '{
    id: .id,
    name: .name,
    path: .path_with_namespace,
    description: (.description // "none"),
    visibility: .visibility,
    default_branch: (.default_branch // "main"),
    web_url: .web_url,
    created: .created_at,
    updated: .last_activity_at,
    creator: (.creator_id // null),
    topics: (.topics // []),
    star_count: (.star_count // 0),
    forks_count: (.forks_count // 0),
    open_issues_count: (.open_issues_count // 0),
    statistics: (.statistics // null)
  }'
}

# --- merge requests ---

cmd_my_mrs() {
  local state="${1:-opened}"
  local limit="${2:-25}"
  local result
  result=$(api_get "/merge_requests" "state=${state}&scope=assigned_to_me&per_page=${limit}&order_by=updated_at&sort=desc")
  local count
  count=$(echo "$result" | jq 'length')
  echo "Merge requests assigned to you (${state}, ${count} shown):"
  echo ""
  echo "$result" | jq -r '
    .[] |
    "!\(.iid) [\(.state)] \(.title)\n  Project: \(.references.full // .web_url)  Author: \(.author.username // "unknown")\n  Updated: \(.updated_at)  Target: \(.target_branch)\n"
  '
}

cmd_mr_review() {
  local state="${1:-opened}"
  local limit="${2:-25}"
  local result
  result=$(api_get "/merge_requests" "state=${state}&scope=assigned_to_me&reviewer_username=$(api_get "/user" "" | jq -r '.username')&per_page=${limit}&order_by=updated_at&sort=desc" 2>/dev/null)
  if [[ -z "$result" || "$result" == "[]" ]]; then
    # Fallback: get MRs where user is reviewer
    result=$(api_get "/merge_requests" "state=${state}&scope=all&reviewer_username=$(api_get "/user" "" | jq -r '.username')&per_page=${limit}&order_by=updated_at&sort=desc" 2>/dev/null || echo "[]")
  fi
  local count
  count=$(echo "$result" | jq 'length')
  echo "Merge requests awaiting your review (${state}, ${count} shown):"
  echo ""
  echo "$result" | jq -r '
    .[] |
    "!\(.iid) [\(.state)] \(.title)\n  Project: \(.references.full // .web_url)  Author: \(.author.username // "unknown")\n  Updated: \(.updated_at)  Target: \(.target_branch)\n"
  '
}

cmd_project_mrs() {
  local project_ref="${1:-}"
  [[ -n "$project_ref" ]] || die "Usage: project-mrs <project-id-or-path> [state] [limit]"
  local state="${2:-opened}"
  local limit="${3:-25}"
  local encoded
  encoded=$(urlencode "$project_ref")
  local result
  result=$(api_get "/projects/${encoded}/merge_requests" "state=${state}&per_page=${limit}&order_by=updated_at&sort=desc")
  echo "$result" | jq -r '
    .[] |
    "!\(.iid) [\(.state)] \(.title)\n  Author: \(.author.username // "unknown")  Updated: \(.updated_at)\n  Source: \(.source_branch) -> \(.target_branch)  Approvals: \(.upvotes // 0)\n"
  '
}

cmd_mr() {
  local project_ref="${1:-}"
  [[ -n "$project_ref" ]] || die "Usage: mr <project-id-or-path> <mr-iid>"
  local mr_iid="${2:-}"
  [[ -n "$mr_iid" ]] || die "Usage: mr <project-id-or-path> <mr-iid>"
  local encoded
  encoded=$(urlencode "$project_ref")
  local result
  result=$(api_get "/projects/${encoded}/merge_requests/${mr_iid}" "")
  echo "$result" | jq '{
    iid: .iid,
    title: .title,
    state: .state,
    author: .author.username,
    assignees: [.assignees[]?.username],
    reviewers: [.reviewers[]?.username],
    source_branch: .source_branch,
    target_branch: .target_branch,
    created: .created_at,
    updated: .updated_at,
    merged_by: (.merged_by.username // null),
    merged_at: (.merged_at // null),
    labels: (.labels // []),
    milestone: (.milestone.title // null),
    draft: (.draft // false),
    merge_status: (.merge_status // "unknown"),
    has_conflicts: (.has_conflicts // false),
    changes_count: (.changes_count // "unknown"),
    web_url: .web_url,
    description: (.description // "none")
  }'
}

cmd_mr_changes() {
  local project_ref="${1:-}"
  [[ -n "$project_ref" ]] || die "Usage: mr-changes <project-id-or-path> <mr-iid>"
  local mr_iid="${2:-}"
  [[ -n "$mr_iid" ]] || die "Usage: mr-changes <project-id-or-path> <mr-iid>"
  local encoded
  encoded=$(urlencode "$project_ref")
  local result
  result=$(api_get "/projects/${encoded}/merge_requests/${mr_iid}/changes" "")
  echo "$result" | jq -r '
    .changes[] |
    "\(.new_path) (\(if .new_file then "added" elif .deleted_file then "deleted" elif .renamed_file then "renamed from \(.old_path)" else "modified" end))\n"
  '
}

# --- issues ---

cmd_my_issues() {
  local state="${1:-opened}"
  local limit="${2:-25}"
  local result
  result=$(api_get "/issues" "state=${state}&scope=assigned_to_me&per_page=${limit}&order_by=updated_at&sort=desc")
  local count
  count=$(echo "$result" | jq 'length')
  echo "Issues assigned to you (${state}, ${count} shown):"
  echo ""
  echo "$result" | jq -r '
    .[] |
    "#\(.iid) [\(.state)] \(.title)\n  Project: \(.references.full // "unknown")  Labels: \((.labels // []) | join(", ") | if . == "" then "none" else . end)\n  Updated: \(.updated_at)\n"
  '
}

cmd_project_issues() {
  local project_ref="${1:-}"
  [[ -n "$project_ref" ]] || die "Usage: project-issues <project-id-or-path> [state] [limit]"
  local state="${2:-opened}"
  local limit="${3:-25}"
  local encoded
  encoded=$(urlencode "$project_ref")
  local result
  result=$(api_get "/projects/${encoded}/issues" "state=${state}&per_page=${limit}&order_by=updated_at&sort=desc")
  echo "$result" | jq -r '
    .[] |
    "#\(.iid) [\(.state)] \(.title)\n  Author: \(.author.username // "unknown")  Assignee: \(.assignee.username // "unassigned")  Labels: \((.labels // []) | join(", ") | if . == "" then "none" else . end)\n  Updated: \(.updated_at)\n"
  '
}

cmd_issue() {
  local project_ref="${1:-}"
  [[ -n "$project_ref" ]] || die "Usage: issue <project-id-or-path> <issue-iid>"
  local issue_iid="${2:-}"
  [[ -n "$issue_iid" ]] || die "Usage: issue <project-id-or-path> <issue-iid>"
  local encoded
  encoded=$(urlencode "$project_ref")
  local result
  result=$(api_get "/projects/${encoded}/issues/${issue_iid}" "")
  echo "$result" | jq '{
    iid: .iid,
    title: .title,
    state: .state,
    author: .author.username,
    assignees: [.assignees[]?.username],
    labels: (.labels // []),
    milestone: (.milestone.title // null),
    created: .created_at,
    updated: .updated_at,
    closed_at: (.closed_at // null),
    due_date: (.due_date // null),
    weight: (.weight // null),
    web_url: .web_url,
    description: (.description // "none")
  }'
}

# --- pipelines ---

cmd_pipelines() {
  local project_ref="${1:-}"
  [[ -n "$project_ref" ]] || die "Usage: pipelines <project-id-or-path> [limit]"
  local limit="${2:-15}"
  local encoded
  encoded=$(urlencode "$project_ref")
  local result
  result=$(api_get "/projects/${encoded}/pipelines" "per_page=${limit}&order_by=updated_at&sort=desc")
  echo "$result" | jq -r '
    .[] |
    "[#\(.id)] \(.status)  Ref: \(.ref)  Source: \(.source // "unknown")\n  Created: \(.created_at)  Updated: \(.updated_at)\n  URL: \(.web_url)\n"
  '
}

cmd_pipeline() {
  local project_ref="${1:-}"
  [[ -n "$project_ref" ]] || die "Usage: pipeline <project-id-or-path> <pipeline-id>"
  local pipeline_id="${2:-}"
  [[ -n "$pipeline_id" ]] || die "Usage: pipeline <project-id-or-path> <pipeline-id>"
  local encoded
  encoded=$(urlencode "$project_ref")
  local result
  result=$(api_get "/projects/${encoded}/pipelines/${pipeline_id}" "")
  echo "$result" | jq '{
    id: .id,
    status: .status,
    ref: .ref,
    sha: .sha,
    source: (.source // "unknown"),
    created: .created_at,
    updated: .updated_at,
    started: (.started_at // null),
    finished: (.finished_at // null),
    duration: (.duration // null),
    user: .user.username,
    web_url: .web_url
  }'
  echo ""
  echo "Jobs:"
  local jobs
  jobs=$(api_get "/projects/${encoded}/pipelines/${pipeline_id}/jobs" "per_page=50")
  echo "$jobs" | jq -r '
    .[] |
    "  [\(.status)] \(.name)  Stage: \(.stage)  Duration: \(.duration // 0)s  Runner: \(.runner.description // "N/A")"
  '
}

# --- branches & commits ---

cmd_branches() {
  local project_ref="${1:-}"
  [[ -n "$project_ref" ]] || die "Usage: branches <project-id-or-path> [limit]"
  local limit="${2:-25}"
  local encoded
  encoded=$(urlencode "$project_ref")
  local result
  result=$(api_get "/projects/${encoded}/repository/branches" "per_page=${limit}&order_by=updated&sort=desc")
  echo "$result" | jq -r '
    .[] |
    "\(.name)\(if .default then " [default]" else "" end)\(if .protected then " [protected]" else "" end)\n  Last commit: \(.commit.short_id) \(.commit.title // "no message") (\(.commit.author_name // "unknown"), \(.commit.committed_date))\n"
  '
}

cmd_commits() {
  local project_ref="${1:-}"
  [[ -n "$project_ref" ]] || die "Usage: commits <project-id-or-path> [ref] [limit]"
  local ref="${2:-}"
  local limit="${3:-15}"
  local encoded
  encoded=$(urlencode "$project_ref")
  local params="per_page=${limit}"
  [[ -z "$ref" ]] || params="${params}&ref_name=${ref}"
  local result
  result=$(api_get "/projects/${encoded}/repository/commits" "$params")
  echo "$result" | jq -r '
    .[] |
    "\(.short_id) \(.title)\n  Author: \(.author_name)  Date: \(.committed_date)\n"
  '
}

# --- groups ---

cmd_groups() {
  local limit="${1:-25}"
  local result
  result=$(api_get "/groups" "per_page=${limit}&order_by=name&sort=asc&min_access_level=10")
  echo "$result" | jq -r '
    .[] |
    "\(.full_path)\n  Visibility: \(.visibility)  Projects: \(.projects // [] | length)  Subgroups: \(.subgroups // [] | length)\n  URL: \(.web_url)\n"
  '
}

cmd_group_projects() {
  local group_ref="${1:-}"
  [[ -n "$group_ref" ]] || die "Usage: group-projects <group-id-or-path> [limit]"
  local limit="${2:-25}"
  local encoded
  encoded=$(urlencode "$group_ref")
  local result
  result=$(api_get "/groups/${encoded}/projects" "per_page=${limit}&order_by=updated_at&sort=desc&include_subgroups=true")
  echo "$result" | jq -r '
    .[] |
    "\(.path_with_namespace)\n  Updated: \(.last_activity_at)  Visibility: \(.visibility)  Stars: \(.star_count // 0)\n"
  '
}

# --- search ---

cmd_search() {
  local query="${1:-}"
  [[ -n "$query" ]] || die "Usage: search <query> [scope: projects|issues|merge_requests|milestones|blobs]"
  local scope="${2:-projects}"
  local limit="${3:-20}"
  local result
  result=$(api_get "/search" "search=$(urlencode "$query")&scope=${scope}&per_page=${limit}")
  case "$scope" in
    projects)
      echo "$result" | jq -r '
        .[] |
        "\(.path_with_namespace)\n  Description: \(.description // "none" | .[0:120])\n  Updated: \(.last_activity_at)\n"
      '
      ;;
    issues)
      echo "$result" | jq -r '
        .[] |
        "#\(.iid) [\(.state)] \(.title)\n  Project: \(.references.full // "unknown")  Updated: \(.updated_at)\n"
      '
      ;;
    merge_requests)
      echo "$result" | jq -r '
        .[] |
        "!\(.iid) [\(.state)] \(.title)\n  Project: \(.references.full // "unknown")  Updated: \(.updated_at)\n"
      '
      ;;
    blobs)
      echo "$result" | jq -r '
        .[] |
        "\(.path) (project: \(.project_id))\n  Ref: \(.ref)  Filename: \(.filename)\n"
      '
      ;;
    *)
      echo "$result" | jq '.'
      ;;
  esac
}

cmd_project_search() {
  local project_ref="${1:-}"
  [[ -n "$project_ref" ]] || die "Usage: project-search <project-id-or-path> <query> [scope: blobs|commits|issues|merge_requests]"
  local query="${2:-}"
  [[ -n "$query" ]] || die "Usage: project-search <project-id-or-path> <query> [scope]"
  local scope="${3:-blobs}"
  local encoded
  encoded=$(urlencode "$project_ref")
  local result
  result=$(api_get "/projects/${encoded}/search" "search=$(urlencode "$query")&scope=${scope}&per_page=20")
  case "$scope" in
    blobs)
      echo "$result" | jq -r '
        .[] |
        "\(.path):\(.startline)\n  \(.data | gsub("\n"; " ") | .[0:200])\n"
      '
      ;;
    commits)
      echo "$result" | jq -r '
        .[] |
        "\(.short_id) \(.title)\n  Author: \(.author_name)  Date: \(.committed_date)\n"
      '
      ;;
    *)
      echo "$result" | jq '.'
      ;;
  esac
}

# --- files ---

cmd_tree() {
  local project_ref="${1:-}"
  [[ -n "$project_ref" ]] || die "Usage: tree <project-id-or-path> [path] [ref]"
  local path="${2:-.}"
  local ref="${3:-}"
  local encoded
  encoded=$(urlencode "$project_ref")
  local params="per_page=100&recursive=false"
  [[ "$path" == "." ]] || params="${params}&path=${path}"
  [[ -z "$ref" ]] || params="${params}&ref=${ref}"
  local result
  result=$(api_get "/projects/${encoded}/repository/tree" "$params")
  echo "$result" | jq -r '
    .[] |
    "\(if .type == "tree" then "📁" else "  " end) \(.name)\(if .type == "tree" then "/" else "" end)"
  '
}

cmd_file() {
  local project_ref="${1:-}"
  [[ -n "$project_ref" ]] || die "Usage: file <project-id-or-path> <file-path> [ref]"
  local file_path="${2:-}"
  [[ -n "$file_path" ]] || die "Usage: file <project-id-or-path> <file-path> [ref]"
  local ref="${3:-}"
  local encoded_project
  encoded_project=$(urlencode "$project_ref")
  local encoded_file
  encoded_file=$(urlencode "$file_path")
  local params=""
  [[ -z "$ref" ]] || params="ref=${ref}"
  local result
  result=$(api_get "/projects/${encoded_project}/repository/files/${encoded_file}" "$params")
  local content
  content=$(echo "$result" | jq -r '.content // empty')
  local encoding
  encoding=$(echo "$result" | jq -r '.encoding // "base64"')
  echo "File: $(echo "$result" | jq -r '.file_path')"
  echo "Size: $(echo "$result" | jq -r '.size') bytes  Ref: $(echo "$result" | jq -r '.ref')"
  echo "Last commit: $(echo "$result" | jq -r '.last_commit_id[0:8]')"
  echo "---"
  if [[ "$encoding" == "base64" ]]; then
    echo "$content" | base64 -d 2>/dev/null || echo "[binary content]"
  else
    echo "$content"
  fi
}

# --- container registry ---

cmd_registries() {
  local project_ref="${1:-}"
  [[ -n "$project_ref" ]] || die "Usage: registries <project-id-or-path>"
  local encoded
  encoded=$(urlencode "$project_ref")
  local result
  result=$(api_get "/projects/${encoded}/registry/repositories" "per_page=50")
  echo "$result" | jq -r '
    .[] |
    "[\(.id)] \(.path)\n  Tags: \(.tags_count // 0)  Created: \(.created_at)\n"
  '
}

# --- main ---

require_jq

COMMAND="${1:-help}"

case "$COMMAND" in
  discover)
    shift; cmd_discover "$@" ;;
  help)
    echo "Usage: gitlab.sh <host> <command> [args...]"
    echo ""
    echo "Discovery:"
    echo "  discover [substring]                             Find GitLab hosts in ~/.netrc"
    echo ""
    echo "Query commands (host = hostname or unique substring from ~/.netrc):"
    echo "  <host> whoami                                    Current user"
    echo "  <host> test                                      Test connection"
    echo ""
    echo "Activity & starred:"
    echo "  <host> starred [limit]                           Starred projects"
    echo "  <host> starred-activity [days] [limit]           Starred projects with recent activity"
    echo "  <host> events [limit]                            Your recent activity feed"
    echo "  <host> project-events <project> [limit]          Project activity"
    echo ""
    echo "Projects:"
    echo "  <host> projects [limit]                          Your projects (by membership)"
    echo "  <host> project-info <project>                    Project details + statistics"
    echo ""
    echo "Merge requests:"
    echo "  <host> my-mrs [state] [limit]                    MRs assigned to you"
    echo "  <host> mr-review [state] [limit]                 MRs awaiting your review"
    echo "  <host> project-mrs <project> [state] [limit]     MRs in a project"
    echo "  <host> mr <project> <iid>                        MR details"
    echo "  <host> mr-changes <project> <iid>                MR changed files"
    echo ""
    echo "Issues:"
    echo "  <host> my-issues [state] [limit]                 Issues assigned to you"
    echo "  <host> project-issues <project> [state] [limit]  Issues in a project"
    echo "  <host> issue <project> <iid>                     Issue details"
    echo ""
    echo "Pipelines:"
    echo "  <host> pipelines <project> [limit]               Recent pipelines"
    echo "  <host> pipeline <project> <id>                   Pipeline details + jobs"
    echo ""
    echo "Code:"
    echo "  <host> branches <project> [limit]                List branches"
    echo "  <host> commits <project> [ref] [limit]           Recent commits"
    echo "  <host> tree <project> [path] [ref]               Directory listing"
    echo "  <host> file <project> <path> [ref]               Read file content"
    echo ""
    echo "Groups:"
    echo "  <host> groups [limit]                            Your groups"
    echo "  <host> group-projects <group> [limit]            Projects in a group"
    echo ""
    echo "Search:"
    echo "  <host> search <query> [scope] [limit]            Global search"
    echo "  <host> project-search <project> <query> [scope]  Project-scoped search"
    echo ""
    echo "Registry:"
    echo "  <host> registries <project>                      Container registry repos"
    echo ""
    echo "Search scopes: projects, issues, merge_requests, milestones, blobs"
    echo "Project refs: use ID (numeric) or URL-encoded path (group%2Fproject)"
    exit 0
    ;;
  *)
    HOST="$1"
    shift
    SUBCMD="${1:-help}"
    shift || true
    resolve_host "$HOST"
    case "$SUBCMD" in
      whoami)            cmd_whoami ;;
      test)              cmd_test ;;
      starred)           cmd_starred "$@" ;;
      starred-activity)  cmd_starred_activity "$@" ;;
      events)            cmd_events "$@" ;;
      project-events)    cmd_project_events "$@" ;;
      projects)          cmd_projects "$@" ;;
      project-info)      cmd_project_info "$@" ;;
      my-mrs)            cmd_my_mrs "$@" ;;
      mr-review)         cmd_mr_review "$@" ;;
      project-mrs)       cmd_project_mrs "$@" ;;
      mr)                cmd_mr "$@" ;;
      mr-changes)        cmd_mr_changes "$@" ;;
      my-issues)         cmd_my_issues "$@" ;;
      project-issues)    cmd_project_issues "$@" ;;
      issue)             cmd_issue "$@" ;;
      pipelines)         cmd_pipelines "$@" ;;
      pipeline)          cmd_pipeline "$@" ;;
      branches)          cmd_branches "$@" ;;
      commits)           cmd_commits "$@" ;;
      tree)              cmd_tree "$@" ;;
      file)              cmd_file "$@" ;;
      groups)            cmd_groups "$@" ;;
      group-projects)    cmd_group_projects "$@" ;;
      search)            cmd_search "$@" ;;
      project-search)    cmd_project_search "$@" ;;
      registries)        cmd_registries "$@" ;;
      help)
        echo "Usage: gitlab.sh <host> <command> [args...]"
        echo "Run 'gitlab.sh help' for full command list."
        ;;
      *)                 die "Unknown command: $SUBCMD (run 'gitlab.sh help')" ;;
    esac
    ;;
esac
