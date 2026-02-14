#!/usr/bin/env python3
"""Extract the Douglas Fir tree from the Cascadia Doug Flag SVG.

Generates two Hugo partials:
  1. tree-defs.html — hidden <svg> with the tree <symbol>, included once per page
  2. tree.html — lightweight per-card SVG using <use> elements in animated bands

Usage:
  python3 extract_tree.py Doug_flag.svg
  # writes themes/timberline/layouts/partials/tree-defs.html
  # writes themes/timberline/layouts/partials/tree.html
"""

import sys
import os
import xml.etree.ElementTree as ET
import re

NS = "{http://www.w3.org/2000/svg}"
PARTIAL_DIR = "themes/timberline/layouts/partials"


def parse_translate(transform_str):
    m = re.search(r"translate\(\s*([\d.eE+-]+)\s+([\d.eE+-]+)\s*\)", transform_str)
    if not m:
        raise ValueError(f"Cannot parse translate: {transform_str}")
    return float(m.group(1)), float(m.group(2))


def parse_matrix(transform_str):
    m = re.search(
        r"matrix\(\s*([\d.eE+-]+)\s+([\d.eE+-]+)\s+([\d.eE+-]+)\s+([\d.eE+-]+)"
        r"\s+([\d.eE+-]+)\s+([\d.eE+-]+)\s*\)",
        transform_str,
    )
    if not m:
        raise ValueError(f"Cannot parse matrix: {transform_str}")
    return tuple(float(m.group(i)) for i in range(1, 7))


def main():
    if len(sys.argv) != 2:
        print(f"Usage: {sys.argv[0]} <input.svg>", file=sys.stderr)
        sys.exit(1)

    svg_tree = ET.parse(sys.argv[1])
    root = svg_tree.getroot()

    # --- Clip-path bounds ---
    clip_b = root.find(f".//{NS}clipPath[@id='b']/{NS}path")
    if clip_b is None:
        raise RuntimeError("Cannot find clipPath#b")
    clip_d = clip_b.get("d")
    clip_m = re.match(r"m([\d.eE+-]+)\s+([\d.eE+-]+)h([\d.eE+-]+)v([\d.eE+-]+)", clip_d)
    if not clip_m:
        raise RuntimeError(f"Cannot parse clip-path d: {clip_d}")
    clip_x = float(clip_m.group(1))
    clip_y_start = float(clip_m.group(2))
    clip_w = float(clip_m.group(3))
    clip_h_neg = float(clip_m.group(4))
    clip_top_y = clip_y_start + clip_h_neg

    # --- Outer matrix ---
    outer_g = root.find(f"{NS}g[@transform]")
    if outer_g is None:
        raise RuntimeError("Cannot find outer <g> with transform")
    a, b, c, d, e, f = parse_matrix(outer_g.get("transform"))

    # --- Tree path ---
    tree_path_el = None
    for p in root.iter(f"{NS}path"):
        if p.get("fill") == "#152615":
            tree_path_el = p
            break
    if tree_path_el is None:
        raise RuntimeError("Cannot find tree path (fill=#152615)")

    path_d = tree_path_el.get("d")
    ttx, tty = parse_translate(tree_path_el.get("transform", ""))

    # --- Collapsed transform ---
    new_e = a * ttx + c * tty + e
    new_f = b * ttx + d * tty + f

    # --- ViewBox from clip-path ---
    corners = [
        (clip_x, clip_top_y),
        (clip_x + clip_w, clip_top_y),
        (clip_x, clip_y_start),
        (clip_x + clip_w, clip_y_start),
    ]
    transformed = [(a * px + c * py + e, b * px + d * py + f) for px, py in corners]
    xs = [p[0] for p in transformed]
    ys = [p[1] for p in transformed]
    vb_x, vb_y = min(xs), min(ys)
    vb_w, vb_h = max(xs) - vb_x, max(ys) - vb_y
    final_e = round(new_e - vb_x, 2)
    final_f = round(new_f - vb_y, 2)
    vb_w_r = round(vb_w)
    vb_h_r = round(vb_h)

    matrix_str = f"matrix({a} {b} {c} {d} {final_e} {final_f})"

    # --- Write tree-defs.html (symbol, included once per page) ---
    defs_path = os.path.join(PARTIAL_DIR, "tree-defs.html")
    with open(defs_path, "w") as fh:
        fh.write(
            f'<svg xmlns="http://www.w3.org/2000/svg" style="position:absolute;width:0;height:0;overflow:hidden">\n'
            f'  <symbol id="fir" viewBox="0 0 {vb_w_r} {vb_h_r}">\n'
            f'    <path transform="{matrix_str}"\n'
            f'          d="{path_d}" fill="currentColor"/>\n'
            f"  </symbol>\n"
            f"</svg>\n"
        )
    print(f"Wrote {defs_path} ({os.path.getsize(defs_path)} bytes)", file=sys.stderr)

    # --- Write tree.html (per-card usage, lightweight) ---
    # Five bands using <use> to reference the symbol.
    # CSS clip-path: inset() on each <g> isolates the band.
    # CSS animation on each <g> creates per-band rustling.
    use_path = os.path.join(PARTIAL_DIR, "tree.html")
    with open(use_path, "w") as fh:
        fh.write(
            f'<svg class="tree-svg" viewBox="0 0 {vb_w_r} {vb_h_r}"\n'
            f'     xmlns="http://www.w3.org/2000/svg" aria-hidden="true">\n'
            f'  <g class="band band-base"><use href="#fir"/></g>\n'
            f'  <g class="band band-lower"><use href="#fir"/></g>\n'
            f'  <g class="band band-mid"><use href="#fir"/></g>\n'
            f'  <g class="band band-upper"><use href="#fir"/></g>\n'
            f'  <g class="band band-top"><use href="#fir"/></g>\n'
            f"</svg>\n"
        )
    print(f"Wrote {use_path} ({os.path.getsize(use_path)} bytes)", file=sys.stderr)


if __name__ == "__main__":
    main()
