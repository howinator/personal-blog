import numpy as np
import timberline_plots as tp
import matplotlib.ticker as mticker

# Parameters
n = 174e9  # unique elements

# Exact set: (user_id: 32-bit + event_id: 53-bit) = 85 bits per pair
# Plus hash table overhead (~1.5x for load factor), so ~16 bytes per entry is realistic
bytes_per_entry_exact = 16
exact_set_gb = n * bytes_per_entry_exact / 1e9  # ~2,784 GB

# Bloom filter: vary target FPR, compute required memory
# m = -n * ln(p) / (ln(2))^2
fpr_targets = np.logspace(-4, -0.3, 200)  # 0.01% to ~50%
bloom_bits = -n * np.log(fpr_targets) / (np.log(2) ** 2)
bloom_gb = bloom_bits / 8e9


def make_chart(filename):
    fig, ax = tp.figure(figsize=(10, 6))

    # Bloom filter curve
    ax.plot(
        fpr_targets * 100, bloom_gb,
        linewidth=2.5, color=tp.COLOR_CYCLE[0], label="Bloom filter",
    )

    # Exact set horizontal line
    ax.axhline(
        y=exact_set_gb, color=tp.COLOR_CYCLE[1], linestyle="--",
        linewidth=2, label=f"Exact hash set (~{exact_set_gb:,.0f} GB)",
    )

    # Annotate key points
    for target_pct in [1.0, 0.1, 0.01]:
        idx = np.argmin(np.abs(fpr_targets * 100 - target_pct))
        mem = bloom_gb[idx]
        savings = exact_set_gb / mem
        ax.annotate(
            f"{target_pct}% FPR: {mem:,.0f} GB ({savings:.0f}x savings)",
            xy=(target_pct, mem),
            xytext=(target_pct * 5, mem * 1.8),
            fontsize=8.5, color=tp.COLORS["text"],
            arrowprops=dict(arrowstyle="->", color=tp.COLORS["muted"], lw=1),
        )

    ax.set_xscale("log")
    ax.set_xlabel("False Positive Rate (%)")
    ax.set_ylabel("Memory Required (GB)")
    ax.set_title(
        "Bloom Filter vs. Exact Set: Memory Requirements\n"
        "174B unique (user_id, event_id) pairs"
    )
    ax.legend(fontsize=9, loc="upper right")
    ax.xaxis.set_major_formatter(mticker.FuncFormatter(lambda x, _: f"{x:g}%"))

    tp.save(fig, filename)


make_chart("bloom-memory-savings.svg")
make_chart("bloom-memory-savings-preview.png")
print("Done: memory savings")
