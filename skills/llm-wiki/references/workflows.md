# Workflows

Procedural detail for each LLM-wiki workflow. SKILL.md selects the workflow; this file executes it.

## Contents

- [init — scaffold a new wiki](#init--scaffold-a-new-wiki)
- [new page — add an atomic page](#new-page--add-an-atomic-page)
- [distill — convert conversation into atomic pages](#distill--convert-conversation-into-atomic-pages)
- [audit — health check the wiki](#audit--health-check-the-wiki)
- [split — break a non-atomic page into atomic pages](#split--break-a-non-atomic-page-into-atomic-pages)
- [merge — combine overlapping pages](#merge--combine-overlapping-pages)
- [link enrichment — backlink an existing page](#link-enrichment--backlink-an-existing-page)
- [reindex — regenerate index.md and llms.txt](#reindex--regenerate-indexmd-and-llmstxt)

---

## init — scaffold a new wiki

Trigger: "start a wiki at `<path>`", "set up a personal wiki", "/llm-wiki init".

Steps:

1. Confirm the path. If it doesn't exist, create it. If it exists and isn't empty, ask before scaffolding.
2. Create:
   - `pages/` (empty directory; add a `.gitkeep`)
   - `README.md` — one-paragraph "this is my LLM-curated wiki" intro plus a pointer to `index.md`. Hand-edited from here on.
   - `CONVENTIONS.md` — copy the contents of `references/conventions.md`; users can edit it to override defaults.
   - `index.md` — empty shell with an "intro" placeholder paragraph and the "By tag / All pages / Recently updated" headings.
   - `llms.txt` — the header (`# <Wiki name>` + one-line description) and empty `## Pages` section.
3. `git init` if not already a repo. Initial commit: "init: scaffold LLM wiki".
4. Tell the user: where the wiki lives, what was created, and what to do next (`add a page on X` is the most common next step).

Ask before adding any starter pages. Don't seed the wiki with example content unless the user explicitly asks.

## new page — add an atomic page

Trigger: "add a page on X", "wiki this concept", "/llm-wiki new <title>".

Steps:

1. **Decide canonical title.** If the user gave a phrase, pick the canonical noun phrase. Check `pages/` for an existing page or alias collision first (`grep -l "^aliases:.*<term>" pages/*.md` and `ls pages/` with fuzzy matching). If a page exists, switch to **link enrichment** instead.
2. **Find related existing pages** before writing. Search `pages/` for the concept term and for terms that are likely prerequisites or consequences. Note the matches — they become the `## Relations` section.
3. **Copy the page template** from `assets/page-template.md`. Fill in:
   - `title`, `aliases`, `tags`, `summary`
   - `created` and `updated` to today's date (the user's `currentDate` if known)
   - `status: stub` if writing less than the definition; `draft` if filling in key points; `stable` only if the user explicitly says so
   - `sources`: cite the prompt or conversation if distilled, else `[]`
4. **Write the body** following the section order in conventions.md. Sections that have nothing to say can be omitted entirely; don't leave empty headings.
5. **Add at least one link** in `## Relations` to an existing page if one exists. If no related pages exist yet, that's fine — note it in `## Open questions`.
6. **Update `index.md` and `llms.txt`** — append the new entry (or run reindex).
7. **Commit** with message `add: <title>`.

If the user provided rich detail, write a fuller draft. If they gave a one-line prompt, write a stub plus a note in `## Open questions` listing what's missing.

## distill — convert conversation into atomic pages

Trigger: "wiki our conversation", "distill this into the wiki", "make notes from this chat".

This is the highest-value workflow but also the easiest to do badly. The failure mode is "create 20 stub pages that all say one sentence each." Avoid that.

Steps:

1. **Scan the conversation** for durable insights — things that will still be true and useful in 6 months. Discard: task-specific debugging context, code that's already in the repo, ephemeral state.
2. **Group findings into concepts.** Each concept gets one atomic page. Aim for 1–4 pages per distill operation. If you find yourself listing 10+, you're being too granular — collapse related findings into one richer page.
3. **For each concept**, check whether a page already exists (`ls pages/` + grep on aliases). If yes, **append to the existing page** rather than creating a new one. Distill into the existing page's `## Key points` or `## Examples` sections.
4. **For new pages**, follow the **new page** workflow above. Set `sources: ["conversation: <one-line description of what was discussed>"]`.
5. **Cross-link aggressively.** Distilled pages are usually about related ideas — the link density between them should be high. Each new page should link to at least one other page from this distill batch and one existing page.
6. **Show the user the diff before committing.** Distill makes more changes than other workflows; a preview catches over-eager note creation.
7. **Commit** with message `distill: <one-line summary of what was captured>`.

**Quality bar:** if a page from a distill operation reads like it was extracted by a script — bullet list with no context, no links, no examples — the distill was too shallow. Rewrite or drop the page.

## audit — health check the wiki

Trigger: "audit my wiki", "find orphans / duplicates / broken links", "/llm-wiki audit".

Report-only by default. Don't make changes; surface findings and let the user decide what to fix.

Check for:

1. **Broken links** — every link in `pages/*.md` should resolve to an existing file. Grep all markdown link targets, verify each file exists.
2. **Orphan pages** — pages with no inbound links from other pages. These are dead ends. (Index/llms.txt links don't count as inbound.)
3. **Stub pages older than 30 days** — `status: stub` with `created` more than 30 days ago. Either flesh out or delete.
4. **Overlong pages** — anything > 400 lines. Candidate for split.
5. **Likely overlap** — pages with similar titles, overlapping aliases, or shared tag sets. Use string similarity on titles and Jaccard similarity on tag/alias sets. Candidate for merge.
6. **Missing frontmatter fields** — required fields (`title`, `summary`, `created`, `updated`) absent or empty.
7. **Tags used by only one page** — a tag with a single page isn't pulling its weight. Either re-tag or accept it'll grow.
8. **`updated` older than `created`** — indicates a malformed frontmatter.
9. **Aliases that collide** — same alias on multiple pages. Forces a rename.
10. **Pages not listed in `index.md` or `llms.txt`** — stale index. Suggest reindex.

Output: a structured report, grouped by severity. Don't write fixes inline — propose them and wait.

## split — break a non-atomic page into atomic pages

Trigger: "split this page", "this page is too big", audit suggesting a split.

Steps:

1. **Identify the split boundaries.** Usually the H2 sections are the candidates. Each candidate atomic page must satisfy the atomicity rules in conventions.md.
2. **Propose the split before doing it.** Show the user: "page X will become pages A, B, C with these titles and summaries." Wait for confirmation if there's any ambiguity.
3. **Create the new pages** following the **new page** workflow. Each gets its own frontmatter, full body structure, and links back to the source page's other split products (so they form a small cluster).
4. **Rewrite the original page** to be either (a) a stub that links to the new pages, with `redirect-to:` if it becomes pure redirect, or (b) a higher-level overview page that links to the split products. Usually (a).
5. **Update inbound links.** Grep `pages/` for links to the original page. Each one needs to be re-pointed to whichever new page is now the right target. This is the highest-risk step — don't skip it.
6. **Reindex.**
7. **Commit** with message `split: <original> into <new1>, <new2>, ...`.

## merge — combine overlapping pages

Trigger: "merge these pages", audit suggesting a merge.

Steps:

1. **Pick the canonical page.** Usually the one with the canonical title, longer body, or more inbound links. The other becomes the redirect stub.
2. **Combine the bodies.** Preserve unique content from both. When sections overlap (both have `## Key points`), merge the bullets and dedupe. When phrasing differs, prefer the clearer phrasing — don't blindly concatenate.
3. **Combine frontmatter.** Union the `aliases` and `tags` lists. Take the earlier `created` date. Update `updated` to today. Take the higher `status` (stable > draft > stub).
4. **Update the non-canonical page** to a redirect stub: only frontmatter with `title`, `redirect-to: <canonical>.md`, original `aliases` preserved, and an empty body.
5. **Update inbound links.** Grep for links to the non-canonical page; rewrite to point at the canonical page.
6. **Reindex.**
7. **Commit** with message `merge: <non-canonical> into <canonical>`.

## link enrichment — backlink an existing page

Trigger: "find missing links for X", "backlink this", a new page is created and existing pages should reference it.

Steps:

1. **Read the target page.** Collect its title and all aliases — these are the terms that, if they appear unlinked in other pages, are candidates for backlinking.
2. **Grep all other pages** for those terms. For each match, check whether the term is already inside a markdown link. If not, it's a candidate.
3. **Filter candidates.**
   - Only link the first meaningful occurrence in each page.
   - Don't link in code blocks or fenced sections.
   - Don't link in headings (they get auto-anchors; that's enough).
   - Don't link if the surrounding sentence makes it clear the term is being used in a different sense.
4. **Propose the edits before applying.** Show the user the list of candidate sites; let them veto any.
5. **Apply edits.** Use a precise replacement (match the surrounding context, not just the term) so you don't change other occurrences accidentally.
6. **Update `updated:` on every modified page.**
7. **Commit** with message `links: backlink <target>`.

## reindex — regenerate index.md and llms.txt

Trigger: "regenerate index", "update llms.txt", end of any workflow that added/removed/renamed pages.

Steps:

1. **Walk `pages/*.md`.** For each, parse frontmatter and extract `title`, `summary`, `tags`, `updated`, `status`, `redirect-to`.
2. **Skip redirect pages** in the visible indexes (they're not browseable destinations).
3. **Rebuild `llms.txt`:**
   - Preserve the header (`# <Wiki name>` + one-line description) — read it from the existing file, don't regenerate.
   - Replace the `## Pages` section with one bullet per non-redirect page, sorted by title:
     `- [<title>](pages/<filename>): <summary>`
4. **Rebuild `index.md`:**
   - Preserve the H1 and the intro paragraph (read existing).
   - Rebuild `## By tag` — group pages by tag, sort tags alphabetically, sort pages within each tag alphabetically.
   - Rebuild `## All pages` — alphabetical by title.
   - Rebuild `## Recently updated` — top 10 pages by `updated`, descending.
5. **Don't commit on its own** if reindex is part of another workflow. If reindex is the only operation, commit with message `reindex`.

Reindex is mechanical. If frontmatter is malformed, surface the bad page and stop — don't silently skip.
