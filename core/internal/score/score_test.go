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

// The calibration the product promises: a typical day is about half a
// hiker's day on medium, easy is a bit more, hard a bit less, and even a
// maximal commit can't finish the shortest real route.
func TestCalibration(t *testing.T) {
	c := Default()
	day := c.TypicalDay(Medium)
	if day < 9000 || day > 11000 {
		t.Fatalf("typical medium day = %v m, want ≈10 km", day)
	}
	if e, h := c.TypicalDay(Easy), c.TypicalDay(Hard); e <= day || h >= day || e > day*1.3 || h < day*0.75 {
		t.Fatalf("easy %v / medium %v / hard %v", e, day, h)
	}
	if maxCommit := c.CommitCap * c.Scale(Easy); maxCommit*2 >= 19400 {
		t.Fatalf("two commits (%v m each) finish a 19.4 km route", maxCommit)
	}
	// Farming 500 maximal commits in one day stays a long day's walk.
	if farmed := c.DailyEffective(500*c.CommitCap*c.Scale(Medium), Medium); farmed > 90000 {
		t.Fatalf("farming gives %v m a day", farmed)
	}
	if !ValidLevel(Hard) || ValidLevel("insane") || LevelFactor("x") != 1 {
		t.Fatal("levels")
	}
}
