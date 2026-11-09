# Craft reference

The rules, the numbers, and why. Rule IDs in `code` match `ui-slop-lint` (see ui-critique) where the lint can check them; rules without an ID need eyes.

Contents: 1. Floor · 2. Reflex · 3. Finish · 4. Color · 5. Typography · 6. Spacing, layout, density · 7. Components & states · 8. Accessibility checklist

---

## 1. Floor — never waived

Accessibility, performance and honesty. These protect users, not taste.

- **Contrast**: body text and placeholder ≥ 4.5:1; large text (≥ 24px, or ≥ 18.66px bold) and UI component boundaries/icons ≥ 3:1. Check muted/secondary text especially — it's where generated UI fails most.
- **Focus**: every interactive element has a visible `:focus-visible` style (2–3px ring or outline with offset, ≥ 3:1 against its background). Removing outlines without a replacement is a Floor violation (`outline-removed`). Focus must not be hidden under sticky headers.
- **Reduced motion**: any motion ships with a `prefers-reduced-motion` path in the same change (`missing-reduced-motion`). See motion.md for what "reduced" means (gentler, not necessarily zero).
- **Performance**: animate only `transform` and `opacity` (plus color/background for hover). Never `transition: all` (`transition-all`) — it animates layout properties you didn't intend and janks.
- **Layout safety**: no `100vw` widths (`viewport-100vw`) — it includes the scrollbar and causes horizontal scroll; use `100%` and `overflow-x: clip` on `html, body` when needed. Grid tracks that hold media use `minmax(0, 1fr)` so content can't blow them out.
- **Zoom**: never disable pinch zoom (`zoom-disabled`).
- **Images**: every `<img>` has `alt` (`alt=""` if decorative) (`img-missing-alt`).
- **Targets**: interactive targets ≥ 24×24 CSS px with ≥ 8px between adjacent targets (WCAG 2.2); 44px for touch-primary UI.
- **Honesty**: no invented metrics, testimonials, customer logos, awards, or user counts presented as real. No placeholder copy (`placeholder-copy`) or placeholder-image services (`placeholder-image`) in shipped output. A missing asset gets an honest, styled empty slot, not a fake.
- **Works without decoration**: content is readable if JS fails, if an effect isn't supported, or with CSS animations disabled. Effects layer on top of content; they never gate it.
- **Semantics**: one `h1`, no skipped heading levels, buttons are `<button>`, links are `<a href>`, form inputs have labels, color is never the only carrier of meaning (status = color + icon/text).

## 2. Reflex — avoid by default, waive with a reason

These are the habits that make UI look generated. Each is fine when *chosen*. If the brief or DESIGN.md wants one, use it and leave `ui-slop-lint: allow <id> -- <reason>` next to it.

| Tell | Lint ID | Do instead |
|---|---|---|
| Two-stop purple→blue (or indigo→pink) gradient | `purple-blue-gradient` | Flat surface, or a tonal gradient within one hue if depth is needed |
| Framework-default indigo/violet as the accent | `default-indigo` | Pick an accent from the product's context; define it as a token |
| Gradient-filled text | `gradient-text` | Solid ink; emphasize with weight or size |
| Emoji as icons | `emoji-icon` | One SVG icon set, one stroke width, sized by token |
| Raw hex scattered through components | `raw-hex-sprawl`, `off-palette-color` | Tokens only past the token block |
| `ease-in` on UI motion | `ease-in-ui` | `ease-out` (custom curve, see motion.md) |
| Entrances from `scale(0)` | `scale-zero-entry` | `scale(0.95)` + opacity 0 |
| UI transitions ≥ 500ms | `long-ui-duration` | ≤ 300ms for UI; see motion.md table |
| Tracked-caps eyebrow label above every heading | `eyebrow-label` | Let the heading carry it; a label, if needed, is body face, one step smaller, sentence case |
| Card/alert with a thick colored side stripe | `colored-side-border` | Full 1px border in a semantic tint, or a leading icon |
| Decorative glass / backdrop blur | `glass-blur` | Opaque surface; blur only for overlays where seeing through has a purpose |
| Generic marketing copy ("seamless", "unlock", "elevate") | `copy-tells` | See copy.md |
| Same-size icon + heading + text card grid as the page's structure | — | Structure from the user's task: a table, a list with detail pane, a single focal chart |
| Cards nested inside cards | — | One container level; group inside with spacing or dividers |
| Hero-metric template (huge number + label + 3 stat tiles) as default dashboard opener | — | Lead with what the user acts on: alerts, the queue, the thing that changed |
| Sparklines / progress rings as filler | — | Only show a trend if someone will act on it |
| 01 / 02 / 03 section numbering | — | Headings that say what the section is |
| Monospace as a "technical" costume | — | Mono for code, IDs, hashes, aligned numeric columns only |
| Light vs dark chosen by category ("it's dev tools, so dark") | — | Choose by use scene; support both via tokens when unsure |
| Every section entering with the same scroll fade | — | No entrance animation in Operate mode; one authored moment at most in Persuade |
| A border *and* a wide soft shadow on the same card ("ghost card") | — | Declare elevation once: border **or** shadow |

