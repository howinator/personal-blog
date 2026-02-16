---
title: "Fix That Bug Now"
date: 2026-02-13T23:40:54-08:00
draft: false
categories: ['']
tags: ['swe']
slug: "fix-that-bug-now"
---

In the AI age, I've noticed that I often see bugs in our backlog and think, "Hm I think I know enough about this bug to point an agent in the right direction here, but like generously 50% sure."
So, being the optimist I am, I tag cursor or open a claude code session, pulling in the context of the ticket and pointing toward how I think it should be fixed.

But then what? The agent submits a PR with a description that's somewhat believable but also a little sus and I'm stuck with a 200 line PR that takes me a couple hours to verify.
In fact, there are times where it takes me significantly longer because I dig into the PR and find out that the agent completely missed the point and now I wasted hours on a cold lead.

Instead, I've shifted to a mental model where, when I'm working in an area of the codebase and the context cache is hot, I will search our issue tracker for bugs related to this area of the codebase.
With a lot of context about the code, it's trivial for me to kick off three agents for three bugs that have a ~95% hit rate.

Before agentic coding, I'd always tell my team "boy scout rule applies to the codebase," but those were often empty words since deadlines mattered and drafting a PR for a bug fix took time.

The chart below is how I now see effort vs. time on bug fixes when working in some part of the codebase.

![Effort to fix a bug over time](/img/charts/fix-bug-effort.svg)

By actively looking for bugs related to your mental warm cache, you can be so much more effective with AI agents, so just fix that bug while you have the context to verify the fix.

In other words, don't let the backlog drive your bugfixing.
Instead, let your current work drive your grooming.

