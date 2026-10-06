"""Original flat-style landscape illustrations for the route uploads (no characters, no names)."""
import math, random, os
from playwright.sync_api import sync_playwright
OUT = '/tmp/uploads'

class Scene:
    def __init__(self, w, h, seed):
        self.w, self.h, self.r, self.defs, self.body, self.n = w, h, random.Random(seed), [], [], 0
    def _id(self):
        self.n += 1; return f"g{self.n}"
    def grad(self, stops, vertical=True):
        i = self._id(); x2, y2 = ("0", "1") if vertical else ("1", "0")
        s = "".join(f'<stop offset="{o}" stop-color="{c}"/>' for o, c in stops)
        self.defs.append(f'<linearGradient id="{i}" x1="0" y1="0" x2="{x2}" y2="{y2}">{s}</linearGradient>'); return f"url(#{i})"
    def radial(self, inner, outer):
        i = self._id(); self.defs.append(f'<radialGradient id="{i}"><stop offset="0" stop-color="{inner}"/><stop offset="1" stop-color="{outer}" stop-opacity="0"/></radialGradient>'); return f"url(#{i})"
    def add(self, s): self.body.append(s)
    def sky(self, top, bottom, mid=None):
        stops = [(0, top)] + ([(0.55, mid)] if mid else []) + [(1, bottom)]
        self.add(f'<rect width="{self.w}" height="{self.h}" fill="{self.grad(stops)}"/>')
    def stars(self, n=120, top=0.5):
        for _ in range(n):
            self.add(f'<circle cx="{self.r.random()*self.w:.0f}" cy="{self.r.random()*self.h*top:.0f}" r="{self.r.random()*1.4+0.3:.1f}" fill="#fff" opacity="{self.r.random()*0.7+0.2:.2f}"/>')
    def sun(self, x, y, rad, color, glow=None):
        if glow: self.add(f'<circle cx="{x}" cy="{y}" r="{rad*4}" fill="{self.radial(glow, glow)}" opacity="0.6"/>')
        self.add(f'<circle cx="{x}" cy="{y}" r="{rad}" fill="{color}"/>')
    def ridge(self, base, amp, color, peaks=7, jag=0.35, opacity=1, smooth=False):
        w, r = self.w, self.r
        xs = sorted([0, w] + [r.random() * w for _ in range(peaks * 2)])
        pts = [(x, base - amp * (0.35 + 0.65 * r.random()) * (1 if i % 2 else 1 - jag)) for i, x in enumerate(xs)]
        if smooth:
            d = f"M0,{self.h} L0,{pts[0][1]:.0f} " + " ".join(f"Q{(a[0]+b[0])/2:.0f},{a[1]:.0f} {b[0]:.0f},{b[1]:.0f}" for a, b in zip(pts, pts[1:])) + f" L{w},{self.h} Z"
        else:
            d = f"M0,{self.h} " + " ".join(f"L{x:.0f},{y:.0f}" for x, y in pts) + f" L{w},{self.h} Z"
        self.add(f'<path d="{d}" fill="{color}" opacity="{opacity}"/>')
    def snowcaps(self, base, amp, color="#f4f7fb"):
        pass
    def peak(self, x, y, wid, hgt, color, snow=None, crater=False):
        if crater:
            d = f"M{x-wid},{y} L{x-wid*0.12},{y-hgt} L{x+wid*0.12},{y-hgt} L{x+wid},{y} Z"
        else:
            d = f"M{x-wid},{y} L{x},{y-hgt} L{x+wid},{y} Z"
        self.add(f'<path d="{d}" fill="{color}"/>')
        if snow:
            s = hgt * 0.3; sw = wid * 0.3
            if crater:
                self.add(f'<path d="M{x-wid*0.12-sw*0.6},{y-hgt+s} L{x-wid*0.12},{y-hgt} L{x+wid*0.12},{y-hgt} L{x+wid*0.12+sw*0.6},{y-hgt+s} L{x+sw*0.3},{y-hgt+s*0.7} L{x-sw*0.2},{y-hgt+s*1.05} Z" fill="{snow}"/>')
            else:
                self.add(f'<path d="M{x-sw},{y-hgt+s} L{x},{y-hgt} L{x+sw},{y-hgt+s} L{x+sw*0.4},{y-hgt+s*0.8} L{x},{y-hgt+s*1.05} L{x-sw*0.5},{y-hgt+s*0.75} Z" fill="{snow}"/>')
    def water(self, top, c1, c2, shimmer="#ffffff", n=40):
        self.add(f'<rect y="{top}" width="{self.w}" height="{self.h-top}" fill="{self.grad([(0, c1), (1, c2)])}"/>')
        for _ in range(n):
            y = top + (self.r.random() ** 1.6) * (self.h - top); x = self.r.random() * self.w; l = 20 + self.r.random() * 90
            self.add(f'<line x1="{x:.0f}" y1="{y:.0f}" x2="{x+l:.0f}" y2="{y:.0f}" stroke="{shimmer}" stroke-width="{1+self.r.random()*1.5:.1f}" opacity="{0.08+self.r.random()*0.25:.2f}" stroke-linecap="round"/>')
    def path(self, pts, color, width):
        d = "M" + " ".join(f"{x:.0f},{y:.0f}" for x, y in pts[:1]) + " " + " ".join(f"Q{a[0]:.0f},{a[1]:.0f} {(a[0]+b[0])/2:.0f},{(a[1]+b[1])/2:.0f}" for a, b in zip(pts[1:], pts[2:])) + f" L{pts[-1][0]:.0f},{pts[-1][1]:.0f}"
        self.add(f'<path d="{d}" fill="none" stroke="{color}" stroke-width="{width}" stroke-linecap="round" stroke-dasharray="{width*2.2},{width*1.6}"/>')
    def pines(self, y0, y1, n, color, size=(18, 40)):
        for _ in range(n):
            x = self.r.random() * self.w; y = y0 + self.r.random() * (y1 - y0); s = size[0] + self.r.random() * (size[1] - size[0]) * (y - y0 + 1) / (y1 - y0 + 1)
            self.add(f'<path d="M{x},{y-s*2.4} L{x+s*0.7},{y} L{x-s*0.7},{y} Z" fill="{color}"/><rect x="{x-s*0.08}" y="{y}" width="{s*0.16}" height="{s*0.3}" fill="{color}"/>')
    def roundtrees(self, y0, y1, n, color, size=(14, 30)):
        for _ in range(n):
            x = self.r.random() * self.w; y = y0 + self.r.random() * (y1 - y0); s = size[0] + self.r.random() * (size[1] - size[0])
            self.add(f'<rect x="{x-s*0.08}" y="{y-s*0.6}" width="{s*0.16}" height="{s*0.9}" fill="#3b2a1e"/><circle cx="{x}" cy="{y-s}" r="{s*0.7}" fill="{color}"/>')
    def mist(self, y, color="#ffffff", opacity=0.35, height=60):
        """A soft band fading out above and below (no hard edges)."""
        h = height * 2.5
        fill = self.grad([(0, color + "00"), (0.5, color), (1, color + "00")]) if color.startswith("#") and len(color) == 7 else color
        self.add(f'<rect y="{y-h/2}" width="{self.w}" height="{h}" fill="{fill}" opacity="{opacity}"/>')
    def ship(self, x, y, s, color="#1a1410", sail="#e9dfc8"):
        self.add(f'<path d="M{x-s*1.6},{y} Q{x},{y+s*0.55} {x+s*1.8},{y-s*0.15} L{x+s*1.4},{y+s*0.15} Q{x},{y+s*0.75} {x-s*1.3},{y+s*0.25} Z" fill="{color}"/>'
                 f'<line x1="{x}" y1="{y}" x2="{x}" y2="{y-s*2.2}" stroke="{color}" stroke-width="{s*0.09}"/>'
                 f'<path d="M{x-s*0.95},{y-s*1.95} L{x+s*0.95},{y-s*1.95} L{x+s*0.85},{y-s*0.45} L{x-s*0.85},{y-s*0.45} Z" fill="{sail}"/>'
                 + "".join(f'<line x1="{x+s*i*0.42:.1f}" y1="{y+s*0.2:.1f}" x2="{x+s*i*0.42-s*0.3:.1f}" y2="{y+s*0.75:.1f}" stroke="{color}" stroke-width="{s*0.05:.1f}"/>' for i in range(-3, 4)))
    def glow(self, x, y, rad, color, opacity=0.8):
        self.add(f'<circle cx="{x}" cy="{y}" r="{rad}" fill="{self.radial(color, color)}" opacity="{opacity}"/>')
    def smoke(self, x, y, color="#55504c", n=7):
        for i in range(n):
            self.add(f'<circle cx="{x + i*i*2.4 + self.r.random()*10:.0f}" cy="{y - i*34}" r="{18 + i*9}" fill="{color}" opacity="{0.55 - i*0.06:.2f}" filter="url(#blur)"/>')
    def figures(self, x, y, n, s, color="#1b1b1b"):
        for i in range(n):
            fx = x + i * s * 1.6
            self.add(f'<circle cx="{fx}" cy="{y-s*1.75}" r="{s*0.28}" fill="{color}"/><path d="M{fx-s*0.3},{y-s*1.4} L{fx+s*0.3},{y-s*1.4} L{fx+s*0.22},{y-s*0.55} L{fx+s*0.3},{y} L{fx+s*0.1},{y} L{fx},{y-s*0.5} L{fx-s*0.1},{y} L{fx-s*0.3},{y} L{fx-s*0.22},{y-s*0.55} Z" fill="{color}"/>'
                     f'<line x1="{fx+s*0.45}" y1="{y-s*1.3}" x2="{fx+s*0.55}" y2="{y}" stroke="{color}" stroke-width="{s*0.08}"/>')
    def mushrooms(self, n, y0, y1, cap="#c9b48a", stem="#e8dcc1"):
        for _ in range(n):
            x = self.r.random() * self.w; y = y0 + self.r.random() * (y1 - y0); s = 30 + self.r.random() * 90 * (y - y0 + 40) / (y1 - y0 + 40)
            self.add(f'<rect x="{x-s*0.12}" y="{y-s*1.6}" width="{s*0.24}" height="{s*1.6}" fill="{stem}" rx="{s*0.06}"/><path d="M{x-s*0.9},{y-s*1.55} Q{x},{y-s*2.6} {x+s*0.9},{y-s*1.55} Z" fill="{cap}"/>')
    def whirlpool(self, x, y, rad, color="#d8e6ef"):
        for i in range(5):
            a = rad * (1 - i * 0.18)
            self.add(f'<ellipse cx="{x}" cy="{y}" rx="{a}" ry="{a*0.32}" fill="none" stroke="{color}" stroke-width="{3-i*0.4:.1f}" opacity="{0.25+i*0.12:.2f}" stroke-dasharray="{a*0.9},{a*0.4}" transform="rotate({i*31} {x} {y})"/>')
    def deer(self, x, y, s, color="#2a221d"):
        self.add(f'<path d="M{x},{y-s} L{x+s*1.4},{y-s} L{x+s*1.55},{y-s*1.45} L{x+s*1.75},{y-s*1.45} L{x+s*1.6},{y-s*0.95} L{x+s*1.4},{y} L{x+s*1.25},{y} L{x+s*1.2},{y-s*0.55} L{x+s*0.3},{y-s*0.55} L{x+s*0.2},{y} L{x+s*0.05},{y} Z" fill="{color}"/>'
                 f'<path d="M{x+s*1.6},{y-s*1.45} L{x+s*1.45},{y-s*2.1} M{x+s*1.52},{y-s*1.8} L{x+s*1.3},{y-s*1.95} M{x+s*1.7},{y-s*1.45} L{x+s*1.95},{y-s*2.05} M{x+s*1.82},{y-s*1.75} L{x+s*2.05},{y-s*1.8}" stroke="{color}" stroke-width="{s*0.08}" fill="none"/>')
    def cottage(self, x, y, s, wall="#f1ece0", roof="#3d4148"):
        self.add(f'<rect x="{x}" y="{y-s}" width="{s*2}" height="{s}" fill="{wall}"/><path d="M{x-s*0.1},{y-s} L{x+s*0.3},{y-s*1.55} L{x+s*1.7},{y-s*1.55} L{x+s*2.1},{y-s} Z" fill="{roof}"/>'
                 f'<rect x="{x+s*0.3}" y="{y-s*0.6}" width="{s*0.3}" height="{s*0.3}" fill="#e8b53d"/><rect x="{x+s*1.4}" y="{y-s*0.6}" width="{s*0.3}" height="{s*0.3}" fill="#e8b53d"/>')
    def svg(self):
        return (f'<svg xmlns="http://www.w3.org/2000/svg" width="{self.w}" height="{self.h}" viewBox="0 0 {self.w} {self.h}">'
                f'<defs><filter id="blur"><feGaussianBlur stdDeviation="14"/></filter>{"".join(self.defs)}</defs>{"".join(self.body)}</svg>')

