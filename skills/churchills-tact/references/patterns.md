# Rhetorical Patterns for Diplomatic Technical Communication

## Table of Contents
- Core Techniques
- Pattern: Redirect a Proposal
- Pattern: Pushback on Architecture/Design
- Pattern: Credit Work While Recommending Change
- Pattern: Summarize Disagreement in Documentation
- Pattern: Decline a Request
- Anti-Patterns to Avoid

---

## Core Techniques

### 1. Lead with Genuine Acknowledgment
Find the specific thing they did well. Not "great work" — name what was great and why it mattered.

**Weak:** "Thanks for the proposal."
**Strong:** "The latency analysis in your proposal surfaced a bottleneck we hadn't quantified before."

### 2. Bridge with Shared Goals
Frame your point as extending their thinking, not replacing it.

**Weak:** "But I think we should do X instead."
**Strong:** "Building on that analysis, one path that might push latency even lower..."

### 3. Use Questions to Redirect
A well-placed question lets someone arrive at your conclusion themselves.

**Weak:** "This won't scale past 10k connections."
**Strong:** "Have we modeled what happens at 10k concurrent connections? I want to make sure we're not surprised."

### 4. Make the Logic Do the Heavy Lifting
State evidence and constraints. Let the conclusion follow naturally rather than asserting it.

**Weak:** "Your approach is wrong because..."
**Strong:** "Given the SLA requires 99.95% uptime and this path introduces a single point of failure..."

### 5. Offer Partnership, Not Correction
Position yourself as joining their effort, not overriding it.

**Weak:** "You should change the retry logic."
**Strong:** "Want to pair on the retry logic? I ran into something similar last quarter that might save us time."

---

## Pattern: Redirect a Proposal

**Situation:** Someone proposes approach A. You believe approach B is better.

**Structure:**
1. Name the specific value in their proposal
2. Introduce a constraint or consideration (not a flaw)
3. Present B as a response to that constraint
4. Invite collaboration on the path forward

**Before:**
> I don't think the message queue approach will work here. We should use event sourcing instead.

**After:**
> The message queue design handles the decoupling problem cleanly — that was the right instinct. One thing I've been turning over: with our current retention requirements (90 days, audit-ready), we'd need to bolt on persistence anyway. Event sourcing might give us that for free while keeping the decoupling you identified. Worth exploring together?

---

## Pattern: Pushback on Architecture/Design

**Situation:** A design has a flaw you need to flag without diminishing the author.

**Structure:**
1. Acknowledge the design's strengths with specifics
2. Raise the concern as a question or constraint
3. Suggest a direction that preserves their good decisions
4. Frame next steps as collaborative

**Before:**
> This design doesn't account for failure modes. We need circuit breakers and the retry strategy is too aggressive.

**After:**
> The service boundary definitions here are sharp — especially the separation between ingestion and processing. As I was tracing the failure paths, I wondered: if the downstream enrichment service goes hard-down for 30 minutes, how do we want ingestion to behave? I think we can keep this architecture intact and add a circuit breaker at that boundary. Happy to sketch out the failure matrix if that'd help.

---

## Pattern: Credit Work While Recommending Change

**Situation:** Someone's work product needs significant changes but the effort was real.

**Structure:**
1. Name what their work accomplished or revealed
2. Explain what changed (new requirement, new data, shifted priority)
3. Propose the change as evolution, not replacement
4. Preserve their contribution in the narrative

**Before:**
> We need to rewrite the caching layer. The current implementation has too many edge cases.

**After:**
> The caching layer Alex built handled our traffic patterns well through the migration — it kept us stable when we needed stability. Now that we're seeing the new access patterns from the mobile clients, some of those original assumptions don't hold. I'd like to evolve the cache strategy to account for the higher read-to-write ratio. Alex, since you know the current implementation best, want to lead the redesign?

---

## Pattern: Summarize Disagreement in Documentation

**Situation:** Writing up a decision where not everyone agreed.

**Structure:**
1. Present all positions as reasoned responses to real tradeoffs
2. Name what each perspective valued (not who was right/wrong)
3. State the decision and the deciding factors
4. Acknowledge what was traded away

**Before:**
> After some debate, we decided to go with Postgres. Some people wanted DynamoDB but that would have been more complex.

**After:**
> Two viable approaches emerged: DynamoDB (optimizing for write throughput and operational simplicity at scale) and Postgres (optimizing for query flexibility and team familiarity). Both addressed the core requirement. We chose Postgres because our current query patterns are exploratory and evolving — schema flexibility reduces iteration cost at this stage. We're accepting the tradeoff of managing write scaling ourselves, with a documented threshold for revisiting the decision at 50k writes/sec.

---

## Pattern: Decline a Request

**Situation:** You need to say no to a feature, timeline, or approach.

**Structure:**
1. Show you understood the need behind the request
2. Explain the constraint honestly
3. Offer what you can do
4. Keep the door open

**Before:**
> We can't add that feature this sprint. We're already overcommitted.

**After:**
> I can see why real-time notifications would unblock the sales team — that's a real pain point. Right now the team is deep in the auth migration, and splitting focus would put both deliverables at risk. What I can do: I'll slot notifications as the top candidate for next sprint planning, and in the meantime we could set up a lightweight polling endpoint that covers the most urgent case. Would that bridge the gap?

---

## Anti-Patterns to Avoid

| Anti-Pattern | Why It Fails | Instead |
|---|---|---|
| "No offense, but..." | Signals offense is coming | Drop the preamble, lead with acknowledgment |
| "With all due respect..." | Universally reads as disrespect | Be genuinely respectful — it shows without announcing |
| Sandwich feedback (praise/criticism/praise) | Transparent and patronizing; people stop trusting your praise | Integrate acknowledgment naturally into your point |
| "I think we all agree that..." | Puts words in people's mouths | State your view, invite theirs |
| Passive voice to avoid attribution | "Mistakes were made" erodes trust | Own positions directly: "I'd recommend..." |
| Over-hedging ("maybe possibly perhaps") | Undermines your credibility and clarity | Be direct about your view while staying open to input |
| False questions ("Don't you think...?") | Manipulation dressed as inquiry | Ask genuine questions you want answers to |
