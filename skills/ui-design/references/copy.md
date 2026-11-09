# Copy

Interface text is part of the design. Generated copy has tells as recognizable as purple gradients, and it costs trust the same way.

## UI copy (Operate mode)

- **Controls name their action and object**: "Restart service", "Export CSV" — not "Submit", "OK", "Proceed". The confirm button in a dialog repeats the verb from the title ("Delete 3 hosts?" → "Delete hosts").
- **Errors** say what happened and what to do, in the user's terms, next to where it happened: "Couldn't reach sensor-04 (timeout after 10s). Check it's powered and on the VPN, then retry." Never "An error occurred." Never blame the user.
- **Empty states**: what goes here + the one action that fills it. "No alerts in the last 24h." is a perfectly good empty state for an alert list — calm is information.
- **Labels**: sentence case, nouns, no trailing colons in stacked layouts. Units in headers ("Latency (ms)").
- **Numbers**: real precision for the task (CPU 37%, not 37.4821%); relative times with absolute on hover/title.
- **Tone**: plain, specific, short. Operators are busy; the interface is not a brand voice opportunity.

## Tells (lint: `copy-tells`)

Each banned item needs a replacement, not just a deletion — a ban with no substitute produces a different cliché.

| Tell | Why it reads as generated | Replace with |
|---|---|---|
| seamless, effortless, robust | Unfalsifiable adjectives | The specific property: "syncs in under 2s", "survives node loss" |
| unlock, elevate, empower, supercharge, revolutionize | Verbs that don't say what happens | The actual verb: "lets you", "cuts", "replaces", "shows" |
| delve, tapestry, landscape, ecosystem, journey | LLM-register vocabulary | Concrete nouns for the actual thing |
| leverage, synergy, cutting-edge, game-changer | Corporate filler | Delete, or name the capability |
| "In today's fast-paced world…", "Look no further", "Join thousands of…" | Stock openers | Start with what it is and who it's for |
| "Unlock the power of X", "Take your X to the next level" | Template slogans | One concrete outcome with a number if you have one |
| "The result? …" / "Here's the thing:" reveals | Manufactured suspense | State the result |
| Trailing "-ing" riders ("…, ensuring reliability") | Padding that restates | Cut, or make it its own sentence with a fact |
| Em-dash every sentence | Rhythm tell | Mix punctuation; most sentences need none |
| Triplets everywhere ("fast, simple, and secure") | Rhythm tell | Say the one that matters, with evidence |

Tells come in clusters. One "robust" isn't the problem; five tells in a hero is.

## Facts

Never invent product facts, metrics, customer names, quotes, or certifications. If the copy needs a number you don't have, leave a clearly marked TODO for the user rather than a plausible fake — a fake stat that survives to production is a real liability (and in regulated or defense contexts, a compliance one).
