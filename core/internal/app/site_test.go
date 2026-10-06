package app

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/Shchusia/commit-hike/core/content"
	"github.com/Shchusia/commit-hike/core/internal/protocol"
	"github.com/Shchusia/commit-hike/core/internal/site"
)

// packOf zips the built-in demo route under another id (as the site would serve it).
func packOf(t *testing.T, id string, broken bool) []byte {
	t.Helper()
	var buf bytes.Buffer
	z := zip.NewWriter(&buf)
	root := "routes/demo-trail"
	err := fs.WalkDir(content.FS, root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, _ := fs.ReadFile(content.FS, p)
		rel := strings.TrimPrefix(p, root+"/")
		if rel == "route.json" {
			var r map[string]any
			_ = json.Unmarshal(data, &r)
			r["id"] = id
			if broken {
				r["length_m"] = -1
			}
			data, _ = json.Marshal(r)
		}
		w, _ := z.Create(path.Join(id, rel))
		_, _ = w.Write(data)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = z.Close()
	return buf.Bytes()
}

// zipFile writes a pack into a temporary .zip file.
func zipFile(t *testing.T, data []byte) string {
	t.Helper()
	p := t.TempDir() + "/pack.zip"
	if err := os.WriteFile(p, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

// fakeSite serves a catalogue of cards and their packs, like commit-hike.dev.
func fakeSite(t *testing.T, versions map[string]int, packs map[string][]byte) {
	t.Helper()
	card := func(id string) map[string]any {
		return map[string]any{
			"id": id, "title": "Title of " + id, "titles": map[string]string{"en": "Title of " + id, "uk": "Назва " + id},
			"author": "Olena", "length_m": 5000, "stops": 3, "version": versions[id], "page": "https://site/routes/" + id,
		}
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/routes":
			var cards []map[string]any
			for id := range versions {
				cards = append(cards, card(id))
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"total": len(cards), "page": 1, "pages": 1, "routes": cards, "q": r.URL.RawQuery})
		case strings.HasPrefix(r.URL.Path, "/api/routes/"):
			id := strings.TrimPrefix(r.URL.Path, "/api/routes/")
			if _, ok := versions[id]; !ok {
				http.NotFound(w, r)
				return
			}
			_ = json.NewEncoder(w).Encode(card(id))
		case strings.HasSuffix(r.URL.Path, "/download"):
			id := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/routes/"), "/download")
			if p, ok := packs[id]; ok {
				_, _ = w.Write(p)
				return
			}
			http.NotFound(w, r)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	t.Setenv("COMMIT_HIKE_SITE", srv.URL)
}

func TestSearchingTheSiteShowsWhatsInstalled(t *testing.T) {
	s, _ := setup(t, "demo-trail")
	fakeSite(t, map[string]int{"lake-walk": 2, "demo-trail": 1, "mine": 1}, map[string][]byte{"lake-walk": packOf(t, "lake-walk", false)})
	if _, err := s.ImportRoute(zipFile(t, packOf(t, "mine", false)), false, "en"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SiteInstall("lake-walk", false, "en", "test"); err != nil {
		t.Fatal(err)
	}
	page, err := s.SiteRoutes(site.Query{Q: "lake"}, "uk", "test")
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]protocol.SiteRoute{}
	for _, r := range page.Routes {
		got[r.ID] = r
	}
	if got["lake-walk"].Installed != "site" || got["lake-walk"].InstalledVersion != 2 || got["lake-walk"].Update {
		t.Fatalf("lake-walk: %+v", got["lake-walk"])
	}
	if got["demo-trail"].Installed != "builtin" || got["mine"].Installed != "local" || got["lake-walk"].Title != "Назва lake-walk" {
		t.Fatalf("states: %+v", got)
	}
}

func TestInstallingAndUpdatingFromTheSite(t *testing.T) {
	s, _ := setup(t, "demo-trail")
	versions := map[string]int{"lake-walk": 1}
	fakeSite(t, versions, map[string][]byte{"lake-walk": packOf(t, "lake-walk", false)})
	got, err := s.SiteInstall("lake-walk", false, "en", "test")
	if err != nil || got.Route.ID != "lake-walk" || got.Version != 1 {
		t.Fatalf("install: %+v %v", got, err)
	}
	if _, err := s.SetJourney(ScopeGlobal, "", "lake-walk", false, ""); err != nil {
		t.Fatalf("an installed route can be walked: %v", err)
	}
	versions["lake-walk"] = 3 // the author published a new version
	page, _ := s.SiteRoutes(site.Query{}, "en", "test")
	if !page.Routes[0].Update {
		t.Fatalf("an update should be offered: %+v", page.Routes[0])
	}
	if got, err = s.SiteInstall("lake-walk", false, "en", "test"); err != nil || got.Version != 3 {
		t.Fatalf("update: %+v %v", got, err)
	}
	if page, _ = s.SiteRoutes(site.Query{}, "en", "test"); page.Routes[0].Update || page.Routes[0].InstalledVersion != 3 {
		t.Fatalf("after the update: %+v", page.Routes[0])
	}
}

func TestSiteInstallRefusals(t *testing.T) {
	s, _ := setup(t, "demo-trail")
	fakeSite(t, map[string]int{"mine": 1, "demo-trail": 1, "broken": 1, "sneaky": 1}, map[string][]byte{
		"mine": packOf(t, "mine", false), "demo-trail": packOf(t, "demo-trail", false),
		"broken": packOf(t, "broken", true), "sneaky": packOf(t, "something-else", false),
	})
	if _, err := s.ImportRoute(zipFile(t, packOf(t, "mine", false)), false, "en"); err != nil {
		t.Fatal(err)
	}
	code := func(err error) string {
		var e *Error
		if errors.As(err, &e) {
			return e.Code
		}
		return fmt.Sprint(err)
	}
	if _, err := s.SiteInstall("mine", false, "en", "t"); code(err) != protocol.CodeRouteExists {
		t.Fatalf("your own route isn't replaced without asking: %v", err)
	}
	if _, err := s.SiteInstall("mine", true, "en", "t"); err != nil {
		t.Fatalf("…but is when asked: %v", err)
	}
	if _, err := s.SiteInstall("demo-trail", true, "en", "t"); code(err) != protocol.CodeRouteExists {
		t.Fatalf("built-in routes are never replaced: %v", err)
	}
	if _, err := s.SiteInstall("broken", false, "en", "t"); code(err) != protocol.CodeInvalidRoute {
		t.Fatalf("a broken pack: %v", err)
	}
	if _, err := s.SiteInstall("sneaky", false, "en", "t"); code(err) != protocol.CodeInvalidRoute {
		t.Fatalf("a pack with another id: %v", err)
	}
	if _, ok := s.routes["something-else"]; ok {
		t.Fatal("the wrong route must not stay installed")
	}
	if _, err := s.SiteInstall("nowhere", false, "en", "t"); code(err) != protocol.CodeUnknownRoute {
		t.Fatalf("not on the site: %v", err)
	}
	if _, err := s.SiteInstall("../etc", false, "en", "t"); err == nil {
		t.Fatal("ids are checked")
	}
	t.Setenv("COMMIT_HIKE_SITE", "http://127.0.0.1:1")
	if _, err := s.SiteRoutes(site.Query{}, "en", "t"); code(err) != protocol.CodeSiteUnreachable {
		t.Fatalf("offline: %v", err)
	}
}

func TestRemovingASiteRouteForgetsIt(t *testing.T) {
	s, _ := setup(t, "demo-trail")
	fakeSite(t, map[string]int{"lake-walk": 1}, map[string][]byte{"lake-walk": packOf(t, "lake-walk", false)})
	if _, err := s.SiteInstall("lake-walk", false, "en", "t"); err != nil {
		t.Fatal(err)
	}
	if err := s.RemoveRoute("lake-walk"); err != nil {
		t.Fatal(err)
	}
	if _, ok := s.siteInstalls()["lake-walk"]; ok {
		t.Fatal("removed routes are forgotten")
	}
}
