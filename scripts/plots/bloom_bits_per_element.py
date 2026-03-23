import numpy as np
import timberline_plots as tp

# Bits per element vs FPR (theoretical, with optimal k)
# FPR = 2^(-k) where k = (m/n) * ln(2), so m/n = -ln(FPR) / (ln(2))^2
# Equivalently: FPR = exp(-(m/n) * (ln(2))^2)

bits_per_element = np.linspace(1, 30, 500)

# Optimal k for each bits/element
k_optimal = bits_per_element * np.log(2)
fpr = np.power(0.5, k_optimal)


def make_chart(filename):
    fig, ax = tp.figure(figsize=(10, 6))

    ax.semilogy(
        bits_per_element, fpr * 100,
        linewidth=2.5, color=tp.COLOR_CYCLE[0],
    )

    # Annotate key points
    annotations = [
        (5, "5 bits/elem"),
        (10, "10 bits/elem"),
        (15, "15 bits/elem"),
        (20, "20 bits/elem"),
    ]
    for bpe, label in annotations:
        idx = np.argmin(np.abs(bits_per_element - bpe))
        fp = fpr[idx] * 100
        k = k_optimal[idx]
        ax.plot(bpe, fp, "o", color=tp.COLOR_CYCLE[1], markersize=6, zorder=5)
        ax.annotate(
            f"{label}\nk={k:.0f}, FPR={fp:.2e}%",
            xy=(bpe, fp),
            xytext=(bpe + 1.5, fp * 4),
            fontsize=8, color=tp.COLORS["text"],
            arrowprops=dict(arrowstyle="->", color=tp.COLORS["muted"], lw=1),
        )

    # Reference lines
    for target_pct, label in [(1.0, "1%"), (0.1, "0.1%"), (0.01, "0.01%")]:
        ax.axhline(y=target_pct, color=tp.COLORS["muted"], linestyle="--",
                    linewidth=1, alpha=0.5)

    ax.set_xlabel("Bits per Element (m/n)")
    ax.set_ylabel("False Positive Rate (%)")
    ax.set_title(
        "Bloom Filter: FPR vs. Bits per Element\n"
        "Optimal number of hash functions at each point"
    )
    ax.set_xlim(1, 30)
    ax.set_ylim(1e-6, 100)

    tp.save(fig, filename)


make_chart("bloom-bits-per-element.svg")
make_chart("bloom-bits-per-element-preview.png")
print("Done: bits per element")
