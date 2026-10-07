// Package achievements evaluates data-driven achievement rules.
//
// Rules live in each route's route.json, so a new route brings its own
// achievements without code changes. Adding a new rule *type* means adding a
// case to Rule.Met and Rule.validate.
package achievements

import (
	"errors"
	"fmt"
)

// Rule types.
const (
	TypeDistance    = "distance"     // walked at least MinM on this journey
	TypeWaypoint    = "waypoint"     // reached Waypoint
	TypeFinish      = "finish"       // completed the route
	TypeCommits     = "commits"      // at least Count counted commits on this journey
	TypeStreak      = "streak"       // commits on Days consecutive days
	TypeDayDistance = "day_distance" // at least MinM in a single day
	TypeAltitude    = "altitude"     // stood at AltitudeM or higher (needs an elevation profile)
	TypeClimb       = "climb"        // climbed at least MinM in total on this journey

	// Added in 1.1.0.
	TypeBiome      = "biome"       // walked at least MinM through Biome
	TypeBiomes     = "biomes"      // walked through at least Count different biomes
	TypeKind       = "kind"        // reached at least Count stops of Kind (Count 0: all of them)
	TypeNight      = "night"       // at least Count commits between midnight and 5 am
	TypeEarly      = "early"       // at least Count commits between 5 and 8 am
	TypeWeekend    = "weekend"     // at least Count commits on Saturdays and Sundays
	TypeDayCommits = "day_commits" // at least Count commits in a single day
)

// Types lists every rule type this core knows.
var Types = []string{
	TypeDistance, TypeWaypoint, TypeFinish, TypeCommits, TypeStreak, TypeDayDistance, TypeAltitude, TypeClimb,
	TypeBiome, TypeBiomes, TypeKind, TypeNight, TypeEarly, TypeWeekend, TypeDayCommits,
}

// Rule is one condition; which fields matter depends on Type.
type Rule struct {
	Type      string  `json:"type"`
	MinM      float64 `json:"min_m,omitempty"`
	Waypoint  string  `json:"waypoint,omitempty"`
	Count     int     `json:"count,omitempty"`
	Days      int     `json:"days,omitempty"`
	AltitudeM float64 `json:"altitude_m,omitempty"`
	Biome     string  `json:"biome,omitempty"`
	Kind      string  `json:"kind,omitempty"`
}

// Def is an achievement as defined in a route pack.
type Def struct {
	ID     string `json:"id"`
	Rule   Rule   `json:"rule"`
	Hidden bool   `json:"hidden,omitempty"` // secret until unlocked
}

// Context is everything a rule may look at.
type Context struct {
	DistanceM   float64
	LengthM     float64
	WaypointAtM map[string]float64 // waypoint id -> position on the route
	Commits     int
	StreakDays  int
	BestDayM    float64
	MaxElevM    float64 // highest point walked so far; 0 when the route has no heights
	AscentM     float64 // total climb walked so far

	BiomeM         map[string]float64 // meters walked in each biome type
	KindsReached   map[string]int     // stops reached, by kind
	KindsTotal     map[string]int     // stops on the route, by kind
	NightCommits   int                // counted commits made 00:00–04:59 local time
	EarlyCommits   int                // 05:00–07:59
	WeekendCommits int                // on Saturdays and Sundays
	BestDayCommits int                // most counted commits in one day
}

// Topology is what validation needs to know about the route.
type Topology struct {
	Waypoints map[string]float64 // id -> position
	Kinds     map[string]int     // stops by kind
	Biomes    map[string]bool    // biome types on the route
}

