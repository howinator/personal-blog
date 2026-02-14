"""Translate theme.py constants into matplotlib rcParams."""

import matplotlib as mpl
import matplotlib.pyplot as plt

from .theme import COLORS, COLOR_CYCLE, FONTS, LAYOUT

# matplotlib rcParams dict — applied once at import time
STYLE: dict = {
    # Figure
    "figure.figsize": LAYOUT["figsize"],
    "figure.dpi": LAYOUT["dpi"],
    "figure.facecolor": COLORS["bg"],
    "figure.edgecolor": "none",
    "savefig.dpi": LAYOUT["dpi"],
    "savefig.facecolor": COLORS["bg"],
    "savefig.edgecolor": "none",
    "savefig.bbox": "tight",
    "savefig.pad_inches": 0.3,

    # Axes
    "axes.facecolor": COLORS["bg"],
    "axes.edgecolor": COLORS["border"],
    "axes.linewidth": LAYOUT["spine_width"],
    "axes.titlesize": LAYOUT["title_size"],
    "axes.titleweight": "bold",
    "axes.titlecolor": COLORS["text"],
    "axes.titlepad": 16,
    "axes.labelsize": LAYOUT["label_size"],
    "axes.labelcolor": COLORS["text"],
    "axes.labelpad": 8,
    "axes.prop_cycle": mpl.cycler(color=COLOR_CYCLE),
    "axes.spines.top": False,
    "axes.spines.right": False,
    "axes.grid": True,
    "axes.axisbelow": True,

    # Grid — horizontal only, subtle
    "grid.color": COLORS["border"],
    "grid.alpha": LAYOUT["grid_alpha"],
    "grid.linewidth": 0.5,
    "axes.grid.axis": "y",

    # Ticks
    "xtick.labelsize": LAYOUT["tick_size"],
    "ytick.labelsize": LAYOUT["tick_size"],
    "xtick.color": COLORS["muted"],
    "ytick.color": COLORS["muted"],
    "xtick.labelcolor": COLORS["text"],
    "ytick.labelcolor": COLORS["text"],
    "xtick.direction": "out",
    "ytick.direction": "out",
    "xtick.major.width": LAYOUT["spine_width"],
    "ytick.major.width": LAYOUT["spine_width"],

    # Lines
    "lines.linewidth": LAYOUT["line_width"],
    "lines.markersize": 6,

    # Legend
    "legend.frameon": True,
    "legend.facecolor": COLORS["surface"],
    "legend.edgecolor": COLORS["border"],
    "legend.framealpha": LAYOUT["legend_alpha"],
    "legend.fontsize": LAYOUT["tick_size"],

    # Font
    "font.family": "sans-serif",
    "font.sans-serif": FONTS["sans"],
    "font.size": LAYOUT["tick_size"],
}


def apply() -> None:
    """Apply the timberline style to matplotlib globally."""
    plt.rcParams.update(STYLE)
