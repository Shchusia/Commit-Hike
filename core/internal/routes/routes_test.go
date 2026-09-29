package routes

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/Shchusia/commit-hike/core/content"
	"github.com/Shchusia/commit-hike/core/internal/i18n"
)

// Every built-in route must load, and every translation must be complete:
// a missing Ukrainian string should fail CI, not show English to users.
func TestBuiltinRoutesAreValidAndFullyTranslated(t *testing.T) {
	rs, err := Load(content.FS, "routes", true)
	if err != nil {
		t.Fatal(err)
	}
	if len(rs) < 2 {
		t.Fatalf("expected at least 2 routes, got %d", len(rs))
	}
	for id, r := range rs {
		for _, loc := range r.Locales() {
			if miss := r.Texts.Missing(r.DefaultLocale, loc); len(miss) > 0 {
				t.Errorf("%s: locales/%s.json is missing %v", id, loc, miss)
			}
			if extra := r.Texts.Missing(loc, r.DefaultLocale); len(extra) > 0 {
				t.Errorf("%s: locales/%s.json has keys not in %s: %v", id, loc, r.DefaultLocale, extra)
			}
		}
	}
	r := rs["chornohora-ridge"]
	if got := r.T(i18n.Chain("uk-UA"), "waypoints.hoverla.name"); got != "Говерла" {
		t.Errorf("uk name = %q", got)
	}
	if got := r.T(i18n.Chain("de"), "waypoints.hoverla.name"); got != "Hoverla" {
		t.Errorf("fallback name = %q", got)
	}
}

