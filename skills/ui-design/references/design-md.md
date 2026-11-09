# DESIGN.md

A DESIGN.md is a design-system spec that lives at the project root and is read by agents and humans before building UI. Format follows Google's `google-labs-code/design.md` spec (Apache-2.0, version "alpha" — expect churn), extended with three sections from the best community examples.

**Why bother**: without a persistent spec, every session re-derives the palette from vibes and the product drifts. With one, the next agent builds from the same tokens and the same don'ts.

## Format

Two layers that must never disagree:
1. **YAML tokens** (front matter between `---` lines, and/or fenced ```yaml blocks — they merge; a top-level key may appear only once). Tokens are normative.
2. **Markdown prose** in `##` sections explaining when and why. Prose references tokens as `{colors.accent}` and also gives the hex, so a human skimming can see the value.

### Token schema (top-level keys)

| Key | Contents |
|---|---|
| `version` | `"alpha"` (optional) |
| `name`, `description` | Product/system name, one-line description |
| `omitted` | Sections intentionally left out, ideally with `{section, reason}` — better than silent gaps |
| `colors` | `name: <CSS color>` — hex recommended for tooling; **must define `primary`** |
| `typography` | `name: {fontFamily, fontSize, fontWeight, lineHeight, letterSpacing, fontFeature?, fontVariation?}` |
| `rounded` | `name: <dimension>` (px/em/rem) |
| `spacing` | `name: <dimension or number>` |
| `components` | `name: {backgroundColor, textColor, typography, rounded, padding, size, height, width}` — values may reference tokens: `"{colors.surface}"`. Variants/states are sibling keys: `button-primary`, `button-primary-hover`, `button-primary-disabled` |

Unknown color/typography names and unknown sections are preserved, not errors; unknown component properties warn. Duplicate `##` headings or duplicate top-level YAML keys invalidate the file.

### Sections (in this order; any may be omitted)

1. `## Overview` — who uses this, where, what it should feel like; 3–5 key characteristics
2. `## Colors` — roles, accent scarcity rule, semantic colors, dark-mode approach
3. `## Typography` — families and why, the scale, usage rules (tabular nums, mono usage)
4. `## Layout` — spacing scale, grid, density setting, breakpoints
5. `## Elevation & Depth` — border *or* shadow, tonal layering recipe
6. `## Shapes` — radius scale and where each applies
7. `## Components` — per component, per state, referencing tokens
8. `## Do's and Don'ts` — ~5–8 each, concrete and checkable
9. `## Motion` *(extension)* — curves, duration scale, frequency decisions (see motion.md)
10. `## Responsive Behavior` *(extension)* — breakpoints, touch targets, what collapses, what's unsupported
11. `## Known Gaps` *(extension)* — what hasn't been decided or observed (dark mode? error states? mobile?). This is the most underrated section: it stops the next agent from inventing an answer and presenting it as the system.

## Writing rules

- Name tokens by role (`surface`, `ink-muted`, `accent`, `danger-subtle`), never by hue.
- Typography: 6–12 levels for an app; each fully specified. Give open-source fallbacks for any proprietary face.
- Write the accent scarcity rule explicitly: "Accent only on: primary action, current nav item, focus ring, selected row."
- State the elevation method once ("1px `{colors.border}` borders; shadows only on overlays").
- Spacing and radius as named scales; say "no off-scale values."
- One component entry per state; reference tokens, never repeat raw hex.
- Do's/Don'ts must be checkable by reading code or a screenshot. "Feel premium" is not checkable; "no gradients on surfaces" is. Include don'ts for the AI defaults the product is prone to.
- Keep it under ~25KB. Every agent session pays for reading it.
- If you're documenting an *existing* product, extract values from the code, mark anything inferred, and list what you couldn't observe under Known Gaps.
- Never reproduce another company's identity (their palette, proprietary fonts, distinctive layouts) as a product's design system. Community DESIGN.md collections of real brands are study material for *structure*, not sources of values.

## Skeleton (operations-dashboard example)

```markdown
---
version: alpha
name: Fieldline Console
description: Operations console for monitoring deployed sensor fleets.
colors:
  primary: "#2f6f5e"        # accent: actions, focus, selection only
  on-primary: "#f6faf8"
  surface: "#f7f8f7"
  surface-raised: "#ffffff"
  surface-sunken: "#eef0ee"
  ink: "#1c2421"
  ink-muted: "#55605b"
  border: "#d9ddda"
  ok: "#2e7d4f"
  warn: "#a86a00"
  danger: "#b3261e"
  danger-subtle: "#fbeae8"
typography:
  body-md:   { fontFamily: "IBM Plex Sans", fontSize: 14px, fontWeight: 400, lineHeight: 1.45, letterSpacing: 0em }
  body-sm:   { fontFamily: "IBM Plex Sans", fontSize: 13px, fontWeight: 400, lineHeight: 1.4,  letterSpacing: 0em }
  label-md:  { fontFamily: "IBM Plex Sans", fontSize: 13px, fontWeight: 500, lineHeight: 1.3,  letterSpacing: 0em }
  heading-md:{ fontFamily: "IBM Plex Sans", fontSize: 20px, fontWeight: 600, lineHeight: 1.25, letterSpacing: -0.01em }
  heading-lg:{ fontFamily: "IBM Plex Sans", fontSize: 24px, fontWeight: 600, lineHeight: 1.2,  letterSpacing: -0.015em }
  mono-sm:   { fontFamily: "IBM Plex Mono", fontSize: 13px, fontWeight: 400, lineHeight: 1.4,  letterSpacing: 0em, fontFeature: "'tnum'" }
rounded: { sm: 4px, md: 6px, lg: 10px, full: 9999px }
spacing: { 1: 4px, 2: 8px, 3: 12px, 4: 16px, 6: 24px, 8: 32px, 12: 48px }
components:
  button-primary:       { backgroundColor: "{colors.primary}", textColor: "{colors.on-primary}", typography: "{typography.label-md}", rounded: "{rounded.md}", height: 32px, padding: "0 12px" }
  button-primary-hover: { backgroundColor: "#275d4f" }
  table-row:            { height: 32px, typography: "{typography.body-sm}" }
---

## Overview
Used by on-call operators on laptops and a wall display, often for hours. Calm by default, loud only when something is wrong. Dense (compact rows), light theme primary, dark theme for the wall display.

## Colors
Accent {colors.primary} (#2f6f5e) appears only on the primary action, focused element, current nav item, and selected rows. Status colors are reserved for status...

## Do's and Don'ts
- Do right-align numeric columns with tabular figures.
- Don't use {colors.danger} for anything that isn't a fault.
- Don't animate polling updates.
...

## Known Gaps
- Mobile layouts: not supported; operators use ≥ 1280px.
- Dark theme tokens: not yet defined.
```

## Using it when building

1. Read it fully before writing UI. Its Do's/Don'ts override this skill's Reflex defaults.
2. Export tokens to the stack:
   - **Tailwind v4**: `npx @google/design.md export --format css-tailwind DESIGN.md > tokens.css` (emits `@theme`), or hand-write the `@theme` block. v3: `--format json-tailwind`.
   - **Go / plain HTML**: generate (or hand-write) one `tokens.css` of `:root` custom properties (`--color-surface`, `--text-body-md-size`, `--space-4`, `--radius-md`). Embed it with `//go:embed` and serve it; components reference only `var(--…)`.
   - **DTCG** (`--format dtcg`) when feeding Style Dictionary or a design tool.
3. Validate after editing: `npx @google/design.md lint DESIGN.md` (JSON findings: broken references are errors; contrast < 4.5:1, missing primary, orphaned tokens, section order are warnings). Requires Node; if Node isn't available (air-gapped), check references and contrast by hand and say so.
4. When you change the spec: `npx @google/design.md diff old.md new.md` and summarize the token changes for the user.
5. Pass `-design DESIGN.md` to `ui-slop-lint` so colors outside the palette are flagged.

## Larger products

For a multi-surface product, keep one `DESIGN.md` (the master) and optional per-surface overrides (`design/pages/<surface>.md`) that only state differences. Pair with a short `PRODUCT.md` (who the users are, their environment, their top tasks) — design decisions that cite the user's environment age better than ones that cite taste.
