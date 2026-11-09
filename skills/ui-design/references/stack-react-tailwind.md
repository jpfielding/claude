# Stack: React + Tailwind

## Tokens
Tailwind v4: put tokens in CSS with `@theme`, so utilities are generated from *your* names and the default palette isn't the path of least resistance.

```css
@import "tailwindcss";
@theme {
  --color-*: initial;               /* drop the default palette: no accidental indigo-500 */
  --color-surface: oklch(0.985 0.003 160);
  --color-ink: oklch(0.24 0.01 160);
  --color-ink-muted: oklch(0.45 0.01 160);
  --color-accent: oklch(0.48 0.07 165);
  --color-danger: oklch(0.52 0.18 27);
  --font-sans: "IBM Plex Sans", ui-sans-serif, system-ui, sans-serif;
  --radius-md: 6px;
  --ease-out: cubic-bezier(0.23, 1, 0.32, 1);
}
@layer base {
  :root { color-scheme: light; accent-color: var(--color-accent); }
  ::selection { background: color-mix(in oklch, var(--color-accent) 25%, transparent); }
}
```

Resetting `--color-*` is the single most effective anti-slop move in Tailwind: `bg-indigo-500` stops compiling. If the project needs some defaults back, re-add only what it uses. Dark mode: redefine the same variables under `.dark` / `[data-theme=dark]` or `@media (prefers-color-scheme: dark)` — components don't change.

v3: `theme.extend` is the trap (it keeps the full default palette). Use `theme.colors` (replace) with your roles.

## Components
- shadcn/ui is a reasonable base: it's copied into the repo, uses CSS variables, and is accessible via Radix. Retheme its variables from DESIGN.md rather than accepting the default zinc/slate + default radius, which is itself now a recognizable "generated" look.
- Compose classes with `cn()` (clsx + tailwind-merge). Long class strings → extract a component, not an `@apply` soup.
- Variants with `cva` mapped to DESIGN.md component states.
- Icons: `lucide-react` (consistent stroke), size via a prop/token.
- Focus: `focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent` — never bare `outline-none`.
- Hover: Tailwind v4's `hover:` already only applies on hover-capable devices; in v3 add a custom variant.
- Motion: `transition-[transform,opacity]` (never `transition-all`), `duration-150`, `ease-[var(--ease-out)]`, `motion-reduce:transition-none`. For orchestrated/exit animation use Motion (`motion/react`) with `AnimatePresence`.

## Data-heavy UI
- Tables: TanStack Table for sorting/filtering/virtualization; render with your own styled `<table>`, `tabular-nums`, `text-right` numeric cells.
- Charts: if the `dataviz` skill is present, follow it. Otherwise Recharts/visx with token colors, no default palettes.
- Loading: Suspense boundaries with layout-matched skeletons.

## Effect libraries (React Bits, Canvas UI, Aceternity-style)
They're catalogs of flashy effects. Rules if the user asks for one:
- One effect, on one element, where it serves the content (a hero headline, a success moment). Never in operational screens.
- Must degrade: content readable with the effect off, under reduced motion, and when WebGL/experimental APIs are unsupported.
- **License**: React Bits and Canvas UI are MIT + Commons Clause — fine to *use* in an app; don't redistribute or repackage the components (e.g. in a shared component kit or template you sell). Install via their shadcn registry (`npx shadcn@latest add @react-bits/<Name>-TS-TW`) rather than copy-pasting.

## Next.js specifics
`next/font` for self-hosted fonts (no layout shift); `next/image` needs `alt`; keep tokens in the root layout's CSS.
