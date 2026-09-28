package app

import (
	"math"
	"time"

	"github.com/Shchusia/commit-hike/core/internal/achievements"
	"github.com/Shchusia/commit-hike/core/internal/i18n"
	"github.com/Shchusia/commit-hike/core/internal/protocol"
	"github.com/Shchusia/commit-hike/core/internal/routes"
	"github.com/Shchusia/commit-hike/core/internal/score"
	"github.com/Shchusia/commit-hike/core/internal/store"
)

const (
	day       = 86400
	dailyDays = 14 // how many days of history Journey.Daily covers
)

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
	byDay    map[int64]float64 // effective meters per local calendar day
}

// dayOf maps a unix time to a day number in the user's time zone, so "today",
// streaks and the daily cap follow the user's calendar, not UTC. Day d
// formatted with time.Unix(d*day, 0).UTC() gives that local date.
func (s *Service) dayOf(t int64) int64 {
	_, off := time.Unix(t, 0).In(s.loc).Zone()
	v := t + int64(off)
	if v < 0 {
		return (v - day + 1) / day
	}
	return v / day
}

func (s *Service) today() int64 { return s.dayOf(s.now().Unix()) }

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

// effective turns stored base points into meters (pace × the difficulty that
// applied when each commit was made), applies the daily discount across all
// tracked projects, then distributes it back to commits proportionally, so
// per-project journeys always sum to the global one.
func (s *Service) effective(cfg *store.Config, st *store.State) map[string]float64 {
	scaled := make(map[string]float64, len(st.Commits))
	dayRaw := map[int64]float64{}
	dayLast := map[int64]int64{} // latest commit of the day decides the day's level
	for id, r := range st.Commits {
		if !s.tracked(cfg, r.Project) {
			continue
		}
		m := r.Meters * s.scaleAt(cfg, r.Time)
		scaled[id] = m
		d := s.dayOf(r.Time)
		dayRaw[d] += m
		if r.Time > dayLast[d] {
			dayLast[d] = r.Time
		}
	}
	out := make(map[string]float64, len(scaled))
	for id, m := range scaled {
		d := s.dayOf(st.Commits[id].Time)
		out[id] = m * s.score.DailyFactor(dayRaw[d], levelAt(cfg, dayLast[d]))
	}
	return out
}

// scaleAt is meters per base point for a commit made at t: the real pace at
// the difficulty of that time, or 1 for commits from before the pace existed.
func (s *Service) scaleAt(cfg *store.Config, t int64) float64 {
	if cfg.PaceFrom == nil || t < *cfg.PaceFrom {
		return 1
	}
	return s.score.Scale(levelAt(cfg, t))
}

