---
title: "Charting with timberline-plots"
date: 2026-02-13T22:48:52-08:00
draft: true
categories: ['engineering']
tags: ['python', 'data-viz']
slug: "charting-with-timberline-plots"
---

I wanted a simple way to drop styled charts into blog posts without fiddling with colors and fonts every time. So I built `timberline-plots` — a thin wrapper around matplotlib that bakes in the blog's visual identity.

## Effort Comparison

Two lines, same slope, different intercepts. The kind of chart you'd sketch on graph paper to make a point about leverage.

![Effort Comparison](/img/charts/effort-comparison.svg)

## Training Loss

A single exponential decay curve with a bit of noise — the canonical deep learning chart.

![Training Loss](/img/charts/training-loss.svg)

## Temperature Control

Dual-axis chart showing a temperature signal and its corresponding heater output. The dashed line on the secondary axis keeps the two series visually distinct.

![Temperature Control](/img/charts/temperature-control.svg)

## How it works

The API is intentionally minimal:

```python
import timberline_plots as tp

tp.line(x, y, title="Training Loss", xlabel="Epoch",
        ylabel="Loss", filename="training-loss.svg")
```

For anything the convenience functions don't cover, `tp.figure()` returns a styled `(fig, ax)` pair:

```python
fig, ax = tp.figure()
ax.fill_between(x, y1, y2, alpha=0.3, color=tp.COLORS["accent"])
tp.save(fig, "custom-chart.svg")
```

All charts render with parchment backgrounds, forest-green-first color cycles, and open spines — so they blend into the page instead of floating in white rectangles.
