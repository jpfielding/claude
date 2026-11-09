---
tags: [ui-design]
max_turns: 40
timeout_seconds: 900
allowed_tools: [Read, Glob, Grep, Skill, Write, Edit, Bash, Agent]
---

I need a fleet status page for our Go service (html/template + htmx, no React). It should show a table of ~40 field sensors with name, site, last seen, battery %, firmware version and status (ok/degraded/offline), auto-refreshing every 10s, with a filter by status. Operators keep it open all shift on laptops. Write the template(s), the CSS, and a small main.go handler with fake data into ./sensorui.
