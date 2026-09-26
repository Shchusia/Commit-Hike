// Package render turns a route and a distance into a drawable scene.
//
// Keeping geometry in the core means every front end (VS Code, JetBrains,
// a terminal, a README badge) draws exactly the same trail. Front ends that
// want full control use the JSON scene; the rest can take the SVG.
package render

import (
	"fmt"
	"hash/fnv"
	"html"
	"math"
	"strings"
)

// Point is in normalized units: 0..1 on both axes, y grows downward.
type Point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// Marker is a laid-out waypoint.
type Marker struct {
	ID         string  `json:"id"`
	Label      string  `json:"label"`
	At         Point   `json:"at"`
	Passed     bool    `json:"passed"`
	LabelRight bool    `json:"label_right"`
	Fraction   float64 `json:"fraction"`
}

// Scene is everything needed to draw a trail.
type Scene struct {
	Path    []Point  `json:"path"`    // polyline, start first
	Walked  float64  `json:"walked"`  // 0..1 share of the path already walked
	Hiker   Point    `json:"hiker"`   // current position
	Markers []Marker `json:"markers"` // waypoints
}

// MarkerInput describes a waypoint before layout.
type MarkerInput struct {
	ID    string
	Label string
	AtM   float64
}

// Build lays out a route. custom is the route's optional hand-drawn path;
// without it a winding trail is generated from the route id, so the same
// route always looks the same.
func Build(routeID string, custom [][2]float64, lengthM, distanceM float64, wps []MarkerInput) Scene {
	pts := make([]Point, 0, len(custom))
	for _, p := range custom {
		pts = append(pts, Point{p[0], p[1]})
	}
	if len(pts) < 2 {
		pts = generate(routeID)
	}
	path := newPolyline(pts)
	walked := clamp01(distanceM / lengthM)
	sc := Scene{Path: pts, Walked: walked, Hiker: path.at(walked)}
	for _, w := range wps {
		f := clamp01(w.AtM / lengthM)
		p := path.at(f)
		sc.Markers = append(sc.Markers, Marker{
			ID: w.ID, Label: w.Label, At: p, Fraction: f,
			Passed: w.AtM <= distanceM, LabelRight: p.X < 0.5,
		})
	}
	return sc
}

// SVG renders a scene as a standalone image (width in px; height follows).
func SVG(sc Scene, title string, width float64) string {
	const aspect = 380.0 / 300.0
	w, h := width, width*aspect
	px := func(p Point) string { return fmt.Sprintf("%.1f,%.1f", p.X*w, p.Y*h) }
	var d strings.Builder
	for i, p := range sc.Path {
		if i == 0 {
			d.WriteString("M" + px(p))
		} else {
			d.WriteString(" L" + px(p))
		}
	}
	scaled := make([]Point, len(sc.Path))
	for i, p := range sc.Path {
		scaled[i] = Point{p.X * w, p.Y * h}
	}
	// Dash length must be measured in pixels, or the pattern repeats and a
	// second "walked" stroke appears near the end of the trail.
	total := newPolyline(scaled).length
	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %.0f %.0f" width="%.0f" height="%.0f" role="img">`, w, h, w, h)
	fmt.Fprintf(&b, `<title>%s</title>`, html.EscapeString(title))
	fmt.Fprintf(&b, `<path d="%s" fill="none" stroke="#8c8f96" stroke-width="2" stroke-dasharray="2 7" stroke-linecap="round"/>`, d.String())
	fmt.Fprintf(&b, `<path d="%s" fill="none" stroke="#e8b53d" stroke-width="4" stroke-linecap="round" stroke-dasharray="%.1f %.1f"/>`,
		d.String(), sc.Walked*total, total)
	for _, m := range sc.Markers {
		fill := "none"
		if m.Passed {
			fill = "#e8b53d"
		}
		fmt.Fprintf(&b, `<rect x="%.1f" y="%.1f" width="7" height="14" rx="1" fill="%s" stroke="#e8b53d" stroke-width="1.5"/>`,
			m.At.X*w-3.5, m.At.Y*h-7, fill)
		anchor, dx := "end", -12.0
		if m.LabelRight {
			anchor, dx = "start", 12.0
		}
		fmt.Fprintf(&b, `<text x="%.1f" y="%.1f" font-family="sans-serif" font-size="12" fill="#8c8f96" text-anchor="%s">%s</text>`,
			m.At.X*w+dx, m.At.Y*h+4, anchor, html.EscapeString(m.Label))
	}
	fmt.Fprintf(&b, `<circle cx="%.1f" cy="%.1f" r="6" fill="#e8b53d"/>`, sc.Hiker.X*w, sc.Hiker.Y*h)
	b.WriteString(`</svg>`)
	return b.String()
}

// ---- geometry ----

// generate builds a smooth serpentine trail, bottom (start) to top (end).
func generate(seed string) []Point {
	h := fnv.New32a()
	_, _ = h.Write([]byte(seed)) // hash writes never fail
	state := h.Sum32() | 1
	rnd := func() float64 { // xorshift32: tiny, deterministic, good enough for shapes
		state ^= state << 13
		state ^= state >> 17
		state ^= state << 5
		return float64(state%10000) / 10000
	}
	const n = 7
	anchors := make([]Point, 0, n+1)
	for i := 0; i <= n; i++ {
		side := -1.0
		if i%2 == 1 {
			side = 1
		}
		x := 0.5 + side*(0.12+rnd()*0.14)
		if i == 0 || i == n {
			x = 0.5 + side*0.06
		}
		anchors = append(anchors, Point{x, 0.95 - 0.9*float64(i)/n})
	}
	return catmullRom(anchors, 16)
}

// catmullRom samples a smooth curve through the anchors.
func catmullRom(a []Point, steps int) []Point {
	get := func(i int) Point { return a[max(0, min(len(a)-1, i))] }
	out := []Point{a[0]}
	for i := 0; i < len(a)-1; i++ {
		p0, p1, p2, p3 := get(i-1), get(i), get(i+1), get(i+2)
		for s := 1; s <= steps; s++ {
			t := float64(s) / float64(steps)
			t2, t3 := t*t, t*t*t
			f := func(a, b, c, d float64) float64 {
				return 0.5 * (2*b + (-a+c)*t + (2*a-5*b+4*c-d)*t2 + (-a+3*b-3*c+d)*t3)
			}
			out = append(out, Point{f(p0.X, p1.X, p2.X, p3.X), f(p0.Y, p1.Y, p2.Y, p3.Y)})
		}
	}
	return out
}

type polyline struct {
	pts    []Point
	cum    []float64
	length float64
}

func newPolyline(pts []Point) polyline {
	cum := make([]float64, len(pts))
	for i := 1; i < len(pts); i++ {
		cum[i] = cum[i-1] + math.Hypot(pts[i].X-pts[i-1].X, pts[i].Y-pts[i-1].Y)
	}
	return polyline{pts: pts, cum: cum, length: cum[len(cum)-1]}
}

// at returns the point at fraction f (0..1) of the arc length.
func (p polyline) at(f float64) Point {
	target := clamp01(f) * p.length
	for i := 1; i < len(p.pts); i++ {
		if p.cum[i] >= target {
			seg := p.cum[i] - p.cum[i-1]
			k := 0.0
			if seg > 0 {
				k = (target - p.cum[i-1]) / seg
			}
			a, b := p.pts[i-1], p.pts[i]
			return Point{a.X + (b.X-a.X)*k, a.Y + (b.Y-a.Y)*k}
		}
	}
	return p.pts[len(p.pts)-1]
}

func clamp01(v float64) float64 { return math.Max(0, math.Min(1, v)) }
