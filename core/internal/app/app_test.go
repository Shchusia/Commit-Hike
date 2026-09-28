package app

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Shchusia/commit-hike/core/internal/score"

	"github.com/Shchusia/commit-hike/core/internal/protocol"
	"github.com/Shchusia/commit-hike/core/internal/store"
)

// ---------- helpers: real git repositories in temp dirs ----------

type repo struct {
	t   *testing.T
	dir string
	ts  int64
}

const base = 1790000000 // fixed "now" region for deterministic dates

func newRepo(t *testing.T) *repo {
	t.Helper()
	r := &repo{t: t, dir: t.TempDir(), ts: base}
	r.git("", "init", "-q", "-b", "main")
	return r
}

func (r *repo) git(email string, args ...string) {
	r.t.Helper()
	if email == "" {
		email = "me@x.io"
	}
	cmd := exec.Command("git", append([]string{"-C", r.dir}, args...)...)
	date := fmt.Sprintf("@%d +0000", r.ts)
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=T", "GIT_AUTHOR_EMAIL="+email,
		"GIT_COMMITTER_NAME=T", "GIT_COMMITTER_EMAIL="+email,
		"GIT_AUTHOR_DATE="+date, "GIT_COMMITTER_DATE="+date,
		"GIT_CONFIG_GLOBAL=/dev/null")
	if out, err := cmd.CombinedOutput(); err != nil {
		r.t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// mustWrite writes a file or fails the test.
func mustWrite(t *testing.T, path, data string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
}

// commit appends `lines` new lines to file and commits them as email.
func (r *repo) commit(email, file string, lines int) {
	r.t.Helper()
	r.ts += 60
	p := filepath.Join(r.dir, file)
	var b strings.Builder
	if old, err := os.ReadFile(p); err == nil {
		b.Write(old)
	}
	for i := 0; i < lines; i++ {
		fmt.Fprintf(&b, "line %d %d\n", r.ts, i)
	}
	mustWrite(r.t, p, b.String())
	r.git(email, "add", "-A")
	r.git(email, "commit", "-q", "-m", "c")
}

func setup(t *testing.T, route string) (*Service, *repo) {
	t.Helper()
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	s.now = func() time.Time { return time.Unix(base+50000, 0) }
	s.loc = time.UTC // tests don't depend on the machine's time zone
	s.score = legacyScore()
	if _, err := s.Init(InitOptions{Emails: []string{" Me@X.io "}, RouteID: route, FromHistory: true}); err != nil {
		t.Fatal(err)
	}
	return s, newRepo(t)
}

func scan(t *testing.T, s *Service, dir, lang string) *protocol.ScanResult {
	t.Helper()
	res, err := s.Scan(dir, lang)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func eventsOf(res *protocol.ScanResult, typ string) []protocol.Event {
	var out []protocol.Event
	for _, e := range res.Events {
		if e.Type == typ {
			out = append(out, e)
		}
	}
	return out
}

// ---------- counting ----------

func TestScanCountsOnlyMyMeaningfulCommits(t *testing.T) {
	s, r := setup(t, "")
	r.commit("", "main.go", 1)              // 30 m
	r.commit("other@x.io", "main.go", 50)   // not mine
	r.commit("", "package-lock.json", 5000) // ignored file
	r.commit("", "node_modules/a/b.js", 10) // ignored dir
	b, err := os.ReadFile(filepath.Join(r.dir, "main.go"))
	if err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(r.dir, "main.go"), strings.ReplaceAll(string(b), " ", "  "))
	r.ts += 60
	r.git("", "commit", "-qam", "whitespace only")

	res := scan(t, s, r.dir, "en")
	if res.NewCommits != 1 || res.TotalM != 30 || res.AddedM != 30 {
		t.Fatalf("unexpected: %+v", res)
	}
	if again := scan(t, s, r.dir, "en"); again.NewCommits != 0 || again.AddedM != 0 {
		t.Fatalf("rescan counted again: %+v", again)
	}
}

func TestAmendAndCloneAreNotDoubleCounted(t *testing.T) {
	s, r := setup(t, "")
	r.commit("other@x.io", "README", 1) // root commit = project identity
	r.commit("", "a.go", 1)
	scan(t, s, r.dir, "")

	mustWrite(t, filepath.Join(r.dir, "a.go"), strings.Repeat("x\n", 3))
	r.git("", "commit", "-q", "-a", "--amend", "--no-edit")
	res := scan(t, s, r.dir, "")
	if res.NewCommits != 0 || res.UpdatedCommits != 1 || res.TotalM != 50 {
		t.Fatalf("amend: %+v", res)
	}
	clone := t.TempDir()
	if out, err := exec.Command("git", "clone", "-q", r.dir, clone).CombinedOutput(); err != nil {
		t.Fatalf("%v %s", err, out)
	}
	if res := scan(t, s, clone, ""); res.NewCommits != 0 || res.TotalM != 50 {
		t.Fatalf("clone: %+v", res)
	}
}

func TestDailyCap(t *testing.T) {
	s, r := setup(t, "")
	for i := 0; i < 20; i++ { // 20 * 200 = 4000 raw in one day
		r.commit("", "big.go", 1000)
	}
	if got := scan(t, s, r.dir, "").TotalM; got != 3250 { // 3000 + 1000*0.25
		t.Fatalf("daily cap: %v", got)
	}
}

// ---------- routes, story, achievements, languages ----------

func TestEventsAreTranslated(t *testing.T) {
	s, r := setup(t, "demo-trail")
	for i := 0; i < 5; i++ {
		r.commit("", "big.go", 1000) // 200 m each -> 1000 m
	}
	res := scan(t, s, r.dir, "uk-UA")
	if res.Locale != "uk" {
		t.Errorf("locale = %q", res.Locale)
	}
	wps := eventsOf(res, protocol.EventWaypoint)
	if len(wps) != 1 || wps[0].Waypoint.Name != "Старий міст" {
		t.Fatalf("waypoint events: %+v", wps)
	}
	var names []string
	for _, e := range eventsOf(res, protocol.EventAchievement) {
		names = append(names, e.Achievement.Name)
	}
	// first-steps (100 m), bridge (waypoint); big-day needs 1500 m
	if strings.Join(names, ",") != "Перші кроки,Через річку" {
		t.Fatalf("achievements: %v", names)
	}
	g := res.Global
	if g.NextWaypoint.Name != "Сосновий ліс" || g.ToNextM != 4000 || g.Route.Name != "Демо-стежка" {
		t.Fatalf("journey: %+v", g)
	}
	// An unknown language falls back to English.
	st, _ := s.Status(r.dir, "de")
	if st.Global.Route.Name != "Demo Trail" || st.Locale != "en" {
		t.Fatalf("fallback: %q %q", st.Global.Route.Name, st.Locale)
	}
	// Achievements unlock once.
	r.commit("", "big.go", 1)
	if again := eventsOf(scan(t, s, r.dir, "uk"), protocol.EventAchievement); len(again) != 0 {
		t.Fatalf("unlocked twice: %+v", again)
	}
}

func TestStoryBeatAndFinish(t *testing.T) {
	s, r := setup(t, "demo-trail")
	for i := 0; i < 13; i++ { // 2600 m, crosses the "birdsong" beat at 2500
		r.commit("", "big.go", 1000)
	}
	res := scan(t, s, r.dir, "en")
	st := eventsOf(res, protocol.EventStory)
	if len(st) != 1 || !strings.Contains(st[0].Story.Text, "birdsong") || res.Global.Story.ID != "birdsong" {
		t.Fatalf("story: %+v", st)
	}
	if len(eventsOf(res, protocol.EventFinished)) != 0 || res.Global.Finished {
		t.Fatal("not finished yet")
	}
}

func TestHiddenAchievementStaysSecret(t *testing.T) {
	s, r := setup(t, "chornohora-ridge")
	r.commit("", "a.go", 1)
	res := scan(t, s, r.dir, "en")
	for _, a := range res.Global.Achievements {
		if a.ID == "hundred" && (a.Name != "" || !a.Hidden) {
			t.Fatalf("hidden achievement leaked: %+v", a)
		}
		if a.ID == "first-steps" && (a.UnlockedAt != 0 || a.Name == "") {
			t.Fatalf("first-steps: locked at 30 m but visible, got %+v", a)
		}
	}
}

func TestStreak(t *testing.T) {
	today := int64(100)
	days := map[int64]float64{97: 1, 98: 1, 99: 1}
	if got := streak(days, today); got != 3 { // today empty: counting from yesterday
		t.Errorf("streak = %d", got)
	}
	days[100] = 1
	if got := streak(days, today); got != 4 {
		t.Errorf("streak = %d", got)
	}
	if got := streak(map[int64]float64{90: 1}, today); got != 0 {
		t.Errorf("streak = %d", got)
	}
}

func TestProjectJourneyAndSelectedMode(t *testing.T) {
	s, r := setup(t, "")
	if _, err := s.Init(InitOptions{Mode: store.ModeSelected}); err != nil {
		t.Fatal(err)
	}
	r.commit("", "a.go", 1)
	if res := scan(t, s, r.dir, ""); res.Tracked {
		t.Fatal("must not track a project that isn't enabled")
	}
	s.now = func() time.Time { return time.Unix(r.ts+1, 0) }
	if _, err := s.SetJourney(ScopeProject, r.dir, "chornohora-ridge", false, ""); err != nil {
		t.Fatal(err)
	}
	r.commit("", "a.go", 3) // 50 m after the project journey started
	res := scan(t, s, r.dir, "")
	if res.Project == nil || res.Project.DistanceM != 50 || res.Global.DistanceM != 80 {
		t.Fatalf("project=%+v global=%+v", res.Project, res.Global)
	}
}

func TestVerifyHealsTamperedState(t *testing.T) {
	s, r := setup(t, "")
	r.commit("", "a.go", 1)
	scan(t, s, r.dir, "")
	st, _ := s.st.LoadState()
	var pid string
	for _, rec := range st.Commits {
		rec.Meters, pid = 99999, rec.Project
	}
	st.Commits["fake"] = &store.CommitRec{Project: pid, Time: base, Meters: 500}
	if err := s.st.SaveState(st); err != nil {
		t.Fatal(err)
	}

	v, err := s.Verify(r.dir)
	if err != nil || v.Updated != 1 || v.Removed != 1 {
		t.Fatalf("verify: %+v %v", v, err)
	}
	if got, _ := s.Status(r.dir, ""); got.TotalM != 30 {
		t.Fatalf("after verify: %v", got.TotalM)
	}
}

func TestErrorsCarryCodes(t *testing.T) {
	s, _ := setup(t, "")
	_, err := s.SetJourney(ScopeGlobal, "", "no-such-route", false, "")
	var e *Error
	if !errors.As(err, &e) || e.Code != protocol.CodeUnknownRoute {
		t.Fatalf("got %v", err)
	}
	if _, err := s.Init(InitOptions{Mode: "sometimes"}); !errors.As(err, &e) || e.Code != protocol.CodeInvalidArgument {
		t.Fatalf("got %v", err)
	}
}

func TestRender(t *testing.T) {
	s, r := setup(t, "chornohora-ridge")
	r.commit("", "a.go", 1)
	scan(t, s, r.dir, "")
	out, err := s.Render(ScopeGlobal, r.dir, "svg", "uk", 300)
	if err != nil || !strings.Contains(out.SVG, "Говерла") {
		t.Fatalf("svg: %v", err)
	}
	sc, err := s.Render(ScopeGlobal, r.dir, "scene", "en", 0)
	if err != nil || len(sc.Scene.Markers) != 7 || sc.Scene.Walked <= 0 {
		t.Fatalf("scene: %+v %v", sc, err)
	}
}

func TestUserRoutes(t *testing.T) {
	dir := t.TempDir()
	custom := filepath.Join(dir, "routes", "my-route")
	mustWrite(t, filepath.Join(custom, "route.json"),
		`{"id":"my-route","length_m":500,"waypoints":[{"id":"a","at_m":0}]}`)
	mustWrite(t, filepath.Join(custom, "locales", "en.json"),
		`{"name":"Mine","description":"d","waypoints":{"a":{"name":"A"}}}`)
	// a broken one next to it must not break anything
	mustWrite(t, filepath.Join(dir, "routes", "broken", "route.json"), `{`)

	s, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(s.RouteWarnings()) != 1 {
		t.Errorf("want 1 warning for the broken route, got %v", s.RouteWarnings())
	}
	found := false
	for _, r := range s.Routes("en") {
		found = found || (r.ID == "my-route" && r.Name == "Mine" && !r.Builtin)
	}
	if !found {
		t.Error("the valid user route should load next to the broken one")
	}
}

// legacyScore is 1 base point = 1 m with the old single-tier daily cap, so
// tests read in plain meters. The real calibration is tested in package score
// and in TestDifficulty.
func legacyScore() score.Config {
	c := score.Default()
	c.Pace, c.DailySoftCapM, c.OverCapFactor, c.FarCapFactor, c.FarFactor = 1, 3000, 0.25, 1e9, 0.25
	return c
}
