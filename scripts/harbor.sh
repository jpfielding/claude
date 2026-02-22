#!/usr/bin/env bash
#
# harbor.sh - Harbor Registry REST API CLI wrapper
#
# Usage: harbor.sh <host> <command> [args...]
#
# Credentials from ~/.netrc

set -euo pipefail

# --- helpers ---

die() { echo "ERROR: $*" >&2; exit 1; }

require_jq() {
  command -v jq >/dev/null 2>&1 || die "jq is required but not installed"
}

parse_netrc() {
  local machine="$1"
  local netrc_file="${HOME}/.netrc"
  [[ -f "$netrc_file" ]] || die "~/.netrc not found. Create it with:\n  machine $machine\n  login <username>\n  password <cli_secret>"

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

# URL-encode a string (for repo names with slashes)
urlencode() {
  python3 -c "import urllib.parse,sys; print(urllib.parse.quote(sys.argv[1], safe=''))" "$1"
}

resolve_host() {
  local input="$1"
  local netrc_file="${HOME}/.netrc"
  [[ -f "$netrc_file" ]] || die "~/.netrc not found"

  if [[ "$input" == *"."* ]]; then
    # Contains a dot: use as-is
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
        if [[ "${1:-}" == *"$input"* && "${1:-}" == *"harbor"* ]]; then
          machines+=("${1:-}")
        fi
        ;;
    esac
    shift
  done

  if [[ ${#machines[@]} -eq 0 ]]; then
    die "No ~/.netrc machine entries matching '$input' (filtered for harbor hosts)"
  fi

  if [[ ${#machines[@]} -gt 1 ]]; then
    echo "Multiple ~/.netrc entries match '$input':" >&2
    for m in "${machines[@]}"; do
      echo "  $m" >&2
    done
    die "Ambiguous host substring '$input'. Use a more specific substring or full hostname."
  fi

  HOSTNAME="${machines[0]}"
  BASE_URL="https://${HOSTNAME}"
  parse_netrc "$HOSTNAME"
}

# --- API ---

api_get() {
  local endpoint="$1"
  shift
  local url="${BASE_URL}/api/v2.0${endpoint}"

  if [[ $# -gt 0 && -n "${1:-}" ]]; then
    local params="$1"
    url="${url}?${params}"
  fi

  local response
  response=$(curl -sS -w "\n%{http_code}" \
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

# Retrieve total count from X-Total-Count header
api_get_with_count() {
  local endpoint="$1"
  shift
  local url="${BASE_URL}/api/v2.0${endpoint}"

  if [[ $# -gt 0 && -n "${1:-}" ]]; then
    local params="$1"
    url="${url}?${params}"
  fi

  local header_file
  header_file=$(mktemp)

  local response
  response=$(curl -sS -w "\n%{http_code}" \
    -D "$header_file" \
    -H "Accept: application/json" \
    "$url" 2>&1) || { rm -f "$header_file"; die "curl failed: $response"; }

  local http_code body total_count
  http_code=$(echo "$response" | tail -1)
  body=$(echo "$response" | sed '$d')
  total_count=$(grep -i 'x-total-count' "$header_file" | tr -d '\r' | awk '{print $2}' || echo "")
  rm -f "$header_file"

  if [[ "$http_code" -ge 400 ]]; then
    die "API returned HTTP $http_code: $body"
  fi

  if [[ -n "$total_count" ]]; then
    echo "TOTAL:${total_count}"
  fi
  echo "$body"
}

# --- commands ---

cmd_discover() {
  # Scan ~/.netrc for machine entries that look like Harbor hosts
  local filter="${1:-harbor}"
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
    echo "To search with a different substring: harbor.sh discover <substring>"
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
  echo "Usage: harbor.sh <hostname-or-substring> <command>"
}

cmd_whoami() {
  local result
  result=$(api_get "/users/current" "")
  echo "$result" | jq '{
    username: .username,
    realname: .realname,
    email: .email,
    admin: .sysadmin_flag,
    user_id: .user_id
  }'
}

cmd_test() {
  echo "Testing connection to ${BASE_URL}..."
  echo ""

  # System info (always available unauthenticated)
  local sysinfo
  sysinfo=$(api_get "/systeminfo" "" 2>/dev/null || echo '{}')
  local version auth_mode
  version=$(echo "$sysinfo" | jq -r '.harbor_version // "unknown"')
  auth_mode=$(echo "$sysinfo" | jq -r '.auth_mode // "unknown"')
  echo "Harbor version: $version (auth_mode: $auth_mode)"

  echo ""
  echo "Testing project listing..."
  local projects
  projects=$(api_get "/projects" "page_size=5")
  echo "$projects" | jq -r '.[] | "  \(.name) (repos: \(.repo_count // 0))"'
  echo ""

  local health
  health=$(api_get "/health" "" 2>/dev/null || echo '{"status":"unknown"}')
  local status
  status=$(echo "$health" | jq -r '.status // "unknown"')
  echo "System health: $status"
  echo "Connection OK."
}

cmd_system_info() {
  local result
  result=$(api_get "/systeminfo" "")
  echo "$result" | jq '{
    harbor_version: .harbor_version,
    registry_url: .registry_url,
    with_notary: .with_notary,
    auth_mode: .auth_mode,
    project_creation_restriction: .project_creation_restriction,
    self_registration: .self_registration,
    has_ca_root: .has_ca_root,
    registry_storage_provider: (.storage_provider_name // "unknown")
  }'
}

cmd_health() {
  local result
  result=$(api_get "/health" "")
  echo "$result" | jq -r '
    "Overall: \(.status)\n\nComponents:" +
    (.components | map("  \(.name): \(.status)") | join("\n"))
  '
}

cmd_projects() {
  local limit="${1:-25}"
  local result
  result=$(api_get "/projects" "page_size=${limit}")
  echo "$result" | jq -r '
    .[] |
    "\(.name)\n  Repos: \(.repo_count // 0)  Public: \(.metadata.public // "false")  Created: \(.creation_time)\n"
  '
}

cmd_project_info() {
  local project_name="${1:-}"
  [[ -n "$project_name" ]] || die "Usage: project-info <project-name>"
  local result
  result=$(api_get "/projects/${project_name}" "")
  echo "$result" | jq '{
    name: .name,
    project_id: .project_id,
    public: (.metadata.public // "false"),
    repo_count: (.repo_count // 0),
    owner: (.owner_name // "unknown"),
    creation_time: .creation_time,
    update_time: .update_time,
    auto_scan: (.metadata.auto_scan // "unknown"),
    severity: (.metadata.severity // "unknown"),
    reuse_sys_cve_allowlist: (.metadata.reuse_sys_cve_allowlist // "unknown")
  }'
}

cmd_repos() {
  local project_name="${1:-}"
  [[ -n "$project_name" ]] || die "Usage: repos <project-name> [limit]"
  local limit="${2:-25}"
  local raw
  raw=$(api_get_with_count "/projects/${project_name}/repositories" "page_size=${limit}")
  local total
  total=$(echo "$raw" | head -1 | grep -oP 'TOTAL:\K.*' || echo "")
  local body
  body=$(echo "$raw" | grep -v '^TOTAL:')
  if [[ -n "$total" ]]; then
    echo "Repositories in ${project_name} (${total} total):"
  else
    echo "Repositories in ${project_name}:"
  fi
  echo ""
  echo "$body" | jq -r '
    .[] |
    "\(.name)\n  Artifacts: \(.artifact_count // 0)  Pulls: \(.pull_count // 0)  Updated: \(.update_time)\n"
  '
}

cmd_artifacts() {
  local repo_ref="${1:-}"
  [[ -n "$repo_ref" ]] || die "Usage: artifacts <project/repo-name> [limit]"
  local limit="${2:-25}"

  # Split project/repo
  local project_name repo_name
  project_name="${repo_ref%%/*}"
  repo_name="${repo_ref#*/}"
  [[ "$project_name" != "$repo_name" ]] || die "Provide full path: <project>/<repo-name>"

  local encoded_repo
  encoded_repo=$(urlencode "$repo_name")

  local result
  result=$(api_get "/projects/${project_name}/repositories/${encoded_repo}/artifacts" "page_size=${limit}&with_tag=true&with_scan_overview=true&with_label=true")
  echo "$result" | jq -r '
    .[] |
    "Digest: \(.digest[0:19])...\n  Tags: \([ .tags[]?.name ] | join(", ") | if . == "" then "<none>" else . end)\n  Type: \(.type // "unknown")  Size: \((.size // 0) / 1048576 | floor)MB  Pushed: \(.push_time)\n  Scan: \(.scan_overview // {} | to_entries | map("\(.key): \(.value.summary.summary // {})") | join(", ") | if . == "" then "not scanned" else . end)\n"
  '
}

cmd_tags() {
  local repo_ref="${1:-}"
  [[ -n "$repo_ref" ]] || die "Usage: tags <project/repo-name> [digest-or-tag]"
  local reference="${2:-}"

  local project_name repo_name
  project_name="${repo_ref%%/*}"
  repo_name="${repo_ref#*/}"
  [[ "$project_name" != "$repo_name" ]] || die "Provide full path: <project>/<repo-name>"

  local encoded_repo
  encoded_repo=$(urlencode "$repo_name")

  if [[ -n "$reference" ]]; then
    # Tags for a specific artifact
    local result
    result=$(api_get "/projects/${project_name}/repositories/${encoded_repo}/artifacts/${reference}/tags" "")
    echo "$result" | jq -r '.[] | "\(.name)  Created: \(.push_time)"'
  else
    # All artifacts with their tags
    local result
    result=$(api_get "/projects/${project_name}/repositories/${encoded_repo}/artifacts" "page_size=50&with_tag=true")
    echo "$result" | jq -r '
      .[] |
      select(.tags != null and (.tags | length) > 0) |
      .tags[] |
      "\(.name)  Pushed: \(.push_time)"
    '
  fi
}

cmd_vulns() {
  local repo_ref="${1:-}"
  [[ -n "$repo_ref" ]] || die "Usage: vulns <project/repo-name> <tag-or-digest>"
  local reference="${2:-}"
  [[ -n "$reference" ]] || die "Usage: vulns <project/repo-name> <tag-or-digest>"

  local project_name repo_name
  project_name="${repo_ref%%/*}"
  repo_name="${repo_ref#*/}"
  [[ "$project_name" != "$repo_name" ]] || die "Provide full path: <project>/<repo-name>"

  local encoded_repo
  encoded_repo=$(urlencode "$repo_name")

  local result
  result=$(api_get "/projects/${project_name}/repositories/${encoded_repo}/artifacts/${reference}/additions/vulnerabilities" "")

  # Harbor returns a map of scanner -> report
  echo "$result" | jq -r '
    to_entries[] |
    "Scanner: \(.key)\nGenerated: \(.value.generated_at // "unknown")\nSeverity: \(.value.severity // "unknown")\n\nSummary:" +
    (.value.summary // {} | to_entries | map("  \(.key): \(.value)") | join("\n")) +
    "\n\nVulnerabilities (\(.value.vulnerabilities // [] | length) total):\n" +
    (.value.vulnerabilities // [] | sort_by(.severity) |
      map("  [\(.severity)] \(.id) - \(.package):\(.version)\n    Fixed: \(.fix_version // "no fix")  Link: \(.links // [] | .[0] // "N/A")") |
      join("\n"))
  '
}

cmd_scan() {
  local repo_ref="${1:-}"
  [[ -n "$repo_ref" ]] || die "Usage: scan <project/repo-name> <tag-or-digest>"
  local reference="${2:-}"
  [[ -n "$reference" ]] || die "Usage: scan <project/repo-name> <tag-or-digest>"

  local project_name repo_name
  project_name="${repo_ref%%/*}"
  repo_name="${repo_ref#*/}"
  [[ "$project_name" != "$repo_name" ]] || die "Provide full path: <project>/<repo-name>"

  local encoded_repo
  encoded_repo=$(urlencode "$repo_name")

  die "scan command requires authentication which is not supported in unauthenticated mode"
}

cmd_labels() {
  local scope="${1:-g}"
  local result
  result=$(api_get "/labels" "scope=${scope}&page_size=50")
  echo "$result" | jq -r '
    .[] |
    "[\(.id)] \(.name)\n  Scope: \(.scope)  Color: \(.color // "none")  Description: \(.description // "none")\n"
  '
}

cmd_search() {
  local query="${1:-}"
  [[ -n "$query" ]] || die "Usage: search <query>"
  local result
  result=$(api_get "/search" "q=$(urlencode "$query")")
  echo "Projects:"
  echo "$result" | jq -r '
    .project // [] | .[] |
    "  \(.name) (repos: \(.repo_count // 0), public: \(.metadata.public // "false"))"
  '
  echo ""
  echo "Repositories:"
  echo "$result" | jq -r '
    .repository // [] | .[] |
    "  \(.repository_name)  Project: \(.project_name)  Pulls: \(.pull_count // 0)"
  '
}

cmd_recent_pushes() {
  local project_name="${1:-}"
  local limit="${2:-20}"

  if [[ -n "$project_name" ]]; then
    # Recent pushes within a project
    local result
    result=$(api_get "/projects/${project_name}/repositories" "page_size=${limit}&sort=-update_time")
    echo "Recently updated repositories in ${project_name}:"
    echo ""
    echo "$result" | jq -r '
      .[] |
      "\(.name)\n  Artifacts: \(.artifact_count // 0)  Pulls: \(.pull_count // 0)  Updated: \(.update_time)\n"
    '
  else
    # Search all projects for recently updated repos
    local projects
    projects=$(api_get "/projects" "page_size=100")
    echo "Recently updated repositories across all projects:"
    echo ""
    local project_names
    project_names=$(echo "$projects" | jq -r '.[].name')
    local count=0
    while IFS= read -r pname; do
      [[ -n "$pname" ]] || continue
      local repos
      repos=$(api_get "/projects/${pname}/repositories" "page_size=5&sort=-update_time" 2>/dev/null) || continue
      local repo_count
      repo_count=$(echo "$repos" | jq 'length')
      if [[ "$repo_count" -gt 0 ]]; then
        echo "$repos" | jq -r '
          .[] |
          "\(.name)\n  Artifacts: \(.artifact_count // 0)  Pulls: \(.pull_count // 0)  Updated: \(.update_time)\n"
        '
        count=$((count + repo_count))
      fi
      [[ $count -lt $limit ]] || break
    done <<< "$project_names"
  fi
}

cmd_replication_policies() {
  local result
  result=$(api_get "/replication/policies" "page_size=25")
  echo "$result" | jq -r '
    .[] |
    "[\(.id)] \(.name)\n  Enabled: \(.enabled)  Trigger: \(.trigger.type // "manual")\n  Src: \(.src_registry.name // "local") -> Dest: \(.dest_registry.name // "local")\n  Filters: \([.filters[]? | "\(.type)=\(.value)"] | join(", ") | if . == "" then "none" else . end)\n"
  '
}

cmd_replication_executions() {
  local policy_id="${1:-}"
  local limit="${2:-10}"
  local params="page_size=${limit}&sort=-start_time"
  if [[ -n "$policy_id" ]]; then
    params="${params}&policy_id=${policy_id}"
  fi
  local result
  result=$(api_get "/replication/executions" "$params")
  echo "$result" | jq -r '
    .[] |
    "[\(.id)] Policy: \(.policy_id)  Status: \(.status)  Trigger: \(.trigger // "unknown")\n  Started: \(.start_time)  Ended: \(.end_time // "running")\n  Success: \(.succeed // 0)  Failed: \(.failed // 0)  In-progress: \(.in_progress // 0)\n"
  '
}

cmd_registries() {
  local result
  result=$(api_get "/registries" "page_size=25")
  echo "$result" | jq -r '
    .[] |
    "[\(.id)] \(.name)\n  URL: \(.url)  Type: \(.type)  Status: \(.status // "unknown")\n"
  '
}

cmd_gc() {
  local result
  result=$(api_get "/system/gc/schedule" "")
  echo "GC Schedule:"
  echo "$result" | jq '.'
  echo ""
  echo "Recent GC runs:"
  local history
  history=$(api_get "/system/gc" "page_size=5&sort=-creation_time")
  echo "$history" | jq -r '
    .[] |
    "[\(.id)] Status: \(.job_status)  Created: \(.creation_time)\n  Updated: \(.update_time)  Deleted: \(.delete // false)\n"
  '
}

cmd_quotas() {
  local result
  result=$(api_get "/quotas" "page_size=50")
  echo "$result" | jq -r '
    .[] |
    "[\(.id)] Ref: \(.ref.name // .ref.id // "unknown")\n  Storage used: \((.used.storage // 0) / 1048576 | floor)MB / \(if (.hard.storage // -1) < 0 then "unlimited" else "\((.hard.storage // 0) / 1048576 | floor)MB" end)\n"
  '
}

cmd_robot_accounts() {
  local result
  result=$(api_get "/robots" "page_size=50")
  echo "$result" | jq -r '
    .[] |
    "[\(.id)] \(.name)\n  Level: \(.level)  Disabled: \(.disable)  Expires: \(.expires_at // -1 | if . < 0 then "never" else todate end)\n  Created: \(.creation_time)\n"
  '
}

cmd_audit_log() {
  local limit="${1:-25}"
  local result
  result=$(api_get "/audit-logs" "page_size=${limit}&sort=-op_time")
  echo "$result" | jq -r '
    .[] |
    "\(.op_time) [\(.operation)] \(.resource_type): \(.resource)\n  User: \(.username)\n"
  '
}

# --- main ---

require_jq

COMMAND="${1:-help}"

case "$COMMAND" in
  discover)
    shift; cmd_discover "$@" ;;
  help)
    echo "Usage: harbor.sh <host> <command> [args...]"
    echo ""
    echo "Global commands:"
    echo "  discover [substring]                                 Find Harbor hosts in ~/.netrc"
    echo "  help                                                 Show this help"
    echo ""
    echo "Query commands (<host> is a hostname or ~/.netrc substring):"
    echo "  <host> whoami                                Current user"
    echo "  <host> test                                  Test connection"
    echo "  <host> system-info                           Harbor version and config"
    echo "  <host> health                                Component health status"
    echo ""
    echo "  <host> projects [limit]                      List projects"
    echo "  <host> project-info <name>                   Project details"
    echo "  <host> repos <project> [limit]               Repositories in a project"
    echo "  <host> artifacts <project/repo> [limit]      Artifacts (images) in a repo"
    echo "  <host> tags <project/repo> [ref]             Tags on repo or specific artifact"
    echo "  <host> search <query>                        Search projects and repos"
    echo "  <host> recent-pushes [project] [limit]       Recently pushed repos"
    echo ""
    echo "  <host> vulns <project/repo> <tag|digest>     Vulnerability report"
    echo "  <host> scan <project/repo> <tag|digest>      Trigger vulnerability scan"
    echo ""
    echo "  <host> labels [scope: g|p]                   List labels (g=global, p=project)"
    echo "  <host> replication-policies                  Replication policies"
    echo "  <host> replication-runs [policy-id] [limit]  Replication execution history"
    echo "  <host> registries                            Connected registries"
    echo "  <host> gc                                    Garbage collection schedule/history"
    echo "  <host> quotas                                Storage quotas"
    echo "  <host> robot-accounts                        Robot accounts"
    echo "  <host> audit-log [limit]                     Recent audit log entries"
    exit 0
    ;;
  *)
    HOST="$1"
    shift
    SUBCMD="${1:-help}"
    shift || true
    resolve_host "$HOST"
    case "$SUBCMD" in
      whoami)                 cmd_whoami ;;
      test)                   cmd_test ;;
      system-info)            cmd_system_info ;;
      health)                 cmd_health ;;
      projects)               cmd_projects "$@" ;;
      project-info)           cmd_project_info "$@" ;;
      repos)                  cmd_repos "$@" ;;
      artifacts)              cmd_artifacts "$@" ;;
      tags)                   cmd_tags "$@" ;;
      search)                 cmd_search "$@" ;;
      recent-pushes)          cmd_recent_pushes "$@" ;;
      vulns)                  cmd_vulns "$@" ;;
      scan)                   cmd_scan "$@" ;;
      labels)                 cmd_labels "$@" ;;
      replication-policies)   cmd_replication_policies ;;
      replication-runs)       cmd_replication_executions "$@" ;;
      registries)             cmd_registries ;;
      gc)                     cmd_gc ;;
      quotas)                 cmd_quotas ;;
      robot-accounts)         cmd_robot_accounts ;;
      audit-log)              cmd_audit_log "$@" ;;
      help)
        echo "Usage: harbor.sh <host> <command> [args...]"
        echo "Run 'harbor.sh help' for full command list."
        exit 0
        ;;
      *)                      die "Unknown command: $SUBCMD (run 'harbor.sh help')" ;;
    esac
    ;;
esac
