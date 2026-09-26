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
)

// Rule is one condition; which fields matter depends on Type.
type Rule struct {
	Type     string  `json:"type"`
	MinM     float64 `json:"min_m,omitempty"`
	Waypoint string  `json:"waypoint,omitempty"`
	Count    int     `json:"count,omitempty"`
	Days     int     `json:"days,omitempty"`
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
	}
	return false
}

func (r Rule) validate(waypoints map[string]float64) error {
	switch r.Type {
	case TypeDistance, TypeDayDistance:
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
func Validate(defs []Def, waypoints map[string]float64) error {
	seen := map[string]bool{}
	for _, d := range defs {
		if d.ID == "" {
			return errors.New("achievement without id")
		}
		if seen[d.ID] {
			return fmt.Errorf("duplicate achievement %q", d.ID)
		}
		seen[d.ID] = true
		if err := d.Rule.validate(waypoints); err != nil {
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
