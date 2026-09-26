package routes

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/commit-hike/commit-hike/core/content"
	"github.com/commit-hike/commit-hike/core/internal/i18n"
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

func TestMergeKeepsBuiltinRoutes(t *testing.T) {
	b := map[string]*Route{"a": {ID: "a", Builtin: true}}
	u := map[string]*Route{"a": {ID: "a"}, "b": {ID: "b"}}
	m, errs := Merge(b, u)
	if len(m) != 2 || !m["a"].Builtin || len(errs) != 1 || !strings.Contains(errs[0].Error(), "built-in") {
		t.Fatalf("m=%v errs=%v", m, errs)
	}
}
