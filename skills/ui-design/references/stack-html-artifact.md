# Stack: standalone HTML artifacts

Single-file pages: explainers, reports, prototypes, internal one-pagers, Claude Artifacts.

- **If publishing as a Claude Artifact**, the `artifact-design` skill (or the Artifact tool's quickstart) owns the page contract: title, allowed CDNs, theming tokens on `:root` with dark-mode overrides, mobile gutters. Load it first; this skill adds the craft rules on top.
- **One file**: `<style>` with a `:root` token block at the top, then components that use only `var(--…)`. A raw hex after the token block is drift (`raw-hex-sprawl`).
- **Both themes**: define tokens under `:root`, override under `@media (prefers-color-scheme: dark)` (and a `[data-theme]` toggle if the page has one). Set `color-scheme` so form controls and scrollbars follow.
- **No framework needed**: modern CSS covers layout (grid, container queries), disclosure (`<details>`), dialogs (`<dialog>`), popovers (Popover API), and motion (transitions, `@starting-style`, scroll-driven animations with `@supports`).
- **Scripts**: only from allowed CDNs with pinned versions; prefer none. Content must render fully with JS disabled except for genuinely interactive parts.
- **Fonts**: Google Fonts or system stack. Two families max.
- **Content**: real content or honest placeholders labeled as such. No placeholder-image services; use a styled empty frame with a caption of what goes there, or an inline SVG diagram that carries real information.
- **Charts**: inline SVG or a pinned charting lib; follow the `dataviz` skill if present.
- **Size**: keep it lean; inline SVG over raster where possible.
