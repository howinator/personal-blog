---
name: generate-chart
description: >
  Use when the user asks to create a chart, plot, graph, or data visualization
  for a blog post. Generates styled SVG/PNG charts matching the blog's
  visual identity using the timberline-plots Python library.
allowed-tools: Bash, Read, Write, Glob
---

# Generate Chart

Create styled charts for howinator.io blog posts using the `timberline-plots` library.

## Setup

Ensure the package is installed (only needed once per session):

```bash
uv sync --project scripts/plots
```

## Running a Chart Script

```bash
uv run --project scripts/plots python <path-to-script>
```

## Output

- Default output directory: `static/img/charts/`
- Filenames: kebab-case, e.g. `my-chart-name.svg`
- SVG is the default format. Use `.png` extension for PNG output.
- Embed in blog posts: `![Alt text](/img/charts/filename.svg)`

## API Reference

```python
import timberline_plots as tp
```

### `tp.line(x, y, *, title, xlabel, ylabel, filename, output_dir, figsize, **kwargs)`

Line chart. Pass `y` as a dict of `{label: values}` for multiple lines.

### `tp.bar(x, y, *, title, xlabel, ylabel, filename, output_dir, figsize, bar_width, **kwargs)`

Bar chart. Pass `y` as a dict of `{label: values}` for grouped bars.

### `tp.scatter(x, y, *, title, xlabel, ylabel, filename, output_dir, figsize, **kwargs)`

Scatter plot. Pass `y` as a dict of `{label: (x_data, y_data)}` for multiple series.

### `tp.figure(figsize=None) -> (fig, ax)`

Returns a styled `(fig, ax)` pair for custom charts. Use this as an escape hatch
when the convenience functions don't cover your use case.

### `tp.save(fig, filename, output_dir=None) -> Path`

Save a figure. Defaults to `static/img/charts/`. Closes the figure after saving.

### Constants

| Constant | Description |
|----------|-------------|
| `tp.COLORS` | Dict of theme colors: `bg`, `text`, `muted`, `accent`, `surface`, `border`, etc. |
| `tp.SYNTAX` | Dict of syntax highlighting colors: `cyan`, `amber`, `purple`, `taupe`, `olive`, `magenta`, `sienna` |
| `tp.COLOR_CYCLE` | List of 7 colors used for data series (forest green, burnt sienna, dark amber, dark magenta, dark purple, dark olive, taupe) |
| `tp.FONTS` | Dict with `sans` and `mono` font family lists |
| `tp.LAYOUT` | Dict of layout constants: `figsize`, `dpi`, font sizes, line widths |

## Example Patterns

### Simple line chart

```python
import numpy as np
import timberline_plots as tp

x = np.arange(1, 51)
y = 2.8 * np.exp(-0.08 * x) + 0.15

tp.line(x, y, title="Training Loss", xlabel="Epoch",
        ylabel="Loss", filename="training-loss.svg")
```

### Multi-line comparison

```python
import numpy as np
import timberline_plots as tp

hours = np.arange(0, 11)
tp.line(hours,
        {"Manual": 2.5 * hours + 5, "AI-Assisted": 2.5 * hours + 1},
        title="Effort Comparison", xlabel="Hours Invested",
        ylabel="Features Shipped", filename="effort-comparison.svg")
```

### Custom chart (escape hatch)

```python
import numpy as np
import timberline_plots as tp

x = np.linspace(0, 10, 100)
y1 = np.sin(x)
y2 = np.cos(x)

fig, ax = tp.figure()
ax.fill_between(x, y1, y2, alpha=0.3, color=tp.COLORS["accent"])
ax.plot(x, y1, label="sin(x)")
ax.plot(x, y2, label="cos(x)")
ax.legend()
ax.set_title("Custom Chart")
tp.save(fig, "custom-chart.svg")
```

### Dual-axis chart

```python
import numpy as np
import timberline_plots as tp

t = np.linspace(0, 24, 200)
temp = 68 + 4 * np.sin(2 * np.pi * t / 24)
control = np.clip(72 - temp, 0, None) / 4

fig, ax1 = tp.figure()
ax1.plot(t, temp, color=tp.COLOR_CYCLE[0], label="Temperature")
ax1.set_xlabel("Hour")
ax1.set_ylabel("Temp (°F)")
ax1.set_title("Temperature Control")

ax2 = ax1.twinx()
ax2.plot(t, control, color=tp.COLOR_CYCLE[1], linestyle="--", label="Heater")
ax2.set_ylabel("Heater Output")
ax2.spines["right"].set_visible(True)
ax2.spines["right"].set_color(tp.COLORS["border"])

lines1, labels1 = ax1.get_legend_handles_labels()
lines2, labels2 = ax2.get_legend_handles_labels()
ax1.legend(lines1 + lines2, labels1 + labels2)
tp.save(fig, "temperature-control.svg")
```

## Workflow

1. Write a Python script using the API above
2. Run it: `uv run --project scripts/plots python <script>`
3. Confirm the SVG/PNG was generated in `static/img/charts/`
4. Embed in blog post: `![Alt text](/img/charts/filename.svg)`
