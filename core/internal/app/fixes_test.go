package app

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Shchusia/commit-hike/core/internal/gitlog"
	"github.com/Shchusia/commit-hike/core/internal/protocol"
	"github.com/Shchusia/commit-hike/core/internal/score"
)

func scanWith(t *testing.T, s *Service, dir string, o ScanOptions) *protocol.ScanResult {
	t.Helper()
	res, err := s.ScanWith(dir, "en", o)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func total(t *testing.T, s *Service) float64 {
	t.Helper()
	st, err := s.Status("", "")
	if err != nil {
		t.Fatal(err)
	}
	return st.TotalM
}

// A squash inside the rescan window is noticed by a plain scan.
func TestSquashIsReconciledByTheNextScan(t *testing.T) {
	s, r := setup(t, "")
	r.commit("other@x.io", "README", 1)
	for i := 0; i < 3; i++ {
		r.commit("", "a.go", 50) // 123.4 m each
	}
	scan(t, s, r.dir, "")
	if got := total(t, s); got != 370.2 {
		t.Fatalf("before squash: %v", got)
	}
	r.git("", "reset", "-q", "--soft", "HEAD~3")
	r.ts += 60
	r.git("", "commit", "-q", "-m", "squashed") // 150 lines: 154.8 m
	res := scan(t, s, r.dir, "")
	if res.RemovedCommits != 3 || res.NewCommits != 1 || total(t, s) != 154.8 {
		t.Fatalf("after squash: removed=%d new=%d total=%v", res.RemovedCommits, res.NewCommits, total(t, s))
	}
}

// A rewrite of old history (beyond the window) needs the previous HEAD.
func TestRewriteBeyondWindowNeedsPrevHead(t *testing.T) {
	s, r := setup(t, "")
	r.ts = base - 20*86400 // three weeks before "now"
	r.commit("other@x.io", "README", 1)
	r.commit("", "a.go", 1)
	r.commit("", "b.go", 1)
	scan(t, s, r.dir, "")
	before := gitlog.Head(r.dir)
	// Drop the last two commits and make a new one today.
	r.git("", "reset", "-q", "--hard", "HEAD~2")
	r.ts = base + 49000
	r.commit("", "c.go", 3) // 50 m
	if res := scan(t, s, r.dir, ""); res.Rewritten || total(t, s) != 110 {
		t.Fatalf("without prev-head the old commits stay: %v %+v", total(t, s), res)
	}
	res := scanWith(t, s, r.dir, ScanOptions{PrevHead: before})
	if !res.Rewritten || res.RemovedCommits != 2 || total(t, s) != 50 {
		t.Fatalf("with prev-head: %v %+v", total(t, s), res)
	}
	// A fast-forward is not a rewrite.
	head := gitlog.Head(r.dir)
	r.commit("", "d.go", 1)
	if res := scanWith(t, s, r.dir, ScanOptions{PrevHead: head}); res.Rewritten {
		t.Fatal("fast-forward treated as a rewrite")
	}
}

// Amending the very first commit changes the project id; progress follows.
func TestRootRewriteMovesTheProject(t *testing.T) {
	s, r := setup(t, "")
	r.commit("", "a.go", 1) // root, mine: 30 m
	r.commit("", "b.go", 1) // 30 m
	if _, err := s.SetJourney(ScopeProject, r.dir, "chornohora-ridge", true, ""); err != nil {
		t.Fatal(err)
	}
	scan(t, s, r.dir, "")
	if err := s.SetTeam(r.dir, true); err != nil {
		t.Fatal(err)
	}
	// Rewrite the root: squash both into one new root commit with a new date.
	r.git("", "reset", "-q", "--soft", "HEAD~1")
	r.ts += 600
	r.git("", "commit", "-q", "--amend", "--reset-author", "-m", "initial")
	res := scan(t, s, r.dir, "")
	if !res.Rewritten || total(t, s) != 41.7 { // one commit, 2 lines: 10 + 20*log2(3)
		t.Fatalf("after root rewrite: total=%v %+v", total(t, s), res)
	}
	if res.Project == nil || res.Project.Route.ID != "chornohora-ridge" || !res.Tracked {
		t.Fatalf("project journey must survive: %+v", res.Project)
	}
	st, _ := s.Status(r.dir, "")
	if !st.Team {
		t.Fatal("team setting must follow the project")
	}
}

// A different repository cloned into a path we already know keeps both.
func TestNewRepoAtKnownPathDoesNotStealProgress(t *testing.T) {
	s, a := setup(t, "")
	a.commit("", "a.go", 1)
	scan(t, s, a.dir, "")
	clone := t.TempDir()
	if out, err := exec.Command("git", "clone", "-q", a.dir, clone).CombinedOutput(); err != nil {
		t.Fatalf("%v %s", err, out)
	}
	scan(t, s, clone, "") // project A is known at two paths now
	if err := os.RemoveAll(a.dir); err != nil {
		t.Fatal(err)
	}
	b := &repo{t: t, dir: a.dir, ts: base + 3000}
	if err := os.MkdirAll(b.dir, 0o750); err != nil {
		t.Fatal(err)
	}
	b.git("", "init", "-q", "-b", "main")
	b.commit("", "x.go", 1)
	scan(t, s, b.dir, "")
	if got := total(t, s); got != 60 {
		t.Fatalf("both projects must count: %v", got)
	}
}

// Two clones of one project with different unpushed commits don't undo each other.
func TestClonesWithDifferentLocalCommits(t *testing.T) {
	s, a := setup(t, "")
	a.commit("other@x.io", "README", 1)
	clone := t.TempDir()
	if out, err := exec.Command("git", "clone", "-q", a.dir, clone).CombinedOutput(); err != nil {
		t.Fatalf("%v %s", err, out)
	}
	a.commit("", "a.go", 1) // only in a
	b := &repo{t: t, dir: clone, ts: base + 5000}
	b.commit("", "b.go", 1) // only in the clone
	scan(t, s, a.dir, "")
	scan(t, s, clone, "")
	scan(t, s, a.dir, "")
	if got := total(t, s); got != 60 {
		t.Fatalf("clones fought over records: %v", got)
	}
}

func TestFutureCommitsAreIgnored(t *testing.T) {
	s, r := setup(t, "")
	r.commit("", "a.go", 1)
	r.ts = base + 50000 + 30*86400 // a month after "now"
	r.commit("", "b.go", 1)
	if got := scan(t, s, r.dir, "").TotalM; got != 30 {
		t.Fatalf("future commit counted: %v", got)
	}
}

func TestDaysFollowTheLocalCalendar(t *testing.T) {
	s, r := setup(t, "")
	s.loc = time.FixedZone("UTC+3", 3*3600)
	// "now" is 12:00 local on some day; a commit at 22:30 UTC the day before
	// is 01:30 local today.
	now := time.Date(2026, 9, 26, 9, 0, 0, 0, time.UTC)
	s.now = func() time.Time { return now }
	r.ts = time.Date(2026, 9, 25, 22, 30, 0, 0, time.UTC).Unix() - 60
	r.commit("", "a.go", 1)
	res := scan(t, s, r.dir, "")
	if res.TodayM != 30 {
		t.Fatalf("today_m = %v, want 30 (01:30 local is today)", res.TodayM)
	}
	daily := res.Global.Daily
	if last := daily[len(daily)-1]; last.Date != "2026-09-26" || last.M != 30 {
		t.Fatalf("last day = %+v", last)
	}
}

func TestTeam(t *testing.T) {
	s, r := setup(t, "chornohora-ridge")
	mustWrite(t, filepath.Join(r.dir, ".mailmap"), "Anna Koval <anna@team.io>\nAnna Koval <anna@team.io> <anna@old.io>\n")
	r.commit("", "a.go", 1)                                            // me: 50 (with .mailmap)
	r.commit("anna@old.io", "b.go", 1)                                 // anna: 30
	r.commit("anna@team.io", "c.go", 3)                                // anna: +50
	r.commit("dependabot[bot]@users.noreply.github.com", "go.mod", 10) // bot
	team, err := s.Team(r.dir)
	if err != nil {
		t.Fatal(err)
	}
	if team.RouteID != "chornohora-ridge" || team.Scope != ScopeGlobal || len(team.Members) != 2 {
		t.Fatalf("team: %+v", team)
	}
	top := team.Members[0]
	if top.Name != "Anna Koval" || top.DistanceM != 80 || top.Commits != 2 || top.Me || top.ElevationM == nil {
		t.Fatalf("anna: %+v", top)
	}
	if me := team.Members[1]; !me.Me || me.DistanceM != 50 { // a.go + the two .mailmap lines
		t.Fatalf("me: %+v", me)
	}
	if strings.Contains(top.ID, "@") {
		t.Fatal("member id must not be an email")
	}
}

func TestLocaleSetting(t *testing.T) {
	s, r := setup(t, "demo-trail")
	uk := "uk"
	info, err := s.Locale(&uk, "en")
	if err != nil || info.Locale != "uk" || info.Effective != "uk" {
		t.Fatalf("set uk: %+v %v", info, err)
	}
	if st, _ := s.Status(r.dir, "en"); st.Locale != "uk" {
		t.Fatalf("a fixed language wins over the IDE's: %q", st.Locale)
	}
	auto := "auto"
	if info, _ = s.Locale(&auto, "en"); info.Locale != "" || info.Effective != "en" {
		t.Fatalf("auto: %+v", info)
	}
	bad := "../../x"
	if _, err := s.Locale(&bad, ""); code(err) != protocol.CodeInvalidArgument {
		t.Fatalf("bad locale: %v", err)
	}
	if len(info.Available) < 2 {
		t.Fatalf("available: %v", info.Available)
	}
}

func TestElevationAndClimbAchievements(t *testing.T) {
	s, r := setup(t, "chornohora-ridge")
	for i := 0; i < 20; i++ { // 20 * 200 m = 4000 m: Hoverla
		r.ts += 3600 * 3 // spread over days so the cap doesn't bite
		r.commit("", "big.go", 1000)
	}
	s.now = func() time.Time { return time.Unix(r.ts+60, 0) }
	res := scan(t, s, r.dir, "en")
	g := res.Global
	if g.ElevationM == nil || *g.ElevationM != 2061 || g.AscentM != 771 || g.MaxElevationM != 2061 {
		t.Fatalf("elevation: %v ascent %v max %v", g.ElevationM, g.AscentM, g.MaxElevationM)
	}
	if g.Route.MaxElevationM != 2061 || len(g.Route.Profile) < 7 || len(g.Route.Facts) == 0 || len(g.Route.Objects) == 0 {
		t.Fatalf("route data: %+v", g.Route)
	}
	got := map[string]bool{}
	for _, e := range eventsOf(res, protocol.EventAchievement) {
		got[e.Achievement.ID] = true
	}
	if !got["above-2000"] {
		t.Fatalf("altitude achievement not unlocked: %v", got)
	}
	if len(eventsOf(res, protocol.EventFact)) == 0 {
		t.Fatal("passing facts must produce events")
	}
}

func TestRouteAssetsAndRemovalClearsAchievements(t *testing.T) {
	s, r := setup(t, "")
	dir, err := s.RouteTemplate("with-art", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.ImportRoute(dir, false, ""); err != nil {
		t.Fatal(err)
	}
	a, err := s.RouteAssets("with-art")
	if err != nil || !strings.HasPrefix(a.Images["assets/signpost.svg"], "data:image/svg+xml;base64,") {
		t.Fatalf("assets: %+v %v", a, err)
	}
	if b, err := s.RouteAssets("chornohora-ridge"); err != nil || len(b.Images) == 0 {
		t.Fatalf("built-in assets: %v %v", b, err)
	}
	r.commit("", "a.go", 50)
	if _, err := s.SetJourney(ScopeGlobal, "", "with-art", true, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SetJourney(ScopeGlobal, "", "demo-trail", true, ""); err != nil {
		t.Fatal(err)
	}
	if err := s.RemoveRoute("with-art"); err != nil {
		t.Fatal(err)
	}
	st, _ := s.st.LoadState()
	for k := range st.Achievements {
		if strings.HasSuffix(k, ":with-art") {
			t.Fatalf("achievements of a removed route stay: %v", k)
		}
	}
}

// Real pace: a difficulty change applies to commits from then on only.
func TestDifficulty(t *testing.T) {
	s, r := setup(t, "")
	s.score = score.Default()
	r.commit("", "a.go", 60) // 128.6 base points
	first := scan(t, s, r.dir, "").TotalM
	if first < 1600 || first > 1700 { // ×13
		t.Fatalf("a 60-line commit on medium = %v m", first)
	}
	info, err := s.Difficulty("easy")
	if err != nil || info.Level != "easy" || info.TypicalDayM["easy"] <= info.TypicalDayM["medium"] {
		t.Fatalf("set easy: %+v %v", info, err)
	}
	if got := total(t, s); got != first {
		t.Fatalf("switching must not rewrite the past: %v -> %v", first, got)
	}
	s.now = func() time.Time { return time.Unix(base+60000, 0) }
	r.ts = base + 59000
	r.commit("", "b.go", 60)
	after := scan(t, s, r.dir, "").TotalM
	if d := after - first; d < first*1.24 || d > first*1.26 {
		t.Fatalf("easy commit = %v m, want ×1.25 of %v", d, first)
	}
	if st, _ := s.Status("", ""); st.Difficulty != "easy" || st.TypicalDayM < 11000 {
		t.Fatalf("status difficulty: %+v", st)
	}
	if _, err := s.Difficulty("nightmare"); code(err) != protocol.CodeInvalidArgument {
		t.Fatalf("bad level: %v", err)
	}
}

func TestUndergroundAndDangers(t *testing.T) {
	// A fixture route instead of real content: tests must not depend on which
	// routes ship with the plugin.
	s, r := setup(t, "")
	if _, err := s.ImportRoute(filepath.Join("testdata", "routes", "tunnel-test"), false, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SetJourney(ScopeGlobal, "", "tunnel-test", true, ""); err != nil {
		t.Fatal(err)
	}
	st, _ := s.Status(r.dir, "en")
	rt := st.Global.Route
	if len(rt.Underground) != 2 || len(rt.Dangers) != 8 || rt.Dangers[0].Text == "" {
		t.Fatalf("route data: underground %v, dangers %d", rt.Underground, len(rt.Dangers))
	}
	tunnel := rt.Underground[0]
	// Walk into the first tunnel: 1 base point = 1 m in tests, so write commits worth it.
	s.score.Pace = tunnel.FromM/200 + 1 // one maximal commit lands just inside
	s.score.DailySoftCapM = 1e12        // no daily discount for this jump
	r.commit("", "big.go", 100000)
	res := scan(t, s, r.dir, "en")
	if !res.Global.Underground {
		t.Fatalf("at %v m the walker should be underground (%v)", res.Global.DistanceM, tunnel)
	}
	if len(eventsOf(res, protocol.EventDanger)) < 5 {
		t.Fatalf("passing the encounters must produce danger events: %v", eventsOf(res, protocol.EventDanger))
	}
}

// Upgrading keeps what was walked: commits from before the real pace keep
// their old distance, commits after it get the new one.
func TestUpgradeKeepsWalkedDistance(t *testing.T) {
	s, r := setup(t, "")
	s.score = score.Default()
	cfg, _ := s.st.LoadConfig()
	cfg.PaceFrom = nil // a config written by the previous version
	if err := s.st.SaveConfig(cfg); err != nil {
		t.Fatal(err)
	}
	r.commit("", "a.go", 3) // 50 base points: 50 m before the upgrade
	if got := scan(t, s, r.dir, "").TotalM; got != 50 {
		t.Fatalf("old commit after upgrade = %v m, want 50", got)
	}
	s.now = func() time.Time { return time.Unix(base+60000, 0) }
	r.ts = base + 59000
	r.commit("", "b.go", 3) // after the upgrade: 50 × 13
	if got := scan(t, s, r.dir, "").TotalM; got != 50+650 {
		t.Fatalf("total = %v, want 700", got)
	}
}