func TestLoadRejectsBrokenPacks(t *testing.T) {
	good := `{"id":"x","length_m":100,"waypoints":[{"id":"a","at_m":0}]}`
	goodText := `{"name":"X","description":"d","waypoints":{"a":{"name":"A"}}}`
	cases := map[string]fstest.MapFS{
		"unknown field": {
			"r/x/route.json":      {Data: []byte(`{"id":"x","length_m":100,"extra_field":1,"waypoints":[]}`)},
			"r/x/locales/en.json": {Data: []byte(goodText)},
		},
		"id mismatch": {
			"r/y/route.json":      {Data: []byte(good)},
			"r/y/locales/en.json": {Data: []byte(goodText)},
		},
		"missing translation": {
			"r/x/route.json":      {Data: []byte(good)},
			"r/x/locales/en.json": {Data: []byte(`{"name":"X","description":"d"}`)},
		},
		"waypoint outside": {
			"r/x/route.json":      {Data: []byte(`{"id":"x","length_m":100,"waypoints":[{"id":"a","at_m":500}]}`)},
			"r/x/locales/en.json": {Data: []byte(goodText)},
		},
		"uppercase id": {
			"r/X/route.json":      {Data: []byte(strings.Replace(good, `"x"`, `"X"`, 1))},
			"r/X/locales/en.json": {Data: []byte(goodText)},
		},
		"asset outside assets/": {
			"r/x/route.json":      {Data: []byte(withObject(good, `"../../etc/passwd.svg"`))},
			"r/x/locales/en.json": {Data: []byte(goodText)},
		},
		"asset missing": {
			"r/x/route.json":      {Data: []byte(withObject(good, `"assets/nope.svg"`))},
			"r/x/locales/en.json": {Data: []byte(goodText)},
		},
		"asset type": {
			"r/x/route.json":      {Data: []byte(withObject(good, `"assets/run.js"`))},
			"r/x/locales/en.json": {Data: []byte(goodText)},
			"r/x/assets/run.js":   {Data: []byte("alert(1)")},
		},
		"map image without path": {
			"r/x/route.json":      {Data: []byte(strings.Replace(good, `"length_m"`, `"map_image":"assets/m.svg","length_m"`, 1))},
			"r/x/locales/en.json": {Data: []byte(goodText)},
			"r/x/assets/m.svg":    {Data: []byte("<svg/>")},
		},
		"fact without text": {
			"r/x/route.json":      {Data: []byte(strings.Replace(good, `"waypoints"`, `"facts":[{"id":"f","at_m":10}],"waypoints"`, 1))},
			"r/x/locales/en.json": {Data: []byte(goodText)},
		},
		"underground outside": {
			"r/x/route.json":      {Data: []byte(strings.Replace(good, `"waypoints"`, `"underground":[{"from_m":50,"to_m":500}],"waypoints"`, 1))},
			"r/x/locales/en.json": {Data: []byte(goodText)},
		},
		"underground overlap": {
			"r/x/route.json":      {Data: []byte(strings.Replace(good, `"waypoints"`, `"underground":[{"from_m":10,"to_m":50},{"from_m":40,"to_m":60}],"waypoints"`, 1))},
			"r/x/locales/en.json": {Data: []byte(goodText)},
		},
		"danger without text": {
			"r/x/route.json":      {Data: []byte(strings.Replace(good, `"waypoints"`, `"dangers":[{"id":"d","at_m":10}],"waypoints"`, 1))},
			"r/x/locales/en.json": {Data: []byte(goodText)},
		},
		"bad achievement rule": {
			"r/x/route.json":      {Data: []byte(strings.Replace(good, `"waypoints"`, `"achievements":[{"id":"h","rule":{"type":"altitude"}}],"waypoints"`, 1))},
			"r/x/locales/en.json": {Data: []byte(`{"name":"X","description":"d","waypoints":{"a":{"name":"A"}},"achievements":{"h":{"name":"H","description":"d"}}}`)},
		},
	}
	for name, fsys := range cases {
		if _, err := Load(fsys, "r", false); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
	ok := fstest.MapFS{
		"r/x/route.json":      {Data: []byte(good)},
		"r/x/locales/en.json": {Data: []byte(goodText)},
	}
	if _, err := Load(ok, "r", false); err != nil {
		t.Fatal(err)
	}
}

func withObject(route, asset string) string {
	return strings.Replace(route, `"waypoints"`, `"objects":[{"id":"o","at_m":10,"asset":`+asset+`}],"waypoints"`, 1)
}

func TestElevationHelpers(t *testing.T) {
	r := &Route{
		LengthM: 100, Waypoints: []Waypoint{{ID: "a", AtM: 0, ElevationM: 100}, {ID: "b", AtM: 100, ElevationM: 120}},
		Profile: []ProfilePoint{{AtM: 50, ElevationM: 200}},
	}
	if e, ok := r.ElevationAt(25); !ok || e != 150 {
		t.Fatalf("elevation at 25 = %v", e)
	}
	if got := r.Ascent(100); got != 100 { // up 100, down 80: only the climb counts
		t.Fatalf("ascent = %v", got)
	}
	if got := r.Ascent(25); got != 50 {
		t.Fatalf("partial ascent = %v", got)
	}
	if got := r.MaxElevation(100); got != 200 {
		t.Fatalf("max = %v", got)
	}
	if _, ok := (&Route{}).ElevationAt(0); ok {
		t.Fatal("a route without heights has no elevation")
	}
}

func TestMergeKeepsBuiltinRoutes(t *testing.T) {
	b := map[string]*Route{"a": {ID: "a", Builtin: true}}
	u := map[string]*Route{"a": {ID: "a"}, "b": {ID: "b"}}
	m, errs := Merge(b, u)
	if len(m) != 2 || !m["a"].Builtin || len(errs) != 1 || !strings.Contains(errs[0].Error(), "built-in") {
		t.Fatalf("m=%v errs=%v", m, errs)
	}
}

func TestTrackValidation(t *testing.T) {
	text := `{"name":"X","description":"d","waypoints":{"a":{"name":"A"},"b":{"name":"B"}}}`
	pack := func(route string) fstest.MapFS {
		return fstest.MapFS{"r/x/route.json": {Data: []byte(route)}, "r/x/locales/en.json": {Data: []byte(text)}}
	}
	good := `{"id":"x","length_m":1000,"track":[[48.16,24.53],[48.15,24.51]],
		"waypoints":[{"id":"a","at_m":0,"lat":48.16,"lon":24.53},{"id":"b","at_m":1000}]}`
	if _, err := Load(pack(good), "r", false); err != nil {
		t.Fatal(err)
	}
	bad := map[string]string{
		"one point":       `{"id":"x","length_m":1000,"track":[[48.1,24.5]],"waypoints":[{"id":"a","at_m":0},{"id":"b","at_m":1000}]}`,
		"lat out":         `{"id":"x","length_m":1000,"track":[[98.1,24.5],[48,24]],"waypoints":[{"id":"a","at_m":0},{"id":"b","at_m":1000}]}`,
		"path and track":  `{"id":"x","length_m":1000,"path":[[0,0],[1,1]],"track":[[48,24],[47,24]],"waypoints":[{"id":"a","at_m":0},{"id":"b","at_m":1000}]}`,
		"lat only":        `{"id":"x","length_m":1000,"track":[[48,24],[47,24]],"waypoints":[{"id":"a","at_m":0,"lat":48},{"id":"b","at_m":1000}]}`,
		"coords no track": `{"id":"x","length_m":1000,"waypoints":[{"id":"a","at_m":0,"lat":48,"lon":24},{"id":"b","at_m":1000}]}`,
	}
	for name, route := range bad {
		if _, err := Load(pack(route), "r", false); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}

func TestMapCoordinatesAndLoop(t *testing.T) {
	text := `{"name":"X","description":"d","waypoints":{"a":{"name":"A"},"b":{"name":"B"}}}`
	pack := func(route string) fstest.MapFS {
		return fstest.MapFS{"r/x/route.json": {Data: []byte(route)}, "r/x/locales/en.json": {Data: []byte(text)}}
	}
	good := `{"id":"x","length_m":1000,"loop":true,"path":[[0.1,0.9],[0.5,0.1],[0.9,0.9]],
		"waypoints":[{"id":"a","at_m":0,"x":0.1,"y":0.9},{"id":"b","at_m":1000}]}`
	rs, err := Load(pack(good), "r", false)
	if err != nil {
		t.Fatal(err)
	}
	if !rs["x"].Loop || *rs["x"].Waypoints[0].X != 0.1 {
		t.Fatalf("loaded: %+v", rs["x"])
	}
	bad := map[string]string{
		"x only":       `{"id":"x","length_m":1000,"path":[[0,0],[1,1]],"waypoints":[{"id":"a","at_m":0,"x":0.5},{"id":"b","at_m":1000}]}`,
		"out of range": `{"id":"x","length_m":1000,"path":[[0,0],[1,1]],"waypoints":[{"id":"a","at_m":0,"x":1.5,"y":0.5},{"id":"b","at_m":1000}]}`,
		"xy, no path":  `{"id":"x","length_m":1000,"waypoints":[{"id":"a","at_m":0,"x":0.5,"y":0.5},{"id":"b","at_m":1000}]}`,
	}
	for name, route := range bad {
		if _, err := Load(pack(route), "r", false); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}
