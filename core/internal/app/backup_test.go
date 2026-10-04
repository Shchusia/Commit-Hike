package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Shchusia/commit-hike/core/internal/protocol"
)

func TestBackupMovesProgressToAnotherComputer(t *testing.T) {
	s, r := setup(t, "demo-trail")
	r.commit("", "a.go", 40)
	r.commit("", "b.go", 25)
	scan(t, s, r.dir, "en")
	if _, err := s.RestDays("sat,sun"); err != nil {
		t.Fatal(err)
	}
	before, _ := s.Status(r.dir, "en")
	// A link in the routes folder must not pull other files into the backup.
	secret := filepath.Join(t.TempDir(), "secret.txt")
	if err := os.WriteFile(secret, []byte("do not copy"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(s.st.UserRoutesDir(), 0o700); err != nil {
		t.Fatal(err)
	}
	if !isWindowsTest() {
		if err := os.Symlink(secret, filepath.Join(s.st.UserRoutesDir(), "link.txt")); err != nil {
			t.Fatal(err)
		}
	}

	file := filepath.Join(t.TempDir(), "backup.json")
	res, err := s.ExportBackup(file, "1.2.3")
	if err != nil {
		t.Fatal(err)
	}
	if res.Commits != 2 || res.Routes != 0 {
		t.Fatalf("export: %+v (a symlinked file must be skipped)", res)
	}
	if info, _ := os.Stat(file); info.Mode().Perm()&0o077 != 0 && !isWindowsTest() {
		t.Errorf("the backup holds the key, it must be private: %v", info.Mode())
	}

	// A new computer: a fresh data directory with its own key.
	s2, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	s2.now, s2.loc, s2.score = s.now, s.loc, s.score
	if _, err := s2.ImportBackup(file, false); err != nil {
		t.Fatal(err)
	}
	after, _ := s2.Status(r.dir, "en")
	if after.Global.DistanceM != before.Global.DistanceM || len(after.RestDays) != 2 {
		t.Fatalf("restored: %v m, rest %v; want %v m", after.Global.DistanceM, after.RestDays, before.Global.DistanceM)
	}
	// The same commits scanned again on the new computer must not count twice.
	if res := scan(t, s2, r.dir, "en"); res.NewCommits != 0 {
		t.Fatalf("restored commits counted again: %d new", res.NewCommits)
	}
	// Restoring over existing progress needs an explicit replace.
	if _, err := s2.ImportBackup(file, false); code(err) != protocol.CodeDataExists {
		t.Fatalf("import over existing progress: %v", err)
	}
	if _, err := s2.ImportBackup(file, true); err != nil {
		t.Fatalf("replace: %v", err)
	}
}

func TestBackupRejectsDamagedFiles(t *testing.T) {
	s, _ := setup(t, "demo-trail")
	dir := t.TempDir()
	write := func(name string, v any) string {
		p := filepath.Join(dir, name)
		data, _ := json.Marshal(v)
		if err := os.WriteFile(p, data, 0o600); err != nil {
			t.Fatal(err)
		}
		return p
	}
	key := make([]byte, 32)
	cases := map[string]string{
		"not a backup":  write("a.json", map[string]any{"format": "zip"}),
		"newer format":  write("b.json", map[string]any{"format": backupFormat, "version": 99, "key": key}),
		"no key":        write("c.json", map[string]any{"format": backupFormat, "version": 1}),
		"escaping path": write("d.json", map[string]any{"format": backupFormat, "version": 1, "key": key, "config": map[string]any{"version": 1}, "state": map[string]any{}, "routes": map[string][]byte{"../../evil": []byte("x")}}),
	}
	for name, p := range cases {
		if _, err := s.ImportBackup(p, true); code(err) != protocol.CodeInvalidBackup {
			t.Errorf("%s: got %v", name, err)
		}
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(s.st.Dir), "evil")); err == nil {
		t.Fatal("a damaged backup wrote outside the data directory")
	}
}

func TestRestDaysAndHistory(t *testing.T) {
	s, r := setup(t, "demo-trail")
	if _, err := s.RestDays("sat,sunday"); err != nil {
		t.Fatal(err)
	}
	info, _ := s.RestDays("")
	if len(info.Days) != 2 || info.Days[0] != int(time.Sunday) || info.Days[1] != int(time.Saturday) {
		t.Fatalf("rest days: %v", info.Days)
	}
	for _, bad := range []string{"funday", "mon,tue,wed,thu,fri,sat,sun"} {
		if _, err := s.RestDays(bad); code(err) != protocol.CodeInvalidArgument {
			t.Errorf("%q: %v", bad, err)
		}
	}
	if info, _ := s.RestDays("none"); len(info.Days) != 0 {
		t.Fatalf("none: %v", info.Days)
	}

	r.commit("", "a.go", 40)
	r.commit("", "b.go", 25)
	scan(t, s, r.dir, "en")
	st, _ := s.Status(r.dir, "en")
	total, commits := 0.0, 0
	for _, d := range st.History {
		total += d.M
		commits += d.Commits
	}
	if len(st.History) == 0 || commits != 2 || total < st.TotalM-0.5 || total > st.TotalM+0.5 {
		t.Fatalf("history %+v doesn't add up to %v m / 2 commits", st.History, st.TotalM)
	}
}

func isWindowsTest() bool { return os.PathSeparator == '\\' }

func TestSettings(t *testing.T) {
	s, r := setup(t, "demo-trail")
	got, err := s.Settings(SettingsChange{})
	if err != nil || *got != (protocol.Settings{ReduceMotion: "auto", HighContrast: "auto", Notifications: "all"}) {
		t.Fatalf("defaults: %+v %v", got, err)
	}
	if got, _ = s.Settings(SettingsChange{ReduceMotion: "on", Notifications: "milestones"}); got.ReduceMotion != "on" || got.HighContrast != "auto" || got.Notifications != "milestones" {
		t.Fatalf("partial change: %+v", got)
	}
	if _, err := s.Settings(SettingsChange{HighContrast: "maybe"}); code(err) != protocol.CodeInvalidArgument {
		t.Fatalf("bad value: %v", err)
	}
	st, _ := s.Status(r.dir, "en")
	if st.Settings.ReduceMotion != "on" || st.Settings.Notifications != "milestones" {
		t.Fatalf("status carries the settings: %+v", st.Settings)
	}
	if got, _ = s.Settings(SettingsChange{ReduceMotion: "auto"}); got.ReduceMotion != "auto" {
		t.Fatalf("back to auto: %+v", got)
	}
	cfg, _ := s.st.LoadConfig()
	if cfg.Prefs.ReduceMotion != "" {
		t.Fatalf("defaults aren't stored: %+v", cfg.Prefs)
	}
}

func TestDiagnosticsHoldNothingPersonal(t *testing.T) {
	s, r := setup(t, "demo-trail")
	r.commit("", "a.go", 40)
	scan(t, s, r.dir, "en")
	d, err := s.Diagnostics("1.2.3")
	if err != nil {
		t.Fatal(err)
	}
	if !d.Initialized || d.Commits != 1 || d.Projects != 1 || d.Emails != 1 || len(d.Journeys) == 0 || d.CoreVersion != "1.2.3" {
		t.Fatalf("diagnostics: %+v", d)
	}
	raw, _ := json.Marshal(d)
	for _, private := range []string{"me@x.io", "x.io", r.dir, s.st.Dir, filepath.Base(r.dir)} {
		if strings.Contains(strings.ToLower(string(raw)), strings.ToLower(private)) {
			t.Fatalf("diagnostics leak %q: %s", private, raw)
		}
	}
	fresh, _ := New(t.TempDir())
	if d, err := fresh.Diagnostics("1.2.3"); err != nil || d.Initialized {
		t.Fatalf("before setup: %+v %v", d, err)
	}
}

func TestBadge(t *testing.T) {
	s, r := setup(t, "demo-trail")
	r.commit("", "a.go", 40)
	scan(t, s, r.dir, "en")
	b, err := s.Badge(r.dir, "en")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(b.SVG, "Commit Hike") || !strings.Contains(b.SVG, "117 m · Demo Trail") || b.FileName != "commit-hike-badge.svg" || !strings.Contains(b.Markdown, b.FileName) {
		t.Fatalf("badge: %+v", b)
	}
	for _, c := range []struct {
		m    float64
		lang string
		want string
	}{
		{34800, "en", "34.8 km"},
		{342400, "en", "342 km"},
		{34800, "uk", "34,8 км"},
		{34800, "pl", "34,8 km"},
		{34800, "de-DE", "34,8 km"},
		{34800, "es", "34,8 km"},
		{640, "uk", "640 м"},
		{2863000, "de", "2863 km"},
		{-5, "en", "0 m"},
	} {
		if got := FormatDistance(c.m, c.lang); got != c.want {
			t.Errorf("FormatDistance(%v, %s) = %q, want %q", c.m, c.lang, got, c.want)
		}
	}
}

func TestPassportKeepsStampsAcrossJourneys(t *testing.T) {
	s, r := setup(t, "demo-trail")
	for i := 0; i < 12; i++ {
		r.commit("", fmt.Sprintf("f%d.go", i), 120)
	}
	scan(t, s, r.dir, "en")
	st, _ := s.Status(r.dir, "en")
	if len(st.Passport) == 0 || st.RouteStops["demo-trail"] == 0 {
		t.Fatalf("stamps on the first trail: %+v", st.Passport)
	}
	first := len(st.Passport)
	for i := 1; i < len(st.Passport); i++ {
		if st.Passport[i].ReachedAt < st.Passport[i-1].ReachedAt {
			t.Fatal("stamps go oldest first")
		}
	}
	for _, p := range st.Passport {
		if p.Name == "" || p.RouteName == "" || p.ReachedAt == 0 {
			t.Fatalf("incomplete stamp: %+v", p)
		}
	}
	// walk another route: the first route's stamps stay
	if _, err := s.SetJourney(ScopeGlobal, "", "chornohora-ridge", false, ""); err != nil {
		t.Fatal(err)
	}
	st, _ = s.Status(r.dir, "en")
	kept := 0
	for _, p := range st.Passport {
		if p.RouteID == "demo-trail" {
			kept++
		}
	}
	if kept != first {
		t.Fatalf("stamps of the route left behind: %d, want %d", kept, first)
	}
	cfg, _ := s.st.LoadConfig()
	if len(cfg.PastJourneys) != 1 || cfg.PastJourneys[0].RouteID != "demo-trail" || cfg.PastJourneys[0].Until == 0 {
		t.Fatalf("past journeys: %+v", cfg.PastJourneys)
	}
}

func TestTheNextRouteOfASeries(t *testing.T) {
	s, r := setup(t, "chornohora-ridge")
	st, _ := s.Status(r.dir, "uk")
	n := st.Global.NextRoute
	if n == nil || n.ID != "svydovets-ridge" || n.SeriesName != "Карпати" || n.Name == "" || n.LengthM == 0 {
		t.Fatalf("next route: %+v", n)
	}
	if _, err := s.SetJourney(ScopeGlobal, "", "demo-trail", false, ""); err != nil {
		t.Fatal(err)
	}
	if st, _ = s.Status(r.dir, "en"); st.Global.NextRoute != nil {
		t.Fatalf("a route in no series has no next: %+v", st.Global.NextRoute)
	}
}

func TestInterfaceLanguagesAreAvailableBeforeRoutesAreTranslated(t *testing.T) {
	s, r := setup(t, "demo-trail")
	info, err := s.Locale(nil, "pl")
	if err != nil {
		t.Fatal(err)
	}
	for _, l := range []string{"en", "uk", "pl", "de", "es"} {
		if !slices.Contains(info.Available, l) {
			t.Errorf("%s missing from %v", l, info.Available)
		}
	}
	st, _ := s.Status(r.dir, "pl")
	if st.Locale != "pl" {
		t.Fatalf("a Polish interface stays Polish: %q", st.Locale)
	}
	if st.Global.Route.Name == "" {
		t.Fatal("route texts fall back to English when there's no Polish yet")
	}
}
