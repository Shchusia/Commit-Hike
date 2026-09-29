"""Draws a schematic Middle-earth map for the two extras routes and writes
their `path` + pinned waypoints. World units: ~3.4 miles, x east, y south."""
import json, math, sys

# Run from the repository root: python3 extras/middle-earth-maps.py .
ROOT = sys.argv[1] if len(sys.argv) > 1 else "."

# ---------------------------------------------------------------- geography
MISTY = [(290, 32), (286, 50), (283, 70), (280, 92), (278, 112), (273, 132), (266, 150), (260, 170), (257, 192), (259, 204)]
GREY = [(298, 60), (315, 56), (335, 55), (352, 57), (365, 60)]
IRON = [(392, 66), (402, 64), (412, 67)]
WHITE = [(252, 262), (268, 256), (285, 258), (300, 262), (318, 266), (334, 272), (342, 278)]
EPHEL = [(366, 224), (369, 238), (370, 252), (369, 266), (368, 282), (370, 298)]
LITHUI = [(368, 221), (384, 218), (400, 216), (418, 214), (432, 216)]
MORGAI = [(376, 246), (377, 258), (377, 272), (376, 286)]
ERED_LUIN = [(120, 70), (116, 95), (118, 125), (114, 150)]
WEATHER_HILLS = [(236, 108), (238, 116)]
EMYN_MUIL = [(326, 226), (332, 222), (338, 227), (331, 231)]

ANDUIN = [(292, 28), (294, 55), (296, 80), (298, 106), (296, 130), (291, 152), (287, 172), (292, 188), (303, 205), (315, 222), (324, 238), (334, 254), (344, 268), (350, 282), (354, 300)]
BRANDYWINE = [(182, 80), (180, 100), (179, 124), (182, 142), (190, 162), (196, 185)]
HOARWELL = [(250, 70), (251, 95), (252, 121), (251, 140), (256, 160)]
BRUINEN = [(276, 104), (270, 112), (265, 119), (260, 135), (255, 152)]
SILVERLODE = [(266, 162), (274, 170), (286, 173)]
FOREST_RIVER = [(344, 86), (356, 87), (365, 88)]
RUNNING = [(372, 73), (370, 80), (369, 86), (369, 96), (372, 118), (380, 150)]
ENTWASH = [(262, 196), (285, 215), (310, 228), (324, 236)]
ISEN = [(258, 206), (250, 230), (238, 262)]

MIRKWOOD = [(312, 73), (330, 70), (352, 70), (372, 74), (377, 90), (374, 110), (368, 130), (356, 150), (340, 170), (326, 180), (316, 172), (314, 150), (312, 125), (310, 100)]
OLD_FOREST = [(183, 126), (190, 123), (197, 126), (198, 132), (191, 136), (184, 133)]
LORIEN = [(270, 166), (280, 163), (288, 168), (285, 178), (274, 179)]
FANGORN = [(256, 182), (266, 178), (274, 186), (271, 198), (260, 199)]
LONG_LAKE = [(366, 82), (371, 81), (372, 90), (370, 97), (366, 95)]
SEA_WEST = [(100, 0), (104, 60), (96, 100), (102, 140), (92, 180), (100, 230), (90, 300), (0, 300), (0, 0)]
DEAD_MARSH = (346, 226)
MIDGEWATER = (224, 121)
GORGOROTH = [(372, 228), (400, 222), (428, 222), (430, 260), (410, 290), (380, 292), (378, 250)]

