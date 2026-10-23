package score

import (
	"math"
	"testing"
)

func TestCommitMeters(t *testing.T) {
	c := Default()
	cases := []struct {
		lines int
		want  float64
	}{
		{0, 0}, {1, 30}, {3, 50}, {100, 143.2}, {100000, 200},
	}
	for _, tc := range cases {
		if got := c.CommitMeters(tc.lines); got != tc.want {
			t.Errorf("CommitMeters(%d) = %v, want %v", tc.lines, got, tc.want)
		}
	}
}

func TestDailyCapTiers(t *testing.T) {
	c := Default()
	if got := c.DailyEffective(10000, Medium); got != 10000 {
		t.Errorf("below cap: %v", got)
	}
	if got := c.DailyEffective(25000, Medium); got != 15000+10000*0.35 {
		t.Errorf("first tier: %v", got)
	}
	if got := c.DailyEffective(130000, Medium); math.Abs(got-(15000+15000*0.35+100000*0.05)) > 1e-6 {
		t.Errorf("far tier: %v", got)
	}
	if f := c.DailyFactor(25000, Medium); math.Abs(f*25000-18500) > 1e-6 {
		t.Errorf("factor: %v", f)
	}
}

func TestCommitPoints(t *testing.T) {
	c := Default()
	cases := []struct {
		lines int
		want  float64
	}{
		{0, 0}, {1, 6.2}, {3, 17}, {10, 45}, {60, 126.3}, {100, 155.7}, {1000, 299.6}, {5000, 300},
	}
	for _, tc := range cases {
		if got := c.CommitPoints(tc.lines); got != tc.want {
			t.Errorf("CommitPoints(%d) = %v, want %v", tc.lines, got, tc.want)
		}
	}
}

// Splitting work into tiny commits gains little: 60 one-line commits are worth
// about 3 commits of 60 lines, not 14 as by the per-commit rules.
func TestSplittingGainsLittle(t *testing.T) {
	c := Default()
	if ratio := 60 * c.CommitPoints(1) / c.CommitPoints(60); ratio > 3 {
		t.Errorf("60 one-liners = %.1f× one 60-line commit", ratio)
	}
	if ratio := 60 * c.CommitMeters(1) / c.CommitMeters(60); ratio < 10 {
		t.Errorf("the per-commit rules changed: %.1f×", ratio)
	}
}

// Records stored before 1.2.0 hold only per-commit points; their lines come
// back from them: within a line or 1 % up to the cap, the cap's 723 lines beyond.
func TestLinesOf(t *testing.T) {
	c := Default()
	for lines := 1; lines <= 723; lines++ {
		got := c.LinesOf(c.CommitMeters(lines))
		if math.Abs(float64(got-lines)) > math.Max(1, float64(lines)/100) {
			t.Fatalf("LinesOf(CommitMeters(%d)) = %d", lines, got)
		}
	}
	if got := c.LinesOf(c.CommitMeters(100000)); got != 723 {
		t.Errorf("a capped commit: %d lines, want 723", got)
	}
	if c.LinesOf(0) != 0 || c.LinesOf(-1) != 0 {
		t.Error("no points, no lines")
	}
}

func TestWalker(t *testing.T) {
	c := Default()
	w := c.NewWalker()
	// The reserve starts full: 10 km on medium.
	if got := w.Step(0, 9000, Medium); got != 9000 {
		t.Fatalf("first commit: %v", got)
	}
	// 1 km left: a burst gets only that.
	if got := w.Step(0, 3000, Medium); math.Abs(got-1000) > 1e-9 {
		t.Fatalf("burst: %v", got)
	}
	if got := w.Step(0, 3000, Medium); got != 0 {
		t.Fatalf("empty reserve: %v", got)
	}
	// An hour later 5 km have come back.
	if got := w.Step(3600, 9000, Medium); math.Abs(got-5000) > 1e-6 {
		t.Fatalf("after an hour: %v", got)
	}
	// It never holds more than the reserve, however long the break.
	if got := w.Step(3600+7*86400, 50000, Medium); math.Abs(got-10000) > 1e-6 {
		t.Fatalf("after a week: %v", got)
	}
	// The level scales the reserve and the speed like the meters.
	e := c.NewWalker()
	if got := e.Step(0, 50000, Easy); math.Abs(got-12500) > 1e-6 {
		t.Fatalf("easy reserve: %v", got)
	}
	if got := e.Step(3600, 50000, Easy); math.Abs(got-6250) > 1e-6 {
		t.Fatalf("easy speed: %v", got)
	}
	// A commit out of order gets no refill and doesn't move time back.
	if got := e.Step(0, 1, Easy); got != 0 {
		t.Fatalf("out of order: %v", got)
	}
}

// The calibration the product promises: a typical day is about half a
// hiker's day on medium, easy is a bit more, hard a bit less, even maximal
// commits can't finish the shortest real route, and farming is bounded by
// walking speed.
func TestCalibration(t *testing.T) {
	c := Default()
	day := c.TypicalDay(Medium)
	if day < 9000 || day > 11000 {
		t.Fatalf("typical medium day = %v m, want ≈10 km", day)
	}
	if e, h := c.TypicalDay(Easy), c.TypicalDay(Hard); e <= day || h >= day || e > day*1.3 || h < day*0.75 {
		t.Fatalf("easy %v / medium %v / hard %v", e, day, h)
	}
	// A typical day never meets the limit, even with all its commits at once.
	w := c.NewWalker()
	for i := 0; i < 6; i++ {
		if m := c.CommitPoints(60) * c.Scale(Medium); w.Step(0, m, Medium) != m {
			t.Fatal("the walker cut a typical day")
		}
	}
	if maxCommit := c.VolumeCap * c.Scale(Easy); maxCommit*2 >= 19400 {
		t.Fatalf("two commits (%v m each) finish a 19.4 km route", maxCommit)
	}
	// Farming: 500 one-line commits in half an hour, and 500 maximal commits
	// over an 8-hour day, stay within what a hiker walks.
	walked := func(n int, lines int, span int64) float64 {
		w, sum := c.NewWalker(), 0.0
		for i := 0; i < n; i++ {
			sum += w.Step(int64(i)*span/int64(n), c.CommitPoints(lines)*c.Scale(Medium), Medium)
		}
		return sum
	}
	if got := walked(500, 1, 1800); got > 12500 {
		t.Fatalf("500 one-liners in 30 min give %v m", got)
	}
	if got := walked(500, 100000, 8*3600); got > 50000 {
		t.Fatalf("500 maximal commits in 8 h give %v m", got)
	}
	if !ValidLevel(Hard) || ValidLevel("insane") || LevelFactor("x") != 1 {
		t.Fatal("levels")
	}
}

// The per-commit rules still count the commits made before 1.2.0.
func TestPerCommitRulesKept(t *testing.T) {
	c := Default()
	if farmed := c.DailyEffective(500*c.CommitCap*c.Scale(Medium), Medium); farmed > 90000 {
		t.Fatalf("farming gives %v m a day", farmed)
	}
}
