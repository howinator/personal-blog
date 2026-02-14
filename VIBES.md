# VIBES.md — howinator.io Visual Identity

This document defines the visual identity, design language, and aesthetic of howinator.io. Use it as the source of truth when building new components or modifying existing ones. Any new UI should feel like it belongs here — no imported frameworks, no utility classes, just handcrafted CSS that matches the parchment-and-forest palette.

To preview the site locally, see `AGENTS.md` for dev server setup (`make dev` → `http://localhost:8004`).

## The Vibe

A technical field journal from the Pacific Northwest. Think graph-paper notebook with precise ink drawings of system architectures alongside hand-pressed Douglas Fir leaves — warm parchment, deep forest green ink, monospace annotations in the margins. The content is engineering-focused: infrastructure diagrams, token counts, session metrics, code blocks. The presentation wraps that precision in something organic and human.

This is a tech blog first. The field journal metaphor means *meticulous and well-kept* — data is presented with care, code is typeset cleanly, metrics are live and exact. Monospace isn't decorative; it's the natural voice for someone who thinks in terminals. The warmth of the palette keeps it from feeling sterile, but the underlying rigor is the point. Every number on the Claude Log page is real, updating in real time, displayed to meaningful precision.

Animations are organic and restrained: branches rustle, digits roll, a green dot breathes. Nothing flashy. They reinforce the sense that this is a living document — a journal that's still being written in.

## Color Palette

All colors are defined as CSS custom properties in `:root` (see `themes/timberline/assets/css/main.css`).

| Token | Hex | Usage |
|-------|-----|-------|
| `--bg` | `#EBE1C3` | Page background — warm cream/parchment |
| `--text` | `#2B2B2B` | Body text — deep charcoal |
| `--muted` | `#6B6860` | Secondary text, dates, labels — warm taupe |
| `--accent` | `#2E4D37` | Links, borders, interactive elements — deep forest green |
| `--accent-hover` | `#3E6349` | Hover states — lighter forest green |
| `--surface` | `#E4DAB9` | Card backgrounds, blockquotes — lighter beige |
| `--border` | `#C4B892` | Subtle separators, card outlines — warm tan |
| `--code-bg` | `#E0D6B3` | Code blocks, inline code — pale parchment |

**Supplementary (not in `:root`):**

| Color | Hex | Usage |
|-------|-----|-------|
| Live active | `#22c55e` | Green status dot, live session border |
| Live border | `#22c55e40` | Live session card border (40% opacity) |
| Redacted block | `#6B6860` | Opaque redacted text background |
| Redacted tooltip | `#4a4740` | Darker tooltip on redacted hover |

**Key rule:** Never introduce blue, red, or saturated colors outside the syntax highlighting theme. The palette is earth tones only — greens, tans, and charcoals.

## Typography

```css
--font-body: ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
--font-mono: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
```

No web fonts. System fonts only — fast, native-feeling, zero layout shift.

| Context | Font | Size | Weight | Notes |
|---------|------|------|--------|-------|
| Body text | `--font-body` | `1rem` | 400 | `line-height: 1.6` |
| Post content | `--font-body` | `1.125rem` | 400 | `line-height: 1.8` for readability |
| Post title (h1) | `--font-body` | `2rem` | 700 | `line-height: 1.3` |
| Post h2 | `--font-body` | `1.5rem` | 400 | Includes tree decoration |
| Post h3 | `--font-body` | `1.25rem` | 400 | Includes tree decoration |
| Post h4 | `--font-body` | `1.1rem` | 400 | Includes tree decoration |
| Nav links | `--font-mono` | `0.85rem` | 400 | `uppercase`, `letter-spacing: 0.05em` |
| Meta (dates, tags) | `--font-mono` | `0.8rem` | 400 | Color: `--muted` |
| Tags | `--font-mono` | `0.7rem` | 400 | `lowercase`, solid accent pill |
| Home tagline | `--font-mono` | `0.95rem` | 400 | Color: `--muted` |
| Footer | `--font-mono` | `0.8rem` | 400 | Color: `--muted` |
| Code | `--font-mono` | `0.85em` | 400 | `--code-bg` background |
| Stats values | `--font-mono` | `1.6rem` | 700 | Color: `--accent` |
| Stats labels | `--font-mono` | `0.75rem` | 400 | `uppercase`, `letter-spacing: 0.05em` |

**Pattern:** Monospace is used for metadata, navigation, and anything "machine-like" (dates, stats, code). Sans-serif is used for reading text and headings. This separation creates clear visual hierarchy.

## Layout

```css
--content-width: 680px;  /* Blog posts, article content */
--site-width: 780px;     /* Outer container with padding */
```

- Body uses CSS Grid: `grid-template-rows: auto 1fr auto` for sticky footer
- Content is centered with `max-width` + `margin: 0 auto`
- Padding: `2rem 1.5rem` on `.site-main`
- Single responsive breakpoint: `max-width: 600px`

## Components