def scenes():
    out = {}
    # ---------------- West Highland Way
    s = Scene(1600, 900, 11); s.sky("#2f3e5c", "#f2b27a", "#b98aa0"); s.sun(1180, 470, 38, "#ffe2a8", "#ffcf7a")
    s.ridge(560, 300, "#5d5f7e", peaks=6, opacity=0.9); s.ridge(600, 230, "#3f4766", peaks=5); s.mist(590, "#f0c6a0", 0.35)
    s.water(600, "#c98f7e", "#2c3550", "#ffd9b0"); s.ridge(900, 120, "#1f2a33", peaks=4, smooth=True)
    s.pines(700, 830, 26, "#141c22", (14, 30)); s.cottage(980, 735, 26); s.path([(0, 880), (300, 820), (600, 780), (900, 760), (1100, 750)], "#e8b53d", 5)
    out["west-highland-way/cover"] = s
    s = Scene(1200, 800, 12); s.sky("#8aa4b8", "#dfe4dc"); s.ridge(470, 200, "#6f7d8c", peaks=5, opacity=0.8); s.ridge(520, 140, "#55626e", peaks=4)
    s.add(f'<rect y="520" width="1200" height="280" fill="#8a7c52"/>')
    for i in range(9): s.add(f'<ellipse cx="{s.r.random()*1200:.0f}" cy="{560+s.r.random()*220:.0f}" rx="{40+s.r.random()*90:.0f}" ry="{8+s.r.random()*14:.0f}" fill="#b9cad6" opacity="0.85"/>')
    s.deer(700, 640, 34); s.mist(500, "#ffffff", 0.4); out["west-highland-way/moor"] = s
    s = Scene(1200, 800, 13); s.sky("#46586b", "#b8c3c9"); s.peak(250, 640, 330, 520, "#2c333b"); s.peak(950, 660, 360, 560, "#252b31"); s.peak(600, 700, 260, 300, "#3a434c")
    s.mist(420, "#dfe6ea", 0.45); s.add('<rect y="640" width="1200" height="160" fill="#4d5a3c"/>'); s.path([(600, 800), (560, 740), (640, 700), (590, 660)], "#e8b53d", 5)
    out["west-highland-way/glen"] = s
    # ---------------- The Odyssey
    s = Scene(1600, 900, 21); s.sky("#2a1b3d", "#f08a4b", "#a33e5c"); s.sun(800, 560, 70, "#ffd27a", "#ff9a52")
    s.ridge(570, 70, "#3b2240", peaks=3, smooth=True, opacity=0.9); s.water(560, "#7a2a4a", "#1c0f26", "#ffb27a", 70)
    s.ship(560, 640, 46); s.add('<path d="M1250,580 Q1330,520 1420,575 Z" fill="#2b1832"/>'); out["the-odyssey/cover"] = s
    s = Scene(1200, 800, 22); s.sky("#0b1026", "#2a2140"); s.stars(160, 0.6); s.peak(620, 600, 420, 360, "#1d1820", crater=True)
    s.glow(620, 245, 120, "#ff6a2a", 0.7); s.smoke(620, 215, "#3b3440"); s.water(600, "#1b1a33", "#07070f", "#ff8a4a", 40); s.ship(260, 650, 22, "#000", "#c9b99a")
    out["the-odyssey/volcano"] = s
    s = Scene(1200, 800, 23); s.sky("#4d6d84", "#b9cbd4"); s.peak(120, 700, 260, 560, "#3a3236"); s.peak(1080, 720, 280, 520, "#433a3e")
    s.water(560, "#3f6b84", "#16303f", "#ffffff", 60); s.whirlpool(760, 640, 170); s.ship(430, 600, 26); out["the-odyssey/strait"] = s
    # ---------------- Journey to the Centre of the Earth
    s = Scene(1600, 900, 31); s.sky("#9fb7c9", "#eef1f0"); s.ridge(640, 120, "#6f6a66", peaks=5, opacity=0.6)
    s.peak(820, 720, 560, 430, "#4e4a48", snow="#f6f8fb", crater=True); s.add('<rect y="700" width="1600" height="200" fill="#5c5650"/>')
    s.mist(700, "#ffffff", 0.35); s.figures(560, 820, 3, 26); s.path([(0, 880), (300, 850), (520, 830)], "#e8b53d", 5); out["journey-to-the-centre-of-the-earth/cover"] = s
    s = Scene(1200, 800, 32); s.sky("#173a3f", "#3f7f78"); s.glow(600, 80, 520, "#b6f0d8", 0.35)
    s.add('<path d="M0,0 L1200,0 L1200,120 Q900,170 600,110 Q300,60 0,140 Z" fill="#0f2427"/>'); s.water(470, "#2b6a66", "#0b2224", "#c7fff0", 50)
    s.mushrooms(9, 430, 480, "#b8a37a", "#e3d7bd"); s.add('<rect x="420" y="560" width="160" height="16" fill="#5a3f28"/><line x1="500" y1="560" x2="500" y2="470" stroke="#5a3f28" stroke-width="5"/><path d="M500,475 L560,520 L500,540 Z" fill="#d9cdb0"/>')
    out["journey-to-the-centre-of-the-earth/sea"] = s
    s = Scene(1200, 800, 33); s.sky("#0c0b1a", "#3a1d25"); s.stars(90, 0.4); s.peak(600, 640, 440, 400, "#1a1416", crater=True)
    s.glow(600, 240, 160, "#ff5a1f", 0.85); s.smoke(600, 220, "#4a3338", 8)
    for i in range(10): s.add(f'<circle cx="{560+s.r.random()*80:.0f}" cy="{150+s.r.random()*90:.0f}" r="{3+s.r.random()*5:.0f}" fill="#ffb347"/>')
    s.water(620, "#2a1418", "#08060a", "#ff8a4a", 30); out["journey-to-the-centre-of-the-earth/eruption"] = s
    # ---------------- Bilbo's Journey (hidden): generic hills, mountains, forest — nothing from the books
    s = Scene(1600, 900, 41); s.sky("#8ec5e8", "#f9e7b8"); s.sun(1250, 230, 46, "#fff3c4", "#ffe08a")
    s.peak(1280, 560, 160, 230, "#7c8aa0"); s.ridge(600, 180, "#8fa5b8", peaks=6, opacity=0.7)
    s.ridge(700, 160, "#7fb069", peaks=5, smooth=True); s.ridge(800, 140, "#5f9a4f", peaks=4, smooth=True); s.ridge(900, 120, "#4c843f", peaks=3, smooth=True)
    s.roundtrees(690, 860, 20, "#3f7a35"); s.path([(80, 900), (320, 830), (700, 790), (1000, 720), (1250, 600)], "#e8d5a0", 7); out["bilbo-journey/cover"] = s
    s = Scene(1200, 800, 42); s.sky("#6f8fb0", "#d7e1ea"); s.peak(300, 640, 300, 460, "#4b5566", snow="#f1f4f8"); s.peak(780, 660, 340, 520, "#3f4857", snow="#eef2f7"); s.peak(1100, 650, 220, 360, "#4b5566", snow="#f1f4f8")
    s.mist(520, "#ffffff", 0.45); s.pines(620, 790, 40, "#22303a", (16, 34)); out["bilbo-journey/mountains"] = s
    s = Scene(1200, 800, 43); s.sky("#23312a", "#5d7a5a"); s.glow(600, 300, 380, "#c9e6b0", 0.25)
    for _ in range(14):
        x = s.r.random() * 1200; w = 18 + s.r.random() * 40; s.add(f'<rect x="{x:.0f}" y="0" width="{w:.0f}" height="800" fill="#15201a" opacity="{0.6+s.r.random()*0.4:.2f}"/>')
    s.add('<rect y="650" width="1200" height="150" fill="#1a281e"/>'); s.path([(600, 800), (560, 720), (620, 660)], "#c9b48a", 5); out["bilbo-journey/forest"] = s
    # ---------------- Frodo's Journey (hidden)
    s = Scene(1600, 900, 51); s.sky("#2e2a33", "#a0583a", "#5b3b3e"); s.peak(1150, 640, 330, 320, "#211b1d", crater=True)
    s.glow(1150, 330, 140, "#ff6a2a", 0.75); s.smoke(1150, 300, "#3a3034"); s.ridge(700, 90, "#3a3236", peaks=6, opacity=0.9)
    s.add('<rect y="700" width="1600" height="200" fill="#4a3f38"/>'); s.path([(0, 900), (300, 840), (650, 790), (950, 730)], "#c9a97a", 6); out["frodo-journey/cover"] = s
    s = Scene(1200, 800, 52); s.sky("#9aa9ba", "#e8edf2"); s.peak(600, 700, 520, 560, "#5a6372", snow="#f7f9fb"); s.peak(150, 720, 260, 380, "#6b7383", snow="#f5f7fa")
    for _ in range(80): s.add(f'<circle cx="{s.r.random()*1200:.0f}" cy="{s.r.random()*800:.0f}" r="{1+s.r.random()*2.5:.1f}" fill="#fff" opacity="0.8"/>')
    s.add('<rect y="690" width="1200" height="110" fill="#eef2f6"/>'); s.figures(520, 760, 4, 18, "#2b2f36"); out["frodo-journey/pass"] = s
    s = Scene(1200, 800, 53); s.sky("#e9d9a6", "#f6efd2"); s.glow(600, 260, 420, "#fff4c9", 0.5)
    for _ in range(12):
        x = s.r.random() * 1200; w = 16 + s.r.random() * 34; s.add(f'<rect x="{x:.0f}" y="0" width="{w:.0f}" height="800" fill="#c9c2b0" opacity="{0.7+s.r.random()*0.3:.2f}"/>')
    for _ in range(60): s.add(f'<circle cx="{s.r.random()*1200:.0f}" cy="{s.r.random()*520:.0f}" r="{20+s.r.random()*50:.0f}" fill="#e3b54a" opacity="{0.35+s.r.random()*0.4:.2f}"/>')
    s.add('<rect y="660" width="1200" height="140" fill="#b9a46a"/>'); out["frodo-journey/forest"] = s
    return out

with sync_playwright() as p:
    b = p.chromium.launch(); page = b.new_page()
    for key, s in scenes().items():
        rid, name = key.split("/"); os.makedirs(f"{OUT}/{rid}", exist_ok=True)
        page.set_viewport_size({"width": s.w, "height": s.h}); page.set_content(f'<html><body style="margin:0">{s.svg()}</body></html>')
        page.screenshot(path=f"{OUT}/{rid}/{name}.jpg", type="jpeg", quality=90)
    b.close()
print("images done")
