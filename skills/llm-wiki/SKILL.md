---
name: llm-wiki
description: Build and maintain a personal LLM-curated wiki of atomic markdown pages with dense backlinks, structured frontmatter, and an llms.txt index — the Karpathy-style "wiki as substrate, LLM as curator" pattern. Use when the user asks to (1) initialize a personal wiki / knowledge base / second brain, (2) add or refine a wiki page on a concept, (3) distill the current conversation into atomic wiki notes, (4) audit a wiki for orphans / overlap / overlong pages, (5) split or merge wiki pages, (6) regenerate the wiki index or llms.txt, or (7) backlink-enrich an existing page. Also triggers on "/llm-wiki", "wiki this", "make atomic notes", "Zettelkasten", and any mention of an "LLM wiki" in the Karpathy sense (atomic pages + LLM curation, not Wikipedia-style topic articles or project documentation).
---

# LLM Wiki

## Overview

A personal wiki where humans and LLMs co-author atomic markdown pages connected by a dense link graph. The wiki — not the chat history — is the durable artifact. Each page captures one concept; pages reference each other by relative markdown links; an `index.md` and `llms.txt` make the wiki navigable to both readers and retrievers.

The skill provides the conventions, page template, and workflows for creating and maintaining such a wiki. It does **not** generate Wikipedia-style topic articles or refactor project documentation — those are different problems.

## Core principles

1. **Atomic pages.** One concept per page. If a page covers two ideas, split it.
2. **Dense backlinks.** Every concept reference is a link. A page in isolation is a smell.
3. **Plain markdown + git.** No proprietary tools. The repo is the wiki.
4. **Frontmatter is the index.** Aliases, tags, summary, updated-on — all in YAML at the top of each page so retrievers can use them.
5. **LLM as curator.** Humans capture; the LLM splits, merges, links, indexes, and audits. Don't ask the user to manage the link graph by hand.
6. **Append + refine, don't rewrite.** Pages grow over time. Preserve prior phrasing when it still holds.
7. **Flat namespace.** All atomic pages live in `pages/`. Categorization happens via tags and links, not folders.

If a request violates these principles (e.g., "make me one long page covering X, Y, Z"), push back briefly and propose the atomic split.

## Workflow selection

Read **references/workflows.md** when executing any of these. It contains the procedural detail, prompts, and decision rules.

| User intent | Workflow |
|---|---|
| "Start / set up a wiki at `<path>`" | **init** — scaffold `pages/`, `index.md`, `llms.txt`, `CONVENTIONS.md` |
| "Add a page on X" / "wiki this concept" | **new page** — create one atomic page; link to existing related pages |
| "Distill this conversation" / "wiki our chat" | **distill** — extract durable atomic notes from current context; file them |
| "Audit my wiki" / "find orphans / duplicates" | **audit** — report orphans, overlong pages, overlapping pages, broken links |
| "Split this page" | **split** — break into atomic pages, preserve and rewire links |
| "Merge these pages" | **merge** — combine into one, redirect inbound links |
| "Backlink this" / "find missing links" | **link enrichment** — find concept mentions that should be links |
| "Regenerate index" / "update llms.txt" | **reindex** — rebuild `index.md` and `llms.txt` from frontmatter |

If the user has not yet pointed at a wiki directory, ask once for the path. After that, remember it for the rest of the session.

## Conventions (page format)

Full schema and rationale in **references/conventions.md**. Quick reference:

- Filename: `kebab-case-of-canonical-title.md` in `pages/`.
- Frontmatter (YAML): `title`, `aliases` (list), `tags` (list), `summary` (one line), `created`, `updated`. Optional: `status` (`stub` | `draft` | `stable`), `sources` (list of URLs/citations).
- Body sections (in order): one-paragraph definition → key points → relations → examples → open questions → sources.
- Links: relative markdown links to other pages, e.g. `[backpropagation](backpropagation.md)`. Wikilink-style `[[...]]` is **not** used (the goal is plain markdown that renders on any host).
- A page MUST cite the prompt or context that produced it in the `sources` field when distilled from a conversation. Otherwise leave `sources: []`.

The page template lives at **assets/page-template.md** — copy it for every new page.

## Operating notes

- **Read `CONVENTIONS.md` in the target wiki before writing.** If the user's wiki has local conventions that differ from the defaults, the local file wins.
- **Never blow away an existing page.** Edits are append-and-refine. If a section needs rewriting, edit it; do not regenerate the whole file.
- **Update `updated:` in frontmatter** on every edit. Leave `created:` alone.
- **Commit at the end of each workflow** unless the user is in a directory that isn't a git repo. One commit per logical operation (one page added, one audit fix applied), not one giant commit per session.
- **For distill, prefer fewer larger atomic notes over many tiny stubs.** A page with two sentences is rarely worth keeping; either flesh it out or fold it into a related page.
- **When linking, prefer existing pages over new ones.** If a concept is referenced and no page exists, create a `status: stub` page rather than leaving the link broken.

## When NOT to use this skill

- The user wants Wikipedia-style explanatory articles ("explain backprop with sections and math") — that's a writing task, not a wiki workflow.
- The user wants to restructure existing project documentation (READMEs, dev docs) — different goal, different conventions.
- The user wants generic note-taking with no structure — overkill; just have them write markdown.
- The user is asking about Claude's persistent memory system — that's the `auto memory` feature, not a wiki.
