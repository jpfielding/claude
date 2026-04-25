# Slide Typography, Transitions, and Nav Chrome

The non-content layer: typography scale, cinematic slide transitions, and navigation chrome (progress indicator, slide counter, keyboard controls).

## Contents

- [Typography Scale](#typography-scale)
- [Cinematic Transitions](#cinematic-transitions)
- [Navigation Chrome](#navigation-chrome)

## Typography Scale

Slide typography is 2–3× larger than scrollable pages. Page-sized text on a viewport-sized canvas looks like a mistake.

```css
.slide__display {
  font-size: clamp(48px, 10vw, 120px);
  font-weight: 800;
  letter-spacing: -3px;
  line-height: 0.95;
  text-wrap: balance;
}

.slide__heading {
  font-size: clamp(28px, 5vw, 48px);
  font-weight: 700;
  letter-spacing: -1px;
  line-height: 1.1;
  text-wrap: balance;
}

.slide__body {
  font-size: clamp(16px, 2.2vw, 24px);
  line-height: 1.6;
  text-wrap: pretty;
}

.slide__label {
  font-family: var(--font-mono);
  font-size: clamp(10px, 1.2vw, 14px);
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 1.5px;
  color: var(--text-dim);
}

.slide__subtitle {
  font-family: var(--font-mono);
  font-size: clamp(14px, 1.8vw, 20px);
  color: var(--text-dim);
  letter-spacing: 0.5px;
}
```

| Element | Size range | Notes |
|---------|-----------|-------|
| Display (title slides) | 48–120px | `10vw` preferred, weight 800 |
| Section numbers | 100–240px | Ultra-light (weight 200), decorative |
| Headings | 28–48px | `5vw` preferred, weight 700 |
| Body / bullets | 16–24px | `2.2vw` preferred, 1.6 line-height |
| Code blocks | 14–18px | `1.8vw` preferred, mono |
| Quotes | 24–48px | `4vw` preferred, serif italic |
| Labels / captions | 10–14px | Mono, uppercase, dimmed |

## Cinematic Transitions

IntersectionObserver adds `.visible` when a slide enters the viewport. Slides animate in once and stay visible when scrolling back.

```css
/* Slide entrance — fade + lift + subtle scale */
.slide {
  opacity: 0;
  transform: translateY(40px) scale(0.98);
  transition:
    opacity 0.6s cubic-bezier(0.16, 1, 0.3, 1),
    transform 0.6s cubic-bezier(0.16, 1, 0.3, 1);
}

.slide.visible {
  opacity: 1;
  transform: none;
}

/* Staggered child reveals — add .reveal to each content element */
.slide .reveal {
  opacity: 0;
  transform: translateY(20px);
  transition:
    opacity 0.5s cubic-bezier(0.16, 1, 0.3, 1),
    transform 0.5s cubic-bezier(0.16, 1, 0.3, 1);
}

.slide.visible .reveal {
  opacity: 1;
  transform: none;
}

/* Stagger delays — up to 6 children per slide */
.slide.visible .reveal:nth-child(1) { transition-delay: 0.1s; }
.slide.visible .reveal:nth-child(2) { transition-delay: 0.2s; }
.slide.visible .reveal:nth-child(3) { transition-delay: 0.3s; }
.slide.visible .reveal:nth-child(4) { transition-delay: 0.4s; }
.slide.visible .reveal:nth-child(5) { transition-delay: 0.5s; }
.slide.visible .reveal:nth-child(6) { transition-delay: 0.6s; }

@media (prefers-reduced-motion: reduce) {
  .slide,
  .slide .reveal {
    opacity: 1 !important;
    transform: none !important;
    transition: none !important;
  }
}
```

## Navigation Chrome

All navigation is `position: fixed` with high z-index, layered above slides. Styled to be visible on any background.

### Progress Bar

```css
.deck-progress {
  position: fixed;
  top: 0;
  left: 0;
  height: 3px;
  background: var(--accent);
  z-index: 100;
  transition: width 0.3s ease;
  pointer-events: none;
}
```

### Nav Dots

```css
.deck-dots {
  position: fixed;
  right: clamp(12px, 2vw, 24px);
  top: 50%;
  transform: translateY(-50%);
  display: flex;
  flex-direction: column;
  gap: 8px;
  z-index: 100;
}

.deck-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--text-dim);
  opacity: 0.3;
  border: none;
  padding: 0;
  cursor: pointer;
  transition: opacity 0.2s ease, transform 0.2s ease;
}

.deck-dot:hover {
  opacity: 0.6;
}

.deck-dot.active {
  opacity: 1;
  transform: scale(1.5);
  background: var(--accent);
}
```

### Slide Counter

```css
.deck-counter {
  position: fixed;
  bottom: clamp(12px, 2vh, 24px);
  right: clamp(12px, 2vw, 24px);
  font-family: var(--font-mono);
  font-size: 12px;
  color: var(--text-dim);
  z-index: 100;
  font-variant-numeric: tabular-nums;
}
```

### Keyboard Hints

Auto-fade after first interaction or after 4 seconds.

```css
.deck-hints {
  position: fixed;
  bottom: clamp(12px, 2vh, 24px);
  left: 50%;
  transform: translateX(-50%);
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--text-dim);
  opacity: 0.6;
  z-index: 100;
  transition: opacity 0.5s ease;
  white-space: nowrap;
}

.deck-hints.faded {
  opacity: 0;
  pointer-events: none;
}
```

### Chrome Visibility on Mixed Backgrounds

For decks where some slides are light and some dark (especially full-bleed slides), nav chrome needs to remain visible. Two approaches:

```css
/* Approach A: subtle backdrop on chrome elements */
.deck-dots,
.deck-counter {
  background: color-mix(in srgb, var(--bg) 70%, transparent 30%);
  padding: 6px;
  border-radius: 20px;
  backdrop-filter: blur(4px);
  -webkit-backdrop-filter: blur(4px);
}

/* Approach B: text shadow for legibility on any background */
.deck-counter,
.deck-hints {
  text-shadow: 0 1px 3px rgba(0, 0, 0, 0.3);
}
```

