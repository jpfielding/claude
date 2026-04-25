# CSS Layout Primitives

Card/section components, code blocks, overflow protection, and Mermaid diagram containers with zoom controls. The core building blocks of every visual page.

## Contents

- [Section / Card Components](#section--card-components)
- [Code Blocks](#code-blocks)
- [Overflow Protection](#overflow-protection)
- [Mermaid Containers](#mermaid-containers)

## Section / Card Components

The fundamental building block. A colored card representing a system component, pipeline step, or data entity.

**IMPORTANT: Never use `.node` as a CSS class name.** Mermaid.js internally uses `.node` on its SVG `<g>` elements with `transform: translate(x, y)` for positioning. Any page-level `.node` styles (hover transforms, box-shadows, transitions) will leak into Mermaid diagrams and break their layout. Use `.ve-card` instead (namespaced to avoid collisions with CSS frameworks like Bootstrap/Tailwind that also use `.card`).

```css
.ve-card {
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: 10px;
  padding: 16px 20px;
  position: relative;
}

/* Colored accent border (left or top) */
.ve-card--accent-a {
  border-left: 3px solid var(--node-a);
}

/* --- Depth tiers: vary card depth to signal importance --- */

/* Elevated: KPIs, key sections, anything that should pop */
.ve-card--elevated {
  background: var(--surface-elevated);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08), 0 1px 2px rgba(0, 0, 0, 0.04);
}

/* Recessed: code blocks, secondary content, detail panels */
.ve-card--recessed {
  background: color-mix(in srgb, var(--bg) 70%, var(--surface) 30%);
  box-shadow: inset 0 1px 3px rgba(0, 0, 0, 0.06);
  border-color: var(--border);
}

/* Hero: executive summaries, focal elements — demands attention */
.ve-card--hero {
  background: color-mix(in srgb, var(--surface) 92%, var(--accent) 8%);
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.08), 0 1px 3px rgba(0, 0, 0, 0.04);
  border-color: color-mix(in srgb, var(--border) 50%, var(--accent) 50%);
}

/* Glass: special-occasion overlay effect (use sparingly) */
.ve-card--glass {
  background: color-mix(in srgb, var(--surface) 60%, transparent 40%);
  backdrop-filter: blur(12px);
  -webkit-backdrop-filter: blur(12px);
  border-color: rgba(255, 255, 255, 0.1);
}

/* Section label (monospace, uppercase, small) */
.ve-card__label {
  font-family: var(--font-mono);
  font-size: 10px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 1.5px;
  color: var(--node-a);
  margin-bottom: 10px;
  display: flex;
  align-items: center;
  gap: 8px;
}

/* Colored dot indicator */
.ve-card__label::before {
  content: '';
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: currentColor;
}
```

## Code Blocks

Code blocks need explicit whitespace preservation and a max-height constraint. Without these, code runs together and long files overwhelm the page.

### Basic Pattern

```css
.code-block {
  font-family: var(--font-mono);
  font-size: 13px;
  line-height: 1.5;
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 16px;
  overflow-x: auto;
  /* CRITICAL: preserve line breaks and indentation */
  white-space: pre-wrap;
  word-break: break-word;
}

/* Constrain height for long code */
.code-block--scroll {
  max-height: 400px;
  overflow-y: auto;
}
```

```html
<pre class="code-block code-block--scroll"><code>// Your code here
function example() {
  return true;
}</code></pre>
```

### With File Header

```css
.code-file {
  border: 1px solid var(--border);
  border-radius: 8px;
  overflow: hidden;
}

.code-file__header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 16px;
  background: var(--surface);
  border-bottom: 1px solid var(--border);
  font-family: var(--font-mono);
  font-size: 12px;
  color: var(--text-dim);
}

.code-file__body {
  font-family: var(--font-mono);
  font-size: 13px;
  line-height: 1.5;
  padding: 16px;
  background: var(--surface-elevated);
  white-space: pre-wrap;
  word-break: break-word;
  max-height: 500px;
  overflow: auto;
}
```

```html
<div class="code-file">
  <div class="code-file__header">
    <span>src/extension.ts</span>
  </div>
  <pre class="code-file__body"><code>export function activate() {
  // ...
}</code></pre>
</div>
```

### Implementation Plans: Don't Dump Full Files

For implementation plans and architecture docs, **don't display entire source files inline**. Instead:

1. **Show structure, not code:**
   ```html
   <div class="file-structure">
     <div class="file-structure__path">src/extension.ts</div>
     <ul class="file-structure__outline">
       <li><code>BOOMERANG_INSTRUCTIONS</code> — System prompt for autonomous mode</li>
       <li><code>clearState()</code> — Reset extension state</li>
       <li><code>updateStatus()</code> — Update UI status indicator</li>
       <li><code>/boomerang</code> command — Start autonomous task</li>
       <li><code>/boomerang-cancel</code> command — Cancel active task</li>
       <li><code>before_agent_start</code> hook — Inject instructions</li>
       <li><code>agent_end</code> hook — Generate summary</li>
     </ul>
   </div>
   ```

2. **Use collapsible sections for full code:**
   ```html
   <details class="collapsible">
     <summary>Full implementation (87 lines)</summary>
     <pre class="code-file__body"><code>...</code></pre>
   </details>
   ```

3. **Show key snippets only:**
   ```html
   <p>The core logic intercepts task completion:</p>
   <pre class="code-block"><code>pi.on("agent_end", async () => {
     const summary = generateSummary(workEntries);
     boomerangComplete = true;
   });</code></pre>
   ```

**Anti-patterns:**
- Displaying full source files inline (100+ lines overwhelming the page)
- Code blocks without `white-space: pre-wrap` (code runs together into unreadable wall)
- No height constraint on long code (page becomes endless scroll)

If someone needs the full file, put it in a collapsible section or link to it.

## Overflow Protection

Grid and flex children default to `min-width: auto`, which prevents them from shrinking below their content width. Long text, inline code badges, and non-wrapping elements will blow out containers.

### Global rules

```css
/* Every grid/flex child must be able to shrink */
.grid > *, .flex > *,
[style*="display: grid"] > *,
[style*="display: flex"] > * {
  min-width: 0;
}

/* Long text wraps instead of overflowing */
body {
  overflow-wrap: break-word;
}
```

### Side-by-side comparison panels

```css
.comparison {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
}

.comparison > * {
  min-width: 0;
  overflow-wrap: break-word;
}

@media (max-width: 768px) {
  .comparison { grid-template-columns: 1fr; }
}
```

### Never use `display: flex` on `<li>` for marker characters

Using `display: flex` on a list item to position a `::before` marker creates an anonymous flex item for the remaining text content. That anonymous flex item gets `min-width: auto` and you **cannot** set `min-width: 0` on anonymous boxes. Lines with many inline `<code>` badges will overflow their container with no CSS fix possible.

Use absolute positioning for markers instead:

```css
/* WRONG — causes overflow with inline code badges */
li {
  display: flex;
  align-items: baseline;
  gap: 6px;
}
li::before {
  content: '›';
  flex-shrink: 0;
}

/* RIGHT — text wraps normally */
li {
  padding-left: 14px;
  position: relative;
}
li::before {
  content: '›';
  position: absolute;
  left: 0;
}
```

### List markers overlapping container borders

By default, `list-style-position: outside` places list markers (bullets, numbers) outside the content box. When lists are inside bordered containers (cards, callout boxes), the markers can overlap or extend beyond the border.

```css
/* WRONG — markers overlap container border */
.card ol, .card ul {
  padding-left: 20px;  /* Not enough for outside markers */
}

/* RIGHT — use inside positioning */
.card ol, .card ul {
  list-style-position: inside;
}

/* OR — adequate padding for outside markers */
.card ol, .card ul {
  padding-left: 2em;  /* ~32px gives room for markers */
}

/* OR — custom markers with absolute positioning (most control) */
.card ol {
  list-style: none;
  padding-left: 0;
  counter-reset: item;
}
.card ol li {
  counter-increment: item;
  padding-left: 2em;
  position: relative;
}
.card ol li::before {
  content: counter(item) ".";
  position: absolute;
  left: 0;
  color: var(--accent);
  font-weight: 600;
}
```

**Rule of thumb:** Any `<ol>` or `<ul>` inside a bordered container needs either `list-style-position: inside` or `padding-left: 2em` minimum. The default 20px padding is not enough for outside-positioned markers.

## Mermaid Containers

Mermaid diagrams have two common layout issues: they render too small to read, and they left-align in their container leaving awkward dead space (especially for narrow vertical flowcharts).

### Centering (Required)

Mermaid SVGs render at a fixed size based on content. Without explicit centering, they default to top-left alignment. **Always center Mermaid diagrams** — narrow vertical flowcharts look particularly bad when left-aligned in a wide container.

```css
/* WRONG — diagram hugs left edge */
.mermaid-container {
  padding: 24px;
  border: 1px solid var(--border);
}

/* RIGHT — diagram centers in container */
.mermaid-wrap {
  display: flex;
  justify-content: center;
  align-items: flex-start;  /* or center for shorter diagrams */
  padding: 24px;
  border: 1px solid var(--border);
}
```

### Scaling Small Diagrams

Mermaid sizes diagrams based on content, not container. Complex diagrams with many nodes render small to fit everything, leaving the text nearly unreadable. Three fixes:

**1. Increase fontSize in themeVariables** (most effective):
```javascript
mermaid.initialize({
  theme: 'base',
  themeVariables: {
    fontSize: '18px',  // default is 16px, bump to 18-20px for complex diagrams
  }
});
```

**2. CSS zoom** for diagrams that still render too small:
```css
.mermaid-wrap--scaled .mermaid {
  zoom: 1.3;
}
```

**3. Constrain container width** so the diagram doesn't float in dead space:
```css
.mermaid-wrap--constrained {
  max-width: 800px;
  margin: 0 auto;
}
```

**Rule of thumb:** If the diagram has 10+ nodes or the text is smaller than 12px rendered, increase fontSize to 18-20px or apply CSS zoom.

### Zoom Controls

Add zoom controls to every `.mermaid-wrap` container for complex diagrams.

**Small diagrams in slides.** If a diagram has fewer than ~7 nodes with no branching, it will render tiny in a full-viewport slide container. For simple linear flows (A → B → C → D), use CSS pipeline cards instead of Mermaid — see `slides-layouts.md` "CSS Pipeline Slide." Reserve Mermaid for complex graphs where automatic edge routing is actually needed.

### Full Pattern

```css
.mermaid-wrap {
  position: relative;
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 32px 24px;
  overflow: auto;
  /* CRITICAL: center the diagram both horizontally and vertically */
  display: flex;
  justify-content: center;
  align-items: center;
  /* Prevent vertical flowcharts from compressing into unreadable thumbnails */
  min-height: 400px;
  scrollbar-width: thin;
  scrollbar-color: var(--border) transparent;
}
.mermaid-wrap::-webkit-scrollbar { width: 6px; height: 6px; }
.mermaid-wrap::-webkit-scrollbar-track { background: transparent; }
.mermaid-wrap::-webkit-scrollbar-thumb { background: var(--border); border-radius: 3px; }
.mermaid-wrap::-webkit-scrollbar-thumb:hover { background: var(--text-dim); }

/* For shorter diagrams that don't need the full height */
.mermaid-wrap--compact { min-height: 200px; }

/* For very tall vertical flowcharts */
.mermaid-wrap--tall { min-height: 600px; }

.mermaid-wrap .mermaid {
  /* Use CSS zoom instead of transform: scale().
     Zoom changes actual layout size, so overflow scrolls normally in all directions.
     Transform only changes visual appearance — content expanding upward/leftward
     goes into negative space which can't be scrolled to.
     Supported in all browsers (Firefox added support in v126, June 2024).
     Note: zoom is not animatable, so no transition. */
  /* Optional: start at >1 for complex diagrams that render too small.
     The diagram stays centered, renders larger, and zoom controls still work. */
  zoom: 1.4;
}

.zoom-controls {
  position: absolute;
  top: 8px;
  right: 8px;
  display: flex;
  gap: 2px;
  z-index: 10;
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: 6px;
  padding: 2px;
}

.zoom-controls button {
  width: 28px;
  height: 28px;
  border: none;
  background: transparent;
  color: var(--text-dim);
  font-family: var(--font-mono);
  font-size: 14px;
  cursor: pointer;
  border-radius: 4px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: background 0.15s ease, color 0.15s ease;
}

.zoom-controls button:hover {
  background: var(--border);
  color: var(--text);
}

.mermaid-wrap { cursor: grab; }
.mermaid-wrap.is-panning { cursor: grabbing; user-select: none; }
```

**Why zoom instead of transform?**

CSS `transform: scale()` only changes visual appearance — the element's layout box stays the same size. When you scale from `center center`, content expands upward and leftward into negative coordinate space. Scroll containers can't scroll to negative positions, so the top and left of the zoomed content get clipped.

CSS `zoom` actually changes the element's layout size. The content grows downward and rightward like any other growing element, staying fully scrollable.

### HTML

```html
<div class="mermaid-wrap">
  <div class="zoom-controls">
    <button onclick="zoomDiagram(this, 1.2)" title="Zoom in">+</button>
    <button onclick="zoomDiagram(this, 0.8)" title="Zoom out">&minus;</button>
    <button onclick="resetZoom(this)" title="Reset zoom">&#8634;</button>
  </div>
  <pre class="mermaid">
    graph TD
      A --> B
  </pre>
</div>
```

### JavaScript

Add once at the end of the page. Handles button clicks and scroll-to-zoom on all `.mermaid-wrap` containers:

```javascript
// Match this to the CSS zoom value (or 1 if not set)
var INITIAL_ZOOM = 1.4;

function zoomDiagram(btn, factor) {
  var wrap = btn.closest('.mermaid-wrap');
  var target = wrap.querySelector('.mermaid');
  var current = parseFloat(target.dataset.zoom || INITIAL_ZOOM);
  var next = Math.min(Math.max(current * factor, 0.5), 5);
  target.dataset.zoom = next;
  target.style.zoom = next;
}

function resetZoom(btn) {
  var wrap = btn.closest('.mermaid-wrap');
  var target = wrap.querySelector('.mermaid');
  target.dataset.zoom = INITIAL_ZOOM;
  target.style.zoom = INITIAL_ZOOM;
}

document.querySelectorAll('.mermaid-wrap').forEach(function(wrap) {
  // Ctrl/Cmd + scroll to zoom
  wrap.addEventListener('wheel', function(e) {
    if (!e.ctrlKey && !e.metaKey) return;
    e.preventDefault();
    var target = wrap.querySelector('.mermaid');
    var current = parseFloat(target.dataset.zoom || INITIAL_ZOOM);
    var factor = e.deltaY < 0 ? 1.1 : 0.9;
    var next = Math.min(Math.max(current * factor, 0.5), 5);
    target.dataset.zoom = next;
    target.style.zoom = next;
  }, { passive: false });

  // Click-and-drag to pan
  var startX, startY, scrollL, scrollT;
  wrap.addEventListener('mousedown', function(e) {
    if (e.target.closest('.zoom-controls')) return;
    wrap.classList.add('is-panning');
    startX = e.clientX;
    startY = e.clientY;
    scrollL = wrap.scrollLeft;
    scrollT = wrap.scrollTop;
  });
  window.addEventListener('mousemove', function(e) {
    if (!wrap.classList.contains('is-panning')) return;
    wrap.scrollLeft = scrollL - (e.clientX - startX);
    wrap.scrollTop = scrollT - (e.clientY - startY);
  });
  window.addEventListener('mouseup', function() {
    wrap.classList.remove('is-panning');
  });
});
```

Scroll-to-zoom requires Ctrl/Cmd+scroll to avoid hijacking normal page scroll. Cursor changes to `grab`/`grabbing` to signal pan mode. The zoom range is capped at 0.5x–5x.

