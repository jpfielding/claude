# Motion

Distilled mainly from Emil Kowalski's skills (animations.dev author), with GSAP and native CSS scroll-driven notes. Motion in an app is a cost the user pays on every interaction; it has to buy something.

## Should this animate at all?

**Frequency rule** — decide by how often a user sees it:

| Seen | Examples | Motion |
|---|---|---|
| 100+ times/day, or keyboard-initiated | command palette, shortcuts, tab switching, typing, table sort | **None.** Instant. |
| Tens of times/day | hover, list navigation, row expand, dropdowns | Minimal (≤ 150ms) or none |
| Occasional | modal, drawer, toast, popover | Standard (table below) |
| Rare | onboarding, first-run, a success moment | Room for one authored moment |

**Purpose test** — every animation must do one of: preserve spatial continuity (where did this come from / go), show a state change, give feedback to an action, explain something, or soften an otherwise jarring change. "It looks nice" is not on the list. In Operate mode, if you can't name the purpose, delete it.

No scroll-reveal entrances on app screens. On marketing pages, one authored moment beats an identical fade on every section.

## Curves

Built-in `ease`/`ease-out` are weak. Define these as tokens:

```css
:root {
  --ease-out:    cubic-bezier(0.23, 1, 0.32, 1);    /* enter & exit */
  --ease-in-out: cubic-bezier(0.77, 0, 0.175, 1);   /* moving something already on screen */
  --ease-drawer: cubic-bezier(0.32, 0.72, 0, 1);    /* sheets/drawers (iOS-like) */
}
```

- Enter and exit: ease-out. On-screen movement: ease-in-out. Hover/color: plain `ease`. Constant motion (spinners, progress): `linear`.
- **Never ease-in for UI** — it starts slow, so the interface feels laggy exactly when the user is waiting for a response.

## Durations

| Interaction | Duration |
|---|---|
| Press feedback | 100–160ms |
| Tooltip | 125–200ms (delay first open ~400ms; adjacent tooltips open at 0ms) |
| Dropdown / popover | 150–250ms |
| Modal / drawer | 200–500ms (drawers longer, modals shorter) |
| Stagger per item | 30–80ms, total stagger ≤ 300ms, never blocks interaction |

Keep UI transitions under ~300ms. Exits run faster than enters (≈ 0.75×) and mirror the enter path.

## Mechanics

- Animate `transform` and `opacity` only. Never `transition: all` — list properties: `transition: transform 160ms var(--ease-out), opacity 160ms var(--ease-out);`
- Enter from `scale(0.95)` + `opacity: 0`, never `scale(0)` — nothing in the physical world appears from a point.
- Popovers/dropdowns scale from their trigger (`transform-origin` at the trigger edge). Modals stay centered.
- Press: `:active { transform: scale(0.97); }` on buttons.
- **Interruptible**: for anything that can be toggled rapidly (hover, open/close), use CSS transitions, not keyframe animations — transitions reverse from where they are; keyframes jump.
- Enter animations without JS: `@starting-style` (Baseline 2024) + `transition-behavior: allow-discrete` for `display`/`overlay` on popovers and dialogs.
- Springs (Motion/Framer) for gesture-driven UI: `{ type: "spring", duration: 0.5, bounce: 0.2 }`; keep bounce 0.1–0.3. No bounce on routine UI.
- Gestures: dismiss on release velocity > ~0.11 px/ms regardless of distance; add rubber-band friction past bounds; use pointer capture.
- Blur ≤ 20px if used at all; `filter: blur(2px)` can mask a crossfade seam.
- `will-change` only to fix a measured issue (e.g. a 1px shift), never preemptively.
- Hover effects gated: `@media (hover: hover) and (pointer: fine) { … }` — otherwise touch devices get sticky hover states.

## Reduced motion

Reduced means **gentler, not necessarily zero**: drop movement (translate, scale, parallax), keep opacity and color changes that communicate state. Ship it with the animation, in the same change.

```css
@media (prefers-reduced-motion: reduce) {
  *, *::before, *::after {
    animation-duration: 0.01ms !important;
    animation-iteration-count: 1 !important;
    scroll-behavior: auto !important;
  }
  .popover { transition: opacity 120ms ease; transform: none; }
}
```

Tailwind: `motion-safe:` / `motion-reduce:` variants. Motion (React): `useReducedMotion()`. GSAP: `gsap.matchMedia()` with `(prefers-reduced-motion: reduce)`.

## Scroll-driven (marketing pages only)

Prefer native CSS where supported — no JS, runs off the main thread:

```css
@supports (animation-timeline: view()) {
  .reveal { animation: reveal linear both; animation-timeline: view(); animation-range: entry 0% cover 30%; }
}
@keyframes reveal { from { opacity: 0; transform: translateY(12px); } }
```

Feature-gate with `@supports`; content must be fully visible without it. Use GSAP ScrollTrigger only when the project already uses GSAP or needs pinning/scrubbed timelines; then: `scrub` with `ease: "none"`, never combine `scrub` with `toggleActions`, animate children of pinned elements, call `ScrollTrigger.refresh()` after fonts/images load, clean up with `useGSAP`/`gsap.context().revert()`.

## Stack notes

- **htmx**: style swaps via the classes htmx adds — `.htmx-added` (new content, before settle), `.htmx-settling`, `.htmx-swapping` (outgoing; pair with `swap:150ms` modifier). Prefer `hx-swap="outerHTML transition:true"` to use the View Transitions API with a crossfade, and keep it short. Polling/refresh swaps should **not** animate — they're high-frequency.
- **View Transitions** (`document.startViewTransition`, cross-document `@view-transition { navigation: auto; }`): good for list→detail continuity; set `view-transition-name` on the shared element only.
- **React**: Motion (`motion/react`). Use `transform` strings or Motion ≥ 14.2 for hardware-accelerated `x`/`y`. `AnimatePresence` for exits.
- **Artifacts**: CSS only; no animation library needed.

## Motion tokens for DESIGN.md

Record: the three curves, a duration scale (`instant 0`, `fast 120ms`, `base 200ms`, `slow 320ms`), press scale (0.97), stagger step (40ms), and the frequency table decisions ("table sort: no motion").
