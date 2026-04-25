# CSS Grid Layouts and SVG Connectors

Grid-based layouts and pure-SVG connector patterns for flow, network, and architecture diagrams.

## Contents

- [Grid Layouts](#grid-layouts)
- [Connectors](#connectors)

## Grid Layouts

### Architecture Diagram (2-column with sidebar)
```css
.arch-grid {
  display: grid;
  grid-template-columns: 260px 1fr;
  grid-template-rows: auto;
  gap: 20px;
  max-width: 1100px;
  margin: 0 auto;
}

.arch-grid__sidebar { grid-column: 1; }
.arch-grid__main { grid-column: 2; }
.arch-grid__full { grid-column: 1 / -1; }
```

### Pipeline (horizontal steps)
```css
.pipeline {
  display: flex;
  align-items: stretch;
  gap: 0;
  overflow-x: auto;
  padding-bottom: 8px;
}

.pipeline__step {
  min-width: 130px;
  flex-shrink: 0;
}

.pipeline__arrow {
  display: flex;
  align-items: center;
  padding: 0 4px;
  color: var(--border-bright);
  font-size: 18px;
  flex-shrink: 0;
}

/* Parallel branch within a pipeline */
.pipeline__parallel {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
```

### Card Grid (dashboard / metrics)
```css
.card-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
  gap: 16px;
}
```

### Data Tables

Use real `<table>` elements for tabular data. Wrap in a scrollable container for wide tables.

```css
/* Scrollable wrapper for wide tables */
.table-wrap {
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: 12px;
  overflow: hidden;
}

.table-scroll {
  overflow-x: auto;
  -webkit-overflow-scrolling: touch;
}

/* Base table */
.data-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 13px;
  line-height: 1.5;
}

/* Header */
.data-table thead {
  position: sticky;
  top: 0;
  z-index: 2;
}

.data-table th {
  background: var(--surface-elevated, var(--surface2, var(--surface)));
  font-family: var(--font-mono);
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 1px;
  color: var(--text-dim);
  text-align: left;
  padding: 12px 16px;
  border-bottom: 2px solid var(--border-bright);
  white-space: nowrap;
}

/* Cells */
.data-table td {
  padding: 12px 16px;
  border-bottom: 1px solid var(--border);
  vertical-align: top;
  color: var(--text);
}

/* Let text-heavy columns wrap naturally */
.data-table .wide {
  min-width: 200px;
  max-width: 500px;
}

/* Right-align numeric columns */
.data-table td.num,
.data-table th.num {
  text-align: right;
  font-variant-numeric: tabular-nums;
  font-family: var(--font-mono);
}

/* Alternating rows */
.data-table tbody tr:nth-child(even) {
  background: var(--accent-dim);
}

/* Row hover */
.data-table tbody tr {
  transition: background 0.15s ease;
}

.data-table tbody tr:hover {
  background: var(--border);
}

/* Last row: no bottom border (container handles it) */
.data-table tbody tr:last-child td {
  border-bottom: none;
}

/* Code inside cells */
.data-table code {
  font-family: var(--font-mono);
  font-size: 11px;
  background: var(--accent-dim);
  color: var(--accent);
  padding: 1px 5px;
  border-radius: 3px;
}

/* Secondary detail text */
.data-table small {
  display: block;
  color: var(--text-dim);
  font-size: 11px;
  margin-top: 2px;
}
```

#### Status Indicators

Styled spans for match/gap/warning states. Never use emoji.

```css
.status {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-family: var(--font-mono);
  font-size: 11px;
  font-weight: 500;
  padding: 3px 10px;
  border-radius: 6px;
  white-space: nowrap;
}

.status--match {
  background: var(--green-dim, rgba(5, 150, 105, 0.1));
  color: var(--green, #059669);
}

.status--gap {
  background: var(--red-dim, rgba(239, 68, 68, 0.1));
  color: var(--red, #ef4444);
}

.status--warn {
  background: var(--orange-dim, rgba(217, 119, 6, 0.1));
  color: var(--orange, #d97706);
}

.status--info {
  background: var(--accent-dim);
  color: var(--accent);
}

/* Dot variant (compact, no text) */
.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  display: inline-block;
}

.status-dot--match { background: var(--green, #059669); }
.status-dot--gap { background: var(--red, #ef4444); }
.status-dot--warn { background: var(--orange, #d97706); }
```

Usage in table cells:
```html
<td><span class="status status--match">Match</span></td>
<td><span class="status status--gap">Gap</span></td>
<td><span class="status status--warn">Partial</span></td>
```

#### Table Summary Row

For totals, counts, or aggregate status at the bottom:

```css
.data-table tfoot td {
  background: var(--surface-elevated, var(--surface2, var(--surface)));
  font-weight: 600;
  font-size: 12px;
  border-top: 2px solid var(--border-bright);
  border-bottom: none;
  padding: 12px 16px;
}
```

#### Sticky First Column (for very wide tables)

```css
.data-table th:first-child,
.data-table td:first-child {
  position: sticky;
  left: 0;
  z-index: 1;
  background: var(--surface);
}

.data-table tbody tr:nth-child(even) td:first-child {
  background: color-mix(in srgb, var(--surface) 95%, var(--accent) 5%);
}
```

## Connectors

### CSS Arrow (vertical, between stacked sections)
```css
.flow-arrow {
  display: flex;
  justify-content: center;
  align-items: center;
  gap: 8px;
  color: var(--text-dim);
  font-family: var(--font-mono);
  font-size: 12px;
  padding: 6px 0;
}

/* Down arrow via SVG icon */
.flow-arrow svg {
  width: 20px;
  height: 20px;
  fill: none;
  stroke: var(--border-bright);
  stroke-width: 2;
  stroke-linecap: round;
  stroke-linejoin: round;
}
```

Down arrow SVG (reuse inline):
```html
<svg viewBox="0 0 20 20"><path d="M10 4 L10 16 M6 12 L10 16 L14 12"/></svg>
```

### CSS Arrow (horizontal, between inline steps)
Use `::after` or a literal arrow character:
```css
.h-arrow::after {
  content: '→';
  color: var(--border-bright);
  font-size: 18px;
  padding: 0 4px;
}
```

### SVG Curved Connector (between arbitrary nodes)
For connections that aren't simple vertical/horizontal, use an absolutely positioned SVG overlay:
```html
<svg class="connectors" style="position:absolute;inset:0;width:100%;height:100%;pointer-events:none;">
  <path d="M 150,100 C 150,200 350,100 350,200" fill="none" stroke="var(--accent)" stroke-width="1.5" stroke-dasharray="4 3"/>
  <!-- Arrowhead -->
  <polygon points="348,195 352,205 356,195" fill="var(--accent)"/>
</svg>
```

Position the parent container as `position: relative` to scope the SVG overlay.

