import numpy as np
import timberline_plots as tp

# Parameters
n = 174e9  # ~174 billion unique (user_id, event_id) pairs (7-day window, ~96% unique)

# Memory range: 50 GB to 800 GB
memory_gb = np.linspace(50, 800, 500)
memory_bits = memory_gb * 8e9  # convert GB to bits

# Optimal k for each memory size: k = (m/n) * ln(2)
k_optimal = (memory_bits / n) * np.log(2)

# False positive rate with optimal k: P = (1 - e^(-k*n/m))^k = (0.5)^k = 2^(-k)
# With optimal k, FPR simplifies to (1/2)^k = (1/2)^((m/n)*ln(2))
fpr = np.power(0.5, k_optimal)

fig, ax = tp.figure(figsize=(10, 6))

ax.semilogy(memory_gb, fpr * 100, color=tp.COLOR_CYCLE[0], linewidth=2.5)

# Reference lines for common FPR targets
targets = [(1.0, "1%"), (0.1, "0.1%"), (0.01, "0.01%")]
for target_pct, label in targets:
    ax.axhline(y=target_pct, color=tp.COLORS["muted"], linestyle="--", linewidth=1, alpha=0.6)
    # Find the memory where FPR hits this target
    idx = np.argmin(np.abs(fpr * 100 - target_pct))
    mem_at_target = memory_gb[idx]
    ax.annotate(
        f"{label} at {mem_at_target:.0f} GB",
        xy=(mem_at_target, target_pct),
        xytext=(mem_at_target + 60, target_pct * 3),
        fontsize=9,
        color=tp.COLORS["text"],
        arrowprops=dict(arrowstyle="->", color=tp.COLORS["muted"], lw=1),
    )

ax.set_xlabel("Total Memory (GB)")
ax.set_ylabel("False Positive Rate (%)")
ax.set_title("Bloom Filter: False Positive Rate vs. Memory\n174B unique elements, optimal hash functions")
ax.set_xlim(50, 800)
ax.set_ylim(1e-4, 100)

tp.save(fig, "bloom-fpr-vs-memory.svg")
print("Saved bloom-fpr-vs-memory.svg")
