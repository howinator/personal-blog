"""Example: two lines, same slope, different intercepts."""

import numpy as np

import timberline_plots as tp

hours = np.arange(0, 11)
manual = 2.5 * hours + 5
assisted = 2.5 * hours + 1

tp.line(
    hours,
    {"Manual": manual, "AI-Assisted": assisted},
    title="Effort Comparison",
    xlabel="Hours Invested",
    ylabel="Features Shipped",
    filename="effort-comparison.svg",
)
