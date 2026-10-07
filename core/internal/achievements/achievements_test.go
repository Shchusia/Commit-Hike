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
	wps := Topology{Waypoints: map[string]float64{"bridge": 1000}, Kinds: map[string]int{"bridge": 1, "lake": 2}, Biomes: map[string]bool{"desert": true, "rock": true}}
	good := []Def{
		{ID: "a", Rule: Rule{Type: TypeWaypoint, Waypoint: "bridge"}},
		{ID: "b", Rule: Rule{Type: TypeBiome, Biome: "desert", MinM: 5}},
		{ID: "c", Rule: Rule{Type: TypeBiomes, Count: 2}},
		{ID: "d", Rule: Rule{Type: TypeKind, Kind: "lake"}},
		{ID: "e", Rule: Rule{Type: TypeWeekend, Count: 3}},
	}
	if err := Validate(good, wps); err != nil {
		t.Fatal(err)
	}
	bad := [][]Def{
		{{ID: "a", Rule: Rule{Type: TypeWaypoint, Waypoint: "nowhere"}}},
		{{ID: "a", Rule: Rule{Type: TypeFinish}}, {ID: "a", Rule: Rule{Type: TypeFinish}}},
		{{ID: "a", Rule: Rule{Type: "magic"}}},
		{{ID: "a", Rule: Rule{Type: TypeStreak}}},
		{{ID: "a", Rule: Rule{Type: TypeBiome, Biome: "jungle", MinM: 10}}},
		{{ID: "a", Rule: Rule{Type: TypeBiome, Biome: "desert"}}},
		{{ID: "a", Rule: Rule{Type: TypeBiomes, Count: 3}}},
		{{ID: "a", Rule: Rule{Type: TypeKind, Kind: "peak"}}},
		{{ID: "a", Rule: Rule{Type: TypeKind, Kind: "lake", Count: 3}}},
		{{ID: "a", Rule: Rule{Type: TypeNight}}},
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

func TestNewRuleTypes(t *testing.T) {
	c := Context{
		BiomeM: map[string]float64{"desert": 5000, "rock": 10}, KindsReached: map[string]int{"lake": 2}, KindsTotal: map[string]int{"lake": 3},
		NightCommits: 2, EarlyCommits: 1, WeekendCommits: 4, BestDayCommits: 9,
	}
	cases := []struct {
		r    Rule
		want bool
	}{
		{Rule{Type: TypeBiome, Biome: "desert", MinM: 5000}, true},
		{Rule{Type: TypeBiome, Biome: "desert", MinM: 5001}, false},
		{Rule{Type: TypeBiome, Biome: "snow", MinM: 1}, false},
		{Rule{Type: TypeBiomes, Count: 2}, true},
		{Rule{Type: TypeBiomes, Count: 3}, false},
		{Rule{Type: TypeKind, Kind: "lake", Count: 2}, true},
		{Rule{Type: TypeKind, Kind: "lake"}, false}, // all three
		{Rule{Type: TypeNight, Count: 2}, true},
		{Rule{Type: TypeEarly, Count: 2}, false},
		{Rule{Type: TypeWeekend, Count: 4}, true},
		{Rule{Type: TypeDayCommits, Count: 10}, false},
	}
	for i, tc := range cases {
		if got := tc.r.Met(c); got != tc.want {
			t.Errorf("case %d (%s): got %v", i, tc.r.Type, got)
		}
	}
}
