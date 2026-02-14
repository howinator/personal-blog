# AGENTS.md - Hugo Blog (site/)

## Overview

Hugo static site for https://howinator.io/. Theme is `timberline` (tracked as regular files, not a submodule).

## Key Files

- `config.toml` — Hugo site configuration (title, menus, social links, permalink structure)
- `archetypes/default.md` — Template used by `hugo new` to scaffold new posts
- `content/blog/` — All blog posts (flat directory, no date-based subdirectories)
- `themes/timberline/` — Active theme
- `static/` — Static assets (images in `static/img/`, PDFs, manifest)
- `assets/js/live-status.js` — Browser WebSocket client for live status dot
- `layouts/shortcodes/cc-status-dot.html` — Inline status dot shortcode
- `data/cc_sessions.json` — Session stats exported by cc-live daemon (auto-generated)
- `Containerfile` — Podman container build (multi-stage: Hugo build + nginx)
- `public/` — Build output (gitignored)

## Blog Post Format

Posts live in `content/blog/<slug>.md` with YAML frontmatter:

```yaml
---
title: "Post Title Here"
date: 2026-02-07T10:00:00-06:00
draft: true
slug: "post-slug"
categories: ['category1']
tags: ['tag1', 'tag2']
---

Markdown content here.
```

- **Filename**: kebab-case, e.g. `my-new-post.md`
- **Date**: ISO 8601 with timezone offset (author is in US Central, -06:00 CST / -05:00 CDT)
- **Permalink pattern**: `/:year/:month/:day/:slug` (defined in config.toml)
- **Emoji**: Enabled site-wide (`:sunglasses:` etc.)

## Creating a New Post

```bash
hugo new content/blog/my-post-slug.md --source site
# or from site/ directory:
cd site && hugo new content/blog/my-post-slug.md
```

This scaffolds from `archetypes/default.md`. Then edit the generated file to set `title`, `categories`, `tags`, and write content. Set `draft: false` when ready to publish.

## Building & Previewing

```bash
# From repo root
hugo server -D --source site    # Local dev server (includes drafts)
hugo server --source site       # Local dev server (published posts only)
hugo --source site              # Build static site into site/public/

# Or from site/ directory
cd site && hugo server -D
```

## Container Build

```bash
# From repo root (via Makefile)
make build    # Builds blog image using site/ as build context
```
