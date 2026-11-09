---
name: ui-design
description: Build user interfaces that look deliberately designed rather than AI-generated — apps, dashboards, internal tools, admin consoles, and (secondarily) landing pages — across React/Tailwind, Go server-rendered HTML (html/template, templ, htmx), and standalone HTML artifacts. Also writes and consumes DESIGN.md design-system specs. Use this skill whenever the user asks to build, style, restyle, redesign, polish, or "make it look better/less generic/less AI" for any page, component, screen, dashboard, or frontend; when they mention design tokens, a design system, DESIGN.md, Tailwind theme, color palette, typography, spacing, or UI animation; or when generating any non-trivial HTML/CSS/JSX even if design isn't mentioned. For reviewing existing UI without building, use ui-critique.
---

# UI Design

Generated UI fails in a recognizable way: indigo accents, purple-to-blue hero gradients, identical icon-heading-text card grids, emoji icons, gradient text, `transition: all`, invented metrics, and copy that says "seamless." None of these are wrong in isolation; together they read as "nobody decided anything." This skill exists to make decisions explicit — tokens first, rules tiered by how negotiable they are, and an independent critique before handing back.

The defaults here are tuned for **apps and dashboards** (dense, operational, used daily). Marketing pages are a secondary mode with their own reference.

## Rule tiers (used throughout)

- **Floor** — accessibility, performance, honesty. Never waived, not even by the brief. (Contrast, focus visibility, reduced motion, no invented facts, no placeholder content shipped.)
- **Reflex** — the AI-default habits. Avoid by default; waive *on the record* when the brief or DESIGN.md genuinely calls for it, with a one-line reason in a code comment (`/* ui-slop-lint: allow gradient-text -- brand wordmark per DESIGN.md */`). The point isn't that eyebrow labels or glass are evil; it's that they should be chosen, not defaulted to.
- **Finish** — craft polish. Do it when there's room; never blocks.

Precedence: Floor > the user's brief > the project's DESIGN.md > this skill's Reflex/Finish defaults. When the brief asks for something a Reflex rule discourages, do it and record the waiver — don't argue.

## Workflow

### 1. Classify the job (one line, to yourself)
- **Mode**: *Operate* (app, dashboard, tool, admin, form-heavy — the default) or *Persuade* (landing, launch, marketing). For Persuade, also read `references/marketing.md`.
- **Change type**: *refine* (preserve the existing system; change as little as needed) or *redesign* (replace it). If unclear and the codebase has an established look, assume refine. Silent redesigns of someone's product are the most expensive mistake this skill can make.
- **Stack**: React/Tailwind, Go server-rendered (html/template, templ, htmx), or standalone HTML artifact. Read the matching `references/stack-*.md`.

### 2. Read before you design
Look for, in order: `DESIGN.md`, `PRODUCT.md`, existing token sources (Tailwind `@theme`/config, `:root` custom properties, theme files), loaded fonts, an existing component library. If a real brand is involved, get its colors/fonts from the project or the user — never from memory; recalled brand values are usually subtly wrong and look worse than a neutral palette.

If you find nothing and the work is more than a single throwaway page, ask the user one question: who uses this and in what setting (a NOC wall display, a field laptop in sunlight, a dev's second monitor)? Theme (light/dark), density, and contrast follow from the use scene, not from the product category.

### 3. Tokens first
Decide the system before writing components: color roles, type scale, spacing scale, radius, elevation, motion. Name tokens by role (`surface`, `ink-muted`, `accent`), not by hue (`blue-500`), so they survive a palette change.

- Project or multi-page work with no spec → write a `DESIGN.md` (read `references/design-md.md`). It is the persistent memory that keeps the next session — or the next agent — consistent.
- Single artifact or small component → a compact token block at `:root` / `@theme` is enough. Don't impose a DESIGN.md on a one-off.
- Then export tokens to the stack (Tailwind v4 `@theme`, CSS custom properties) and reference only tokens in components. A raw hex past the token block is a smell: it's where drift starts.

Read `references/craft.md` for the numbers (contrast, type scale, spacing, radius, density) and the full Floor/Reflex/Finish lists. This is the core reference; read it for every non-trivial build.

### 4. Build, most important view first
Build the primary view first — for a dashboard, the main data table or chart, not the header and nav. Get it right, look at it cold (re-read the code as if someone else wrote it), then build outward. One component at a time, each referencing tokens.

For every interactive component, cover the states, not just the happy path: default, hover, focus-visible, active, disabled, loading, empty, error. Empty and error states are where generated UI most visibly gives up.

Content: use realistic, domain-shaped content. For dashboards, sample data is fine and necessary — make it plausible (real-looking hostnames, timestamps, units) and keep it obviously fixture data in code (`sampleRows`), never presented as real metrics in copy. For marketing, never invent metrics, testimonials, logos, or customer counts; leave an honest slot.

Motion: read `references/motion.md` before adding any animation. In Operate mode most things should not animate at all.

Copy: read `references/copy.md` when writing any user-facing text longer than a label.

Charts: if the `dataviz` skill is available, use it for any chart; it owns chart color and form.

### 5. Gate: critique before handback
Run the `ui-critique` workflow on what you built (deterministic lint + an independent fresh-context review). Fix every Floor finding. Fix each Reflex finding or waive it with a reason. Re-run the lint after fixes; stop after 3 rounds and report what's left rather than looping.

Why a separate reviewer: the builder rationalizes its own defaults. A fresh context that only sees the output and the spec catches the "looks fine to me" failures.

### 6. Hand back
Brief summary: what was built, the token decisions in one or two lines, waivers taken (rule + reason), known gaps (states not covered, assumptions about brand or users). Don't claim visual quality you haven't verified — say whether you rendered it.

## References
- `references/craft.md` — rule tiers in full, numeric scales, color/type/layout/density for apps. **Read for every build.**
- `references/design-md.md` — DESIGN.md format (Google spec + extensions), how to write one, how to export tokens per stack.
- `references/motion.md` — when to animate (mostly: don't), curves, durations, reduced motion, scroll-driven, htmx swaps.
- `references/copy.md` — UI copy, banned tells with substitutes.
- `references/stack-react-tailwind.md`, `references/stack-go-ssr.md`, `references/stack-html-artifact.md` — stack specifics.
- `references/marketing.md` — Persuade mode: hero, page structure, word budgets.
- `references/catalog.md` — the upstream repos this skill distills, what each is good for, licenses, and when to reach for one directly.
