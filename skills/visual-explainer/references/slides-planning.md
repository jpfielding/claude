# Planning a Slide Deck

**Before generating**, also read `./css-layout.md`, `./css-motion.md`, `./css-components.md` for shared patterns (Mermaid zoom, overflow protection, depth tiers, badges) and `./libraries.md` for Mermaid theming, Chart.js, and font pairings.

How to plan a deck from a source document and govern composition, readability, and content density. Read this first when starting a slide deck.

## Contents

- [Planning a Deck from a Source Document](#planning-a-deck-from-a-source-document)
- [Compositional Variety](#compositional-variety)
- [Presentation Readability](#presentation-readability)
- [Content Density Limits](#content-density-limits)

## Planning a Deck from a Source Document

When converting a plan, spec, review, or any structured document into slides, follow this process before writing any HTML. Skipping it leads to polished-looking decks that silently drop 30–40% of the source material.

**Step 1 — Inventory the source.** Read the entire source document and enumerate every section, subsection, card, table row, decision, specification, collapsible detail, and footnote. Count them. A plan with 7 sections, 6 decision cards, a 7-row file table, 4 presets, 6 technique guides, and an engine spec with 3 sub-specs and 2 collapsibles is ~25 distinct content items that all need slide real estate.

**Step 2 — Map source to slides.** Assign each inventory item to one or more slides. Every item must appear somewhere. Rules:
- If a section has 6 decisions, all 6 need slides — not the 2 that fit on one split slide.
- If a table has 7 rows, all 7 rows show up.
- Collapsible/expandable details in the source are not optional in the deck — they become their own slides.
- Subsections with multiple cards (e.g., "6 Visual Technique cards") may need 2–3 slides to cover at readable density.
- Each plan section typically needs a divider slide + 1–3 content slides depending on density.

**Step 3 — Choose layouts.** For each planned slide, pick a slide type and spatial composition. Vary across the sequence (see Compositional Variety below). This is where narrative pacing happens — alternate dense slides with sparse ones.

**Step 4 — Plan images.** Run `which surf`. If surf-cli is available, plan 2–4 generated images for the deck. At minimum, target the **title slide** (16:9 background that sets the visual tone) and **one full-bleed slide** (immersive background for a key moment). Content slides with conceptual topics also benefit from a 1:1 illustration in the aside area. Generate these images early — before writing HTML — so you can embed them as base64 data URIs. See the Proactive Imagery section below for the full workflow. If surf isn't available, degrade to CSS gradients and SVG decorations — note the fallback in a comment but don't error.

**Step 5 — Verify before writing HTML.** Scan the inventory from Step 1. Is anything unmapped? Would a reader of the source document notice something missing from the deck? If yes, add slides. A source document with 7 sections typically produces 18–25 slides, not 10–13.

**The test:** After generating the deck, a reader who has never seen the source document should be able to reconstruct every major point from the slides alone. If they'd miss entire sections, the deck is incomplete.


## Compositional Variety

Consecutive slides must vary their spatial approach. Three centered slides in a row means push one off-axis.

**Composition patterns to alternate between:**
- Centered (title slides, quotes)
- Left-heavy: content on the left 60%, breathing room on the right
- Right-heavy: content on the right 60%, visual or whitespace on the left
- Edge-aligned: content pushed to bottom or top, large empty space opposite
- Split: two distinct panels filling the viewport
- Full-bleed: background dominates, minimal overlaid text

The agent should plan the slide sequence considering layout rhythm, not just content order. When outlining a deck, assign a composition to each slide before writing HTML.

## Presentation Readability

Slides get projected, screen-shared, viewed at distance. Design accordingly:

- **Minimum body text: 16px.** Nothing smaller except labels and captions.
- **One focal point per slide.** Not three competing elements.
- **Higher contrast than pages.** Dimmed text (`--text-dim`) should still be easily readable at distance — test against the background.
- **Nav chrome opacity.** Dots and progress bar must be visible on any slide background (light or dark) without being distracting. Use the backdrop blur or text-shadow approach from the Nav Chrome section.
- **Simpler Mermaid diagrams.** Max 8–10 nodes, 18px+ labels, 2px+ edges. The diagram should be readable without zoom at presentation distance. Zoom controls remain available for detail inspection.

## Content Density Limits

Each slide must fit in exactly 100dvh. If content exceeds these limits, the agent splits across multiple slides — never scrolls within a slide.

| Slide type | Max content |
|-----------|-------------|
| Title | 1 heading + 1 subtitle |
| Section Divider | 1 number + 1 heading + optional subhead |
| Content | 1 heading + 5–6 bullets (max 2 lines each) |
| Split | 1 heading + 2 panels, each follows its inner type's limits |
| Diagram | 1 heading + 1 Mermaid diagram (max 8–10 nodes) |
| Dashboard | 1 heading + 6 KPI cards. Hero values ≤6 chars (numbers, %, short labels). Longer strings belong in the label row. |
| Table | 1 heading + 8 rows; overflow paginates to next slide |
| Code | 1 heading + 10 lines of code |
| Quote | 1 short quote (~25 words / ~150 chars max) + 1 attribution. Longer quotes are content slides, not quote slides. |
| Full-Bleed | 1 heading + 1 subtitle over background |