// Met reports whether the rule is satisfied.
func (r Rule) Met(c Context) bool {
	switch r.Type {
	case TypeDistance:
		return c.DistanceM >= r.MinM
	case TypeWaypoint:
		at, ok := c.WaypointAtM[r.Waypoint]
		return ok && c.DistanceM >= at
	case TypeFinish:
		return c.LengthM > 0 && c.DistanceM >= c.LengthM
	case TypeCommits:
		return c.Commits >= r.Count
	case TypeStreak:
		return c.StreakDays >= r.Days
	case TypeDayDistance:
		return c.BestDayM >= r.MinM
	case TypeAltitude:
		return c.MaxElevM > 0 && c.MaxElevM >= r.AltitudeM
	case TypeClimb:
		return c.AscentM >= r.MinM
	case TypeBiome:
		return c.BiomeM[r.Biome] >= r.MinM
	case TypeBiomes:
		n := 0
		for _, m := range c.BiomeM {
			if m > 0 {
				n++
			}
		}
		return n >= r.Count
	case TypeKind:
		want := r.Count
		if want == 0 {
			want = c.KindsTotal[r.Kind]
		}
		return want > 0 && c.KindsReached[r.Kind] >= want
	case TypeNight:
		return c.NightCommits >= r.Count
	case TypeEarly:
		return c.EarlyCommits >= r.Count
	case TypeWeekend:
		return c.WeekendCommits >= r.Count
	case TypeDayCommits:
		return c.BestDayCommits >= r.Count
	}
	return false
}

func (r Rule) validate(t Topology) error {
	waypoints := t.Waypoints
	switch r.Type {
	case TypeBiome:
		if r.MinM <= 0 {
			return errors.New("min_m must be positive")
		}
		if !t.Biomes[r.Biome] {
			return fmt.Errorf("the route has no %q biome", r.Biome)
		}
	case TypeBiomes:
		if r.Count < 2 || r.Count > len(t.Biomes) {
			return fmt.Errorf("count must be between 2 and the route's %d biome types", len(t.Biomes))
		}
	case TypeKind:
		if t.Kinds[r.Kind] == 0 {
			return fmt.Errorf("the route has no stops of kind %q", r.Kind)
		}
		if r.Count < 0 || r.Count > t.Kinds[r.Kind] {
			return fmt.Errorf("count must be between 0 (all) and %d", t.Kinds[r.Kind])
		}
	case TypeNight, TypeEarly, TypeWeekend, TypeDayCommits:
		if r.Count <= 0 {
			return errors.New("count must be positive")
		}
	case TypeAltitude:
		if r.AltitudeM <= 0 {
			return errors.New("altitude_m must be positive")
		}
	case TypeDistance, TypeDayDistance, TypeClimb:
		if r.MinM <= 0 {
			return errors.New("min_m must be positive")
		}
	case TypeWaypoint:
		if _, ok := waypoints[r.Waypoint]; !ok {
			return fmt.Errorf("unknown waypoint %q", r.Waypoint)
		}
	case TypeFinish:
	case TypeCommits:
		if r.Count <= 0 {
			return errors.New("count must be positive")
		}
	case TypeStreak:
		if r.Days <= 0 {
			return errors.New("days must be positive")
		}
	default:
		return fmt.Errorf("unknown rule type %q", r.Type)
	}
	return nil
}

// Validate checks a route's achievement list.
func Validate(defs []Def, t Topology) error {
	seen := map[string]bool{}
	for _, d := range defs {
		if d.ID == "" {
			return errors.New("achievement without id")
		}
		if seen[d.ID] {
			return fmt.Errorf("duplicate achievement %q", d.ID)
		}
		seen[d.ID] = true
		if err := d.Rule.validate(t); err != nil {
			return fmt.Errorf("achievement %q: %w", d.ID, err)
		}
	}
	return nil
}

// Newly returns ids of achievements that are met now but not yet unlocked,
// in definition order.
func Newly(defs []Def, c Context, unlocked map[string]int64) []string {
	var out []string
	for _, d := range defs {
		if _, done := unlocked[d.ID]; !done && d.Rule.Met(c) {
			out = append(out, d.ID)
		}
	}
	return out
}