## 3. Finish — polish

- `will-change` only as a last resort for a measured problem (`will-change`).
- Tinted neutrals instead of pure `#000` / `#fff` (`pure-black-white`): e.g. ink `oklch(0.22 0.01 250)`, surface `oklch(0.985 0.003 250)` — a hint of the accent's hue in the grays makes the palette cohere.
- No text below 12px (`tiny-text`); 13px only for dense table metadata.
- `font-variant-numeric: tabular-nums` on any column, counter, timer or changing number.
- Theme the browser surfaces: `::selection`, `caret-color`, `accent-color` for native controls, `color-scheme`, scrollbar color, `text-underline-offset`.
- `text-wrap: balance` on headings, `text-wrap: pretty` on short paragraphs.
- Shadows have a y-offset and soft blur, tinted toward the surface hue; no zero-offset colored glows.

## 4. Color

- **Define in OKLCH** where the stack allows (all modern browsers do). Equal L values give perceptually equal lightness, so scales and dark-mode pairs are predictable. Hex is fine in DESIGN.md for tool compatibility; convert in the token layer.
- **Roles, not swatches**: `surface`, `surface-raised`, `surface-sunken`, `ink`, `ink-muted`, `ink-faint`, `border`, `accent`, `on-accent`, `focus`, plus semantic `ok`, `warn`, `danger`, `info` each with a `-subtle` background variant.
- **Accent scarcity**: the accent marks the primary action, the current selection, and focus — little else. If more than ~5 elements on a screen use it, it stops meaning anything. Write the scarcity rule down in DESIGN.md.
- **Semantic colors are reserved**: in an ops dashboard, red means something is wrong. Don't use the danger hue decoratively, and don't make the brand accent red/amber/green if the product shows status.
- **Secondary text on colored surfaces** is tinted from the surface hue, not neutral gray (gray on blue looks dirty and often fails contrast).
- **Dark mode** is a second token set, not inverted colors: raise surface lightness in steps for elevation (darker = further back), desaturate the accent slightly, and add a touch more line-height/weight to light-on-dark body text.

## 5. Typography

