---
tags: [hillclimb]
max_turns: 120
timeout_seconds: 3600
allowed_tools: [Read, Edit, Write, Glob, Grep, Agent, Skill, SendMessage, Bash]
---

hillclimb the README in this repo until three judges all rate it at least 8 out of 10 for clarity and usefulness to a first-time user. Use three Claude judges, not council. Cap it at 3 rounds. Don't change main.go.
Input files: README.md, main.go, go.mod
