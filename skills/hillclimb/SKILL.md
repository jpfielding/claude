---
name: hillclimb
description: Trigger this if the user says 'hillclimb' or 'hill climb'. Iterates work through blind worker subagents and independent judges until every judge scores at or above a target the workers never see. Often combined with council (as the judges) and full-send, full-stop, or file-pr (for shipping).
---

# Hillclimb

Improve an artifact (UI, code, docs, a design) in rounds until every judge
scores it at or above the user's target. The target is a secret held by you,
the controller. Workers and judges never learn it, so workers cannot aim for
"just enough" and judges cannot round up to end the loop.

Run this from the main session. Subagents cannot start their own subagents,
so the controller must be the thread that spawns workers and judges.

## Roles

Keep the roles disjoint. Most loop failures come from one role holding two
powers: a worker that sees the scorecard learns the scorer, and a judge that
sees the history grades the improvement instead of the result.

- **Controller (you):** holds the target, the ledger, and the stop rule.
  Runs gates, collects evidence, decides accept or revert, writes the report.
- **Worker:** a fresh subagent each round. Edits the artifact. Sees the goal,
  the quality bar in words, constraints, and a defect list. Never sees
  scores, the target, judge identities, the ledger, or the number of rounds.
- **Judge:** a fresh subagent each round. Read-only. Sees the goal, the
  rubric, and the evidence. Never sees the target, prior scores, prior
  critiques, or the worker's description of what it did.

## Round 0: setup

1. **Parse the request.** Extract the goal, the target score, the judges, and
   constraints. Example: "client and retained views are kiosk (ease of use);
   attorney and staff views are terminal (speed for expert users)."
2. **Judges.** If the user says council, use the council skill's delegates as
   judges and follow its rules on providers and scope. Otherwise use the
   council panel if reachable. If not, use three judges on the strongest
   available model, and note in the report that the panel lacks provider
   diversity. Judges from the worker's model family tend to favor its output.
3. **Evidence spec.** Decide exactly what judges will see, and freeze it for
   every round, so scores stay comparable. For UI: the same views, viewports,
   seed data, and app state each time. For code: the same diff base and
   check output.
4. **Gates.** List the deterministic checks that apply (build, typecheck,
   lint, tests, a launch that does not crash). A failing gate always beats a
   passing judge.
5. **Ledger.** Keep it in the scratchpad, outside the repo. Never put scores
   in the repo, commit messages, branch names, or the PR until the final
   report. A fresh worker that runs `git log` or `gh pr view` must find
   nothing about scores.
6. **Baseline.** Collect evidence and judge the starting state. This is the
   first "best" version, and it shows whether later rounds actually climb.

## Each round

1. **Brief a fresh worker.** Default to a cheaper model than the controller
   (for example Sonnet), per the user's advisor strategy. Use this brief:

   ```
   Goal: <goal, in the user's words>
   Quality bar: <the bar in words, e.g. "polished enough that a demanding
   design reviewer would call it excellent">. No number.
   Constraints: <constraints, scope limits, files not to touch>
   Current defects (most important first):
   - <concrete defect, where it is, what is wrong>
   Deliver finished work. Do not change tests, fixtures, the screenshot or
   evidence harness, or seed data. Do not remove features or content to
   avoid a defect. Report what you changed.
   ```

   Round 1 has no defect list, so the worker works from the goal alone.
   Leave out rounds, scores, judges, and any hint that a stop rule exists.

2. **Run gates.** On failure, send the worker the exact error and retry, up
   to 3 times. Gate retries do not count as rounds.
3. **Check integrity.** Read the diff before judging. Revert changes to
   tests, fixtures, the evidence harness, or seed data. Also revert removed
   functionality and out-of-scope edits. If you reverted something material,
   send it back to the worker instead of judging.
4. **Collect evidence** per the frozen spec. Use the real artifact, never the
   worker's summary.
5. **Judge in parallel** with fresh judges. Use this brief:

   ```
   Review only. Do not edit anything.
   Goal: <goal> Constraints: <constraints>
   Evidence: <paths or attachments>
   Reason first, then score. List concrete defects, most important first,
   each with location and why it matters. Then give one integer score 1-10:
   10 = you cannot name a meaningful improvement
   9  = only minor nits remain
   7-8 = solid, with clear defects
   5-6 = notable problems
   1-4 = broken or far from the goal
   Your score must match your defect list. Length or effort is not quality.
   Reply: DEFECTS: ... then SCORE: <n>
   ```

6. **Decide.**
   - **Pass:** every judge reaches the target in this round. Stop.
   - **Accept:** the lowest score is higher than the best version's lowest,
     or the same with a higher mean. Commit with a neutral message (no
     scores). This is now the best version.
   - **Revert:** otherwise, reset to the best version. Pass the next worker
     the best version's defects plus a note about what the rejected attempt
     broke, written as defects.
7. **Distill critiques** into the next defect list. Merge duplicates. Rewrite
   each one in your own words as a concrete problem. Drop scores, judge
   names, and quoted phrasing. Drop suggestions outside the goal's scope, and
   list them as follow-ups. Workers that get raw judge text start echoing the
   judges instead of fixing the work.

## Watch for gaming

- **Canary jump:** if the lowest score rises by 3 or more in one round,
  re-read the diff before accepting. Look for harness edits, hidden or
  deleted content, or views that are no longer reachable.
- Never lower the target, reword the rubric, or swap judges mid-run to get a
  pass. Never tell judges the work is close, or that this is a final round.
- If a judge's score contradicts its own defect list (a 9 with major
  defects), ask that judge once to reconcile them. Record both answers.

## Stop

- **Pass:** all judges at or above the target in the same round.
- **Plateau:** two accepted or reverted rounds in a row with no gain in the
  lowest score. Escalate once: run the next round with the strongest worker
  model. If it still does not gain, stop.
- **Cap:** 6 judged rounds after the baseline, unless the user set a
  different cap.

Plateau and cap are not passes. Leave the best version in place and report
the gap.

## Report

Give a final score table in council's format, with one column per round,
baseline first. If the work is in a PR, put it in the description at the end
(per council). Mark rounds that were reverted.

```markdown
## Hillclimb

Target: 9/10 · Result: passed in round 4 · Best: <sha>

| Judge | Base | R1 | R2 (reverted) | R3 | R4 |
| --- | --- | --- | --- | --- | --- |
| Claude Opus | 5 | 7 | 6 | 8 | 9 |
```

Also report: outcome (pass, plateau, or cap), the remaining defects if not
passed, integrity reverts, gate failures, judge contradictions, and declined
out-of-scope suggestions. If a judge gave no score, write `no score`.

## Composing with other skills

- **council:** supplies the judge panel and the scope rule. Its "fix known
  issues before audit" step is covered by gates and the integrity check.
- **full-send, full-stop, file-pr:** own the branch, PR, and merge. Hillclimb
  owns only the loop and the score table. If the user asks for a PR up front,
  create it with neutral text and add the table at the end.
