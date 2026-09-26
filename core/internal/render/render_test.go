package render

import (
	"fmt"
	"math"
	"reflect"
	"strings"
	"testing"
)

func TestBuildIsDeterministicAndOrdered(t *testing.T) {
	wps := []MarkerInput{{ID: "a", Label: "Start", AtM: 0}, {ID: "b", Label: "End", AtM: 100}}
	s1 := Build("route-x", nil, 100, 50, wps)
	s2 := Build("route-x", nil, 100, 50, wps)
	if !reflect.DeepEqual(s1, s2) {
		t.Fatal("same input must give the same scene")
	}
	if reflect.DeepEqual(s1.Path, Build("route-y", nil, 100, 50, wps).Path) {
		t.Error("different routes should look different")
	}
	start, end := s1.Path[0], s1.Path[len(s1.Path)-1]
	if start.Y < end.Y {
		t.Error("the trail should start at the bottom")
	}
	if !s1.Markers[0].Passed || s1.Markers[1].Passed || s1.Walked != 0.5 {
		t.Errorf("markers/walked wrong: %+v", s1)
	}
	for _, p := range s1.Path {
		if p.X < 0 || p.X > 1 || p.Y < 0 || p.Y > 1 {
			t.Fatalf("point outside unit square: %+v", p)
		}
	}
}

func TestCustomPathAndArcLength(t *testing.T) {
	sc := Build("r", [][2]float64{{0, 1}, {0, 0}}, 100, 25, nil)
	if math.Abs(sc.Hiker.Y-0.75) > 1e-9 || sc.Hiker.X != 0 {
		t.Errorf("hiker = %+v", sc.Hiker)
	}
}

func TestSVGEscapesLabels(t *testing.T) {
	sc := Build("r", nil, 100, 10, []MarkerInput{{ID: "a", Label: `<script>`, AtM: 50}})
	svg := SVG(sc, "T & T", 300)
	if strings.Contains(svg, "<script>") || !strings.Contains(svg, "&lt;script&gt;") || !strings.Contains(svg, "T &amp; T") {
		t.Error("labels must be escaped")
	}
	if !strings.HasPrefix(svg, "<svg") || !strings.HasSuffix(svg, "</svg>") {
		t.Error("not a standalone svg")
	}
}

func TestSVGWalkedDashCoversWholePath(t *testing.T) {
	sc := Build("r", nil, 100, 10, nil)
	svg := SVG(sc, "t", 300)
	// dasharray is "walked total": the gap must be at least the real length.
	var walked, gap float64
	idx := strings.LastIndex(svg, `stroke-dasharray="`)
	if n, err := fmt.Sscanf(svg[idx+len(`stroke-dasharray="`):], "%f %f", &walked, &gap); n != 2 || err != nil {
		t.Fatalf("cannot parse dasharray: %v", err)
	}
	scaled := make([]Point, len(sc.Path))
	for k, p := range sc.Path {
		scaled[k] = Point{p.X * 300, p.Y * 380}
	}
	real := newPolyline(scaled).length
	if math.Abs(gap-real) > 0.5 || math.Abs(walked-0.1*real) > 0.5 {
		t.Fatalf("dash %v %v, path length %v", walked, gap, real)
	}
}
