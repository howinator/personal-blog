"""Convenience chart functions: line(), bar(), scatter(), figure(), save()."""

from __future__ import annotations

import os
from pathlib import Path
from typing import Any

import matplotlib.pyplot as plt
import numpy as np
from numpy.typing import ArrayLike

from .style import apply

# Default output directory (relative to repo root)
_REPO_ROOT = Path(__file__).resolve().parents[4]
_CHARTS_DIR = _REPO_ROOT / "static" / "img" / "charts"


def _ensure_style() -> None:
    """Apply the timberline style if not already applied."""
    apply()


def figure(
    figsize: tuple[float, float] | None = None,
) -> tuple[plt.Figure, plt.Axes]:
    """Create a styled (fig, ax) pair. Escape hatch for custom charts."""
    _ensure_style()
    fig, ax = plt.subplots(figsize=figsize)
    return fig, ax


def save(
    fig: plt.Figure,
    filename: str,
    output_dir: str | Path | None = None,
) -> Path:
    """Save a figure to static/img/charts/ (or a custom directory).

    Returns the path to the saved file.
    """
    dest = Path(output_dir) if output_dir else _CHARTS_DIR
    dest.mkdir(parents=True, exist_ok=True)
    path = dest / filename
    fig.savefig(path)
    plt.close(fig)
    return path


def line(
    x: ArrayLike,
    y: ArrayLike | dict[str, ArrayLike],
    *,
    title: str | None = None,
    xlabel: str | None = None,
    ylabel: str | None = None,
    filename: str | None = None,
    output_dir: str | Path | None = None,
    figsize: tuple[float, float] | None = None,
    **kwargs: Any,
) -> tuple[plt.Figure, plt.Axes]:
    """Line chart. Pass a dict of {label: y_values} for multiple lines."""
    fig, ax = figure(figsize=figsize)

    if isinstance(y, dict):
        for label, y_data in y.items():
            ax.plot(x, y_data, label=label, **kwargs)
        ax.legend()
    else:
        ax.plot(x, y, **kwargs)

    if title:
        ax.set_title(title)
    if xlabel:
        ax.set_xlabel(xlabel)
    if ylabel:
        ax.set_ylabel(ylabel)

    if filename:
        save(fig, filename, output_dir)

    return fig, ax


def bar(
    x: ArrayLike,
    y: ArrayLike | dict[str, ArrayLike],
    *,
    title: str | None = None,
    xlabel: str | None = None,
    ylabel: str | None = None,
    filename: str | None = None,
    output_dir: str | Path | None = None,
    figsize: tuple[float, float] | None = None,
    bar_width: float = 0.8,
    **kwargs: Any,
) -> tuple[plt.Figure, plt.Axes]:
    """Bar chart. Pass a dict of {label: y_values} for grouped bars."""
    fig, ax = figure(figsize=figsize)

    x_arr = np.asarray(x)

    if isinstance(y, dict):
        n = len(y)
        width = bar_width / n
        offsets = np.linspace(-(n - 1) / 2 * width, (n - 1) / 2 * width, n)
        for offset, (label, y_data) in zip(offsets, y.items()):
            ax.bar(x_arr + offset, y_data, width=width, label=label, **kwargs)
        ax.legend()
    else:
        ax.bar(x_arr, y, width=bar_width, **kwargs)

    if title:
        ax.set_title(title)
    if xlabel:
        ax.set_xlabel(xlabel)
    if ylabel:
        ax.set_ylabel(ylabel)

    if filename:
        save(fig, filename, output_dir)

    return fig, ax


def scatter(
    x: ArrayLike,
    y: ArrayLike | dict[str, tuple[ArrayLike, ArrayLike]],
    *,
    title: str | None = None,
    xlabel: str | None = None,
    ylabel: str | None = None,
    filename: str | None = None,
    output_dir: str | Path | None = None,
    figsize: tuple[float, float] | None = None,
    **kwargs: Any,
) -> tuple[plt.Figure, plt.Axes]:
    """Scatter plot. Pass a dict of {label: (x, y)} for multiple series."""
    fig, ax = figure(figsize=figsize)

    if isinstance(y, dict):
        for label, (x_data, y_data) in y.items():
            ax.scatter(x_data, y_data, label=label, **kwargs)
        ax.legend()
    else:
        ax.scatter(x, y, **kwargs)

    if title:
        ax.set_title(title)
    if xlabel:
        ax.set_xlabel(xlabel)
    if ylabel:
        ax.set_ylabel(ylabel)

    if filename:
        save(fig, filename, output_dir)

    return fig, ax
