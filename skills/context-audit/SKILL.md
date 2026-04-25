---
name: context-audit
description: Audit the user's ~/.claude/ setup for context-management hygiene and report prunable bloat. Use when the user invokes /context-audit, asks to "audit context", "review my context", "clean up memory/skills", "prune CLAUDE.md", or asks how to shrink their context footprint. Inspects MEMORY.md truncation risk, dead/orphan memory pointers, oversized CLAUDE.md, bloated SKILL.md bodies, and reference files without a TOC. Read-only — surfaces findings with prioritized recommendations; the user decides what to prune.
---

# Context Audit

Walks `~/.claude/` read-only and produces a prioritized report of context
bloat. Complements (does not replace) the `/context` slash command — this
skill audits the static artifacts that load every session; `/context`
shows the live token breakdown of the current conversation.

## Workflow

1. Run the Go audit binary against `~/.claude/`:

   ```bash
   cd ~/.claude/scripts/context-audit && go run .
   ```

   The tool prints an inventory (file counts, sizes) followed by findings
   grouped by severity (HIGH / MED / LOW).

2. Ask the user to paste the output of `/context` if they haven't already.
   The audit script cannot see the live conversation; `/context` reveals
   what's actually consuming tokens right now (system tools, memory files,
   custom agents, messages).

3. Cross-reference findings against `references/best-practices.md` — it
   explains the *why* behind each threshold and the recommended remedy
   for each category (MEMORY, CLAUDE_MD, SKILLS).

4. Present a prioritized recommendation list. Report only — do not
   modify files. Structure:

   - **HIGH** findings first, with the concrete action (e.g., "delete
     these 3 dead pointers from MEMORY.md", "split visual-explainer's
     css-patterns.md into variant files").
   - **MED** next, grouped by category.
   - **LOW** as a tail, often just "add a TOC to these reference files".
   - Live-conversation observations last (from `/context` output if
     provided).

5. Let the user pick what to act on. If they ask for edits, make them
   in a follow-up turn — this skill is diagnostic only.

## What the audit checks

| Category | Check |
|---|---|
| MEMORY | MEMORY.md line count vs. 200-line truncation limit |
| MEMORY | MEMORY.md lines exceeding 150 chars (index-only discipline) |
| MEMORY | Orphan files in `memory/` not linked from MEMORY.md |
| MEMORY | Dead pointers in MEMORY.md whose target file is missing |
| CLAUDE_MD | Global CLAUDE.md exceeding 400 lines |
| SKILLS | SKILL.md body > 500 (MED) or > 750 (HIGH) lines |
| SKILLS | Skill description < 60 chars (weak trigger signal) |
| SKILLS | Reference files > 100 lines without a visible TOC |
| AGENTS | Count of installed agents (inventory only) |
| SETTINGS | Line counts for settings.json / settings.local.json |

## Script location

`~/.claude/scripts/context-audit/` — stdlib-only Go module. Build with
`go build ./...` or run directly with `go run .`. Flag `--root DIR`
overrides the default `~/.claude` root (useful when auditing a
per-project `.claude/` directory).

## When to re-run

Occasional — not continuous. Good trigger points:
- After a burst of skill creation or memory additions
- When MEMORY.md starts feeling full
- Before publishing a skill with `package_skill.py`
- When the user notices `/context` usage climbing
