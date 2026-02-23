---
title: "The Return of the QA Engineer"
date: 2026-02-22T14:15:44-08:00
draft: true
categories: ['']
tags: ['']
slug: "qa"
---

I had another one of those "humans are so cooked" moments while using an LLM recently, y'all.
I've been reflecting on what made the agent so successful and figured I'd share, but first some backstory.

I've been working on a project that required compiling a Linux kernel with custom compile time options.
Claude wrote an initial implementation and I fired up a VM on GCP to test it.
Of course the compiled Linux kernel didn't work for what I needed first try, so I copied the logs, switched over to my local Claude window and asked it to fix it.

It was after midnight at this point, which was a great forcing function for me to realize this was an absolutely terrible use of my, now, Sunday morning.
Instead, I came up with a plan and the plan was simple.
I'd let Claude Code absolutely rip on this VM until the thing worked.

There were three main ingredients for this:
  1. You already know it — we're running `--dangerously-skip-permissions`. (VM had no IAM roles so no GCP bills were harmed during the writing of this blog post)
  2. I gave it instructions that absolutely every test needed to be done by modifying a bash script in the repo. I probably could've changed CC settings to enforce that, but, again, 12 AM.
  3. I threw together a little verification script. If the script printed "Test Succeeded," the e2e flow (kernel compilation, networking config, etc.) was working.

I fired up Claude Code, gave it the instructions and told it to be _relentless_ until `./verify.sh` printed "Test Succeeded".
After watching it for 5 minutes, I went to sleep.

I woke up the next morning and wouldn't you know it, I had a working set of shell scripts for what I needed.

## The QA Engineer Has Entered the Chat

What made this successful?
A couple things:
  1. A sandbox where the agent could do its worst.
  2. Guardrails to describe what was permissible and what was not for the agent, e.g., "if you run a shell command that is not a script in this repo, I swear on my life, I'll change your weights permanently."
  3. A clear value function that it could work against.

This story sounds familiar — where have I seen it before?
Well, in my 12-year career, I've worked with a QA engineer for a grand total of 6 months.
And during those 6 months, I engaged with them on exactly one project.

During that one project, I was the damn LLM!
The QA engineer told me, "Okay in the QA environment, for your project, I was able to generate these 5 failure scenarios. Please go fix them."
The sandbox was his QA env for that project.
The guardrails were my professionalism as a SWE.
And the value function was "fix these 5 scenarios."

I see a lot of people (including me) working on sandboxes for agents, but I don't see the same fervor for new verification primitives.
Taking that a step further, I really think the SDLC 2.0 will be defined by the outer and inner loops of verification while the sandbox will be a small commoditized piece in that story.

Put another way, my non-consensus, but probably right investment thesis is that a sharp QA engineer is probably the highest ROI hire in tech right now.

Let me put y'all on game about this (yes, I'm quoting Kendrick Lamar in a post about software engineering, deal with it).
A problem I see now is that AI PRs are getting bunched up in review because we're trying to apply the SDLC playbook of 2015 to 2026 and it simply doesn't work.
With the volume of code that is being generated, it would take an immense amount of time to verify it all for correctness.
This must be automated.

When I asked Claude to compile that Linux kernel so that it would boot how I wanted it to, I didn't care one lick about the contents of the shell scripts.
I only cared that the verification script that I wrote printed "Test Succeeded."
The agent was able to run fast because of the sandbox and I was able to move fast because of the formalized success criteria.
This is the future.

I'd bet almost anything that if a sharp QA engineer came into a team with some mix of SDET practices, formal verification and data science/statistical analysis, they would accelerate a team far more than the marginal SWE hire.

