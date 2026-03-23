import numpy as np
import timberline_plots as tp

# Parameters
n = 174e9  # ~174 billion unique (user_id, event_id) pairs

# Memory range: 50 GB to 800 GB
memory_gb = np.linspace(50, 800, 500)
memory_bits = memory_gb * 8e9

# Optimal k: k = (m/n) * ln(2); FPR = 2^(-k)
k_optimal = (memory_bits / n) * np.log(2)
fpr = np.power(0.5, k_optimal)

fig, ax = tp.figure(figsize=(10, 6))
ax.semilogy(memory_gb, fpr * 100, color=tp.COLOR_CYCLE[0], linewidth=2.5)

targets = [(1.0, "1%"), (0.1, "0.1%"), (0.01, "0.01%")]
for target_pct, label in targets:
    ax.axhline(y=target_pct, color=tp.COLORS["muted"], linestyle="--", linewidth=1, alpha=0.6)
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

tp.save(fig, "bloom-fpr-vs-memory-preview.png")
print("Saved preview PNG")
