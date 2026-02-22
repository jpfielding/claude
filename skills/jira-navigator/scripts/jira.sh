#!/usr/bin/env bash
#
# jira.sh - Jira REST API CLI wrapper
#
# Usage: jira.sh <host> <command> [args...]
#
# Credentials from ~/.netrc, Bearer auth

set -euo pipefail

# --- helpers ---

die() { echo "ERROR: $*" >&2; exit 1; }

require_jq() {
  command -v jq >/dev/null 2>&1 || die "jq is required but not installed"
}

parse_netrc() {
  local machine="$1"
  local netrc_file="${HOME}/.netrc"
  [[ -f "$netrc_file" ]] || die "~/.netrc not found. Create it with:\n  machine $machine\n  login <username>\n  password <token>"

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

extract_hostname() {
  echo "$1" | sed -E 's|^https?://||' | cut -d/ -f1 | cut -d: -f1
}

resolve_host() {
  local input="$1"
  local netrc_file="${HOME}/.netrc"
  [[ -f "$netrc_file" ]] || die "~/.netrc not found"

  if [[ "$input" == *"."* ]]; then
    # Input contains a dot - use as-is
    HOSTNAME="$input"
    BASE_URL="https://${HOSTNAME}"
    parse_netrc "$HOSTNAME"
    return
  fi

  # Substring match against machine entries in ~/.netrc
  local machines=()
  local tokens
  tokens=$(tr '\n' ' ' < "$netrc_file")
  set -- $tokens
  while [[ $# -gt 0 ]]; do
    case "$1" in
      machine)
        shift
        if [[ "${1:-}" == *"$input"* && "${1:-}" == *"jira"* ]]; then
          machines+=("${1:-}")
        fi
        ;;
    esac
    shift
  done

  if [[ ${#machines[@]} -eq 0 ]]; then
    die "No ~/.netrc machine entries matching '$input' (filtered for jira hosts)"
  fi

  if [[ ${#machines[@]} -gt 1 ]]; then
    echo "Multiple ~/.netrc entries match '$input':" >&2
    for m in "${machines[@]}"; do
      echo "  $m" >&2
    done
    die "Ambiguous host substring '$input'"
  fi

  HOSTNAME="${machines[0]}"
  BASE_URL="https://${HOSTNAME}"
  parse_netrc "$HOSTNAME"
}

api_get() {
  local endpoint="$1"
  shift
  local url="${BASE_URL}/rest/api/2${endpoint}"

  if [[ $# -gt 0 && -n "${1:-}" ]]; then
    local params="$1"
    url="${url}?${params}"
  fi

  local response
  response=$(curl -sS -w "\n%{http_code}" \
    -H "Authorization: Bearer $NETRC_PASSWORD" \
    -H "Content-Type: application/json" \
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

api_get_agile() {
  local endpoint="$1"
  shift
  local url="${BASE_URL}/rest/agile/1.0${endpoint}"

  if [[ $# -gt 0 && -n "${1:-}" ]]; then
    local params="$1"
    url="${url}?${params}"
  fi

  local response
  response=$(curl -sS -w "\n%{http_code}" \
    -H "Authorization: Bearer $NETRC_PASSWORD" \
    -H "Content-Type: application/json" \
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

# --- commands ---

cmd_discover() {
  # Scan ~/.netrc for machine entries that look like Jira hosts
  local filter="${1:-jira}"
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
    echo "To search with a different substring: jira.sh discover <substring>"
    return
  fi

  echo "Found ~/.netrc entries matching '$filter':"
  echo ""
  local i=1
  for m in "${machines[@]}"; do
    echo "  [$i] $m"
    i=$((i + 1))
  done
  echo ""
  echo "Usage: jira.sh <hostname-or-substring> <command>"
}

cmd_whoami() {
  local result
  result=$(api_get "/myself" "")
  echo "$result" | jq '{
    name: .name,
    displayName: .displayName,
    emailAddress: .emailAddress,
    key: .key,
    timeZone: .timeZone
  }'
}

cmd_test() {
  echo "Testing connection..."
  local result
  result=$(api_get "/myself" "")
  local name
  name=$(echo "$result" | jq -r '.displayName // .name // "unknown"')
  echo "Connected as: $name"
  echo ""
  echo "Testing project listing..."
  local projects
  projects=$(api_get "/project" "maxResults=3")
  echo "$projects" | jq -r '.[] | "  \(.key) - \(.name)"'
  echo ""
  echo "Connection OK."
}

cmd_recent() {
  # Recently updated issues across the instance
  local limit="${1:-20}"
  local jql="ORDER BY updated DESC"
  local encoded_jql
  encoded_jql=$(python3 -c "import urllib.parse,sys; print(urllib.parse.quote(sys.argv[1]))" "$jql")
  local result
  result=$(api_get "/search" "jql=${encoded_jql}&maxResults=${limit}&fields=summary,status,assignee,updated,priority,issuetype,project")
  echo "$result" | jq -r '
    .issues[] |
    "[\(.fields.project.key)] \(.key): \(.fields.summary)\n  Status: \(.fields.status.name)  Priority: \(.fields.priority.name // "None")  Type: \(.fields.issuetype.name)\n  Assignee: \(.fields.assignee.displayName // "Unassigned")  Updated: \(.fields.updated)\n"
  '
}

cmd_my_issues() {
  # Issues assigned to the current user
  local limit="${1:-25}"
  local jql="assignee = currentUser() AND resolution = Unresolved ORDER BY updated DESC"
  local encoded_jql
  encoded_jql=$(python3 -c "import urllib.parse,sys; print(urllib.parse.quote(sys.argv[1]))" "$jql")
  local result
  result=$(api_get "/search" "jql=${encoded_jql}&maxResults=${limit}&fields=summary,status,priority,issuetype,project,updated")
  local count
  count=$(echo "$result" | jq '.total')
  echo "Open issues assigned to you (${count} total):"
  echo ""
  echo "$result" | jq -r '
    .issues[] |
    "[\(.fields.project.key)] \(.key): \(.fields.summary)\n  Status: \(.fields.status.name)  Priority: \(.fields.priority.name // "None")  Type: \(.fields.issuetype.name)\n  Updated: \(.fields.updated)\n"
  '
}

cmd_watched() {
  # Issues the current user is watching
  local limit="${1:-25}"
  local jql="watcher = currentUser() AND resolution = Unresolved ORDER BY updated DESC"
  local encoded_jql
  encoded_jql=$(python3 -c "import urllib.parse,sys; print(urllib.parse.quote(sys.argv[1]))" "$jql")
  local result
  result=$(api_get "/search" "jql=${encoded_jql}&maxResults=${limit}&fields=summary,status,assignee,priority,issuetype,project,updated")
  local count
  count=$(echo "$result" | jq '.total')
  echo "Unresolved watched issues (${count} total):"
  echo ""
  echo "$result" | jq -r '
    .issues[] |
    "[\(.fields.project.key)] \(.key): \(.fields.summary)\n  Status: \(.fields.status.name)  Priority: \(.fields.priority.name // "None")  Assignee: \(.fields.assignee.displayName // "Unassigned")\n  Updated: \(.fields.updated)\n"
  '
}

cmd_watch_changes() {
  # Recently updated issues that the current user is watching
  local days="${1:-7}"
  local jql="watcher = currentUser() AND updated >= -${days}d ORDER BY updated DESC"
  local encoded_jql
  encoded_jql=$(python3 -c "import urllib.parse,sys; print(urllib.parse.quote(sys.argv[1]))" "$jql")
  local result
  result=$(api_get "/search" "jql=${encoded_jql}&maxResults=50&fields=summary,status,assignee,priority,issuetype,project,updated")
  local count
  count=$(echo "$result" | jq '.total')
  echo "Watched issues updated in last ${days} days (${count} results):"
  echo ""
  echo "$result" | jq -r '
    .issues[] |
    "[\(.fields.project.key)] \(.key): \(.fields.summary)\n  Status: \(.fields.status.name)  Priority: \(.fields.priority.name // "None")  Assignee: \(.fields.assignee.displayName // "Unassigned")\n  Updated: \(.fields.updated)\n"
  '
}

cmd_search() {
  local jql="${1:-}"
  [[ -n "$jql" ]] || die "Usage: search <JQL query> [limit]"
  local limit="${2:-20}"
  local encoded_jql
  encoded_jql=$(python3 -c "import urllib.parse,sys; print(urllib.parse.quote(sys.argv[1]))" "$jql")
  local result
  result=$(api_get "/search" "jql=${encoded_jql}&maxResults=${limit}&fields=summary,status,assignee,priority,issuetype,project,updated")
  local count
  count=$(echo "$result" | jq '.total')
  echo "Results (${count} total):"
  echo ""
  echo "$result" | jq -r '
    .issues[] |
    "[\(.fields.project.key)] \(.key): \(.fields.summary)\n  Status: \(.fields.status.name)  Priority: \(.fields.priority.name // "None")  Assignee: \(.fields.assignee.displayName // "Unassigned")\n  Updated: \(.fields.updated)\n"
  '
}

cmd_issue() {
  local issue_key="${1:-}"
  [[ -n "$issue_key" ]] || die "Usage: issue <issue-key>"
  local result
  result=$(api_get "/issue/${issue_key}" "expand=renderedFields,names,changelog")
  echo "$result" | jq -r '
    "Key: \(.key)\nSummary: \(.fields.summary)\nType: \(.fields.issuetype.name)\nStatus: \(.fields.status.name)\nPriority: \(.fields.priority.name // "None")\nProject: \(.fields.project.key) - \(.fields.project.name)\nAssignee: \(.fields.assignee.displayName // "Unassigned")\nReporter: \(.fields.reporter.displayName // "Unknown")\nCreated: \(.fields.created)\nUpdated: \(.fields.updated)\nResolution: \(.fields.resolution.name // "Unresolved")\nLabels: \((.fields.labels // []) | join(", "))\nComponents: \([.fields.components[]?.name] | join(", "))\nFix Versions: \([.fields.fixVersions[]?.name] | join(", "))\n\n--- Description ---\n\(.renderedFields.description // .fields.description // "No description")"
  '
}

cmd_issue_info() {
  # Compact metadata view (no description)
  local issue_key="${1:-}"
  [[ -n "$issue_key" ]] || die "Usage: issue-info <issue-key>"
  local result
  result=$(api_get "/issue/${issue_key}" "fields=summary,status,priority,issuetype,project,assignee,reporter,created,updated,resolution,labels,components,fixVersions,subtasks,issuelinks,parent")
  echo "$result" | jq '{
    key: .key,
    summary: .fields.summary,
    type: .fields.issuetype.name,
    status: .fields.status.name,
    priority: (.fields.priority.name // "None"),
    project: {key: .fields.project.key, name: .fields.project.name},
    assignee: (.fields.assignee.displayName // "Unassigned"),
    reporter: (.fields.reporter.displayName // "Unknown"),
    created: .fields.created,
    updated: .fields.updated,
    resolution: (.fields.resolution.name // "Unresolved"),
    labels: (.fields.labels // []),
    components: [.fields.components[]?.name],
    fixVersions: [.fields.fixVersions[]?.name],
    parent: (.fields.parent | if . then {key: .key, summary: .fields.summary} else null end),
    subtasks: [.fields.subtasks[]? | {key: .key, summary: .fields.summary, status: .fields.status.name}],
    links: [.fields.issuelinks[]? | {
      type: .type.name,
      direction: (if .outwardIssue then "outward" else "inward" end),
      issue: ((.outwardIssue // .inwardIssue) | {key: .key, summary: .fields.summary, status: .fields.status.name})
    }]
  }'
}

cmd_comments() {
  local issue_key="${1:-}"
  [[ -n "$issue_key" ]] || die "Usage: comments <issue-key> [limit]"
  local limit="${2:-20}"
  local result
  result=$(api_get "/issue/${issue_key}/comment" "maxResults=${limit}&orderBy=-created")
  echo "$result" | jq -r '
    .comments[] |
    "\(.author.displayName) (\(.created)):\n\(.body)\n---\n"
  '
}

cmd_transitions() {
  local issue_key="${1:-}"
  [[ -n "$issue_key" ]] || die "Usage: transitions <issue-key>"
  local result
  result=$(api_get "/issue/${issue_key}/transitions" "")
  echo "Available transitions for ${issue_key}:"
  echo ""
  echo "$result" | jq -r '
    .transitions[] |
    "  [\(.id)] \(.name) -> \(.to.name)"
  '
}

cmd_projects() {
  local result
  result=$(api_get "/project" "")
  echo "$result" | jq -r '
    .[] |
    "\(.key) - \(.name)\n  Lead: \(.lead.displayName // "Unknown")  Type: \(.projectTypeKey // "unknown")\n"
  '
}

cmd_project_info() {
  local project_key="${1:-}"
  [[ -n "$project_key" ]] || die "Usage: project-info <project-key>"
  local result
  result=$(api_get "/project/${project_key}" "expand=description,lead,issueTypes,components,versions")
  echo "$result" | jq '{
    key: .key,
    name: .name,
    description: (.description // "None"),
    lead: (.lead.displayName // "Unknown"),
    projectType: (.projectTypeKey // "unknown"),
    issueTypes: [.issueTypes[]? | .name],
    components: [.components[]? | {name: .name, lead: (.lead.displayName // null)}],
    versions: [.versions[]? | {name: .name, released: .released, releaseDate: (.releaseDate // null)}]
  }'
}

cmd_filters() {
  local result
  result=$(api_get "/filter/favourite" "")
  echo "Favourite filters:"
  echo ""
  echo "$result" | jq -r '
    .[] |
    "[\(.id)] \(.name)\n  JQL: \(.jql)\n  Owner: \(.owner.displayName // "Unknown")\n"
  '
}

cmd_boards() {
  local result
  result=$(api_get_agile "/board" "maxResults=50")
  echo "$result" | jq -r '
    .values[] |
    "[\(.id)] \(.name)\n  Type: \(.type)  Project: \(.location.projectKey // "N/A")\n"
  '
}

cmd_sprints() {
  local board_id="${1:-}"
  [[ -n "$board_id" ]] || die "Usage: sprints <board-id> [state: active|closed|future]"
  local state="${2:-active}"
  local result
  result=$(api_get_agile "/board/${board_id}/sprint" "state=${state}&maxResults=10")
  echo "$result" | jq -r '
    .values[] |
    "[\(.id)] \(.name)\n  State: \(.state)  Start: \(.startDate // "N/A")  End: \(.endDate // "N/A")\n"
  '
}

cmd_sprint_issues() {
  local sprint_id="${1:-}"
  [[ -n "$sprint_id" ]] || die "Usage: sprint-issues <sprint-id> [limit]"
  local limit="${2:-50}"
  local result
  result=$(api_get_agile "/sprint/${sprint_id}/issue" "maxResults=${limit}&fields=summary,status,assignee,priority,issuetype,project")
  echo "$result" | jq -r '
    .issues[] |
    "[\(.fields.project.key)] \(.key): \(.fields.summary)\n  Status: \(.fields.status.name)  Priority: \(.fields.priority.name // "None")  Assignee: \(.fields.assignee.displayName // "Unassigned")\n"
  '
}

cmd_changelog() {
  local issue_key="${1:-}"
  [[ -n "$issue_key" ]] || die "Usage: changelog <issue-key> [limit]"
  local limit="${2:-10}"
  local result
  result=$(api_get "/issue/${issue_key}" "expand=changelog&fields=summary")
  echo "Changelog for ${issue_key}: $(echo "$result" | jq -r '.fields.summary')"
  echo ""
  echo "$result" | jq -r --argjson limit "$limit" '
    .changelog.histories | sort_by(.created) | reverse | .[:$limit] | .[] |
    "\(.author.displayName) (\(.created)):" +
    (.items | map("  \(.field): \(.fromString // "none") -> \(.toString // "none")") | join("\n")) +
    "\n"
  '
}

cmd_statuses() {
  local project_key="${1:-}"
  if [[ -n "$project_key" ]]; then
    local result
    result=$(api_get "/project/${project_key}/statuses" "")
    echo "$result" | jq -r '
      .[] |
      "\(.name):" +
      (.statuses | map("  [\(.id)] \(.name) (\(.statusCategory.name))") | join("\n")) +
      "\n"
    '
  else
    local result
    result=$(api_get "/status" "")
    echo "$result" | jq -r '
      .[] |
      "[\(.id)] \(.name) (\(.statusCategory.name))"
    '
  fi
}

# --- main ---

require_jq

COMMAND="${1:-help}"

case "$COMMAND" in
  discover)
    shift; cmd_discover "$@" ;;
  help)
    echo "Usage: jira.sh <host> <command> [args...]"
    echo ""
    echo "Discovery:"
    echo "  discover [substring]                  Find Jira hosts in ~/.netrc"
    echo ""
    echo "Commands (first arg is hostname or substring matching ~/.netrc):"
    echo "  <host> whoami                         Show current user"
    echo "  <host> test                           Test connection"
    echo "  <host> recent [limit]                 Recently updated issues"
    echo "  <host> my-issues [limit]              Issues assigned to you"
    echo "  <host> watched [limit]                Unresolved watched issues"
    echo "  <host> watch-changes [days]           Watched issues updated recently (default: 7d)"
    echo "  <host> search <JQL> [limit]           Search via JQL"
    echo "  <host> issue <key>                    Full issue details + description"
    echo "  <host> issue-info <key>               Compact issue metadata (JSON)"
    echo "  <host> comments <key> [limit]         Issue comments"
    echo "  <host> transitions <key>              Available status transitions"
    echo "  <host> changelog <key> [limit]        Issue change history"
    echo "  <host> projects                       List all projects"
    echo "  <host> project-info <key>             Project details"
    echo "  <host> statuses [project-key]         List statuses"
    echo "  <host> filters                        Favourite/saved filters"
    echo "  <host> boards                         List agile boards"
    echo "  <host> sprints <board-id> [state]     List sprints (active|closed|future)"
    echo "  <host> sprint-issues <sprint-id>      Issues in a sprint"
    exit 0
    ;;
  *)
    HOST="$1"
    shift
    SUBCMD="${1:-help}"
    shift || true
    resolve_host "$HOST"
    case "$SUBCMD" in
      whoami)          cmd_whoami ;;
      test)            cmd_test ;;
      recent)          cmd_recent "$@" ;;
      my-issues)       cmd_my_issues "$@" ;;
      watched)         cmd_watched "$@" ;;
      watch-changes)   cmd_watch_changes "$@" ;;
      search)          cmd_search "$@" ;;
      issue)           cmd_issue "$@" ;;
      issue-info)      cmd_issue_info "$@" ;;
      comments)        cmd_comments "$@" ;;
      transitions)     cmd_transitions "$@" ;;
      changelog)       cmd_changelog "$@" ;;
      projects)        cmd_projects ;;
      project-info)    cmd_project_info "$@" ;;
      statuses)        cmd_statuses "$@" ;;
      filters)         cmd_filters ;;
      boards)          cmd_boards ;;
      sprints)         cmd_sprints "$@" ;;
      sprint-issues)   cmd_sprint_issues "$@" ;;
      help)
        echo "Usage: jira.sh <host> <command> [args...]"
        echo "Run 'jira.sh help' for full command list."
        ;;
      *)               die "Unknown command: $SUBCMD (run 'jira.sh help')" ;;
    esac
    ;;
esac
