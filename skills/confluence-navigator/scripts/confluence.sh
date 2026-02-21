#!/usr/bin/env bash
#
# confluence.sh - Confluence REST API CLI wrapper
#
# Usage: confluence.sh <instance> <command> [args...]
#
# Configuration: ~/.confluence-navigator/instances.json
# Format (netrc auth - credentials read from ~/.netrc):
# {
#   "default": "psdo",
#   "instances": {
#     "psdo": {
#       "base_url": "https://confluence.psdo.lsre.launchpad-leidos.com",
#       "auth_type": "netrc"
#     }
#   }
# }
#
# ~/.netrc entry:
#   machine confluence.psdo.lsre.launchpad-leidos.com
#   login your_username
#   password your_pat_token
#
# Auth types:
#   netrc - Read credentials from ~/.netrc by hostname (Bearer token from password field)
#   pat   - Personal Access Token (Bearer header), stored in config
#   basic - Username:password (Basic auth), stored in config

set -euo pipefail

CONFIG_FILE="${CONFLUENCE_NAV_CONFIG:-$HOME/.confluence-navigator/instances.json}"

# --- helpers ---

die() { echo "ERROR: $*" >&2; exit 1; }

require_jq() {
  command -v jq >/dev/null 2>&1 || die "jq is required but not installed"
}

load_instance() {
  local name="$1"
  [[ -f "$CONFIG_FILE" ]] || die "Config not found: $CONFIG_FILE. Run setup first."

  if [[ "$name" == "default" ]]; then
    name=$(jq -r '.default // empty' "$CONFIG_FILE")
    [[ -n "$name" ]] || die "No default instance configured"
  fi

  BASE_URL=$(jq -r ".instances[\"$name\"].base_url // empty" "$CONFIG_FILE")
  AUTH_TYPE=$(jq -r ".instances[\"$name\"].auth_type // empty" "$CONFIG_FILE")
  TOKEN=$(jq -r ".instances[\"$name\"].token // empty" "$CONFIG_FILE")
  USERNAME=$(jq -r ".instances[\"$name\"].username // empty" "$CONFIG_FILE")
  PASSWORD=$(jq -r ".instances[\"$name\"].password // empty" "$CONFIG_FILE")

  [[ -n "$BASE_URL" ]] || die "Instance '$name' not found in config"
  [[ -n "$AUTH_TYPE" ]] || die "auth_type not set for instance '$name'"

  # Strip trailing slash
  BASE_URL="${BASE_URL%/}"
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
  case "$AUTH_TYPE" in
    netrc)
      local hostname
      hostname=$(extract_hostname "$BASE_URL")
      parse_netrc "$hostname"
      echo "Authorization: Bearer $NETRC_PASSWORD"
      ;;
    netrc-basic)
      local hostname
      hostname=$(extract_hostname "$BASE_URL")
      parse_netrc "$hostname"
      [[ -n "$NETRC_LOGIN" ]] || die "No login found in ~/.netrc for $(extract_hostname "$BASE_URL")"
      echo "Authorization: Basic $(echo -n "$NETRC_LOGIN:$NETRC_PASSWORD" | base64)"
      ;;
    pat)
      [[ -n "$TOKEN" ]] || die "token not set for PAT auth"
      echo "Authorization: Bearer $TOKEN"
      ;;
    basic)
      [[ -n "$USERNAME" && -n "$PASSWORD" ]] || die "username/password not set for basic auth"
      echo "Authorization: Basic $(echo -n "$USERNAME:$PASSWORD" | base64)"
      ;;
    *)
      die "Unknown auth_type: $AUTH_TYPE"
      ;;
  esac
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
    -H "$(auth_header)" \
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