# ------------------------------------------------------------------ routes
BILBO = [  # (x, y, waypoint id or None)
    (160, 126, "bag-end"), (174, 123, None), (192, 122, None), (213, 124, None), (238, 120, None),
    (252, 120, None), (259, 116, "trollshaws"), (269, 116, "rivendell"),
    (276, 112, None), (281, 109, "misty-mountains"), (284, 108, "goblin-town"), (286, 110, "gollum-lake"),
    (289, 107, None), (292, 103, "eagles-eyrie"), (298, 104, None), (306, 105, "beorn"),
    (310, 98, None), (316, 94, "mirkwood"), (330, 93, None), (343, 91, None), (352, 86, "elvenking-halls"),
    (358, 88, "forest-river"), (367, 89, "lake-town"), (371, 81, None), (373, 70, "lonely-mountain"),
    # home again: round the north of the forest, winter at Beorn's, the High Pass, Rivendell, the Road
    (364, 66, None), (345, 66, None), (325, 67, None), (309, 71, None), (304, 84, None), (303, 100, None),
    (303, 110, "beorn-return"), (294, 114, None), (283, 117, None), (270, 121, "rivendell-return"),
    (258, 121, None), (240, 124, None), (214, 128, None), (192, 127, None), (175, 128, None), (161, 130, "home-again"),
]
FRODO = [
    (160, 126, "bag-end"), (167, 129, None), (174, 130, "farmer-maggot"), (179, 128, "bucklebury-ferry"),
    (182, 125, "crickhollow"), (186, 129, "old-forest"), (193, 132, "bombadil"), (201, 132, "barrow-downs"),
    (207, 128, None), (213, 124, "bree"), (224, 120, "midgewater"), (238, 118, "weathertop"),
    (246, 121, None), (252, 121, "last-bridge"), (257, 117, "stone-trolls"), (265, 119, "ford-of-bruinen"),
    (269, 116, "rivendell"), (267, 126, None), (263, 138, None), (258, 149, "hollin"),
    (264, 152, "caradhras"), (259, 155, None), (256, 158, "moria-gate"), (263, 160, "khazad-dum-bridge"),
    (267, 162, "dimrill-dale"), (278, 170, "lothlorien"), (287, 174, None), (292, 188, None),
    (303, 205, None), (315, 222, None), (322, 233, "argonath"), (325, 238, "amon-hen"),
    (332, 233, "emyn-muil"), (345, 229, "dead-marshes"), (355, 226, None), (362, 225, "morannon"),
    (360, 234, None), (360, 242, "ithilien"), (360, 254, "henneth-annun"), (359, 263, None),
    (360, 268, "crossroads"), (364, 267, "minas-morgul"), (367, 263, "cirith-ungol-stairs"),
    (369, 261, "shelobs-lair"), (372, 260, "cirith-ungol-tower"), (374, 252, None), (374, 242, None),
    (373, 234, "isenmouthe"), (381, 240, None), (392, 250, "mount-doom"),
]

CROPS = {"bilbo-journey": (140, 32, 260, 130), "frodo-journey": (140, 96, 280, 204)}

# -------------------------------------------------------------- drawing
def catmull_d(pts, closed=False):
    if closed:
        pts = pts + pts[:3]
    d = "M%.1f %.1f" % pts[0]
    for i in range(len(pts) - 1 if not closed else len(pts) - 3):
        p0 = pts[i - 1] if i > 0 else pts[i]
        p1, p2 = pts[i], pts[i + 1]
        p3 = pts[i + 2] if i + 2 < len(pts) else p2
        c1 = (p1[0] + (p2[0] - p0[0]) / 6, p1[1] + (p2[1] - p0[1]) / 6)
        c2 = (p2[0] - (p3[0] - p1[0]) / 6, p2[1] - (p3[1] - p1[1]) / 6)
        d += " C%.1f %.1f %.1f %.1f %.1f %.1f" % (*c1, *c2, *p2)
    return d + (" Z" if closed else "")

def peaks(chain, size=5.5, step=5.0):
    out = []
    for (x1, y1), (x2, y2) in zip(chain, chain[1:]):
        n = max(1, int(math.hypot(x2 - x1, y2 - y1) / step))
        for k in range(n):
            t = k / n
            x, y = x1 + (x2 - x1) * t, y1 + (y2 - y1) * t
            j = ((int(x * 7 + y * 13) % 5) - 2) * 0.6
            s = size * (0.85 + (int(x * 3 + y) % 4) * 0.08)
            out.append(f"M{x - s + j:.1f} {y + s * .55:.1f}L{x + j:.1f} {y - s * .7:.1f}L{x + s + j:.1f} {y + s * .55:.1f}")
            out.append(f"M{x + j:.1f} {y - s * .7:.1f}L{x + s * .35 + j:.1f} {y + s * .55:.1f}")
    return " ".join(out)

def trees(poly, step=6):
    xs, ys = [p[0] for p in poly], [p[1] for p in poly]
    def inside(x, y):
        c = False
        for (x1, y1), (x2, y2) in zip(poly, poly[1:] + poly[:1]):
            if (y1 > y) != (y2 > y) and x < (x2 - x1) * (y - y1) / (y2 - y1) + x1:
                c = not c
        return c
    out = []
    y = min(ys) + step / 2
    row = 0
    while y < max(ys):
        x = min(xs) + (step / 2 if row % 2 else 0)
        while x < max(xs):
            if inside(x, y):
                out.append(f"M{x:.1f} {y + 2:.1f}l1.6-3.6 1.6 3.6z")
            x += step
        y += step * 0.8
        row += 1
    return " ".join(out)

def dots(c, r=5, n=9):
    return " ".join(f"M{c[0] + math.cos(i * 2.4) * r * (i / n):.1f} {c[1] + math.sin(i * 2.4) * r * .6 * (i / n):.1f}h2" for i in range(1, n))

