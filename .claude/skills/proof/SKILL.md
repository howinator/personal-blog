---
name: proof
description: >
  Use when the user asks to proofread, review, or check a blog post for voice,
  clarity, or authenticity. Compares the target post against existing published
  posts to ensure consistency with the author's established writing voice.
allowed-tools: Read, Glob, Grep
---

# Proof — Blog Post Proofreader & Voice Check

Proofread a blog post and check it for authenticity against the author's established voice.

## Usage

The user will point you at a blog post file (e.g. `content/blog/my-new-post.md`). You will:

1. Read the target post
2. Read all existing published posts in `content/blog/` for voice comparison
3. Produce a structured review

## Workflow

### Step 1 — Gather Context

- Use `Glob` to find all `.md` files in `content/blog/`
- Use `Read` to read every existing blog post (these are the voice reference corpus)
- Use `Read` to read the target post

### Step 2 — Proofread

Check the target post for:

- **Spelling & typos** — flag misspelled words (but respect intentional slang/casual language)
- **Grammar** — subject-verb agreement, dangling modifiers, comma splices, etc.
- **Punctuation** — missing or misplaced commas, periods, semicolons, em-dashes
- **Broken markdown** — unclosed links, malformed code blocks, bad frontmatter
- **Factual consistency** — if the post references other posts or claims, verify where possible

### Step 3 — Voice & Authenticity Check

Compare the target post against the corpus of existing posts. The author's established voice has these characteristics:

**Tone & Personality:**
- Conversational and irreverent with a technical edge
- Self-deprecating humor used naturally, not forced
- Direct address to the reader ("Do you remember...", "Just look at that...")
- Occasional profanity for emphasis (authentic, not gratuitous)
- Balances playfulness with intellectual rigor

**Structure & Rhythm:**
- Varied sentence length: short punchy sentences for emphasis, longer complex ones for nuance
- Parenthetical asides for tangential-but-relevant info
- Em-dashes for interjections and elaboration
- Oxford commas consistently
- Headers to break up longer posts
- Footnotes for tangential jokes or references

**Rhetorical Patterns:**
- Opens with a hook — personal anecdote, provocative question, or vivid scene
- Uses metaphor and analogy as primary teaching tools (oven/desert, viral GPL, leverage)
- Hedges appropriately before bold claims ("I would say", "I suspect")
- Acknowledges complexity and counterarguments (intellectual honesty)
- Connects personal experience to larger systemic implications

**Technical Writing:**
- Precise terminology without over-explaining to the audience
- Runnable code examples with context
- Specificity over vagueness (actual numbers, names, tools)

**Recurring Themes:**
- Leverage — creating outsized impact through building
- Systems thinking — second-order effects, incentive structures
- Joy in building and understanding
- Personal growth and reflection

Flag anything in the target post that:
- Feels stilted, overly formal, or corporate compared to the author's natural voice
- Uses passive voice where the author would typically use active
- Over-explains concepts the author's audience would already understand
- Under-explains concepts that need more context
- Lacks the personal anecdotes or connections that characterize the author's style
- Uses filler phrases or hedging beyond what's natural for the author
- Misses opportunities for the author's characteristic humor or directness
- Feels like it was written by an LLM (generic phrasing, list-heavy structure, "In conclusion" style wrapping)

### Step 4 — Clarity Review

Evaluate the post for:
- **Argument flow** — does each section logically follow the previous one?
- **Redundancy** — are any points repeated without adding value?
- **Jargon** — is technical language used appropriately for the audience?
- **Transitions** — do paragraphs connect smoothly?
- **Opening strength** — does the first paragraph hook the reader?
- **Closing strength** — does the ending land with impact rather than trailing off?

## Output Format

Present your review as a single structured report:

```
## Proofreading

[List specific errors with line references and suggested fixes]

## Voice & Authenticity

[Assessment of how well the post matches the author's voice, with specific
examples from the post and comparisons to existing posts. Call out passages
that feel off and suggest how to make them more authentic.]

## Clarity & Structure

[Feedback on argument flow, transitions, opening/closing strength, and
any passages that could be tightened or expanded.]

## Summary

[2-3 sentence overall assessment. Is this ready to publish? What are the
highest-priority changes?]
```

## Important Notes

- Be direct and specific. Vague feedback like "consider tightening this section" is useless. Quote the passage and explain what's wrong and how to fix it.
- Distinguish between errors (must fix) and suggestions (could improve). Not everything needs to change.
- Respect the author's intentional style choices. Casual grammar, sentence fragments, and profanity are features, not bugs.
- Do NOT rewrite the post. Point out issues and let the author fix them in their own voice.
- Compare against real patterns from the corpus, not a generic style guide.
- DO not mention the lack of categories or tags. Some posts will not have these.
