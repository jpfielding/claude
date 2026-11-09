# Stack: Go server-rendered (html/template, templ, htmx)

Server-rendered UI has an advantage generated React UI lacks: it works without JS by default. Keep it that way — htmx enhances, it doesn't gate.

## Tokens and CSS
- One `tokens.css` of `:root` custom properties generated from DESIGN.md (or hand-written), plus a small number of component CSS files. No build step required; if the project uses Tailwind standalone CLI, use v4 `@theme` as in the React reference.
- Embed and serve with long cache + content hash:

```go
//go:embed static
var static embed.FS
// serve /static/ with http.FileServerFS(static); fingerprint filenames or add ?v=<hash> in the layout template
```

- Use modern CSS freely (nesting, `:has()`, container queries, `@layer`, `color-mix()`, OKLCH) — no preprocessor needed. Order layers: `@layer reset, tokens, base, components, utilities;`.
- Avoid inline `style=""` attributes — they fight CSP (`style-src` without `'unsafe-inline'`) and scatter raw values. Use classes and data attributes (`data-state="error"`) styled in CSS.

## Templates
- **html/template**: one base layout with `{{block "content" .}}`; partials for components (`{{template "button" dict ...}}` via a small `dict` helper). Keep presentational decisions (classes, states) in templates, not in handler Go code.
- **templ**: components as functions with typed params — model states explicitly (`type ButtonState int`), so "disabled" and "loading" can't be forgotten. Lint `.templ` files, not generated `_templ.go`.
- Escape-safe by default; never `template.HTML` user content.

## htmx patterns
- **Loading states**: `hx-indicator` pointing at a spinner/skeleton; disable the trigger during requests with `hx-disabled-elt="this"`. Style `.htmx-request` states in CSS.
- **Swaps**: `hx-swap="outerHTML"` for components that own their markup; target stable IDs. Keep layout stable during swaps (reserve height for regions that load).
- **Motion on swaps**: `.htmx-added` / `.htmx-settling` / `.htmx-swapping` classes, or `hx-swap="outerHTML transition:true"` for View Transitions. Polling refreshes (`hx-trigger="every 5s"`) must not animate — high-frequency motion is noise.
- **Accessibility**: regions updated by swaps get `aria-live="polite"` (or `role="status"`) when the user needs to know; return focus sensibly after modal/dialog swaps; errors render server-side next to fields with `aria-describedby`.
- **Errors**: handle `htmx:responseError` globally to show a toast; render validation errors as 422 + the form partial so the swap shows them in place.
- **No-JS baseline**: forms have real `action`/`method`; links have real `href`. `hx-boost` for progressive enhancement of navigation.

## Native over JS
- `<dialog>` with `showModal()` for modals (focus trap, Esc, inert background for free); `::backdrop` for the scrim.
- Popover API (`popover`, `popovertarget`) for menus/tooltips; `@starting-style` for their enter animation.
- `<details>`/`<summary>` for disclosure.
- `<table>` for tabular data — sortable via server round-trip with `hx-get="?sort=latency"` and `aria-sort` on the header.

## Air-gapped / restricted deployments
Self-host fonts (WOFF2 in `static/fonts`, `font-display: swap`) or use a system stack; no CDN scripts — vendor htmx into `static/`. The DESIGN.md lint (`npx`) is a dev-time tool; it isn't needed at runtime.
