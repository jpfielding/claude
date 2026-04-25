# CSS Foundation for Diagrams

Theme tokens, background atmosphere, and link styling. Load first when setting up a new diagram page.

## Contents

- [Theme Setup](#theme-setup)
- [Background Atmosphere](#background-atmosphere)
- [Link Styling](#link-styling)

## Theme Setup

Always define both light and dark palettes via custom properties. Start with whichever fits the chosen aesthetic, ensure both work.

```css
:root {
  --font-body: 'Outfit', system-ui, sans-serif;
  --font-mono: 'Space Mono', 'SF Mono', Consolas, monospace;

  --bg: #f8f9fa;
  --surface: #ffffff;
  --surface-elevated: #ffffff;
  --border: rgba(0, 0, 0, 0.08);
  --border-bright: rgba(0, 0, 0, 0.15);
  --text: #1a1a2e;
  --text-dim: #6b7280;
  --accent: #0891b2;
  --accent-dim: rgba(8, 145, 178, 0.1);
  /* Semantic accents for diagram elements */
  --node-a: #0891b2;
  --node-a-dim: rgba(8, 145, 178, 0.1);
  --node-b: #059669;
  --node-b-dim: rgba(5, 150, 105, 0.1);
  --node-c: #d97706;
  --node-c-dim: rgba(217, 119, 6, 0.1);
}

@media (prefers-color-scheme: dark) {
  :root {
    --bg: #0d1117;
    --surface: #161b22;
    --surface-elevated: #1c2333;
    --border: rgba(255, 255, 255, 0.06);
    --border-bright: rgba(255, 255, 255, 0.12);
    --text: #e6edf3;
    --text-dim: #8b949e;
    --accent: #22d3ee;
    --accent-dim: rgba(34, 211, 238, 0.12);
    --node-a: #22d3ee;
    --node-a-dim: rgba(34, 211, 238, 0.12);
    --node-b: #34d399;
    --node-b-dim: rgba(52, 211, 153, 0.12);
    --node-c: #fbbf24;
    --node-c-dim: rgba(251, 191, 36, 0.12);
  }
}
```

## Background Atmosphere

Flat backgrounds feel dead. Use subtle gradients or patterns.

```css
/* Radial glow behind focal area */
body {
  background: var(--bg);
  background-image: radial-gradient(ellipse at 50% 0%, var(--accent-dim) 0%, transparent 60%);
}

/* Faint dot grid */
body {
  background-color: var(--bg);
  background-image: radial-gradient(circle, var(--border) 1px, transparent 1px);
  background-size: 24px 24px;
}

/* Diagonal subtle lines */
body {
  background-color: var(--bg);
  background-image: repeating-linear-gradient(
    -45deg, transparent, transparent 40px,
    var(--border) 40px, var(--border) 41px
  );
}

/* Gradient mesh (pick 2-3 positioned radials) */
body {
  background: var(--bg);
  background-image:
    radial-gradient(at 20% 20%, var(--node-a-dim) 0%, transparent 50%),
    radial-gradient(at 80% 60%, var(--node-b-dim) 0%, transparent 50%);
}
```

## Link Styling

**Never rely on browser default link colors.** The default blue (`#0000EE`) has poor contrast on dark backgrounds. Style links with `color: var(--accent)` and keep underlines for discoverability. On dark backgrounds, use bright accents (`#22d3ee`, `#34d399`, `#fbbf24`). On light backgrounds, use deeper tones (`#0891b2`, `#059669`, `#d97706`).

