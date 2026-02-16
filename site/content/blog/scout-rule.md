---
title: "Scout Rule is So Back"
date: 2026-02-13T23:40:54-08:00
draft: false
categories: ['']
tags: ['swe']
slug: "scout-rule"
---

In the AI age, I've noticed that I often see bugs in our backlog and think, "Hm I think I know enough about this bug to point an agent in the right direction here, but like generously 50% sure."
So, being the optimist I am, I tag Cursor or open a Claude Code session, pulling in the context of the ticket and pointing toward how I think it should be fixed.

But then what? The agent submits a PR with a description that's somewhat believable but also a little sus and I'm stuck with a 200-line PR that takes me a couple hours to verify in the best case.
At worst, I'm _that guy_ with 12 open draft PRs just sitting there in the Pull Requests tab.

In fact, there are times when these LLM-created bug fixes take significantly longer to fix the bug than if I had done it myself. 
This is often because I'll spend time verifying the PR before realizing the model was just completely wrong about the solution, so I'm down an hour and back at square one.

Instead, I've shifted to a mental model where, when I'm working in an area of the codebase and the context cache is hot, I will search our issue tracker for bugs related to that area of the code.
With a lot of context about the code, it's trivial for me to kick off three agents for three bugs that have a ~95% hit rate.

Before agentic coding, I'd always tell my team "boy scout rule applies to the codebase," but those were often empty words since deadlines mattered and drafting a PR for a bug fix took time.

![Effort to fix a bug over time](/img/charts/fix-bug-effort.svg)

This is a subtle shift in thinking.
Instead of letting the priority of bugs being the highest order bit for determining what to work on, it now often makes sense to make the last thing you worked on drive the bugs you pick up.


Say what you want about agentic coding, but there's no denying the strike zone is when you, the human, have full context about the code that's being changed.
So, it obviously has limits, but it's more important than ever to be a good scout and fix some stuff while you're there.

