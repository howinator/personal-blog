#!/usr/bin/env python3
"""Extract the Douglas Fir tree path from the Cascadia Doug Flag SVG.

Parses the flag SVG, finds the tree <path> by its fill color (#152615),
collapses transforms, computes a 0-origin viewBox from the clip-path bounds,
and writes a standalone SVG to stdout.

Usage: python3 extract_tree.py Doug_flag.svg > themes/timberline/layouts/partials/tree.html
"""

import sys
import xml.etree.ElementTree as ET
import re

NS = "{http://www.w3.org/2000/svg}"


def parse_translate(transform_str):
    """Extract tx, ty from 'translate(tx ty)'."""
    m = re.search(r"translate\(\s*([\d.eE+-]+)\s+([\d.eE+-]+)\s*\)", transform_str)
    if not m:
        raise ValueError(f"Cannot parse translate: {transform_str}")
    return float(m.group(1)), float(m.group(2))


def parse_matrix(transform_str):
    """Extract (a, b, c, d, e, f) from 'matrix(a b c d e f)'."""
    m = re.search(
        r"matrix\(\s*([\d.eE+-]+)\s+([\d.eE+-]+)\s+([\d.eE+-]+)\s+([\d.eE+-]+)\s+([\d.eE+-]+)\s+([\d.eE+-]+)\s*\)",
        transform_str,
    )
    if not m:
        raise ValueError(f"Cannot parse matrix: {transform_str}")
    return tuple(float(m.group(i)) for i in range(1, 7))


def main():
    if len(sys.argv) != 2:
        print(f"Usage: {sys.argv[0]} <input.svg>", file=sys.stderr)
        sys.exit(1)

    tree = ET.parse(sys.argv[1])
    root = tree.getroot()

    # Find clip-path 'b' to get visible tree region bounds
    clip_b = root.find(f".//{NS}clipPath[@id='b']/{NS}path")
    if clip_b is None:
        raise RuntimeError("Cannot find clipPath#b")

    # Parse clip rect from 'd' attr: "m540.185 838.08h362.963v-838.08h-362.963z"
    clip_d = clip_b.get("d")
    clip_m = re.match(
        r"m([\d.eE+-]+)\s+([\d.eE+-]+)h([\d.eE+-]+)v([\d.eE+-]+)", clip_d
    )
    if not clip_m:
        raise RuntimeError(f"Cannot parse clip-path d: {clip_d}")

    clip_x = float(clip_m.group(1))
    clip_y_start = float(clip_m.group(2))
    clip_w = float(clip_m.group(3))
    clip_h_neg = float(clip_m.group(4))  # negative
    clip_h = abs(clip_h_neg)
    # Clip rect top-left in SVG user coords: (clip_x, clip_y_start + clip_h_neg)
    clip_top_y = clip_y_start + clip_h_neg  # = 0.0

    # Find the outer <g> with the matrix transform
    outer_g = root.find(f"{NS}g[@transform]")
    if outer_g is None:
        raise RuntimeError("Cannot find outer <g> with transform")
    a, b, c, d, e, f = parse_matrix(outer_g.get("transform"))

    # Find the tree path by fill="#152615"
    tree_path = None
    for path_el in root.iter(f"{NS}path"):
        if path_el.get("fill") == "#152615":
            tree_path = path_el
            break
    if tree_path is None:
        raise RuntimeError("Cannot find tree path (fill=#152615)")

    path_d = tree_path.get("d")
    tx, ty = parse_translate(tree_path.get("transform", ""))

    # Collapse transforms: outer matrix * path translate
    # Matrix (a,b,c,d,e,f) applied to point (x,y):
    #   x' = a*x + c*y + e
    #   y' = b*x + d*y + f
    # The path has translate(tx, ty), so effective transform on path coords is:
    #   matrix(a, b, c, d, a*tx + c*ty + e, b*tx + d*ty + f)
    new_e = a * tx + c * ty + e
    new_f = b * tx + d * ty + f

    # Transform clip-path bounds through the outer matrix to get viewBox
    # Clip rect corners in SVG user space (before outer matrix):
    # Top-left: (clip_x, clip_top_y), Bottom-right: (clip_x + clip_w, clip_y_start)
    corners = [
        (clip_x, clip_top_y),
        (clip_x + clip_w, clip_top_y),
        (clip_x, clip_y_start),
        (clip_x + clip_w, clip_y_start),
    ]

    transformed = []
    for px, py in corners:
        tx2 = a * px + c * py + e
        ty2 = b * px + d * py + f
        transformed.append((tx2, ty2))

    xs = [p[0] for p in transformed]
    ys = [p[1] for p in transformed]
    vb_x = min(xs)
    vb_y = min(ys)
    vb_w = max(xs) - min(xs)
    vb_h = max(ys) - min(ys)

    # Shift the collapsed transform so the viewBox starts at origin
    final_e = new_e - vb_x
    final_f = new_f - vb_y

    # Round for clean output
    vb_w_r = round(vb_w)
    vb_h_r = round(vb_h)
    final_e_r = round(final_e, 2)
    final_f_r = round(final_f, 2)

    svg = f"""\
<svg class="tree-svg" viewBox="0 0 {vb_w_r} {vb_h_r}"
     xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
  <path transform="matrix({a} {b} {c} {d} {final_e_r} {final_f_r})"
        d="{path_d}" fill="currentColor"/>
</svg>
"""
    sys.stdout.write(svg)


if __name__ == "__main__":
    main()
