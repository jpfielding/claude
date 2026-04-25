# Slide Engine (CSS + JS + Auto-Fit)

The slide engine's base CSS, the JavaScript runtime (SlideEngine), and the auto-fit mechanism that keeps content inside the viewport.

## Contents

- [Slide Engine Base](#slide-engine-base)
- [SlideEngine JavaScript](#slideengine-javascript)
- [Auto-Fit](#auto-fit)

## Slide Engine Base

The deck is a scroll-snap container. Each slide is exactly one viewport tall.

```html
<body>
<div class="deck">
  <section class="slide slide--title"> ... </section>
  <section class="slide slide--content"> ... </section>
  <section class="slide slide--diagram"> ... </section>
  <!-- one <section> per slide -->
</div>
</body>
```

```css
/* Scroll-snap container */
.deck {
  height: 100dvh;
  overflow-y: auto;
  scroll-snap-type: y mandatory;
  scroll-behavior: smooth;
  -webkit-overflow-scrolling: touch;
}

/* Individual slide */
.slide {
  height: 100dvh;
  scroll-snap-align: start;
  overflow: hidden;
  position: relative;
  display: flex;
  flex-direction: column;
  justify-content: center;
  padding: clamp(40px, 6vh, 80px) clamp(40px, 8vw, 120px);
  isolation: isolate; /* contain z-index stacking */
}
```


## SlideEngine JavaScript

Add once at the end of the page. Handles navigation, chrome updates, and scroll-triggered reveals. Event delegation ensures slide-internal interactions (Mermaid zoom, scrollable code, overflow tables) don't trigger slide navigation.

```javascript
class SlideEngine {
  constructor() {
    this.deck = document.querySelector('.deck');
    this.slides = [...document.querySelectorAll('.slide')];
    this.current = 0;
    this.total = this.slides.length;
    this.buildChrome();
    this.bindEvents();
    this.observe();
    this.update();
  }

  buildChrome() {
    // Progress bar
    var bar = document.createElement('div');
    bar.className = 'deck-progress';
    document.body.appendChild(bar);
    this.bar = bar;

    // Nav dots
    var dots = document.createElement('div');
    dots.className = 'deck-dots';
    var self = this;
    this.slides.forEach(function(_, i) {
      var d = document.createElement('button');
      d.className = 'deck-dot';
      d.title = 'Slide ' + (i + 1);
      d.onclick = function() { self.goTo(i); };
      dots.appendChild(d);
    });
    document.body.appendChild(dots);
    this.dots = [].slice.call(dots.children);

    // Counter
    var ctr = document.createElement('div');
    ctr.className = 'deck-counter';
    document.body.appendChild(ctr);
    this.counter = ctr;

    // Keyboard hints
    var hints = document.createElement('div');
    hints.className = 'deck-hints';
    hints.textContent = '\u2190 \u2192 or scroll to navigate';
    document.body.appendChild(hints);
    this.hints = hints;
    this.hintTimer = setTimeout(function() {
      hints.classList.add('faded');
    }, 4000);
  }

  bindEvents() {
    var self = this;
    // Keyboard — skip if focus is inside interactive content
    document.addEventListener('keydown', function(e) {
      if (e.target.closest('.mermaid-wrap, .table-scroll, .code-scroll, input, textarea, [contenteditable]')) return;
      if (['ArrowDown', 'ArrowRight', ' ', 'PageDown'].includes(e.key)) {
        e.preventDefault(); self.next();
      } else if (['ArrowUp', 'ArrowLeft', 'PageUp'].includes(e.key)) {
        e.preventDefault(); self.prev();
      } else if (e.key === 'Home') {
        e.preventDefault(); self.goTo(0);
      } else if (e.key === 'End') {
        e.preventDefault(); self.goTo(self.total - 1);
      }
      self.fadeHints();
    });

    // Touch swipe
    var touchY;
    this.deck.addEventListener('touchstart', function(e) {
      touchY = e.touches[0].clientY;
    }, { passive: true });
    this.deck.addEventListener('touchend', function(e) {
      var dy = touchY - e.changedTouches[0].clientY;
      if (Math.abs(dy) > 50) { dy > 0 ? self.next() : self.prev(); }
    });
  }

  observe() {
    var self = this;
    var obs = new IntersectionObserver(function(entries) {
      entries.forEach(function(entry) {
        if (entry.isIntersecting) {
          entry.target.classList.add('visible');
          self.current = self.slides.indexOf(entry.target);
          self.update();
        }
      });
    }, { threshold: 0.5 });
    this.slides.forEach(function(s) { obs.observe(s); });
  }

  goTo(i) {
    this.slides[Math.max(0, Math.min(i, this.total - 1))]
      .scrollIntoView({ behavior: 'smooth' });
  }

  next() { if (this.current < this.total - 1) this.goTo(this.current + 1); }
  prev() { if (this.current > 0) this.goTo(this.current - 1); }

  update() {
    this.bar.style.width = ((this.current + 1) / this.total * 100) + '%';
    var self = this;
    this.dots.forEach(function(d, i) { d.classList.toggle('active', i === self.current); });
    this.counter.textContent = (this.current + 1) + ' / ' + this.total;
  }

  fadeHints() {
    clearTimeout(this.hintTimer);
    this.hints.classList.add('faded');
  }
}
```

**Usage:** Instantiate after the DOM is ready and any libraries (Mermaid, Chart.js) have rendered. Always call `autoFit()` before `new SlideEngine()` so content is sized correctly before intersection observers fire.

```html
<script>
  // After Mermaid/Chart.js initialization (if used), or at end of <body>:
  document.addEventListener('DOMContentLoaded', function() {
    autoFit();
    new SlideEngine();
  });
</script>
```

## Auto-Fit

A single post-render function that handles all known content overflow cases. Agents can't perfectly predict how text reflows at every viewport size, so `autoFit()` is a required safety net. Call it after Mermaid/Chart.js render but before SlideEngine init.

```javascript
function autoFit() {
  // Mermaid SVGs: fill container instead of rendering at intrinsic size
  document.querySelectorAll('.mermaid svg').forEach(function(svg) {
    svg.removeAttribute('height');
    svg.style.width = '100%';
    svg.style.maxWidth = '100%';
    svg.style.height = 'auto';
    svg.parentElement.style.width = '100%';
  });

  // KPI values: visually scale down text that overflows card width
  document.querySelectorAll('.slide__kpi-val').forEach(function(el) {
    if (el.scrollWidth > el.clientWidth) {
      var s = el.clientWidth / el.scrollWidth;
      el.style.transform = 'scale(' + s + ')';
      el.style.transformOrigin = 'left top';
    }
  });

  // Blockquotes: reduce font proportionally for long text
  document.querySelectorAll('.slide--quote blockquote').forEach(function(el) {
    var len = el.textContent.trim().length;
    if (len > 100) {
      var scale = Math.max(0.5, 100 / len);
      var fs = parseFloat(getComputedStyle(el).fontSize);
      el.style.fontSize = Math.max(16, Math.round(fs * scale)) + 'px';
    }
  });
}
```

Three cases, one function:
- **Mermaid:** SVGs render with fixed dimensions inside flex containers — force them to fill available width.
- **KPI values:** Long text strings at hero scale overflow card boundaries — `transform: scale()` shrinks visually without reflow.
- **Blockquotes:** Quotes longer than ~100 characters get proportionally smaller font. The 0.5 floor prevents unreadably small text; if it needs more than 50% shrink, it should have been a content slide.


