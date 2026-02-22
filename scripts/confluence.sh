#!/usr/bin/env bash
#
# confluence.sh - Confluence REST API CLI wrapper
#
# Usage: confluence.sh <hostname-or-substring> <command> [args...]
#
# Credentials from ~/.netrc, Bearer auth

set -euo pipefail

# --- helpers ---

die() { echo "ERROR: $*" >&2; exit 1; }

require_jq() {
  command -v jq >/dev/null 2>&1 || die "jq is required but not installed"
}

resolve_host() {
  # Takes a hostname or substring, scans ~/.netrc for matches.
  # Sets HOSTNAME and BASE_URL, then calls parse_netrc.
  local input="$1"
  local netrc_file="${HOME}/.netrc"
  [[ -f "$netrc_file" ]] || die "~/.netrc not found"

  if [[ "$input" == *.* ]]; then
    # Input contains a dot — use as-is
    HOSTNAME="$input"
  else
    # Substring match against machine entries in ~/.netrc
    local machines=()
    local tokens
    tokens=$(tr '\n' ' ' < "$netrc_file")
    set -- $tokens
    while [[ $# -gt 0 ]]; do
      case "$1" in
        machine)
          shift
          if [[ "${1:-}" == *"$input"* && "${1:-}" == *"confluence"* ]]; then
            machines+=("${1:-}")
          fi
          ;;
      esac
      shift
    done

    if [[ ${#machines[@]} -eq 0 ]]; then
      die "No ~/.netrc machine entries matching '$input'"
    elif [[ ${#machines[@]} -gt 1 ]]; then
      echo "Multiple matches for '$input':" >&2
      for m in "${machines[@]}"; do
        echo "  $m" >&2
      done
      die "Ambiguous host substring '$input' — be more specific or use the full hostname"
    fi

    HOSTNAME="${machines[0]}"
  fi

  BASE_URL="https://${HOSTNAME}"
  parse_netrc "$HOSTNAME"
}

parse_netrc() {
  # Extract login and password for a given machine from ~/.netrc
  # Handles both single-line and multi-line formats:
  #   machine host login user password pass
  #   machine host\n  login user\n  password pass
  local machine="$1"
  local netrc_file="${HOME}/.netrc"
  [[ -f "$netrc_file" ]] || die "~/.netrc not found. Create it with:\n  machine $machine\n  login <username>\n  password <token>"

  NETRC_LOGIN=""
  NETRC_PASSWORD=""

  # Tokenize the entire file and walk through tokens
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
  # Extract hostname from a URL
  echo "$1" | sed -E 's|^https?://||' | cut -d/ -f1 | cut -d: -f1
}

auth_header() {
  echo "Authorization: Bearer $NETRC_PASSWORD"
}

api_get() {
  local endpoint="$1"
  shift
  local url="${BASE_URL}/rest/api${endpoint}"

  # Append query params if provided
  if [[ $# -gt 0 ]]; then
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
  # Scan ~/.netrc for machine entries that look like Confluence hosts
  local filter="${1:-confluence}"
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
    echo "To search with a different substring: confluence.sh discover <substring>"
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
  echo "Usage: confluence.sh <hostname-or-substring> <command>"
  echo ""
  echo "Example using the first match:"
  echo "  confluence.sh ${machines[0]} test"
}

cmd_recent() {
  local limit="${1:-20}"
  local result
  result=$(api_get "/content" "orderby=history.lastUpdated%20desc&limit=${limit}&expand=history.lastUpdated,space,version")
  echo "$result" | jq -r '
    .results[] |
    "[\(.space.key)] \(.title)\n  Updated: \(.version.when // .history.lastUpdated) by \(.version.by.displayName // "unknown")\n  URL: \(.["_links"].self // "N/A")\n"
  '
}

cmd_watched() {
  # Get current user's watches via the user/watch API
  local result
  result=$(api_get "/user/watch" "limit=50")
  if echo "$result" | jq -e '.results' >/dev/null 2>&1; then
    echo "$result" | jq -r '
      .results[] |
      "[\(.content.space.key // "?")] \(.content.title // .space.name // "unknown")\n  Type: \(.type // "unknown")\n"
    '
  else
    # Fallback: try content watches for spaces
    echo "Direct watch API not available. Trying alternative approach..."
    # Get current user key first
    local user_result
    user_result=$(api_get "/user/current" "")
    local user_key
    user_key=$(echo "$user_result" | jq -r '.userKey // .key // empty')
    if [[ -n "$user_key" ]]; then
      echo "Current user: $(echo "$user_result" | jq -r '.displayName // .username')"
      echo ""
      echo "Use 'search' command with CQL to find recently modified content in your watched spaces."
      echo "Example: confluence.sh <instance> search 'watcher = currentUser() order by lastModified desc'"
    else
      echo "Could not determine current user. Check authentication."
    fi
  fi
}

cmd_watch_changes() {
  # Search for recent changes in content the current user is watching
  local days="${1:-7}"
  local cql="watcher = currentUser() AND lastModified >= now(\"-${days}d\") ORDER BY lastModified DESC"
  local encoded_cql
  encoded_cql=$(python3 -c "import urllib.parse; print(urllib.parse.quote('$cql'))" 2>/dev/null || echo "$cql")
  local result
  result=$(api_get "/content/search" "cql=${encoded_cql}&limit=25&expand=history.lastUpdated,space,version")
  if echo "$result" | jq -e '.results' >/dev/null 2>&1; then
    local count
    count=$(echo "$result" | jq '.results | length')
    echo "Changes to watched content in last ${days} days (${count} results):"
    echo ""
    echo "$result" | jq -r '
      .results[] |
      "[\(.space.key // "?")] \(.title)\n  Updated: \(.version.when // "unknown") by \(.version.by.displayName // "unknown")\n  ID: \(.id)\n"
    '
  else
    echo "No results or CQL not supported. Raw response:"
    echo "$result" | jq '.' 2>/dev/null || echo "$result"
  fi
}

cmd_search() {
  local cql="${1:-}"
  [[ -n "$cql" ]] || die "Usage: search <CQL query>"
  local limit="${2:-20}"
  local encoded_cql
  encoded_cql=$(python3 -c "import urllib.parse,sys; print(urllib.parse.quote(sys.argv[1]))" "$cql" 2>/dev/null || echo "$cql")
  local result
  result=$(api_get "/content/search" "cql=${encoded_cql}&limit=${limit}&expand=space,version")
  echo "$result" | jq -r '
    .results[] |
    "[\(.space.key // "?")] \(.title) (id: \(.id))\n  Updated: \(.version.when // "unknown") by \(.version.by.displayName // "unknown")\n"
  '
}

cmd_page() {
  local page_id="${1:-}"
  [[ -n "$page_id" ]] || die "Usage: page <page-id> [format: storage|view]"
  local format="${2:-view}"
  local result
  result=$(api_get "/content/${page_id}" "expand=body.${format},space,version,ancestors")
  echo "$result" | jq -r '
    "Title: \(.title)\nSpace: \(.space.key) - \(.space.name)\nVersion: \(.version.number) by \(.version.by.displayName) at \(.version.when)\nAncestors: \([.ancestors[]?.title] | join(" > "))\n\n--- Content ---\n\(.body[env.format // "view"].value // "No content")"
  ' --arg format "$format"
}

cmd_page_info() {
  local page_id="${1:-}"
  [[ -n "$page_id" ]] || die "Usage: page-info <page-id>"
  local result
  result=$(api_get "/content/${page_id}" "expand=space,version,ancestors,children.page,history,metadata.labels")
  echo "$result" | jq '{
    id: .id,
    title: .title,
    space: {key: .space.key, name: .space.name},
    version: {number: .version.number, by: .version.by.displayName, when: .version.when},
    ancestors: [.ancestors[]? | {id: .id, title: .title}],
    children: [.children.page.results[]? | {id: .id, title: .title}],
    labels: [.metadata.labels.results[]? | .name],
    created: .history.createdDate,
    creator: .history.createdBy.displayName
  }'
}

cmd_spaces() {
  local limit="${1:-50}"
  local result
  result=$(api_get "/space" "limit=${limit}&expand=description.plain")
  echo "$result" | jq -r '
    .results[] |
    "\(.key) - \(.name)\n  Type: \(.type)\n  Description: \(.description.plain.value // "none" | gsub("\n"; " ") | .[0:120])\n"
  '
}

cmd_space_pages() {
  local space_key="${1:-}"
  [[ -n "$space_key" ]] || die "Usage: space-pages <space-key> [limit]"
  local limit="${2:-25}"
  local result
  result=$(api_get "/space/${space_key}/content/page" "limit=${limit}&expand=version&orderby=history.lastUpdated%20desc")
  echo "$result" | jq -r '
    .results[] |
    "\(.title) (id: \(.id))\n  Version: \(.version.number) by \(.version.by.displayName // "unknown") at \(.version.when // "unknown")\n"
  '
}

cmd_whoami() {
  local result
  result=$(api_get "/user/current" "")
  echo "$result" | jq '{
    username: .username,
    displayName: .displayName,
    email: .email,
    userKey: (.userKey // .key)
  }'
}

cmd_children() {
  local page_id="${1:-}"
  [[ -n "$page_id" ]] || die "Usage: children <page-id>"
  local result
  result=$(api_get "/content/${page_id}/child/page" "limit=50&expand=version")
  echo "$result" | jq -r '
    .results[] |
    "\(.title) (id: \(.id))\n  Version: \(.version.number) by \(.version.by.displayName // "unknown")\n"
  '
}

cmd_labels() {
  local page_id="${1:-}"
  [[ -n "$page_id" ]] || die "Usage: labels <page-id>"
  local result
  result=$(api_get "/content/${page_id}/label" "")
  echo "$result" | jq -r '.results[] | "\(.prefix):\(.name)"'
}

cmd_history() {
  local page_id="${1:-}"
  [[ -n "$page_id" ]] || die "Usage: history <page-id> [limit]"
  local limit="${2:-10}"
  local result
  result=$(api_get "/content/${page_id}/version" "limit=${limit}")
  echo "$result" | jq -r '
    .results[] |
    "v\(.number) - \(.when) by \(.by.displayName // "unknown")\n  Message: \(.message // "<no message>")\n"
  '
}

cmd_test() {
  echo "Testing connection..."
  local result
  result=$(api_get "/user/current" "")
  local name
  name=$(echo "$result" | jq -r '.displayName // .username // "unknown"')
  echo "Connected as: $name"
  echo ""
  echo "Testing space listing..."
  local spaces
  spaces=$(api_get "/space" "limit=3")
  echo "$spaces" | jq -r '.results[] | "  \(.key) - \(.name)"'
  echo ""
  echo "Connection OK."
}

# --- main ---

require_jq

COMMAND="${1:-help}"

case "$COMMAND" in
  discover)
    shift; cmd_discover "$@" ;;
  help)
    echo "Usage: confluence.sh <host> <command> [args...]"
    echo ""
    echo "Discovery:"
    echo "  discover [substring]              Find Confluence hosts in ~/.netrc"
    echo ""
    echo "Commands (first arg is hostname or unique substring from ~/.netrc):"
    echo "  <host> whoami                     Show current user"
    echo "  <host> test                       Test connection"
    echo "  <host> recent [limit]             Recent content changes"
    echo "  <host> watched                    List watched content"
    echo "  <host> watch-changes [days]       Changes to watched content (default: 7 days)"
    echo "  <host> search <CQL> [limit]       Search via CQL"
    echo "  <host> spaces [limit]             List spaces"
    echo "  <host> space-pages <key> [limit]  Pages in a space"
    echo "  <host> page <id> [format]         Get page content"
    echo "  <host> page-info <id>             Get page metadata"
    echo "  <host> children <id>              List child pages"
    echo "  <host> labels <id>                List page labels"
    echo "  <host> history <id> [limit]       Page version history"
    exit 0
    ;;
  *)
    # Host-scoped command: confluence.sh <host> <command> [args...]
    HOST="$1"
    shift
    SUBCMD="${1:-help}"
    shift || true
    resolve_host "$HOST"
    case "$SUBCMD" in
      whoami)        cmd_whoami ;;
      test)          cmd_test ;;
      recent)        cmd_recent "$@" ;;
      watched)       cmd_watched ;;
      watch-changes) cmd_watch_changes "$@" ;;
      search)        cmd_search "$@" ;;
      spaces)        cmd_spaces "$@" ;;
      space-pages)   cmd_space_pages "$@" ;;
      page)          cmd_page "$@" ;;
      page-info)     cmd_page_info "$@" ;;
      children)      cmd_children "$@" ;;
      labels)        cmd_labels "$@" ;;
      history)       cmd_history "$@" ;;
      help)
        echo "Usage: confluence.sh <host> <command> [args...]"
        echo "Run 'confluence.sh help' for full command list."
        exit 0
        ;;
      *)             die "Unknown command: $SUBCMD (run 'confluence.sh help')" ;;
    esac
    ;;
esac