// levelAt is the difficulty that applied at unix time t (medium by default).
func levelAt(cfg *store.Config, t int64) string {
	lvl := score.Medium
	for _, c := range cfg.Difficulty {
		if c.At <= t {
			lvl = c.Level
		}
	}
	return lvl
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
		js.byDay[s.dayOf(r.Time)] += m
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
	d := math.Min(js.distance, j.route.LengthM)
	return achievements.Context{
		DistanceM:   js.distance,
		LengthM:     j.route.LengthM,
		WaypointAtM: j.route.WaypointPositions(),
		Commits:     js.commits,
		StreakDays:  streak(js.byDay, s.today()),
		BestDayM:    best,
		MaxElevM:    j.route.MaxElevation(d),
		AscentM:     j.route.Ascent(d),
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
	today := s.today()
	for id, m := range eff {
		out.TotalM += m
		if s.dayOf(st.Commits[id].Time) == today {
			out.TodayM += m
		}
	}
	out.TotalM, out.TodayM = score.Round1(out.TotalM), score.Round1(out.TodayM)
	out.Difficulty = levelAt(cfg, s.now().Unix())
	out.TypicalDayM = s.score.TypicalDay(out.Difficulty)
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
		StreakDays: streak(js.byDay, s.today()),
	}
	out.Underground = r.UndergroundAt(d)
	if e, ok := r.ElevationAt(d); ok {
		e = math.Round(e)
		out.ElevationM = &e
		out.AscentM = math.Round(r.Ascent(d))
		out.MaxElevationM = math.Round(r.MaxElevation(d))
	}
	for i, w := range r.Waypoints {
		if w.AtM <= js.distance {
			out.LastWaypoint = &out.Route.Waypoints[i]
		} else if out.NextWaypoint == nil {
			out.NextWaypoint = &out.Route.Waypoints[i]
			out.ToNextM = score.Round1(w.AtM - js.distance)
			out.ToNextClimbM = math.Round(r.Ascent(w.AtM) - r.Ascent(d))
		}
	}
	for _, b := range r.Story {
		if b.AtM <= js.distance {
			out.Story = &protocol.Story{ID: b.ID, Text: r.T(chain, "story."+b.ID), AtM: b.AtM}
		}
	}
	today := s.today()
	// Day 1 is the day the journey started: the assignment date, or the first
	// counted commit when history was included.
	start := today
	if j.assign.Since > 0 {
		start = s.dayOf(j.assign.Since)
	} else {
		for d := range js.byDay {
			start = min(start, d)
		}
	}
	out.Day = int(today-start) + 1
	for d := today - dailyDays + 1; d <= today; d++ {
		out.Daily = append(out.Daily, protocol.Day{
			Date: time.Unix(d*day, 0).UTC().Format(time.DateOnly),
			M:    score.Round1(js.byDay[d]),
		})
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
			out = append(out, protocol.Event{
				Type: protocol.EventStory, Journey: scope,
				Story: &protocol.Story{ID: b.ID, Text: r.T(chain, "story."+b.ID), AtM: b.AtM},
			})
		}
	}
	for _, d := range r.Dangers {
		if d.AtM > before && d.AtM <= after {
			out = append(out, protocol.Event{
				Type: protocol.EventDanger, Journey: scope,
				Danger: &protocol.Danger{ID: d.ID, AtM: d.AtM, Text: r.T(chain, "dangers."+d.ID)},
			})
		}
	}
	for _, f := range r.Facts {
		if f.AtM > before && f.AtM <= after {
			out = append(out, protocol.Event{
				Type: protocol.EventFact, Journey: scope,
				Fact: &protocol.Fact{ID: f.ID, AtM: f.AtM, Text: r.T(chain, "facts."+f.ID)},
			})
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
	for _, b := range r.Biomes {
		out.Biomes = append(out.Biomes, protocol.Biome{AtM: b.AtM, Type: b.Type})
	}
	for i, p := range r.Elevations() {
		out.Profile = append(out.Profile, protocol.ProfilePoint{AtM: p.AtM, ElevationM: p.ElevationM})
		if i == 0 || p.ElevationM < out.MinElevationM {
			out.MinElevationM = p.ElevationM
		}
		out.MaxElevationM = math.Max(out.MaxElevationM, p.ElevationM)
	}
	if len(out.Profile) > 0 {
		out.AscentM = math.Round(r.Ascent(r.LengthM))
	}
	for _, f := range r.Facts {
		out.Facts = append(out.Facts, protocol.Fact{ID: f.ID, AtM: f.AtM, Text: r.T(chain, "facts."+f.ID)})
	}
	for _, o := range r.Objects {
		dto := protocol.Object{
			ID: o.ID, AtM: o.AtM, Asset: o.Asset, HeightP: o.HeightP, LiftP: o.LiftP, OffsetM: o.OffsetM,
			FadeM: o.FadeM, Layer: o.Layer, Caption: r.Tmaybe(chain, "objects."+o.ID),
		}
		if dto.HeightP == 0 {
			dto.HeightP = 90
		}
		if dto.FadeM == 0 {
			dto.FadeM = 350
		}
		if dto.Layer == "" {
			dto.Layer = "trail"
		}
		out.Objects = append(out.Objects, dto)
	}
	out.Path, out.Track, out.MapImage = r.Path, r.Track, r.MapImage
	for _, u := range r.Underground {
		out.Underground = append(out.Underground, protocol.Span{FromM: u.FromM, ToM: u.ToM})
	}
	for _, d := range r.Dangers {
		out.Dangers = append(out.Dangers, protocol.Danger{ID: d.ID, AtM: d.AtM, Text: r.T(chain, "dangers."+d.ID)})
	}
	return out
}

func waypointDTO(r *routes.Route, chain []string, w routes.Waypoint) protocol.Waypoint {
	return protocol.Waypoint{
		ID: w.ID, AtM: w.AtM, Kind: w.Kind, ElevationM: w.ElevationM, Lat: w.Lat, Lon: w.Lon,
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
