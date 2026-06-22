# Wiki Conventions

Detailed conventions for the LLM wiki. SKILL.md has the summary; this file is the source of truth when conventions conflict or edge cases arise. A wiki's own `CONVENTIONS.md` (if present) overrides this file.

## Contents

- [Repository layout](#repository-layout)
- [Filename rules](#filename-rules)
- [Frontmatter schema](#frontmatter-schema)
- [Body structure](#body-structure)
- [Linking rules](#linking-rules)
- [Tagging](#tagging)
- [Atomicity rules](#atomicity-rules)
- [llms.txt format](#llmstxt-format)
- [index.md format](#indexmd-format)

## Repository layout

```
my-wiki/
├── CONVENTIONS.md     # local conventions (optional; overrides defaults)
├── README.md          # human-facing intro, not part of the page graph
├── index.md           # generated table of contents, grouped by tag
├── llms.txt           # generated machine-readable index for retrievers
└── pages/
    ├── backpropagation.md
    ├── chain-rule.md
    ├── gradient-descent.md
    └── ...
```

No subfolders inside `pages/`. The link graph and tags do the work that folders would. Folders bias toward premature categorization; tags are cheap to add and remove.

## Filename rules

- `kebab-case-of-canonical-title.md`.
- ASCII only. Strip diacritics. No spaces, no underscores.
- Acronyms lowercased: `gpu.md`, not `GPU.md`.
- Versions go after the concept: `transformer-architecture.md`, not `2017-transformer.md`.
- If a title would collide, disambiguate parenthetically inside the title and in the filename: `bias-statistics.md`, `bias-ml.md`.
- Renames are allowed but rare. When renaming, leave a one-line redirect stub at the old filename: `redirect-to: new-name.md` in frontmatter.

## Frontmatter schema

YAML between two `---` delimiters at the very top of the file. Every page must have it.

```yaml
---
title: Backpropagation
aliases: [backprop, reverse-mode autodiff]
tags: [neural-networks, training, calculus]
summary: Gradient computation in neural networks via the chain rule applied in reverse topological order.
created: 2026-05-11
updated: 2026-05-11
status: draft
sources: []
---
```

Field rules:

- `title` — required. Human-readable canonical title (not the filename). Title case for proper nouns, sentence case for everything else.
- `aliases` — list of alternate names a reader might search for. Lowercase. Include common abbreviations, full forms, and historical names.
- `tags` — list, lowercase, kebab-case. 1–4 tags. Tags are for cross-cutting categorization (`neural-networks`, `optimization`, `linear-algebra`); they are not a folder system.
- `summary` — one sentence, under 200 chars, suitable for use in `llms.txt` and link previews. Describes the concept, not the page's purpose ("Backpropagation is …", not "This page covers …").
- `created` — ISO date. Set once; never modify.
- `updated` — ISO date. Update on every meaningful edit (not on typo fixes).
- `status` — `stub` (one-paragraph placeholder), `draft` (has content but rough), `stable` (reviewed and complete enough). Defaults to `draft`.
- `sources` — list of strings: URLs, citations, or `"conversation: <one-line context>"` when the page was distilled from chat.
- `redirect-to` — used only on rename stubs. If present, the page's body should be empty.

## Body structure

Sections in this order, each as an H2. Omit any that aren't relevant; keep the order.

1. **Definition** — one paragraph, no heading needed (the H1 from the title is implicit). Lead with what the concept *is*, in plain language. Aim for ~3 sentences.
2. **`## Key points`** — bulleted list of the load-bearing facts. Each bullet stands alone; no nested bullets unless unavoidable.
3. **`## Relations`** — bulleted list of how this concept connects to others. Each bullet links to another page: `- [chain rule](chain-rule.md) is the mathematical foundation.` This is where dense backlinking shows up.
4. **`## Examples`** — concrete examples, code snippets, or worked problems. Code blocks fenced and language-tagged.
5. **`## Open questions`** — things the page doesn't answer, dead-ends, areas where the author is unsure. This is a *feature* — it invites future refinement and is honest about uncertainty.
6. **`## Sources`** — same content as the frontmatter `sources` field, but in human-readable list form. Optional if `sources: []`.

A page that omits everything except the definition is fine for a stub. A page longer than ~400 lines should probably be split.

## Linking rules

- Use relative markdown links: `[chain rule](chain-rule.md)`.
- Do **not** use wikilink syntax (`[[chain rule]]`). The wiki must render correctly on GitHub, plain markdown viewers, and static site generators with no preprocessing.
- Link the first meaningful mention of a concept in each page. Subsequent mentions can be plain text — over-linking is noise.
- Anchor links inside a page: `[see open questions](#open-questions)`. Use sparingly.
- External links use the same markdown syntax; they don't count toward the link graph.

## Tagging

- Tags are flat, lowercase, kebab-case.
- 1–4 tags per page. More than 4 is usually a sign the page covers too much.
- Common tag categories: domain (`neural-networks`, `linux`, `cryptography`), kind (`algorithm`, `data-structure`, `tool`, `concept`), context (`production`, `historical`, `theoretical`).
- A new tag should be used by at least 2 pages within a week, or it's not pulling its weight — re-tag with an existing tag.

## Atomicity rules

A page is atomic when:

- It can be summarized in one sentence (the `summary` frontmatter field).
- A reader who knows the prerequisites can read the page in under 3 minutes.
- Removing any section would lose information *about that concept*, not information about a related concept.

Signs a page is not atomic:

- The body has two H2 sections of comparable size that could each become their own page.
- The `summary` requires "and" to connect two unrelated ideas.
- Tags span unrelated domains (e.g., both `cryptography` and `web-design` on one page).
- The page is over 400 lines.

When in doubt, split. Two well-linked atomic pages are better than one sprawling page.

## llms.txt format

`llms.txt` is the machine-readable index at the wiki root. Format:

```
# <Wiki name>

> <One-line description of what this wiki is about.>

## Pages

- [backpropagation](pages/backpropagation.md): Gradient computation in neural networks via the chain rule applied in reverse topological order.
- [chain rule](pages/chain-rule.md): Derivative of a composition of functions; the foundation of backpropagation.
- ...
```

One line per page. The bullet is `[title](relative-path): summary`. Sort alphabetically by title. Regenerated, not hand-edited.

## index.md format

`index.md` is the human-readable table of contents. Format:

```markdown
# <Wiki name>

<one-paragraph intro, hand-written, edited rarely>

## By tag

### neural-networks

- [Backpropagation](pages/backpropagation.md)
- [Transformer architecture](pages/transformer-architecture.md)

### optimization

- [Gradient descent](pages/gradient-descent.md)
- ...

## All pages

- [Backpropagation](pages/backpropagation.md)
- ...

## Recently updated

- 2026-05-11 — [Backpropagation](pages/backpropagation.md)
- ...
```

The intro paragraph is preserved across regenerations. Everything else is rebuilt from frontmatter.
