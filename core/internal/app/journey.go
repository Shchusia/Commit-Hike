package app

import (
	"math"

	"github.com/commit-hike/commit-hike/core/internal/achievements"
	"github.com/commit-hike/commit-hike/core/internal/i18n"
	"github.com/commit-hike/commit-hike/core/internal/protocol"
	"github.com/commit-hike/commit-hike/core/internal/routes"
	"github.com/commit-hike/commit-hike/core/internal/score"
	"github.com/commit-hike/commit-hike/core/internal/store"
)

const day = 86400

// journey is one active route: the global one, or a project's own.
type journey struct {
	scope  string
	key    string // key into State.Achievements
	pid    string // "" for global
	assign *store.Assignment
	route  *routes.Route
}

// journeyStats is everything computed from counted commits.
type journeyStats struct {
	distance float64
	commits  int
	byDay    map[int64]float64 // effective meters per UTC day
}

func (s *Service) journeys(cfg *store.Config, pid string) []journey {
	var out []journey
	if a := cfg.GlobalJourney; a != nil && s.routes[a.RouteID] != nil {
		out = append(out, journey{ScopeGlobal, "global:" + a.RouteID, "", a, s.routes[a.RouteID]})
	}
	if a := cfg.ProjectJourneys[pid]; pid != "" && a != nil && s.routes[a.RouteID] != nil {
		out = append(out, journey{ScopeProject, pid + ":" + a.RouteID, pid, a, s.routes[a.RouteID]})
	}
	return out
}

// effective applies the daily soft cap across all tracked projects, then
// distributes it back to commits proportionally, so per-project journeys
// always sum to the global one.
func (s *Service) effective(cfg *store.Config, st *store.State) map[string]float64 {
	rawByDay := map[int64]float64{}
	for _, r := range st.Commits {
		if s.tracked(cfg, r.Project) {
			rawByDay[r.Time/day] += r.Meters
		}
	}
	out := make(map[string]float64, len(st.Commits))
	for id, r := range st.Commits {
		if s.tracked(cfg, r.Project) {
			out[id] = r.Meters * s.score.DailyFactor(rawByDay[r.Time/day])
		}
	}
	return out
}

func (s *Service) stats(st *store.State, eff map[string]float64, j journey) journeyStats {
	js := journeyStats{byDay: map[int64]float64{}}
	for id, m := range eff {
		r := st.Commits[id]
		if r.Time < j.assign.Since || (j.pid != "" && r.Project != j.pid) {
			continue
		}
		js.distance += m
		js.commits++
		js.byDay[r.Time/day] += m
	}
	return js
}

// streak counts consecutive days with commits, ending today or yesterday
// (so the streak doesn't look broken first thing in the morning).
func streak(byDay map[int64]float64, today int64) int {
	d := today
	if byDay[d] == 0 {
		d--
	}
	n := 0
	for byDay[d] > 0 {
		n++
		d--
	}
	return n
}

func (s *Service) achievementContext(j journey, js journeyStats) achievements.Context {
	best := 0.0
	for _, m := range js.byDay {
		best = math.Max(best, m)
	}
	return achievements.Context{
		DistanceM:   js.distance,
		LengthM:     j.route.LengthM,
		WaypointAtM: j.route.WaypointPositions(),
		Commits:     js.commits,
		StreakDays:  streak(js.byDay, s.now().Unix()/day),
		BestDayM:    best,
	}
}

// unlockAchievements records newly met achievements and returns them as events.
func (s *Service) unlockAchievements(cfg *store.Config, st *store.State, pid string, chain []string) []protocol.Event {
	eff := s.effective(cfg, st)
	var events []protocol.Event
	for _, j := range s.journeys(cfg, pid) {
		ctx := s.achievementContext(j, s.stats(st, eff, j))
		got := st.Achievements[j.key]
		if got == nil {
			got = map[string]int64{}
			st.Achievements[j.key] = got
		}
		for _, id := range achievements.Newly(j.route.Achievements, ctx, got) {
			got[id] = s.now().Unix()
			a := achievementDTO(j.route, chain, id, false, got[id])
			events = append(events, protocol.Event{Type: protocol.EventAchievement, Journey: j.scope, Achievement: &a})
		}
	}
	return events
}

