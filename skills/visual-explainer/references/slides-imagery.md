# Slide Decoration and Imagery

Decorative SVG accents and the proactive-imagery workflow: check `which surf` at the start, generate 2–4 images before writing HTML.

## Contents

- [Decorative SVG Elements](#decorative-svg-elements)
- [Proactive Imagery](#proactive-imagery)

## Decorative SVG Elements

Inline SVG accents lift slides from functional to editorial. Use sparingly — one or two per slide, never on every slide.

### Corner Accent

```html
<!-- Top-right corner mark -->
<svg class="slide__decor slide__decor--corner" width="120" height="120" viewBox="0 0 120 120">
  <line x1="120" y1="0" x2="120" y2="40" stroke="var(--accent)" stroke-width="2" opacity="0.2"/>
  <line x1="80" y1="0" x2="120" y2="0" stroke="var(--accent)" stroke-width="2" opacity="0.2"/>
</svg>
```

```css
.slide__decor {
  position: absolute;
  pointer-events: none;
  z-index: 0;
}

.slide__decor--corner {
  top: 0;
  right: 0;
}
```

### Section Divider Mark

```html
<!-- Horizontal rule with diamond -->
<svg class="slide__decor slide__decor--divider" width="200" height="20" viewBox="0 0 200 20">
  <line x1="0" y1="10" x2="85" y2="10" stroke="var(--accent)" stroke-width="1" opacity="0.3"/>
  <rect x="92" y="3" width="14" height="14" transform="rotate(45 99 10)" fill="none" stroke="var(--accent)" stroke-width="1" opacity="0.3"/>
  <line x1="115" y1="10" x2="200" y2="10" stroke="var(--accent)" stroke-width="1" opacity="0.3"/>
</svg>
```

### Geometric Background Pattern

```css
/* Faint grid dots behind a slide */
.slide--with-grid::before {
  content: '';
  position: absolute;
  inset: 0;
  background-image: radial-gradient(circle, var(--border) 1px, transparent 1px);
  background-size: 32px 32px;
  opacity: 0.5;
  pointer-events: none;
  z-index: 0;
}
```

### Per-Slide Background Variation

Vary gradient direction and accent glow position across slides to create visual rhythm. Don't use a uniform background for every slide.

```css
/* Vary these per slide via inline style or nth-child */
.slide:nth-child(odd) {
  background-image: radial-gradient(ellipse at 20% 80%, var(--accent-dim) 0%, transparent 50%);
}

.slide:nth-child(even) {
  background-image: radial-gradient(ellipse at 80% 20%, var(--accent-dim) 0%, transparent 50%);
}
```

## Proactive Imagery

Slides should reach for visuals before defaulting to text alone. If a slide could be more compelling with an image, chart, or diagram, add one.

**surf-cli integration:** Check `which surf` at the start of every slide deck generation. If available, **generate 2–4 images minimum** for any deck over 10 slides. This is not optional when surf is available — a deck with AI-generated imagery is dramatically more compelling than one with only CSS gradients. Target these slides in priority order:

1. **Title slide** (always): background image that sets the deck's visual tone. Match the topic and palette. Use `--aspect-ratio 16:9`. Prompt example: "abstract dark geometric pattern with green accent lines, technical and minimal" for Terminal Mono preset.
2. **Full-bleed slide** (always if deck has one): immersive background for the deck's visual anchor moment. Style should match the preset — photo-realistic for Midnight Editorial, abstract/geometric for Swiss Clean, circuit-board or terminal aesthetic for Terminal Mono.
3. **Content slides with conceptual topics** (1–2 if the deck has room): illustration in the `.slide__aside` area for slides about abstract concepts. Use `--aspect-ratio 1:1`.

**Generate images before writing HTML** so they're ready to embed. The workflow:

```bash
# Check availability
which surf

# Generate (one per target slide)
surf gemini "descriptive prompt matching deck palette" --generate-image /tmp/ve-slide-title.png --aspect-ratio 16:9

# Base64 encode for self-containment (macOS)
TITLE_IMG=$(base64 -i /tmp/ve-slide-title.png)
# Linux: TITLE_IMG=$(base64 -w 0 /tmp/ve-slide-title.png)

# Embed in the slide
# <div class="slide__bg" style="background-image:url('data:image/png;base64,${TITLE_IMG}')"></div>

# Clean up
rm /tmp/ve-slide-title.png
```

**Prompt craft for slides:** Be specific about style, dominant colors, and mood. Pull colors from the preset's CSS variables. Examples:
- Terminal Mono: "dark abstract circuit board pattern, green (#50fa7b) traces on near-black (#0a0e14), minimal, technical"
- Midnight Editorial: "deep navy abstract composition, warm gold accent light, cinematic depth of field, premium editorial feel"
- Warm Signal: "warm cream textured paper with terracotta geometric accents, confident modern design"

**When surf fails or isn't available:** Degrade gracefully to CSS gradients and SVG decorations. Use the `.slide__bg--gradient` pattern with bold `linear-gradient` or `radial-gradient` backgrounds. The deck should stand on its own visually without generated images — they enhance, they don't carry. Note the fallback in an HTML comment (`<!-- surf unavailable, using CSS gradient fallback -->`) so future edits know to retry.

**Inline data visualizations:** Proactively add SVG sparklines next to numbers, mini-charts on dashboard slides, and small Mermaid diagrams on split slides even when not explicitly requested. A number with a sparkline next to it tells a better story than a number alone.

**When to skip images:** If surf isn't available, degrade gracefully — use CSS gradients and SVG decorations instead. Never error on missing surf. Pure structural or data-heavy decks (code reviews, table comparisons) may not need generated images.

