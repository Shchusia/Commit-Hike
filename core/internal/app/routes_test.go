package app

import (
	"archive/zip"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Shchusia/commit-hike/core/internal/protocol"
)

func code(err error) string {
	var e *Error
	if errors.As(err, &e) {
		return e.Code
	}
	return ""
}

func TestTemplateImportReplaceRemove(t *testing.T) {
	s, _ := setup(t, "")
	work := t.TempDir()

	dir, err := s.RouteTemplate("my-trail", work)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.RouteTemplate("Bad Id", work); code(err) != protocol.CodeInvalidArgument {
		t.Fatalf("bad id: %v", err)
	}
	if _, err := s.RouteTemplate("demo-trail", work); code(err) != protocol.CodeRouteExists {
		t.Fatalf("built-in id: %v", err)
	}

	r, err := s.ImportRoute(dir, false, "uk")
	if err != nil {
		t.Fatalf("the template must be a valid route: %v", err)
	}
	if r.ID != "my-trail" || r.Builtin || r.Name != "Моя стежка" || r.Waypoints[1].Kind != "viewpoint" || len(r.Biomes) != 5 ||
		len(r.Facts) != 1 || len(r.Objects) != 1 || r.Objects[0].Caption != "Вказівник" || r.AscentM != 480 {
		t.Fatalf("imported: %+v", r)
	}
	if _, err := s.ImportRoute(dir, false, ""); code(err) != protocol.CodeRouteExists {
		t.Fatalf("second import without --replace: %v", err)
	}
	if _, err := s.ImportRoute(dir, true, ""); err != nil {
		t.Fatalf("replace: %v", err)
	}

	// A fresh process sees the route too: it was copied into the data dir.
	s2, err := New(s.st.Dir)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, rt := range s2.Routes("en") {
		found = found || rt.ID == "my-trail"
	}
	if !found {
		t.Fatal("imported route not persisted")
	}

	if _, err := s.SetJourney(ScopeGlobal, "", "my-trail", true, ""); err != nil {
		t.Fatal(err)
	}
	if err := s.RemoveRoute("my-trail"); code(err) != protocol.CodeRouteInUse {
		t.Fatalf("remove while in use: %v", err)
	}
	if _, err := s.SetJourney(ScopeGlobal, "", "demo-trail", true, ""); err != nil {
		t.Fatal(err)
	}
	if err := s.RemoveRoute("my-trail"); err != nil {
		t.Fatal(err)
	}
	if err := s.RemoveRoute("demo-trail"); code(err) != protocol.CodeInvalidArgument {
		t.Fatalf("removing a built-in route: %v", err)
	}
	if _, err := os.Stat(filepath.Join(s.st.UserRoutesDir(), "my-trail")); !os.IsNotExist(err) {
		t.Fatal("route folder should be gone")
	}
}

func writeZip(t *testing.T, files map[string]string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "route.zip")
	f, err := os.Create(p)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	for name, body := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestImportZip(t *testing.T) {
	s, _ := setup(t, "")
	route := `{"id":"zipped","length_m":1000,"waypoints":[{"id":"a","at_m":0,"kind":"hut"}]}`
	text := `{"name":"Zipped","description":"d","waypoints":{"a":{"name":"A"}}}`

	// Pack inside one top-level folder, plus files that must be ignored.
	good := writeZip(t, map[string]string{
		"zipped/route.json":      route,
		"zipped/locales/en.json": text,
		"zipped/notes.txt":       "ignored",
	})
	r, err := s.ImportRoute(good, false, "")
	if err != nil || r.ID != "zipped" {
		t.Fatalf("zip import: %+v %v", r, err)
	}
	if _, err := os.Stat(filepath.Join(s.st.UserRoutesDir(), "zipped", "notes.txt")); !os.IsNotExist(err) {
		t.Error("unknown files must not be copied")
	}

	// A malicious entry name must never be written outside the routes dir.
	evil := writeZip(t, map[string]string{
		"route.json":      strings.Replace(route, "zipped", "evil", 1),
		"locales/en.json": text,
		"../escape.json":  "{}",
	})
	if _, err := s.ImportRoute(evil, false, ""); err != nil {
		t.Fatalf("evil zip with a valid pack at the root: %v", err)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(s.st.UserRoutesDir()), "escape.json")); !os.IsNotExist(err) {
		t.Fatal("zip slip: a file escaped the routes folder")
	}

	cases := map[string]string{
		"no route.json": writeZip(t, map[string]string{"readme.txt": "hi"}),
		"missing text": writeZip(t, map[string]string{
			"route.json":      strings.Replace(route, "zipped", "broken", 1),
			"locales/en.json": `{"name":"B","description":"d"}`,
		}),
		"unknown kind": writeZip(t, map[string]string{
			"route.json":      strings.Replace(strings.Replace(route, "zipped", "k", 1), `"hut"`, `"spaceport"`, 1),
			"locales/en.json": text,
		}),
		"built-in id": writeZip(t, map[string]string{
			"route.json":      strings.Replace(route, "zipped", "demo-trail", 1),
			"locales/en.json": text,
		}),
	}
	for name, p := range cases {
		_, err := s.ImportRoute(p, false, "")
		if c := code(err); c != protocol.CodeInvalidRoute && c != protocol.CodeRouteExists {
			t.Errorf("%s: want invalid_route or route_exists, got %v", name, err)
		}
	}
	if _, err := s.ImportRoute(filepath.Join(t.TempDir(), "x.txt"), false, ""); code(err) != protocol.CodeInvalidRoute {
		t.Errorf("missing file: %v", err)
	}
}

func TestDailyStatsAndMapData(t *testing.T) {
	s, r := setup(t, "chornohora-ridge")
	now := time.Unix(base+50000, 0)
	// one commit two days ago, two today
	r.ts = now.Unix() - 2*86400
	r.commit("", "a.go", 1) // 30 m
	r.ts = now.Unix() - 600
	r.commit("", "a.go", 3) // 50 m
	r.commit("", "b.go", 3) // 50 m
	res := scan(t, s, r.dir, "en")
	d := res.Global.Daily
	if len(d) != 14 || d[13].Date != now.UTC().Format(time.DateOnly) || d[13].M != 100 || d[11].M != 30 || d[12].M != 0 {
		t.Fatalf("daily: %+v", d)
	}
	w := res.Global.Route.Waypoints[1]
	if w.Kind != "peak" || w.ElevationM != 2061 || len(res.Global.Route.Biomes) == 0 {
		t.Fatalf("map data: %+v", res.Global.Route)
	}
}
