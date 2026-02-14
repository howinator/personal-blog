"""Example: dual-axis chart — temperature + control signal."""

import numpy as np

import timberline_plots as tp

t = np.linspace(0, 24, 200)
temp = 68 + 4 * np.sin(2 * np.pi * t / 24) + np.random.default_rng(7).normal(0, 0.3, len(t))
control = np.clip(72 - temp, 0, None) / 4

fig, ax1 = tp.figure()

color_temp = tp.COLOR_CYCLE[0]
color_ctrl = tp.COLOR_CYCLE[1]

ax1.plot(t, temp, color=color_temp, label="Temperature (°F)")
ax1.set_xlabel("Hour of Day")
ax1.set_ylabel("Temperature (°F)")
ax1.set_title("Temperature Control System")

ax2 = ax1.twinx()
ax2.plot(t, control, color=color_ctrl, linestyle="--", label="Heater Output")
ax2.set_ylabel("Heater Output (0–1)")
ax2.spines["right"].set_visible(True)
ax2.spines["right"].set_color(tp.COLORS["border"])
ax2.spines["right"].set_linewidth(0.8)

# Combined legend
lines1, labels1 = ax1.get_legend_handles_labels()
lines2, labels2 = ax2.get_legend_handles_labels()
ax1.legend(lines1 + lines2, labels1 + labels2, loc="upper right")

tp.save(fig, "temperature-control.svg")