- **Scale**: one modular scale, ~1.2 ratio for apps (denser), 1.25–1.333 for marketing. App defaults: 12 / 13 / 14 (body-sm, dense tables) / 16 (body) / 20 / 24 / 30. Display tops out around 6rem even on marketing pages.
- **Body**: 16px floor for reading text; 14px acceptable for dense app UI chrome and tables. Line-height 1.5 for body, 1.2–1.3 for headings, tighter as size grows.
- **Measure**: 60–75ch for reading text. App panels are often narrower; that's fine — don't stretch prose across a 1600px content area.
- **Tracking**: never negative on body text. Headings may go to −0.02em; −0.04em is the floor even for display.
- **Weights**: two or three (e.g. 400, 500/600). Hierarchy comes from size + weight + color together, not from weight alone.
- **Faces**: one family is enough for most apps; at most two (display + text). Pick deliberately and say why in DESIGN.md. Inter/system UI is acceptable for dense apps — it was designed for that — but it's the default everyone reaches for, so if the product has any identity, consider alternatives (IBM Plex Sans, Source Sans 3, Public Sans, Geist, Atkinson Hyperlegible for accessibility-first). Use mono (JetBrains Mono, IBM Plex Mono) for code, IDs and aligned numbers only.
- **Load responsibly**: `font-display: swap`, subset, ≤ 2 families × ≤ 3 weights. In air-gapped or CSP-strict deployments, self-host or use a system stack.

## 6. Spacing, layout, density

- **Spacing scale on 4px**: 4, 8, 12, 16, 24, 32, 48, 64, 96. Off-scale values are drift.
- **Proximity carries grouping**: tight within a group (4–8), generous between groups (24–48). More space above a heading than below it, so the heading belongs to what follows.
- **Density dial** for apps: *compact* (row height 32px, gaps 4–16), *comfortable* (40px rows, 8–24), *spacious* (48px+, 16–48). Ops/monitoring tools usually want compact; settings and forms comfortable. Pick one and record it.
- **Radius**: one scale (e.g. 4 / 6 / 8 / 12). Apps lean small (4–8); cards 8–12; pills only for small controls and tags. Nested radii: inner = outer − padding.
- **z-index scale**: 0 base, 10 sticky, 20 dropdown, 40 overlay, 100 modal, 1000 toast. No `z-index: 9999`.
- **Tables are the dashboard**: left-align text, right-align numbers with tabular figures, align decimals, units in the header not every cell, sticky header, zebra or row dividers (not both), row hover, and a visible sort state. Truncate with a tooltip/title rather than wrapping IDs.
- **Layout follows the task**: list → detail split, master table with a filter bar, a single focal chart with supporting stats. Not a grid of identical cards.
- **Breakpoints**: design for the narrowest realistic viewport of the actual users. An ops console may never be used on a phone — say so in DESIGN.md (Known gaps) rather than half-supporting it.

## 7. Components & states

Every interactive component specifies: default, hover (pointer devices only — gate with `@media (hover: hover) and (pointer: fine)`), focus-visible, active/pressed, disabled, loading, and where relevant selected, empty, error.

- **Buttons**: one primary per view region. Loading = disabled + spinner/label change, width stable (no layout shift). Destructive actions use the danger role and confirm only when irreversible.
- **Empty states**: say what this area is for and the one action that fills it. Not a cartoon.
- **Errors**: name the problem and the recovery, next to the field or region it concerns. Preserve user input.
- **Loading**: skeletons that match the final layout for content regions; spinners only for actions. Reserve space so content doesn't jump (CLS < 0.1).
- **Icons**: one set (Lucide, Heroicons, Phosphor, Tabler…), one stroke width, sizes from a token (16/20/24). Decorative icons `aria-hidden="true"`; icon-only buttons get `aria-label`.
- **Live regions**: content that updates in place (htmx swaps, polling) announces via `aria-live="polite"` when the change matters to the user.

## 8. Accessibility checklist (quick pass)

Priority order when trading off: accessibility > interaction/touch > performance > visual style > layout > type/color > motion.

- [ ] Contrast checked for ink-muted and on-accent text, in both themes
- [ ] Visible focus on every interactive element; logical tab order; no focus traps except modals (which trap and restore)
- [ ] Keyboard: everything reachable and operable; Esc closes overlays
- [ ] Headings hierarchical; landmarks (`header`, `nav`, `main`); skip link on multi-section pages
- [ ] Form fields labeled; errors associated (`aria-describedby`); required marked in text, not only color
- [ ] Status never color-only
- [ ] Reduced motion honored
- [ ] Zoom to 200% without loss; no horizontal scroll at 320px CSS width for content pages
