package score

import "testing"

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

func TestDailyCap(t *testing.T) {
	c := Default()
	if got := c.DailyEffective(1000); got != 1000 {
		t.Errorf("below cap: %v", got)
	}
	if got := c.DailyEffective(7000); got != 4000 { // 3000 + 4000*0.25
		t.Errorf("above cap: %v", got)
	}
	if f := c.DailyFactor(7000); f*7000 != 4000 {
		t.Errorf("factor: %v", f)
	}
}
