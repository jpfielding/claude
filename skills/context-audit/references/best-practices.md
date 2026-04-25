# Context Management Best Practices

Reference loaded by the `context-audit` skill to explain *why* each flagged
finding matters and what the recommended remediation is. Keep concise.

## Contents

- [MEMORY.md hygiene](#memorymd-hygiene)
- [CLAUDE.md hygiene](#claudemd-hygiene)
- [Skills hygiene](#skills-hygiene)
- [Live conversation hygiene](#live-conversation-hygiene)
- [Thresholds at a glance](#thresholds-at-a-glance)

## MEMORY.md hygiene

**Truncation at 200 lines.** The auto-memory system prompt states:
`MEMORY.md` is always loaded into context, and lines after 200 are truncated.
Anything past that line is invisible to Claude in every future session.

**Index-only discipline.** `MEMORY.md` is an index, not a memory store.
Each entry should be one line (`- [Title](file.md) — one-line hook`) under
~150 characters. Long lines mean content is leaking out of the individual
memory files and into the always-loaded index.

**Orphaned memory files** (present in `memory/` but not linked from
`MEMORY.md`) are dead weight on disk. Either link them or delete them.

**Dead pointers** (linked from `MEMORY.md` but file missing) waste the
index's precious 200-line budget and confuse future Claude sessions that
try to follow the link.

**Recommended actions:**
- Condense or move content out of long `MEMORY.md` lines into the target file.
- Delete orphan files that are no longer useful; link the ones that are.
- Remove dead pointers immediately.
- If near truncation, audit each entry: consolidate duplicates, drop stale
  project memories (projects end; their memory should too).

## CLAUDE.md hygiene

**Global CLAUDE.md loads every session.** Every byte in the global
`~/.claude/CLAUDE.md` is a tax on every conversation. Keep it lean: personal
preferences, architectural rules that apply everywhere, and non-obvious
conventions. Project-specific rules belong in per-project `CLAUDE.md`.

**Watch for duplication with MEMORY.md.** If a rule is a permanent preference,
CLAUDE.md is correct. If it's a running record of "I learned X about the
user," MEMORY.md is correct. Never both.

**Recommended actions:**
- Move project-specific guidance out of the global file.
- Delete stale rules (preferences change; rules rot).
- Replace verbose prose with the compact imperative form Claude expects.

## Skills hygiene

**SKILL.md body under 500 lines.** Per skill-creator guidance, keep the body
lean. Split variant-specific detail (framework patterns, API specs, large
example sets) into `references/*.md` so they load only when relevant.

**Description is the only trigger.** The frontmatter `description` is the
single signal Claude uses to decide whether to invoke the skill. Short or
vague descriptions (< 60 chars) produce silent failures: the skill exists
but never fires on the prompts it should. Include concrete triggers —
verbs, filenames, slash commands the user might type.

**Reference files over 100 lines need a TOC.** When Claude previews a
reference file, it sees only the first chunk. A table of contents at the
top lets Claude decide whether the file is worth fully loading. Without
one, Claude may load the whole file just to find out it's irrelevant.

**Recommended actions:**
- Split oversized `SKILL.md` bodies by variant, domain, or workflow phase.
- Rewrite thin descriptions to include triggers ("Use when X, Y, or Z").
- Add a `## Contents` section with anchor links to the top of any
  reference file over 100 lines.
- Delete skills that have never triggered and whose purpose is unclear —
  their metadata costs context on every conversation.

## Live conversation hygiene

The audit script cannot see the live conversation. When running
`/context-audit`, ask the user to paste the output of the `/context`
slash command to inspect real token usage.

**Patterns to flag in `/context` output:**
- `System tools` + `Memory files` + `Custom agents` taking a disproportionate
  share — points to bloated agents/ or MEMORY.md.
- `Messages` dominating — suggests compact (`/compact`) or a fresh
  session (`/clear`) before the next unrelated task.
- Large tool result chunks from past turns still resident — prefer
  delegating research to the `Explore` subagent next time so results
  never enter the main context.

**Recommended actions:**
- `/compact` at natural checkpoints.
- `/clear` between unrelated tasks.
- Use the `Explore` subagent for broad codebase searches to keep raw
  results out of the main thread.
- Write findings to a file, then read only what's needed — don't carry
  large tool outputs forward.

## Thresholds at a glance

| Artifact | Soft limit | Hard limit | Why |
|---|---|---|---|
| `MEMORY.md` total lines | 160 | 200 | system truncates past 200 |
| `MEMORY.md` per-line chars | 150 | — | index-only discipline |
| Global `CLAUDE.md` lines | 400 | — | loads every session |
| `SKILL.md` body lines | 500 | 750 | skill-creator guidance |
| Skill `description` chars | 60 | — | trigger reliability |
| Reference file lines | 100 (add TOC) | — | Claude previews only the head |
