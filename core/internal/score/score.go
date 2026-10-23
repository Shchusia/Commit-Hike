// Package score turns commit sizes into meters of travel.
//
// Two sets of rules exist; the config says from when the newer one applies.
//
// Volume rules (1.2.0 on):
//   - a commit is worth more the more it changes, from almost nothing for a
//     one-line commit to VolumeCap base points at about a thousand lines, so
//     splitting work into many tiny commits gains little;
//   - the hiker can't walk faster than WalkSpeed: a reserve of trail refills at
//     that speed up to WalkReserveM, and a commit is credited at most what the
//     reserve holds. A burst of commits is cut, while commits spread over a
//     working day count in full however late in the day they come;
//   - a typical developer's day (6 commits of ~60 lines) is still ≈10 km on
//     medium, so real-length routes (Frodo walked ~2,863 km) take months;
//   - difficulty scales everything a little: easy ×1.25, hard ×0.8.
//
// Per-commit rules (before 1.2.0), kept for commits counted with them:
//   - BaseMeters + LogFactor·log2(1+lines) base points, at most CommitCap;
//   - past DailySoftCapM per day progress is discounted, and further on almost
//     stops.
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
	// Per-commit rules (before 1.2.0).
	BaseMeters float64 // flat base points for any meaningful commit
	LogFactor  float64 // weight of log2(1+lines)
	CommitCap  float64 // max base points per single commit
	// Daily discount on the scaled meters of one day (medium; scaled by the level):
	// full value up to DailySoftCapM, OverCapFactor up to FarCapFactor×DailySoftCapM,
	// FarFactor beyond.
	DailySoftCapM float64
	OverCapFactor float64
	FarCapFactor  float64
	FarFactor     float64

	// Volume rules (1.2.0 on): VolumeFactor·log2(1 + lines/VolumeLines) base
	// points, at most VolumeCap.
	VolumeFactor float64
	VolumeLines  float64
	VolumeCap    float64
	// The walking-speed limit (medium; scaled by the level): the reserve
	// refills at WalkSpeed meters per second up to WalkReserveM.
	WalkSpeed    float64
	WalkReserveM float64

	// Pace turns base points into meters on the trail (medium difficulty).
	Pace float64
}

// Default returns the scoring used by the app.
//
// Calibration: a typical day of ~6 commits of ~60 changed lines is ≈758 base
// points; ×13 ≈ 9.9 km on medium (≈770 and 10 km by the per-commit rules).
// One commit is at most 300×13 = 3.9 km; the walker covers 5 km an hour and
// carries a reserve of 10 km.
func Default() Config {
	return Config{
		BaseMeters:    10,
		LogFactor:     20,
		CommitCap:     200,
		DailySoftCapM: 15000,
		OverCapFactor: 0.35,
		FarCapFactor:  2,
		FarFactor:     0.05,

		VolumeFactor: 45,
		VolumeLines:  10,
		VolumeCap:    300,
		WalkSpeed:    5000.0 / 3600,
		WalkReserveM: 10000,

		Pace: 13,
	}
}

// CommitMeters returns the base points of the per-commit rules for a commit
// with `lines` significant changed lines (added + deleted, after filtering).
// Zero lines -> zero. Base points are what the state stores; Scale turns them
// into meters.
func (c Config) CommitMeters(lines int) float64 {
	if lines <= 0 {
		return 0
	}
	m := c.BaseMeters + c.LogFactor*math.Log2(1+float64(lines))
	return Round1(math.Min(m, c.CommitCap))
}

// LinesOf recovers the changed lines from per-commit base points, for records
// stored before lines were: exact up to the cap (≈723 lines), the cap's
// lines beyond it.
func (c Config) LinesOf(points float64) int {
	if points <= 0 {
		return 0
	}
	return max(1, int(math.Round(math.Exp2((points-c.BaseMeters)/c.LogFactor)-1)))
}

// CommitPoints returns the base points of the volume rules for a commit with
// `lines` significant changed lines. Zero lines -> zero.
func (c Config) CommitPoints(lines int) float64 {
	if lines <= 0 {
		return 0
	}
	p := c.VolumeFactor * math.Log2(1+float64(lines)/c.VolumeLines)
	return Round1(math.Min(p, c.VolumeCap))
}

// Scale is the meters per base point at a difficulty level.
func (c Config) Scale(level string) float64 { return c.Pace * LevelFactor(level) }

// DailyEffective applies the daily discount of the per-commit rules to a
// day's scaled total.
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

// Walker is the walking-speed limit of the volume rules. Feed it commits in
// time order: each is credited at most the trail in its reserve, which starts
// full and refills at walking speed. One walker per hiker: the user's commits
// in all tracked projects together, or one teammate's.
type Walker struct {
	c       Config
	reserve float64
	last    int64
	started bool
}

// NewWalker returns a walker with a full reserve.
func (c Config) NewWalker() *Walker { return &Walker{c: c} }

// Step credits a commit made at unix time t worth `meters` (already scaled to
// the level) and returns the meters walked. The level that applied at t
// scales the walking speed and the reserve like it scales the meters.
func (w *Walker) Step(t int64, meters float64, level string) float64 {
	f := LevelFactor(level)
	capM := w.c.WalkReserveM * f
	if !w.started {
		w.reserve, w.started = capM, true
	} else if t > w.last {
		w.reserve += float64(t-w.last) * w.c.WalkSpeed * f
	}
	w.reserve = math.Min(w.reserve, capM)
	w.last = max(w.last, t)
	got := math.Max(0, math.Min(meters, w.reserve))
	w.reserve -= got
	return got
}

// TypicalDay is the distance of a typical day of commits (6 × ~60 lines) by
// the volume rules; spread over a day, the walker never limits it.
func (c Config) TypicalDay(level string) float64 {
	return Round1(6 * c.CommitPoints(60) * c.Scale(level))
}

// Round1 rounds to one decimal place, the precision shown to users.
func Round1(v float64) float64 { return math.Round(v*10) / 10 }
