// Package score turns commit sizes into meters of travel.
//
// Design goals:
//   - small commits still count, huge commits are only slightly better (log curve);
//   - one commit is never worth more than CommitCap base points;
//   - a typical developer's day covers about half of a hiker's day (≈10 km on
//     medium), so real-length routes (Frodo walked ~2,863 km) take months, and
//     no route can be finished with a couple of commits;
//   - past DailySoftCapM per day progress is discounted, and further on almost
//     stops, so farming thousands of commits a day is pointless;
//   - difficulty scales everything a little: easy ×1.25, hard ×0.8.
package score

import "math"

// Difficulty levels.
const (
	Easy   = "easy"
	Medium = "medium"
	Hard   = "hard"
)

// Levels lists the difficulty levels in order.
var Levels = []string{Easy, Medium, Hard}

// LevelFactor is the multiplier of a difficulty level (medium for unknown).
func LevelFactor(level string) float64 {
	switch level {
	case Easy:
		return 1.25
	case Hard:
		return 0.8
	}
	return 1
}

// ValidLevel reports whether level is a known difficulty.
func ValidLevel(level string) bool { return level == Easy || level == Medium || level == Hard }

// Config holds the scoring constants.
type Config struct {
	BaseMeters float64 // flat base points for any meaningful commit
	LogFactor  float64 // weight of log2(1+lines)
	CommitCap  float64 // max base points per single commit
	// Pace turns base points into meters on the trail (medium difficulty).
	Pace float64
	// Daily discount on the scaled meters of one day (medium; scaled by the level):
	// full value up to DailySoftCapM, OverCapFactor up to FarCapFactor×DailySoftCapM,
	// FarFactor beyond.
	DailySoftCapM float64
	OverCapFactor float64
	FarCapFactor  float64
	FarFactor     float64
}

// Default returns the scoring used by the app.
//
// Calibration: a typical day of ~6 commits of ~60 changed lines is ≈770 base
// points; ×13 ≈ 10 km on medium. One commit is at most 200×13 = 2.6 km.
func Default() Config {
	return Config{
		BaseMeters:    10,
		LogFactor:     20,
		CommitCap:     200,
		Pace:          13,
		DailySoftCapM: 15000,
		OverCapFactor: 0.35,
		FarCapFactor:  2,
		FarFactor:     0.05,
	}
}

// CommitMeters returns the base points for a commit with `lines` significant
// changed lines (added + deleted, after filtering). Zero lines -> zero. Base
// points are what the state stores; Scale turns them into meters.
func (c Config) CommitMeters(lines int) float64 {
	if lines <= 0 {
		return 0
	}
	m := c.BaseMeters + c.LogFactor*math.Log2(1+float64(lines))
	return Round1(math.Min(m, c.CommitCap))
}

// Scale is the meters per base point at a difficulty level.
func (c Config) Scale(level string) float64 { return c.Pace * LevelFactor(level) }

// DailyEffective applies the daily discount to a day's scaled total.
func (c Config) DailyEffective(raw float64, level string) float64 {
	soft := c.DailySoftCapM * LevelFactor(level)
	if raw <= soft {
		return raw
	}
	far := soft * c.FarCapFactor
	if raw <= far {
		return soft + (raw-soft)*c.OverCapFactor
	}
	return soft + (far-soft)*c.OverCapFactor + (raw-far)*c.FarFactor
}

// DailyFactor is the multiplier applied to every commit of a day whose scaled
// total is `raw`. A per-day factor keeps results independent of the order in
// which projects are scanned, and makes per-project journeys sum exactly to
// the global one.
func (c Config) DailyFactor(raw float64, level string) float64 {
	if raw <= 0 {
		return 1
	}
	return c.DailyEffective(raw, level) / raw
}

// TypicalDay is the distance of a typical day of commits (6 × ~60 lines).
func (c Config) TypicalDay(level string) float64 {
	return Round1(c.DailyEffective(6*c.CommitMeters(60)*c.Scale(level), level))
}

// Round1 rounds to one decimal place, the precision shown to users.
func Round1(v float64) float64 { return math.Round(v*10) / 10 }
