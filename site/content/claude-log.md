---
title: "Claug"
date: 2026-02-12T10:00:00-06:00
layout: "claude-log"
---

A log of my [Claude Code](https://docs.anthropic.com/en/docs/claude-code) sessions. A lightweight Go daemon hooks into session lifecycle events — `register` on start, `unregister` on end — streaming live stats over WebSocket while a session is active. The {{< cc-status-dot >}} in the nav pulses when I'm in a session. Between sessions, a `sync` pass reparses all transcripts and generates the historical stats below.

{{< cc-sessions >}}
