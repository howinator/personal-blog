---
title: "Weird Claude Things"
date: 2026-02-11T21:14:23-08:00
draft: false
categories: ['tech']
tags: ['ai']
slug: "weird-claude-things"
---

I'm as AI optimist as it gets (mainly because I believe in humans), but uh Claude does some weird shit sometimes. 

This is a running list of weird things I've seen Claude do. I'll update it as I encounter new things:

- During a long hacking session (14k LOC change in 200k+ codebase), it decided to rename all .d.ts files in <root>/apps/admin/node_modules to .d.rs because tsc was complaining (Opus 4.6).
- For this blog, I was working on getting the infra for it into Pulumi. I needed to import existing Cloudflare resources into Pulumi, so it kept suggesting that I fetch IDs for those resources using `gh api --method GET 'https://cloudflare...' (Opus 4.6).

