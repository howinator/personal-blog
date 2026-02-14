import numpy as np
import timberline_plots as tp

# X positions for 5 qualitative time markers
x = np.array([0, 1, 2, 3, 4])
labels = ["Just now", "Last week", "Last month", "Last quarter", "Last year"]

# Acoustic coding: higher y-intercept, lower slope
# Agentic coding: lower y-intercept, shallower slope (still steeper than acoustic)

acoustic = 5.0 + 0.7 * x
agentic = 4.0 + 1.0 * x

# Use smooth interpolation for a nicer look
x_smooth = np.linspace(0, 4, 200)
acoustic_smooth = 5.0 + 0.7 * x_smooth
agentic_smooth = 4.0 + 1.0 * x_smooth

fig, ax = tp.figure()

ax.plot(x_smooth, acoustic_smooth, color=tp.COLOR_CYCLE[0], linewidth=2.5, label="Acoustic Coding")
ax.plot(x_smooth, agentic_smooth, color=tp.COLOR_CYCLE[1], linewidth=2.5, label="Agentic Coding")

ax.set_xlabel("Time Since Writing Code")
ax.set_ylabel("Effort")
ax.set_xticks(x)
ax.set_xticklabels(labels, fontsize=9)
ax.set_yticks([])  # No numeric y-axis labels — keep it conceptual
ax.legend()

tp.save(fig, "fix-bug-effort.svg")
