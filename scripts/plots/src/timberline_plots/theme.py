"""Pure data: colors, fonts, and layout constants from VIBES.md.

No library imports — this module defines the visual identity as plain
Python dicts and lists so any consumer (matplotlib, D3, Plotly) can use it.
"""

# Core palette (from CSS custom properties in VIBES.md)
COLORS = {
    "bg": "#EBE1C3",
    "text": "#2B2B2B",
    "muted": "#6B6860",
    "accent": "#2E4D37",
    "accent_hover": "#3E6349",
    "surface": "#E4DAB9",
    "border": "#C4B892",
    "code_bg": "#E0D6B3",
}

# Syntax highlighting colors (from Chroma monokailight)
SYNTAX = {
    "cyan": "#00a8c8",
    "amber": "#A26200",
    "purple": "#7021FF",
    "taupe": "#75715e",
    "olive": "#496D00",
    "magenta": "#D1064F",
    "sienna": "#A0522D",
}

# Data color cycle — forest green first, then darkened syntax colors
# All pass WCAG AA graphical contrast (3:1+) against parchment #EBE1C3
COLOR_CYCLE = [
    COLORS["accent"],    # #2E4D37 — forest green   (7.21:1)
    SYNTAX["sienna"],    # #A0522D — burnt sienna    (4.30:1)
    SYNTAX["amber"],     # #A26200 — dark amber      (3.76:1)
    SYNTAX["magenta"],   # #D1064F — dark magenta    (4.18:1)
    SYNTAX["purple"],    # #7021FF — dark purple      (4.80:1)
    SYNTAX["olive"],     # #496D00 — dark olive       (4.64:1)
    SYNTAX["taupe"],     # #75715e — taupe            (3.76:1)
]

# System fonts — translated from CSS identifiers to names matplotlib recognizes.
# The CSS values (ui-sans-serif, system-ui, etc.) are in VIBES.md but matplotlib
# needs actual font family names.
FONTS = {
    "sans": [
        "Helvetica Neue", "Helvetica", "Arial",
        "Segoe UI", "Roboto", "sans-serif",
    ],
    "mono": [
        "SF Mono", "SFMono-Regular", "Menlo",
        "Monaco", "Consolas", "monospace",
    ],
}

# Chart layout constants
LAYOUT = {
    "figsize": (8.5, 5.0),    # ~680px at 80dpi, matches --content-width
    "dpi": 80,
    "title_size": 14,
    "label_size": 11,
    "tick_size": 9,
    "line_width": 2.0,
    "spine_width": 0.8,
    "grid_alpha": 0.5,
    "legend_alpha": 0.9,
}
