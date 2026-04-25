# CSS for Prose Pages and Images

Prose-page typography and layout (for READMEs, articles, essays) and image container styles (hero banners, inline illustrations, captions).

## Contents

- [Prose Page Elements](#prose-page-elements)
- [Generated Images](#generated-images)

## Prose Page Elements

Patterns for documentation, articles, blog posts, and other reading-first content. The key difference from visual explanations: optimize for sustained reading, not scanning.

### Body Text Settings

```css
/* Comfortable reading baseline */
.prose {
  font-size: clamp(17px, 1.1vw + 14px, 19px);
  line-height: 1.7;
  max-width: 65ch;  /* ~600-680px */
  text-wrap: pretty;
}

.prose p {
  margin-bottom: 1.5em;
}

/* Narrow column for essays/literary content */
.prose--narrow {
  max-width: 60ch;
  line-height: 1.8;
}

/* Wide column for technical content with code */
.prose--wide {
  max-width: 75ch;
  line-height: 1.6;
}
```

### Lead Paragraph

Opening paragraph styled distinctly from body text.

```css
/* Larger size */
.lead {
  font-size: 20px;
  line-height: 1.6;
  color: var(--text-bright);
  margin-bottom: 32px;
}

/* With drop cap */
.lead--dropcap::first-letter {
  float: left;
  font-family: var(--font-display);
  font-size: 64px;
  font-weight: 600;
  line-height: 0.85;
  padding-right: 12px;
  padding-top: 6px;
  color: var(--accent);
}
```

### Pull Quotes

Key insights pulled out for emphasis. Use sparingly — one or two per article maximum.

```css
/* Border left — most versatile */
.pullquote {
  margin: 48px 0;
  padding-left: 24px;
  border-left: 3px solid var(--accent);
}
.pullquote p {
  font-size: 22px;
  font-style: italic;
  line-height: 1.4;
  color: var(--text-bright);
  margin: 0;
}

/* Centered with quotation mark */
.pullquote--centered {
  margin: 56px 0;
  padding: 32px 40px;
  border-top: 1px solid var(--border);
  border-bottom: 1px solid var(--border);
  text-align: center;
  position: relative;
}
.pullquote--centered::before {
  content: '"';
  position: absolute;
  top: -12px;
  left: 50%;
  transform: translateX(-50%);
  background: var(--bg);
  padding: 0 16px;
  font-family: var(--font-display);
  font-size: 48px;
  color: var(--accent);
  line-height: 1;
}
```

### Section Dividers

```css
/* Horizontal rule */
hr {
  border: none;
  height: 1px;
  background: var(--border);
  margin: 48px 0;
}

/* Ornamental divider — use: <div class="divider">✦ ✦ ✦</div> */
.divider {
  text-align: center;
  margin: 48px 0;
  color: var(--text-dim);
  font-size: 18px;
  letter-spacing: 12px;
}
```

### Article Hero Patterns

```css
/* Centered minimal — essays, personal posts */
.hero--centered {
  text-align: center;
  padding: 80px 24px 64px;
  max-width: 800px;
  margin: 0 auto;
}
.hero__category {
  font-size: 12px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 2px;
  color: var(--accent);
  margin-bottom: 16px;
}
.hero__title {
  font-size: clamp(32px, 5vw, 48px);
  font-weight: 600;
  line-height: 1.15;
  margin-bottom: 16px;
}
.hero__subtitle {
  font-size: 20px;
  font-style: italic;
  color: var(--text-dim);
  max-width: 600px;
  margin: 0 auto 24px;
}
.hero__meta {
  font-size: 13px;
  color: var(--text-dim);
}

/* Left-aligned editorial — features, documentation */
.hero--editorial {
  padding: 100px 40px 60px;
  max-width: 1000px;
  margin: 0 auto;
}
.hero--editorial .hero__title {
  font-size: clamp(40px, 7vw, 72px);
  font-weight: 800;
  line-height: 1.0;
  letter-spacing: -2px;
}
```

### Author Byline

```css
.byline {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-top: 24px;
}
.byline__avatar {
  width: 40px;
  height: 40px;
  border-radius: 50%;
}
.byline__name {
  font-weight: 600;
  color: var(--text-bright);
  display: block;
}
.byline__meta {
  font-size: 13px;
  color: var(--text-dim);
}
```

### Callout Boxes

For warnings, tips, notes, and key takeaways.

```css
.callout {
  padding: 16px 20px;
  border-radius: 8px;
  border-left: 4px solid var(--callout-border);
  background: var(--callout-bg);
  margin: 24px 0;
}

.callout--info {
  --callout-border: var(--accent);
  --callout-bg: color-mix(in srgb, var(--accent) 10%, transparent);
}

.callout--warning {
  --callout-border: var(--amber);
  --callout-bg: color-mix(in srgb, var(--amber) 10%, transparent);
}

.callout--success {
  --callout-border: var(--green);
  --callout-bg: color-mix(in srgb, var(--green) 10%, transparent);
}

.callout__title {
  font-weight: 600;
  margin-bottom: 8px;
  color: var(--callout-border);
}

/* Lists inside callouts need padding fix */
.callout ul, .callout ol {
  padding-left: 1.5em;
  margin: 8px 0 0 0;
}
```

### Theme Toggle

Use `data-theme` attribute for user-controllable light/dark modes. Random initial theme adds variety.

```css
:root, [data-theme="light"] {
  --bg: #fafaf9;
  --surface: #ffffff;
  --text: #1c1917;
  --text-dim: #78716c;
  --border: #e7e5e4;
  --accent: #0d9488;
}

[data-theme="dark"] {
  --bg: #0c0a09;
  --surface: #1c1917;
  --text: #fafaf9;
  --text-dim: #a8a29e;
  --border: #292524;
  --accent: #14b8a6;
}
```

```javascript
// Random initial theme
const themes = ['light', 'dark'];
document.documentElement.setAttribute('data-theme', themes[Math.floor(Math.random() * 2)]);

// Toggle function
function toggleTheme() {
  const current = document.documentElement.getAttribute('data-theme');
  document.documentElement.setAttribute('data-theme', current === 'light' ? 'dark' : 'light');
}
```

```html
<button class="theme-toggle" onclick="toggleTheme()" aria-label="Toggle theme">
  <svg class="theme-toggle__sun" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
    <circle cx="12" cy="12" r="5"/><path d="M12 1v2M12 21v2M4.22 4.22l1.42 1.42M18.36 18.36l1.42 1.42M1 12h2M21 12h2M4.22 19.78l1.42-1.42M18.36 5.64l1.42-1.42"/>
  </svg>
  <svg class="theme-toggle__moon" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
    <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"/>
  </svg>
</button>
```

```css
.theme-toggle {
  position: fixed;
  top: 20px;
  right: 20px;
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 8px;
  cursor: pointer;
  z-index: 100;
}
[data-theme="light"] .theme-toggle__moon { display: none; }
[data-theme="dark"] .theme-toggle__sun { display: none; }
```

### Prose Anti-Patterns

Avoid these in reading-first content:
- Body text smaller than 16px
- Line-height below 1.5
- Measure wider than 75ch (text spanning full viewport)
- Pull quotes every other paragraph
- Drop caps on every section
- Busy background patterns behind text

## Generated Images

For AI-generated illustrations embedded as base64 data URIs via `surf gemini --generate-image`. Use sparingly — hero banners, conceptual illustrations, educational diagrams, decorative accents.

### Hero Banner

Full-width image cropped to a fixed height with a gradient fade into the page background. Place at the top of the page before the title, or between the title and the first content section.

```css
.hero-img-wrap {
  position: relative;
  border-radius: 12px;
  overflow: hidden;
  margin-bottom: 24px;
}

.hero-img-wrap img {
  width: 100%;
  height: 240px;
  object-fit: cover;
  display: block;
}

/* Gradient fade into page background */
.hero-img-wrap::after {
  content: '';
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  height: 50%;
  background: linear-gradient(to top, var(--bg), transparent);
  pointer-events: none;
}
```

```html
<div class="hero-img-wrap">
  <img src="data:image/png;base64,..." alt="Descriptive alt text">
</div>
```

Generate with `--aspect-ratio 16:9` for hero banners.

### Inline Illustration

Centered image with border, shadow, and optional caption. Use within content sections for conceptual or educational illustrations.

```css
.illus {
  text-align: center;
  margin: 24px 0;
}

.illus img {
  max-width: 480px;
  width: 100%;
  border-radius: 10px;
  border: 1px solid var(--border);
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08);
}

.illus figcaption {
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--text-dim);
  margin-top: 8px;
}
```

```html
<figure class="illus">
  <img src="data:image/png;base64,..." alt="Descriptive alt text">
  <figcaption>How the message queue routes events between services</figcaption>
</figure>
```

Generate with `--aspect-ratio 1:1` or `--aspect-ratio 4:3` for inline illustrations.

### Side Accent

Small image floated beside a section. Use when the illustration supports but doesn't dominate the content.

```css
.accent-img {
  float: right;
  max-width: 200px;
  margin: 0 0 16px 24px;
  border-radius: 10px;
  border: 1px solid var(--border);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
}

@media (max-width: 768px) {
  .accent-img {
    float: none;
    max-width: 100%;
    margin: 0 0 16px 0;
  }
}
```

```html
<img class="accent-img" src="data:image/png;base64,..." alt="Descriptive alt text">
```
