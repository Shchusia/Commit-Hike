package routes

import (
	"testing"
	"testing/fstest"

	"github.com/Shchusia/commit-hike/core/content"
)

func TestBuiltinSeries(t *testing.T) {
	builtin, err := Load(content.FS, "routes", true)
	if err != nil {
		t.Fatal(err)
	}
	series, err := LoadSeries(content.FS, "series.json", builtin)
	if err != nil || len(series) == 0 {
		t.Fatalf("built-in series: %v %v", series, err)
	}
	next, s, ok := Next(series, "chornohora-ridge")
	if !ok || next != "svydovets-ridge" || SeriesName(s, []string{"uk", "en"}) != "Карпати" {
		t.Fatalf("after Chornohora: %q in %+v", next, s)
	}
	if _, _, ok := Next(series, "gorgany-popadia-ring"); ok {
		t.Fatal("the last route of a series has no next")
	}
	if _, _, ok := Next(series, "demo-trail"); ok {
		t.Fatal("a route in no series has no next")
	}
}

func TestSeriesAreChecked(t *testing.T) {
	known := map[string]*Route{"a": {ID: "a"}, "b": {ID: "b"}, "c": {ID: "c"}}
	for name, body := range map[string]string{
		"unknown route":   `{"series":[{"id":"x","routes":["a","zz"],"name":{"en":"X"}}]}`,
		"one route":       `{"series":[{"id":"x","routes":["a"],"name":{"en":"X"}}]}`,
		"no english name": `{"series":[{"id":"x","routes":["a","b"],"name":{"uk":"Х"}}]}`,
		"in two series":   `{"series":[{"id":"x","routes":["a","b"],"name":{"en":"X"}},{"id":"y","routes":["b","c"],"name":{"en":"Y"}}]}`,
		"not json":        `{`,
	} {
		if _, err := LoadSeries(fstest.MapFS{"s.json": {Data: []byte(body)}}, "s.json", known); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}