cmd_setup() {
  local name="${1:-}"
  [[ -n "$name" ]] || die "Usage: confluence.sh setup <instance-name> <base-url> <auth-type> [token|username password]"
  local base_url="${2:-}"
  local auth_type="${3:-pat}"

  mkdir -p "$(dirname "$CONFIG_FILE")"

  # Create config if it doesn't exist
  if [[ ! -f "$CONFIG_FILE" ]]; then
    echo '{"default":"","instances":{}}' > "$CONFIG_FILE"
    chmod 600 "$CONFIG_FILE"
  fi

  local tmp
  case "$auth_type" in
    netrc|netrc-basic)
      # Validate that .netrc has an entry for this host
      local hostname
      hostname=$(extract_hostname "$base_url")
      parse_netrc "$hostname"
      echo "Found ~/.netrc entry for $hostname (login: ${NETRC_LOGIN:-<none>})"
      tmp=$(jq --arg n "$name" --arg u "$base_url" --arg a "$auth_type" \
        '.instances[$n] = {"base_url":$u,"auth_type":$a}' "$CONFIG_FILE")
      ;;
    pat)
      local token="${4:-}"
      [[ -n "$token" ]] || die "PAT auth requires a token argument"
      tmp=$(jq --arg n "$name" --arg u "$base_url" --arg t "$token" \
        '.instances[$n] = {"base_url":$u,"auth_type":"pat","token":$t}' "$CONFIG_FILE")
      ;;
    basic)
      local user="${4:-}" pass="${5:-}"
      [[ -n "$user" && -n "$pass" ]] || die "Basic auth requires username and password"
      tmp=$(jq --arg n "$name" --arg u "$base_url" --arg user "$user" --arg pass "$pass" \
        '.instances[$n] = {"base_url":$u,"auth_type":"basic","username":$user,"password":$pass}' "$CONFIG_FILE")
      ;;
    *)
      die "Unknown auth_type: $auth_type (use 'netrc', 'netrc-basic', 'pat', or 'basic')"
      ;;
  esac

  # Set as default if first instance
  local count
  count=$(echo "$tmp" | jq '.instances | length')
  if [[ "$count" -eq 1 ]]; then
    tmp=$(echo "$tmp" | jq --arg n "$name" '.default = $n')
  fi

  echo "$tmp" > "$CONFIG_FILE"
  echo "Instance '$name' configured. Config: $CONFIG_FILE"
}

cmd_set_default() {
  local name="${1:-}"
  [[ -n "$name" ]] || die "Usage: confluence.sh <instance> set-default"
  [[ -f "$CONFIG_FILE" ]] || die "Config not found"
  local tmp
  tmp=$(jq --arg n "$name" '.default = $n' "$CONFIG_FILE")
  echo "$tmp" > "$CONFIG_FILE"
  echo "Default instance set to '$name'"
}

cmd_list_instances() {
  [[ -f "$CONFIG_FILE" ]] || die "Config not found"
  local default_inst
  default_inst=$(jq -r '.default // ""' "$CONFIG_FILE")
  echo "Configured instances:"
  jq -r '.instances | to_entries[] | "  \(.key): \(.value.base_url) [\(.value.auth_type)]"' "$CONFIG_FILE"
  echo "Default: ${default_inst:-<none>}"
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
  setup)
    shift; cmd_setup "$@" ;;
  list-instances)
    cmd_list_instances ;;
  help)
    echo "Usage: confluence.sh <command> [args...]"
    echo "       confluence.sh <instance> <command> [args...]"
    echo ""
    echo "Setup commands:"
    echo "  setup <name> <base-url> <auth-type> [token|user pass]  Configure an instance"
    echo "  list-instances                                          List configured instances"
    echo "  <instance> set-default                                  Set default instance"
    echo ""
    echo "Query commands (use instance name or 'default'):"
    echo "  <inst> whoami                     Show current user"
    echo "  <inst> test                       Test connection"
    echo "  <inst> recent [limit]             Recent content changes"
    echo "  <inst> watched                    List watched content"
    echo "  <inst> watch-changes [days]       Changes to watched content (default: 7 days)"
    echo "  <inst> search <CQL> [limit]       Search via CQL"
    echo "  <inst> spaces [limit]             List spaces"
    echo "  <inst> space-pages <key> [limit]  Pages in a space"
    echo "  <inst> page <id> [format]         Get page content"
    echo "  <inst> page-info <id>             Get page metadata"
    echo "  <inst> children <id>              List child pages"
    echo "  <inst> labels <id>                List page labels"
    echo "  <inst> history <id> [limit]       Page version history"
    exit 0
    ;;
  *)
    # Instance-scoped command: confluence.sh <instance> <command> [args...]
    INSTANCE="$1"
    shift
    SUBCMD="${1:-help}"
    shift || true
    load_instance "$INSTANCE"
    case "$SUBCMD" in
      set-default)  cmd_set_default "$INSTANCE" ;;
      whoami)       cmd_whoami ;;
      test)         cmd_test ;;
      recent)       cmd_recent "$@" ;;
      watched)      cmd_watched ;;
      watch-changes) cmd_watch_changes "$@" ;;
      search)       cmd_search "$@" ;;
      spaces)       cmd_spaces "$@" ;;
      space-pages)  cmd_space_pages "$@" ;;
      page)         cmd_page "$@" ;;
      page-info)    cmd_page_info "$@" ;;
      children)     cmd_children "$@" ;;
      labels)       cmd_labels "$@" ;;
      history)      cmd_history "$@" ;;
      *)            die "Unknown command: $SUBCMD (run 'confluence.sh help')" ;;
    esac
    ;;
esac
