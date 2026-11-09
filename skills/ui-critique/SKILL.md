---
name: ui-critique
description: Review existing UI code or pages for design quality and AI-generated "slop" — accessibility floor violations, generic AI defaults (indigo accents, purple-blue gradients, emoji icons, card-grid layouts, gradient text), motion and copy problems — using a deterministic Go lint plus an independent fresh-context design review, then report tiered findings and optionally fix them. Use whenever the user asks to review, critique, audit, roast, grade, or sanity-check a UI, page, component, dashboard, template, stylesheet, or frontend PR; asks "does this look AI-generated / generic / good?"; or wants an accessibility or polish pass. Also invoked by ui-design as its pre-handback gate.
---

# UI Critique

Two independent assessments, merged: a deterministic lint that never gets tired or talked out of a finding, and a design review by a reviewer with fresh context that judges what a regex can't (hierarchy, restraint, whether the layout fits the task). They run separately on purpose — if the reviewer sees the lint output first, it anchors on the lint's list and stops looking.

The rulebook is shared with ui-design: `~/.claude/skills/ui-design/references/craft.md` (Floor / Reflex / Finish tiers). Read it before reviewing.

## 1. Scope and context

Establish, briefly:
- **What**: files, directory, or a diff (for a PR, review the changed UI files plus the token/theme files they depend on).
- **Spec**: is there a `DESIGN.md` / `PRODUCT.md`? Its Do's/Don'ts and tokens are the standard; craft.md fills the gaps.
- **Mode**: Operate (app/dashboard — default) or Persuade (marketing; also read ui-design's `references/marketing.md`).
- **Ask**: review only, or review and fix? Default to review only unless the user asked for fixes or ui-design invoked you as its gate.

## 2. Deterministic lint

```bash
LINT=~/.claude/scripts/ui-slop-lint
[ -x "$LINT/ui-slop-lint" ] || go -C "$LINT" build -o ui-slop-lint .
"$LINT/ui-slop-lint" -format json [-design /abs/path/DESIGN.md] /abs/path/to/target > "${TMPDIR:-/tmp}/ui-lint.json"
```

- Exit 0 = no Floor findings, 1 = Floor findings present, 2 = usage error.
- `"$LINT/ui-slop-lint" -list-rules` prints every rule with its tier.
- Waivers: `ui-slop-lint: allow <rule-id> -- <reason>` in a comment on the same or previous line. Reflex/Finish only; Floor can't be waived inline. A waiver without a reason is itself a finding.
- The lint is static and heuristic. Treat a finding as "look here", confirm it in context, and drop false positives explicitly (say why) rather than silently.

If the lint can't run (no Go toolchain, build error), say so at the top of the report — **"Degraded review: lint not run (reason)"** — and continue with the design review. A review that silently skipped half its checks is worse than one that says so.

## 3. Independent design review

Spawn a subagent with a fresh context (if the Agent tool is available) using the brief in `references/reviewer-brief.md`. Give it: the target paths, DESIGN.md path if any, the mode, and the path to craft.md. **Do not** give it the lint output, and don't tell it what you suspect.

If subagents aren't available, do the design review yourself *before* reading the lint JSON, and note "single-context review" in the report.

If you can render the UI (a running dev server plus a browser/screenshot tool), have the reviewer look at it at 1280px and the narrowest supported width. Otherwise review from code and say so — code-only review misses visual hierarchy problems, so don't overclaim.

## 4. Merge and report

Combine: dedupe overlapping findings (lint and reviewer flagging the same line), confirm or reject each lint finding, keep reviewer findings that cite a specific location. Then report in this format:

```markdown
## UI critique: <target>
<degraded-run banner if any> · Mode: Operate|Persuade · Spec: DESIGN.md|none · Rendered: yes|no

**Verdict**: <one sentence: the single most important thing>

### Scores (1–5)
| Hierarchy | Restraint | Specificity | Execution | Accessibility | State coverage |
|---|---|---|---|---|---|
| n | n | n | n | n | n |

### Floor (must fix)
- `path:line` — <finding> → <concrete fix>

### Reflex (fix or waive with reason)
- `path:line` — <finding> → <fix> *(or: acceptable if <condition>; waive with `allow <id> -- …`)*

### Finish
- …

### What's working
- <1–3 specific things to keep, so a fix pass doesn't destroy them>

### Dropped lint findings
- <rule> at `path:line` — false positive because …
```

Score meanings: **Hierarchy** — one obvious first read and primary action per view. **Restraint** — every color, effect, and animation earns its place. **Specificity** — would this fit any other product unchanged? (low = generic). **Execution** — spacing/type on scale, alignment, tokens used. **Accessibility** — Floor rules and the craft.md checklist. **State coverage** — empty/loading/error/disabled/focus present.

Order findings by impact, not by file. Every finding needs a location and a concrete fix; "consider improving the visual hierarchy" is not a finding.

## 5. Fix loop (only when asked, or as ui-design's gate)

1. Fix all Floor findings.
2. Fix Reflex findings, or add a waiver with a real reason (the brief or DESIGN.md asked for it). "Looks nice" is not a reason.
3. Re-run the lint. Repeat at most 3 rounds; then stop and report what remains and why.
4. Don't redesign while fixing — refinement preserves. If the real problem is structural (wrong layout for the task), say so in the verdict and ask before restructuring.
