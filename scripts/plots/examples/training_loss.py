"""Example: training loss curve with exponential decay."""

import numpy as np

import timberline_plots as tp

epochs = np.arange(1, 51)
loss = 2.8 * np.exp(-0.08 * epochs) + 0.15 + np.random.default_rng(42).normal(0, 0.03, len(epochs))

tp.line(
    epochs,
    loss,
    title="Training Loss",
    xlabel="Epoch",
    ylabel="Loss",
    filename="training-loss.svg",
)
