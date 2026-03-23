import numpy as np
import timberline_plots as tp

# Parameters
n = 174e9

# Memory range: 50 GB to 800 GB
memory_gb = np.linspace(50, 800, 500)
memory_bits = memory_gb * 8e9

# Optimal k = (m/n) * ln(2)
k_optimal = (memory_bits / n) * np.log(2)

# Also compute the FPR at each point for the secondary axis
fpr = np.power(0.5, k_optimal)


def make_chart(filename):
    fig, ax1 = tp.figure(figsize=(10, 6))

    ax1.plot(
        memory_gb, k_optimal,
        linewidth=2.5, color=tp.COLOR_CYCLE[0], label="Optimal k",
    )

    # Mark integer k values
    for k_int in range(1, int(k_optimal.max()) + 1):
        idx = np.argmin(np.abs(k_optimal - k_int))
        mem = memory_gb[idx]
        fp = fpr[idx]
        ax1.plot(mem, k_int, "o", color=tp.COLOR_CYCLE[1], markersize=5, zorder=5)
        if k_int <= 6 or k_int % 2 == 0:
            ax1.annotate(
                f"k={k_int} (FPR={fp*100:.1e}%)" if fp * 100 < 1 else f"k={k_int} (FPR={fp*100:.1f}%)",
                xy=(mem, k_int),
                xytext=(mem + 30, k_int + 0.3),
                fontsize=7.5, color=tp.COLORS["text"],
                arrowprops=dict(arrowstyle="->", color=tp.COLORS["muted"], lw=0.8),
            )

    ax1.set_xlabel("Total Memory (GB)")
    ax1.set_ylabel("Optimal Hash Functions (k)")
    ax1.set_title(
        "Bloom Filter: Optimal Hash Functions vs. Memory\n"
        "174B unique elements | integer k values annotated with resulting FPR"
    )
    ax1.set_xlim(50, 800)
    ax1.set_ylim(0, k_optimal.max() + 1)

    tp.save(fig, filename)


make_chart("bloom-optimal-k.svg")
make_chart("bloom-optimal-k-preview.png")
print("Done: optimal k")
