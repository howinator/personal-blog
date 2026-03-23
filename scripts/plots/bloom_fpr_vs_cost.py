import numpy as np
import timberline_plots as tp
import matplotlib.ticker as mticker

# Parameters
n = 174e9  # ~174 billion unique elements

# Use r7g.4xlarge as representative node (all r7g sizes have ~same $/GB)
node_name = "r7g.4xlarge"
mem_gib = 105.81
price_hr = 1.745
HOURS_PER_MONTH = 730

mem_gb = mem_gib * (1024**3) / (1000**3)  # GiB to GB
price_month = price_hr * HOURS_PER_MONTH


def make_chart(filename):
    fig, ax = tp.figure(figsize=(10, 6))

    # Enough nodes to get well below 0.01% FPR
    max_nodes = 1
    while True:
        m = max_nodes * mem_gb * 8e9
        k = (m / n) * np.log(2)
        if 0.5**k * 100 < 1e-4 or max_nodes > 100:
            break
        max_nodes += 1

    node_counts = np.arange(1, max_nodes + 1)
    total_mem_bits = node_counts * mem_gb * 8e9
    total_cost = node_counts * price_month

    k_optimal = (total_mem_bits / n) * np.log(2)
    fpr = np.power(0.5, k_optimal)

    # Only show points with FPR < 100%
    mask = fpr < 1.0
    total_cost = total_cost[mask]
    fpr = fpr[mask]
    node_counts_filtered = node_counts[mask]

    ax.semilogy(
        total_cost, fpr * 100,
        marker="o", markersize=5, linewidth=2.2,
        color=tp.COLOR_CYCLE[0],
    )

    # Annotate node counts at select points
    for nc, cost, fp in zip(node_counts_filtered, total_cost, fpr):
        if fp * 100 < 5e-4:
            break
        if nc <= 5 or nc % 2 == 0:
            ax.annotate(
                f"{nc} node{'s' if nc > 1 else ''}",
                xy=(cost, fp * 100),
                fontsize=7, color=tp.COLORS["muted"],
                textcoords="offset points", xytext=(12, 4),
                ha="left",
            )

    # Reference lines
    for target_pct, label in [(1.0, "1%"), (0.1, "0.1%"), (0.01, "0.01%")]:
        ax.axhline(y=target_pct, color=tp.COLORS["muted"], linestyle="--",
                    linewidth=1, alpha=0.5)
        ax.text(
            total_cost[0] * 0.5, target_pct * 1.5, label,
            fontsize=9, color=tp.COLORS["muted"], va="bottom",
        )

    ax.set_xlabel("Monthly Cost (USD)")
    ax.set_ylabel("False Positive Rate (%)")
    ax.set_title(
        "Bloom Filter: FPR vs. Monthly ElastiCache Cost\n"
        f"174B elements | {node_name} nodes @ ${price_month:,.0f}/mo each"
    )
    ax.set_ylim(1e-4, 100)
    ax.set_xlim(left=0)
    ax.xaxis.set_major_formatter(mticker.FuncFormatter(lambda x, _: f"${x:,.0f}"))

    tp.save(fig, filename)


make_chart("bloom-fpr-vs-cost.svg")
make_chart("bloom-fpr-vs-cost-preview.png")
print("Done")