// status builds the full, translated picture for plugins.
func (s *Service) status(cfg *store.Config, st *store.State, pid, lang string) protocol.Status {
	chain := i18n.Chain(lang)
	eff := s.effective(cfg, st)
	out := protocol.Status{Tracked: true, Locale: s.resolvedLocale(chain)}
	today := s.now().Unix() / day
	for id, m := range eff {
		out.TotalM += m
		if st.Commits[id].Time/day == today {
			out.TodayM += m
		}
	}
	out.TotalM, out.TodayM = score.Round1(out.TotalM), score.Round1(out.TodayM)
	for _, j := range s.journeys(cfg, pid) {
		dto := s.journeyDTO(j, s.stats(st, eff, j), st.Achievements[j.key], chain)
		if j.scope == ScopeGlobal {
			out.Global = dto
		} else {
			out.Project = dto
		}
	}
	return out
}

func (s *Service) journeyDTO(j journey, js journeyStats, unlocked map[string]int64, chain []string) *protocol.Journey {
	r := j.route
	d := math.Min(js.distance, r.LengthM)
	out := &protocol.Journey{
		Scope:      j.scope,
		Route:      routeDTO(r, chain),
		DistanceM:  score.Round1(d),
		Percent:    math.Round(d/r.LengthM*1000) / 10,
		Finished:   js.distance >= r.LengthM,
		Commits:    js.commits,
		StreakDays: streak(js.byDay, s.now().Unix()/day),
	}
	for i, w := range r.Waypoints {
		if w.AtM <= js.distance {
			out.LastWaypoint = &out.Route.Waypoints[i]
		} else if out.NextWaypoint == nil {
			out.NextWaypoint = &out.Route.Waypoints[i]
			out.ToNextM = score.Round1(w.AtM - js.distance)
		}
	}
	for _, b := range r.Story {
		if b.AtM <= js.distance {
			out.Story = &protocol.Story{ID: b.ID, Text: r.T(chain, "story."+b.ID), AtM: b.AtM}
		}
	}
	out.Achievements = make([]protocol.Achievement, 0, len(r.Achievements))
	for _, a := range r.Achievements {
		out.Achievements = append(out.Achievements, achievementDTO(r, chain, a.ID, a.Hidden, unlocked[a.ID]))
	}
	return out
}

// resolvedLocale is the first language in chain that any route provides.
func (s *Service) resolvedLocale(chain []string) string {
	for _, loc := range chain {
		for _, r := range s.routes {
			if _, ok := r.Texts[loc]; ok {
				return loc
			}
		}
	}
	return i18n.DefaultLocale
}

// events compares two snapshots of the same journey.
func events(scope string, r *routes.Route, before, after float64, chain []string) []protocol.Event {
	var out []protocol.Event
	for _, w := range r.Waypoints {
		if w.AtM > before && w.AtM <= after {
			wp := waypointDTO(r, chain, w)
			out = append(out, protocol.Event{Type: protocol.EventWaypoint, Journey: scope, Waypoint: &wp})
		}
	}
	for _, b := range r.Story {
		if b.AtM > before && b.AtM <= after {
			out = append(out, protocol.Event{Type: protocol.EventStory, Journey: scope,
				Story: &protocol.Story{ID: b.ID, Text: r.T(chain, "story."+b.ID), AtM: b.AtM}})
		}
	}
	if before < r.LengthM && after >= r.LengthM {
		out = append(out, protocol.Event{Type: protocol.EventFinished, Journey: scope})
	}
	return out
}

func routeDTO(r *routes.Route, chain []string) protocol.Route {
	out := protocol.Route{
		ID: r.ID, Name: r.T(chain, "name"), Description: r.T(chain, "description"),
		LengthM: r.LengthM, Locales: r.Locales(), Builtin: r.Builtin, Achievements: len(r.Achievements),
	}
	for _, w := range r.Waypoints {
		out.Waypoints = append(out.Waypoints, waypointDTO(r, chain, w))
	}
	return out
}

func waypointDTO(r *routes.Route, chain []string, w routes.Waypoint) protocol.Waypoint {
	return protocol.Waypoint{
		ID: w.ID, AtM: w.AtM,
		Name: r.T(chain, "waypoints."+w.ID+".name"),
		Text: r.T(chain, "waypoints."+w.ID+".text"),
	}
}

func achievementDTO(r *routes.Route, chain []string, id string, hidden bool, unlockedAt int64) protocol.Achievement {
	a := protocol.Achievement{ID: id, Hidden: hidden, UnlockedAt: unlockedAt}
	if !hidden || unlockedAt > 0 { // secret achievements stay secret until earned
		a.Name = r.T(chain, "achievements."+id+".name")
		a.Description = r.T(chain, "achievements."+id+".description")
	}
	return a
}
