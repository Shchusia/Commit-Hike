package achievements

import (
	"reflect"
	"testing"
)

func TestRules(t *testing.T) {
	c := Context{
		DistanceM: 1500, LengthM: 2000, WaypointAtM: map[string]float64{"bridge": 1000, "peak": 2000},
		Commits: 12, StreakDays: 3, BestDayM: 700,
	}
	tests := []struct {
		rule Rule
		want bool
	}{
		{Rule{Type: TypeDistance, MinM: 1000}, true},
		{Rule{Type: TypeDistance, MinM: 1600}, false},
		{Rule{Type: TypeWaypoint, Waypoint: "bridge"}, true},
		{Rule{Type: TypeWaypoint, Waypoint: "peak"}, false},
		{Rule{Type: TypeFinish}, false},
		{Rule{Type: TypeCommits, Count: 10}, true},
		{Rule{Type: TypeStreak, Days: 7}, false},
		{Rule{Type: TypeDayDistance, MinM: 500}, true},
		{Rule{Type: "nope"}, false},
	}
	for _, tt := range tests {
		if got := tt.rule.Met(c); got != tt.want {
			t.Errorf("%+v: got %v, want %v", tt.rule, got, tt.want)
		}
	}
}

func TestValidate(t *testing.T) {
	wps := map[string]float64{"bridge": 1000}
	good := []Def{{ID: "a", Rule: Rule{Type: TypeWaypoint, Waypoint: "bridge"}}}
	if err := Validate(good, wps); err != nil {
		t.Fatal(err)
	}
	bad := [][]Def{
		{{ID: "a", Rule: Rule{Type: TypeWaypoint, Waypoint: "nowhere"}}},
		{{ID: "a", Rule: Rule{Type: TypeFinish}}, {ID: "a", Rule: Rule{Type: TypeFinish}}},
		{{ID: "a", Rule: Rule{Type: "magic"}}},
		{{ID: "a", Rule: Rule{Type: TypeStreak}}},
	}
	for i, defs := range bad {
		if Validate(defs, wps) == nil {
			t.Errorf("case %d should fail", i)
		}
	}
}

func TestNewly(t *testing.T) {
	defs := []Def{
		{ID: "first", Rule: Rule{Type: TypeCommits, Count: 1}},
		{ID: "ten", Rule: Rule{Type: TypeCommits, Count: 10}},
		{ID: "done", Rule: Rule{Type: TypeFinish}},
	}
	got := Newly(defs, Context{Commits: 10}, map[string]int64{"first": 1})
	if !reflect.DeepEqual(got, []string{"ten"}) {
		t.Errorf("got %v", got)
	}
}
