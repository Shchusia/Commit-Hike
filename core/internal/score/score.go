// Package score turns commit sizes into meters of travel.
//
// Design goals:
//   - small commits still count, huge commits are only slightly better (log curve);
//   - one commit can never be worth more than CommitCap;
//   - after DailySoftCap meters per day further progress is heavily discounted,
//     so farming thousands of commits a day is pointless.
package score

import "math"

// Config holds the scoring constants.
type Config struct {
	BaseMeters    float64 // flat reward for any meaningful commit
	LogFactor     float64 // weight of log2(1+lines)
	CommitCap     float64 // max meters per single commit
	DailySoftCap  float64 // meters per UTC day before the discount kicks in
	OverCapFactor float64 // multiplier for meters above DailySoftCap
}

// Default returns the scoring used by the app.
func Default() Config {
	return Config{
		BaseMeters:    10,
		LogFactor:     20,
		CommitCap:     200,
		DailySoftCap:  3000,
		OverCapFactor: 0.25,
	}
}

// CommitMeters returns raw meters for a commit with `lines` significant
// changed lines (added + deleted, after filtering). Zero lines -> zero meters.
func (c Config) CommitMeters(lines int) float64 {
	if lines <= 0 {
		return 0
	}
	m := c.BaseMeters + c.LogFactor*math.Log2(1+float64(lines))
	return Round1(math.Min(m, c.CommitCap))
}

// DailyEffective applies the soft daily cap to a day's raw total.
func (c Config) DailyEffective(raw float64) float64 {
	if raw <= c.DailySoftCap {
		return raw
	}
	return c.DailySoftCap + (raw-c.DailySoftCap)*c.OverCapFactor
}

// DailyFactor is the multiplier applied to every commit of a day whose raw
// total is `raw`. A per-day factor keeps results independent of the order in
// which projects are scanned, and makes per-project journeys sum exactly to
// the global one.
func (c Config) DailyFactor(raw float64) float64 {
	if raw <= 0 {
		return 1
	}
	return c.DailyEffective(raw) / raw
}

// Round1 rounds to one decimal place, the precision shown to users.
func Round1(v float64) float64 { return math.Round(v*10) / 10 }
