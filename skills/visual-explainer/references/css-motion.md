# CSS Animations and SVG Charts

Keyframe animations (respecting prefers-reduced-motion) and pure-SVG sparklines for inline trend visualizations.

## Contents

- [Animations](#animations)
- [Sparklines and Simple Charts (Pure SVG)](#sparklines-and-simple-charts-pure-svg)

## Animations

### Staggered Fade-In on Load

Define the keyframe once, then stagger via a `--i` CSS variable set per element. This approach works regardless of DOM nesting or interleaved non-animated elements (unlike `nth-child` which breaks when siblings aren't all the same type).

```css
@keyframes fadeUp {
  from { opacity: 0; transform: translateY(12px); }
  to { opacity: 1; transform: translateY(0); }
}

.ve-card {
  animation: fadeUp 0.4s ease-out both;
  animation-delay: calc(var(--i, 0) * 0.05s);
}
```

Set `--i` per element in the HTML to control stagger order:

```html
<div class="ve-card" style="--i: 0">First</div>
<div class="connector">...</div>
<div class="ve-card" style="--i: 1">Second</div>
<div class="connector">...</div>
<div class="ve-card" style="--i: 2">Third</div>
```

### Hover Lift
```css
.ve-card {
  transition: transform 0.2s ease, box-shadow 0.2s ease;
}

.ve-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
}
```

### Scale-Fade (for KPI cards, badges, status indicators)

```css
@keyframes fadeScale {
  from { opacity: 0; transform: scale(0.92); }
  to { opacity: 1; transform: scale(1); }
}

.kpi-card {
  animation: fadeScale 0.35s ease-out both;
  animation-delay: calc(var(--i, 0) * 0.06s);
}
```

### SVG Draw-In (for connectors, progress rings, path elements)

```css
@keyframes drawIn {
  from { stroke-dashoffset: var(--path-length); }
  to { stroke-dashoffset: 0; }
}

/* Set --path-length to the path's getTotalLength() value */
.connector path {
  stroke-dasharray: var(--path-length);
  animation: drawIn 0.8s ease-in-out both;
  animation-delay: calc(var(--i, 0) * 0.1s);
}
```

### CSS Counter (for hero numbers without JS)

Uses `@property` to animate a custom property as an integer, then display it via `counter()`. No JS required. Falls back to showing the final value immediately in browsers without `@property` support.

```css
@property --count {
  syntax: '<integer>';
  initial-value: 0;
  inherits: false;
}

@keyframes countUp {
  to { --count: var(--target); }
}

.kpi-card__value--animated {
  --target: 247;
  counter-reset: val var(--count);
  animation: countUp 1.2s ease-out forwards;
}

.kpi-card__value--animated::after {
  content: counter(val);
}
```

### Choreography

Don't use the same animation for everything. Mix types by element role, with easing stagger (fast-then-slow, not linear):

- **Cards**: `fadeUp` — the default entrance, reliable and subtle
- **KPI / badges**: `fadeScale` — scale draws the eye to important numbers
- **SVG connectors**: `drawIn` — reveals flow direction, pairs with card stagger
- **Hero numbers**: `countUp` — counting motion signals "this number matters"
- **Stagger timing**: `calc(var(--i) * 0.06s)` with lower `--i` values on important elements so they appear first

### Respect Reduced Motion
```css
@media (prefers-reduced-motion: reduce) {
  *, *::before, *::after {
    animation-duration: 0.01ms !important;
    animation-iteration-count: 1 !important;
    transition-duration: 0.01ms !important;
  }
}
```

## Sparklines and Simple Charts (Pure SVG)

For simple inline visualizations without a library:

```html
<!-- Sparkline -->
<svg viewBox="0 0 100 30" style="width:100px;height:30px;">
  <polyline points="0,25 15,20 30,22 45,10 60,15 75,5 90,12 100,8"
    fill="none" stroke="var(--accent)" stroke-width="1.5" stroke-linecap="round"/>
</svg>

<!-- Progress bar -->
<div style="height:6px;background:var(--border);border-radius:3px;overflow:hidden;">
  <div style="height:100%;width:72%;background:var(--accent);border-radius:3px;"></div>
</div>
```