### Navigation
- Top bar with 2px accent bottom border
- Home icon (left) + monospace links (right)
- Hamburger menu on mobile (CSS-only, checkbox toggle)
- "Claude Log" link includes a live status dot

### Blog Cards (Archive)
- `--surface` background, `--border` outline, 6px radius
- Douglas Fir SVG tree decoration in bottom-right corner (50px x 100px)
- Branches fold in a cascading wave on hover (see Animations)
- Right padding accommodates the tree: `padding-right: 5rem`

### Blog Posts
- Generous `1.125rem` content with `1.8` line-height
- Headings include inline Douglas Fir tree SVGs (sized 17-24px by heading level)
- Blockquotes: 3px left accent border, italic, `--surface` background
- Code blocks: `--code-bg` with 3px left accent border
- Tags: small solid pills, accent background, cream text

### Claude Code Stats Dashboard
- 4-column grid of stat boxes (2-col on mobile)
- Collapsible session cards using `<details>`/`<summary>`
- Caret rotates 90deg on expand
- Live sessions: green border, "LIVE" label in `#22c55e`
- Token counts use abbreviated suffixes (k, M)
- Slot-machine digit transition on value changes
- Typewriter animation for latest prompt display

### Redacted Text
- Grey block (`#6B6860`) with transparent text
- Hover reveals tooltip above with arrow pointer
- Tooltip uses darker background (`#4a4740`) with `--bg` colored text
- 0.6s fade transition

### Footer
- 2px accent top border (mirrors nav)
- Centered monospace text, muted color
- "Powered by Hugo" attribution

## Animations

All animations are CSS-only. No JavaScript animation libraries.

### Branch Fold (Signature)
The most distinctive visual element. Douglas Fir tree SVGs have six clip-path segments (br, mr, tr, bl, ml, tl) that animate independently in a cascading wave when the parent element is hovered.

- **Archive cards:** 5s duration, subtle motion (2px translate, 0.91 scaleX, 0.75deg rotate)
- **Heading trees:** 6.5s duration, exaggerated motion (3px translate, 0.87 scaleX, 1deg rotate)
- **Wave pattern:** bottom-right starts first, propagates upward with 10% stagger per tier
- **Ease:** `ease-in-out`, single cycle, plays only while hovering

### Status Dot Pulse
```css
@keyframes cc-pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.4; }
}
/* 2s, ease-in-out, infinite — breathing effect */
```

### Slot-Machine Digits
```css
.cc-digit-enter {
  transform: translateY(0.5em);
  opacity: 0;
}
/* 300ms ease-out transition on transform + opacity */
```
New digits slide up from below when values update.

### Typewriter
- Characters appear at 25ms intervals (JS-driven)
- Blinking block cursor (`\2588`) while typing
- Cursor disappears 2s after completion
- Blink rate: 1.06s step-end

### Hover Feedback
- Links: bottom border appears (1px solid accent), 0.2s transition
- Social icons: `translateY(-2px)` lift
- Tags: background lightens to `--accent-hover`

## Syntax Highlighting

Chroma monokailight, adapted to sit on parchment (`--code-bg`):

| Element | Color |
|---------|-------|
| Keywords | `#00a8c8` (cyan) |
| Strings | `#d88200` (amber) |
| Numbers | `#ae81ff` (purple) |
| Comments | `#75715e` (taupe) |
| Functions/names | `#75af00` (olive) |
| Operators/tags | `#f92672` (magenta) |
| Default text | `#111` |

## Design Principles

1. **Technical precision first.** This is an engineering blog. Numbers are exact, metrics are live, code is clean. The design serves the content, not the other way around.
2. **No external dependencies.** No Tailwind, no Bootstrap, no Google Fonts. Everything is hand-written CSS with system fonts. Build it yourself — that's the ethos of the whole site.
3. **Warm, not sterile.** The palette is earthy because a field journal is warmer than a whiteboard. But the underlying rigor is non-negotiable — avoid anything that trades clarity for decoration.
4. **Monospace is the native voice.** It's not an accent font — it's how this person thinks. Terminals, metrics, dates, labels, nav. Sans-serif is for the prose that connects the technical pieces.
5. **Animations serve meaning.** The tree rustles because it's alive. The dot breathes because something is active. Digits roll because values change. Nothing animates just to animate.
6. **Mobile-first simplicity.** Single breakpoint at 600px. Hamburger nav, 2-col grids, smaller trees. No complex responsive gymnastics.
7. **Content-width constraint.** Blog text never exceeds 680px for optimal reading line length.

## File Map

| File | What it styles |
|------|---------------|
| `themes/timberline/assets/css/main.css` | Base theme: palette, typography, nav, posts, archive cards, tree animations, syntax highlighting |
| `assets/css/cc-stats.css` | Claude Code dashboard: stat boxes, session cards, live status, digit animation, typewriter |
| `assets/css/redacted.css` | Redacted text blocks and hover tooltips |
| `assets/js/live-status.js` | WebSocket client: live status dot, typewriter, digit roller (JS behavior, not styling) |
