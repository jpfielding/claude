# Slide Responsive Breakpoints and Curated Presets

Height-based responsive breakpoints (for short viewports) and the four curated slide presets: Midnight Editorial, Warm Signal, Terminal Mono, Swiss Clean.

## Contents

- [Responsive Height Breakpoints](#responsive-height-breakpoints)
- [Curated Presets](#curated-presets)

## Responsive Height Breakpoints

Height-based scaling is more critical for slides than width. Each breakpoint progressively reduces padding, font sizes, and hides decorative elements.

```css
/* Compact viewports */
@media (max-height: 700px) {
  .slide {
    padding: clamp(24px, 4vh, 40px) clamp(32px, 6vw, 80px);
  }
  .slide__display { font-size: clamp(36px, 8vw, 72px); }
  .slide--divider .slide__number { font-size: clamp(80px, 16vw, 160px); }
}

/* Small tablets / landscape phones */
@media (max-height: 600px) {
  .slide__decor { display: none; } /* hide decorative SVGs */
  .slide--quote { padding: clamp(32px, 6vh, 60px) clamp(40px, 8vw, 100px); }
  .slide__quote-mark { display: none; }
}

/* Aggressive: landscape phones */
@media (max-height: 500px) {
  .slide {
    padding: clamp(16px, 3vh, 24px) clamp(24px, 5vw, 48px);
  }
  .deck-dots { display: none; } /* dots clutter tiny viewports */
  .slide__display { font-size: clamp(28px, 7vw, 48px); }
}

/* Width breakpoint for grids */
@media (max-width: 768px) {
  .slide--content .slide__inner { grid-template-columns: 1fr; }
  .slide--content .slide__aside { display: none; }
  .slide--split .slide__panels { grid-template-columns: 1fr; }
  .slide--dashboard .slide__kpis { grid-template-columns: repeat(2, 1fr); }
}
```

## Curated Presets

Starting points the agent can riff on. Each defines a font pairing, palette, and background treatment. The agent adapts these to the content — different decks with the same preset should still feel distinct.

### Midnight Editorial

Deep navy, serif display, warm gold accents. Cinematic, premium. Dark-first.

```css
:root {
  --font-body: 'Instrument Serif', Georgia, serif;
  --font-mono: 'JetBrains Mono', 'SF Mono', monospace;
  --bg: #0f1729;
  --surface: #162040;
  --surface2: #1d2b52;
  --surface-elevated: #243362;
  --border: rgba(200, 180, 140, 0.08);
  --border-bright: rgba(200, 180, 140, 0.16);
  --text: #e8e4d8;
  --text-dim: #9a9484;
  --accent: #d4a73a;
  --accent-dim: rgba(212, 167, 58, 0.1);
  --code-bg: #0a0f1e;
  --code-text: #d4d0c4;
}
@media (prefers-color-scheme: light) {
  :root {
    --bg: #faf8f2;
    --surface: #ffffff;
    --surface2: #f5f0e6;
    --surface-elevated: #fffdf5;
    --border: rgba(30, 30, 50, 0.08);
    --border-bright: rgba(30, 30, 50, 0.16);
    --text: #1a1814;
    --text-dim: #7a7468;
    --accent: #b8860b;
    --accent-dim: rgba(184, 134, 11, 0.08);
    --code-bg: #2a2520;
    --code-text: #e8e4d8;
  }
}
```

Background: radial gold glow at top center. Decorative corner marks in gold. Title slides use dramatic serif at max scale.

### Warm Signal

Cream paper, bold sans, terracotta/coral accents. Confident and modern. Light-first.

```css
:root {
  --font-body: 'Plus Jakarta Sans', system-ui, sans-serif;
  --font-mono: 'Azeret Mono', 'SF Mono', monospace;
  --bg: #faf6f0;
  --surface: #ffffff;
  --surface2: #f5ece0;
  --surface-elevated: #fffdf5;
  --border: rgba(60, 40, 20, 0.08);
  --border-bright: rgba(60, 40, 20, 0.16);
  --text: #2c2a25;
  --text-dim: #7c756a;
  --accent: #c2410c;
  --accent-dim: rgba(194, 65, 12, 0.08);
  --code-bg: #2c2520;
  --code-text: #f5ece0;
}
@media (prefers-color-scheme: dark) {
  :root {
    --bg: #1c1916;
    --surface: #262220;
    --surface2: #302b28;
    --surface-elevated: #3a3430;
    --border: rgba(200, 180, 160, 0.08);
    --border-bright: rgba(200, 180, 160, 0.16);
    --text: #f0e8dc;
    --text-dim: #a09888;
    --accent: #e85d2a;
    --accent-dim: rgba(232, 93, 42, 0.1);
    --code-bg: #141210;
    --code-text: #f0e8dc;
  }
}
```

Background: warm radial glow at bottom left. Terracotta accent borders on cards. Section divider numbers in ultra-light coral.

### Terminal Mono

Dark, monospace everything, green/cyan accents, faint grid. Developer-native. Dark-first.

```css
:root {
  --font-body: 'Geist Mono', 'SF Mono', Consolas, monospace;
  --font-mono: 'Geist Mono', 'SF Mono', Consolas, monospace;
  --bg: #0a0e14;
  --surface: #12161e;
  --surface2: #1a1f2a;
  --surface-elevated: #222836;
  --border: rgba(80, 250, 123, 0.06);
  --border-bright: rgba(80, 250, 123, 0.12);
  --text: #c8d6e5;
  --text-dim: #5a6a7a;
  --accent: #50fa7b;
  --accent-dim: rgba(80, 250, 123, 0.08);
  --code-bg: #060a10;
  --code-text: #c8d6e5;
}
@media (prefers-color-scheme: light) {
  :root {
    --bg: #f4f6f8;
    --surface: #ffffff;
    --surface2: #eaecf0;
    --surface-elevated: #f8f9fa;
    --border: rgba(0, 80, 40, 0.08);
    --border-bright: rgba(0, 80, 40, 0.16);
    --text: #1a2332;
    --text-dim: #5a6a7a;
    --accent: #0d7a3e;
    --accent-dim: rgba(13, 122, 62, 0.08);
    --code-bg: #1a2332;
    --code-text: #c8d6e5;
  }
}
```

Background: faint dot grid. Everything in mono. Title slides use large weight-400 mono instead of bold display. Code slides feel native.

### Swiss Clean

White, geometric sans, single bold accent, visible grid. Minimal and precise. Light-first.

```css
:root {
  --font-body: 'DM Sans', system-ui, sans-serif;
  --font-mono: 'Fira Code', 'SF Mono', monospace;
  --bg: #ffffff;
  --surface: #f8f8f8;
  --surface2: #f0f0f0;
  --surface-elevated: #ffffff;
  --border: rgba(0, 0, 0, 0.08);
  --border-bright: rgba(0, 0, 0, 0.16);
  --text: #111111;
  --text-dim: #666666;
  --accent: #0055ff;
  --accent-dim: rgba(0, 85, 255, 0.06);
  --code-bg: #18181b;
  --code-text: #e4e4e7;
}
@media (prefers-color-scheme: dark) {
  :root {
    --bg: #111111;
    --surface: #1a1a1a;
    --surface2: #222222;
    --surface-elevated: #2a2a2a;
    --border: rgba(255, 255, 255, 0.08);
    --border-bright: rgba(255, 255, 255, 0.16);
    --text: #f0f0f0;
    --text-dim: #888888;
    --accent: #3b82f6;
    --accent-dim: rgba(59, 130, 246, 0.08);
    --code-bg: #0a0a0a;
    --code-text: #e4e4e7;
  }
}
```

Background: clean white or near-black, no gradients. Visible grid lines (the `--with-grid` pattern). Tight geometric layouts. Single accent color used sparingly for emphasis. Data-heavy and analytical content shines here.
