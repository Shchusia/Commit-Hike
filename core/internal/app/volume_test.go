package app

import (
	"math"
	"testing"
	"time"

	"github.com/Shchusia/commit-hike/core/internal/protocol"
)

// With legacyScore's pace of 1 and medium difficulty, base points read as
// meters: 3 changed lines are 50 m by the per-commit rules and 17 m by the
// volume rules (45·log2(1 + 3/10)).

func near(a, b float64) bool { return math.Abs(a-b) < 0.05 }

// A new install counts its whole history by the volume rules.
func TestNewInstallCountsHistoryByVolume(t *testing.T) {
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	s.now = func() time.Time { return time.Unix(base+50000, 0) }
	if _, err := s.Init(InitOptions{Emails: []string{"me@x.io"}, FromHistory: true}); err != nil {
		t.Fatal(err)
	}
	cfg, _ := s.st.LoadConfig()
	if cfg.ScoringFrom == nil || *cfg.ScoringFrom != 0 || cfg.ScoringUpgradedAt != 0 {
		t.Fatalf("scoring_from %v, upgraded at %d", cfg.ScoringFrom, cfg.ScoringUpgradedAt)
	}
	if _, err := s.Scoring("off", ""); code(err) != protocol.CodeInvalidArgument {
		t.Fatalf("history off on a new install: %v", err)
	}
}

// Upgrading keeps what was walked: commits from before 1.2.0 keep their
// per-commit points, later ones follow the volume rules, and `scoring
// --history on|off` moves the line both ways.
func TestUpgradeToVolumeRules(t *testing.T) {
	s, r := setup(t, "")
	cfg, _ := s.st.LoadConfig()
	cfg.ScoringFrom = nil // a config written by 1.1
	if err := s.st.SaveConfig(cfg); err != nil {
		t.Fatal(err)
	}
	r.commit("", "a.go", 3) // before the upgrade: 50 m
	if got := scan(t, s, r.dir, "").TotalM; got != 50 {
		t.Fatalf("old commit after the upgrade = %v m, want 50", got)
	}
	upgraded := base + 50000 // the scan migrated the config at "now"
	s.now = func() time.Time { return time.Unix(base+60000, 0) }
	r.ts = base + 59000
	r.commit("", "b.go", 3) // after the upgrade: 17 m
	if got := scan(t, s, r.dir, "").TotalM; got != 67 {
		t.Fatalf("total = %v, want 50 + 17", got)
	}

	info, err := s.Scoring("", "")
	if err != nil || info.From != int64(upgraded) || info.History || info.UpgradedAt != int64(upgraded) || info.TotalM != 67 {
		t.Fatalf("scoring: %+v %v", info, err)
	}
	if info, err = s.Scoring("on", ""); err != nil || !info.History || info.From != 0 || info.TotalM != 34 || info.PreviousM != 67 {
		t.Fatalf("history on: %+v %v", info, err)
	}
	if st, _ := s.Status("", ""); st.TotalM != 34 {
		t.Fatalf("status after history on: %v", st.TotalM)
	}
	if info, err = s.Scoring("off", ""); err != nil || info.History || info.From != int64(upgraded) || info.TotalM != 67 || info.PreviousM != 34 {
		t.Fatalf("history off: %+v %v", info, err)
	}
	if _, err := s.Scoring("maybe", ""); code(err) != protocol.CodeInvalidArgument {
		t.Fatalf("bad value: %v", err)
	}
}

// A record stored before 1.2.0 has no lines: they come back from its points,
// a scan stores them without reporting the commit as updated, and `scoring
// --repo` re-reads the whole project for the records outside the scan window.
func TestOldRecordsLearnTheirLines(t *testing.T) {
	s, r := setup(t, "")
	volumeRules(t, s)
	r.commit("", "a.go", 3)
	scan(t, s, r.dir, "")
	forgetLines := func() {
		t.Helper()
		st, _ := s.st.LoadState()
		for _, rec := range st.Commits {
			if rec.Lines != 3 || rec.Meters != 50 {
				t.Fatalf("stored: %+v", rec)
			}
			rec.Lines = 0 // as 1.1 stored it
		}
		if err := s.st.SaveState(st); err != nil {
			t.Fatal(err)
		}
	}
	lines := func() int {
		st, _ := s.st.LoadState()
		for _, rec := range st.Commits {
			return rec.Lines
		}
		return -1
	}

	forgetLines()
	if st, _ := s.Status("", ""); st.TotalM != 17 {
		t.Fatalf("lines recovered from points: %v m, want 17", st.TotalM)
	}
	if res := scan(t, s, r.dir, ""); res.UpdatedCommits != 0 || lines() != 3 || res.TotalM != 17 {
		t.Fatalf("scan: updated %d, lines %d, total %v", res.UpdatedCommits, lines(), res.TotalM)
	}

	forgetLines()
	info, err := s.Scoring("on", r.dir)
	if err != nil || info.Verified == nil || info.Verified.Updated != 0 || lines() != 3 || info.TotalM != 17 {
		t.Fatalf("scoring --repo: %+v %v, lines %d", info, err, lines())
	}
}

// One walker for all tracked projects: a burst is cut wherever its commits
// land, and the reserve comes back with time.
func TestWalkerSpansProjects(t *testing.T) {
	s, a := setup(t, "")
	volumeRules(t, s)
	s.score.WalkReserveM, s.score.WalkSpeed = 40, 0.01 // 40 m, 36 m an hour
	b := newRepo(t)
	b.ts = base + 90 // the same author time in two projects would be one commit

	a.commit("", "a.go", 3) // base+60: 17 of 40, 23 left
	a.commit("", "b.go", 3) // base+120: +0.6 → 23.6, 17 taken, 6.6 left
	b.commit("", "c.go", 3) // base+150: +0.3 → 6.9, all of it taken
	if got := scan(t, s, a.dir, "").TotalM; got != 34 {
		t.Fatalf("project a alone: %v", got)
	}
	if got := scan(t, s, b.dir, "").TotalM; !near(got, 40.9) {
		t.Fatalf("both projects: %v, want 40.9", got)
	}
	// Hours later the reserve is full again: a commit counts in full.
	a.ts = base + 10000
	a.commit("", "d.go", 3)
	if got := scan(t, s, a.dir, "").TotalM; !near(got, 57.9) {
		t.Fatalf("after a break: %v, want 57.9", got)
	}
}
