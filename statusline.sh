#!/bin/bash
# Claude Code statusline focused on context-window management.
# Line 1: identity (model, dir, branch, cost/duration).
# Line 2: context state (bar, %, used/total, cache hit %, actionable hints).

set -euo pipefail

input=$(cat)

jqv() { echo "$input" | jq -r "$1 // empty"; }

DIM=$'\033[2m'
RST=$'\033[0m'
GREEN=$'\033[32m'
CYAN=$'\033[36m'
YELLOW=$'\033[33m'
RED=$'\033[31m'

# ── Line 1: identity ────────────────────────────────────────────
MODEL=$(jqv '.model.display_name')
DIR=$(jqv '.workspace.current_dir')
CWSIZE=$(jqv '.context_window.context_window_size')
EFFORT=$(jqv '.effort.level')
THINKING=$(jqv '.thinking.enabled')
COST_USD=$(jqv '.cost.total_cost_usd')
DUR_MS=$(jqv '.cost.total_duration_ms')

MODEL_LABEL="$MODEL"
[[ "$CWSIZE" == "1000000" ]] && MODEL_LABEL+=" 1M"
[[ -n "$EFFORT" && "$EFFORT" != "medium" ]] && MODEL_LABEL+=":$EFFORT"
[[ "$THINKING" == "true" ]] && MODEL_LABEL+="✻"

BRANCH=""
if [[ -n "$DIR" ]] && git -C "$DIR" rev-parse --git-dir >/dev/null 2>&1; then
  B=$(git -C "$DIR" branch --show-current 2>/dev/null || true)
  [[ -n "$B" ]] && BRANCH=" | 🌿 $B"
fi

COST_STR=""
if [[ -n "$COST_USD" ]]; then
  COST_STR=$(awk -v c="$COST_USD" 'BEGIN { if (c+0 >= 0.01) printf "$%.2f", c }')
fi
DUR_STR=""
if [[ -n "$DUR_MS" && "$DUR_MS" != "0" ]]; then
  SECS=$((DUR_MS / 1000))
  H=$((SECS / 3600)); M=$(( (SECS % 3600) / 60 )); S=$((SECS % 60))
  if   (( H > 0 )); then DUR_STR=$(printf "%dh%02dm" "$H" "$M")
  elif (( M > 0 )); then DUR_STR=$(printf "%dm%02ds" "$M" "$S")
  else                   DUR_STR=$(printf "%ds" "$S")
  fi
fi
SESSION_STR=""
if [[ -n "$COST_STR" || -n "$DUR_STR" ]]; then
  sep=""
  [[ -n "$COST_STR" && -n "$DUR_STR" ]] && sep=" "
  SESSION_STR=" | 💰 ${COST_STR}${sep}${DUR_STR}"
fi

printf "[%s] 📁 %s%s%s%s%s\n" \
  "$MODEL_LABEL" "${DIR##*/}" "$BRANCH" \
  "$DIM" "$SESSION_STR" "$RST"

# ── Line 2: context ─────────────────────────────────────────────
PCT=$(jqv '.context_window.used_percentage')
IN=$(jqv '.context_window.current_usage.input_tokens')
CREATE=$(jqv '.context_window.current_usage.cache_creation_input_tokens')
READ=$(jqv '.context_window.current_usage.cache_read_input_tokens')
EXCEEDS_200K=$(jqv '.exceeds_200k_tokens')

if [[ -z "$PCT" ]]; then
  printf "%s▱▱▱▱▱▱▱▱▱▱  — awaiting first API call%s\n" "$DIM" "$RST"
  exit 0
fi

PCT_INT=$(printf "%.0f" "$PCT")

FILLED=$((PCT_INT / 10))
(( FILLED > 10 )) && FILLED=10
BAR=""
for ((i=0; i<FILLED; i++)); do BAR+="▰"; done
for ((i=FILLED; i<10; i++)); do BAR+="▱"; done

if   (( PCT_INT >= 90 )); then PCT_COLOR=$RED
elif (( PCT_INT >= 75 )); then PCT_COLOR=$YELLOW
elif (( PCT_INT >= 40 )); then PCT_COLOR=$CYAN
else                           PCT_COLOR=$GREEN
fi

human_tokens() {
  local t=$1
  if   (( t >= 1000000 )); then awk -v t="$t" 'BEGIN { printf "%.1fM", t/1000000 }'
  elif (( t >= 1000 ));    then awk -v t="$t" 'BEGIN { printf "%.0fk", t/1000 }'
  else echo "$t"
  fi
}
USED_TOK=$(( ${IN:-0} + ${CREATE:-0} + ${READ:-0} ))
USED_STR=$(human_tokens "$USED_TOK")
TOTAL_STR=$(human_tokens "${CWSIZE:-200000}")

CACHE_STR=""
CACHE_PCT=0
if (( USED_TOK > 0 )); then
  CACHE_PCT=$(( ${READ:-0} * 100 / USED_TOK ))
  CACHE_COLOR=$GREEN
  (( CACHE_PCT < 70 )) && CACHE_COLOR=$CYAN
  (( CACHE_PCT < 40 )) && CACHE_COLOR=$YELLOW
  CACHE_STR=$(printf " %scache %d%%%s" "$CACHE_COLOR" "$CACHE_PCT" "$RST")
fi

# Hints: cold-cache prefix, then threshold-based primary hint.
HINT=""
if (( USED_TOK > 10000 )) && (( CACHE_PCT < 40 )); then
  HINT="❄️  cold cache"
fi
primary=""
if (( PCT_INT >= 90 )); then
  primary="🔴 /compact or /clear now"
elif (( PCT_INT >= 75 )); then
  primary="⚠️  /compact or offload research to Explore"
elif (( PCT_INT >= 60 )); then
  primary="💡 consider /compact soon"
elif [[ "$EXCEEDS_200K" == "true" && "$CWSIZE" != "1000000" ]]; then
  primary="⚠️  over 200k autocompact threshold"
fi
if [[ -n "$primary" ]]; then
  HINT="${HINT:+$HINT  }$primary"
fi

printf "%s  %s%d%%%s  %s/%s%s%s\n" \
  "$BAR" "$PCT_COLOR" "$PCT_INT" "$RST" \
  "$USED_STR" "$TOTAL_STR" "$CACHE_STR" \
  "${HINT:+  $HINT}"