def svg(x0, y0, w, h):
    s = 320 / w
    W, H = 320, round(320 * h / w)
    rivers = [ANDUIN, BRANDYWINE, HOARWELL, BRUINEN, SILVERLODE, FOREST_RIVER, RUNNING, ENTWASH, ISEN]
    g = []
    g.append(f'<path d="{catmull_d(SEA_WEST, True)}" fill="#bcd3d8" stroke="#6b5230" stroke-width="1.2"/>')
    g.append(f'<path d="{catmull_d(GORGOROTH, True)}" fill="#a88f72" opacity=".55"/>')
    for f in (MIRKWOOD, OLD_FOREST, LORIEN, FANGORN):
        g.append(f'<path d="{catmull_d(f, True)}" fill="#a9b07a" opacity=".55" stroke="#6f7446" stroke-width=".6"/>')
        g.append(f'<path d="{trees(f)}" fill="#5d6a3c" opacity=".7"/>')
    for r in rivers:
        g.append(f'<path d="{catmull_d(r)}" fill="none" stroke="#5f86a0" stroke-width="{1.4 if r is ANDUIN else .9}" stroke-linecap="round"/>')
    g.append(f'<path d="{catmull_d(LONG_LAKE, True)}" fill="#a9c6d0" stroke="#5f86a0" stroke-width=".8"/>')
    g.append(f'<path d="{dots(DEAD_MARSH, 9, 14)} {dots(MIDGEWATER, 6, 8)}" stroke="#5f86a0" stroke-width=".8" stroke-linecap="round"/>')
    ridge = " ".join(peaks(c) for c in (MISTY, GREY, IRON, WHITE, ERED_LUIN))
    g.append(f'<path d="{ridge}" fill="none" stroke="#6e5232" stroke-width=".9" stroke-linejoin="round"/>')
    dark = " ".join(peaks(c, 4.5, 4.2) for c in (EPHEL, LITHUI, MORGAI))
    g.append(f'<path d="{dark}" fill="none" stroke="#4a3322" stroke-width="1" stroke-linejoin="round"/>')
    hills = " ".join(peaks(c, 3, 3.5) for c in (WEATHER_HILLS, EMYN_MUIL, [(198, 131), (204, 134)]))
    g.append(f'<path d="{hills}" fill="none" stroke="#7a5b33" stroke-width=".8"/>')
    g.append('<path d="M367 77l6-10 6 10z" fill="#8f7550" stroke="#4a3322" stroke-width=".8"/>')  # the lone mountain
    g.append('<path d="M388 254l4-7 4 7z" fill="#7b3a26" stroke="#3b2418" stroke-width=".8"/>')  # the fire mountain
    body = "\n    ".join(g)
    return f'''<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 {W} {H}">
  <defs><radialGradient id="p" cx="50%" cy="45%" r="80%"><stop offset="0" stop-color="#f3e3bd"/><stop offset=".75" stop-color="#e3c98f"/><stop offset="1" stop-color="#b8904f"/></radialGradient></defs>
  <rect width="{W}" height="{H}" fill="url(#p)"/>
  <g transform="scale({s:.5f}) translate({-x0} {-y0})">
    {body}
  </g>
  <path d="M1.5 1.5h{W - 3}v{H - 3}h-{W - 3}z" fill="none" stroke="#8a6431" stroke-width="3"/>
</svg>
'''

for rid, pts in (("bilbo-journey", BILBO), ("frodo-journey", FRODO)):
    x0, y0, w, h = CROPS[rid]
    norm = lambda x, y: [round((x - x0) / w, 4), round((y - y0) / h, 4)]
    folder = f"{ROOT}/extras/routes/{rid}"
    with open(f"{folder}/assets/middle-earth-map.svg", "w") as f:
        f.write(svg(x0, y0, w, h))
    route = json.load(open(f"{folder}/route.json"))
    pins = {wid: norm(x, y) for x, y, wid in pts if wid}
    ids = [wp["id"] for wp in route["waypoints"]]
    assert set(pins) == set(ids), (rid, set(ids) ^ set(pins))
    order = [wid for _, _, wid in pts if wid]
    assert order == sorted(order, key=lambda i: next(w["at_m"] for w in route["waypoints"] if w["id"] == i)), rid
    for wp in route["waypoints"]:
        wp["x"], wp["y"] = pins[wp["id"]]
    new = {}
    for k, v in route.items():
        new[k] = v
        if k == "length_m" and "path" not in route:
            new["map_image"] = "assets/middle-earth-map.svg"
            new["path"] = [norm(x, y) for x, y, _ in pts]
    if "path" not in route:
        new["version"] = route["version"] + 1
    with open(f"{folder}/route.json", "w") as f:
        json.dump(new, f, indent=2, ensure_ascii=False)
        f.write("\n")
    print(rid, "ok", len(pts), "points")
