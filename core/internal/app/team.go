package app

import (
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Shchusia/commit-hike/core/internal/filter"
	"github.com/Shchusia/commit-hike/core/internal/gitlog"
	"github.com/Shchusia/commit-hike/core/internal/i18n"
	"github.com/Shchusia/commit-hike/core/internal/protocol"
	"github.com/Shchusia/commit-hike/core/internal/routes"
	"github.com/Shchusia/commit-hike/core/internal/score"
	"github.com/Shchusia/commit-hike/core/internal/store"
)

const maxTeam = 40

// Team puts everyone who commits to the project on the same route: the
// project's own journey if it has one, otherwise the global one. It uses the
// same rules as the user's own progress (filters, per-commit and daily caps,
// the journey's start date), counted only within this repository. It also
// sums up the team's week and, when one is set, the team goal.
//
// Privacy: names come from the local git history (after .mailmap), are
// computed on request and are never written anywhere.
func (s *Service) Team(repo, lang string) (*protocol.Team, error) {
	cfg, err := s.st.LoadConfig()
	if err != nil {
		return nil, err
	}
	pid, top, err := s.project(repo)
	if err != nil {
		return nil, err
	}
	js := s.journeys(cfg, pid)
	if len(js) == 0 {
		return nil, fail(protocol.CodeInvalidArgument, "no journey to put the team on")
	}
	j := js[len(js)-1] // the project journey comes last when there is one
	chain := i18n.Chain(s.lang(cfg, lang))

	today := s.today()
	weekStart := today - int64((weekdayOf(today)+6)%7) // Monday
	prevStart := weekStart - 7
	goal := cfg.TeamGoals[pid]
	var goalRoute *routes.Route
	if goal != nil {
		goalRoute = s.routes[goal.RouteID] // nil when that route was removed
	}

	// One pass over the history covers the journey, the last two weeks and the goal.
	from := s.now().Add(-16 * 24 * time.Hour).Unix()
	if j.assign.Since < from {
		from = j.assign.Since
	}
	if goalRoute != nil && goal.Since < from {
		from = goal.Since
	}
	var since time.Time
	if from > 0 {
		since = time.Unix(from, 0).Add(-querySlack)
	}
	commits, err := gitlog.Log(top, since)
	if err != nil {
		return nil, err
	}

	mine := map[string]bool{}
	for _, e := range cfg.Emails {
		mine[e] = true
	}
	type entry struct {
		t int64
		m float64
	}
	type person struct {
		id, name string
		me       bool
		seen     map[string]entry // dedup: the same commit on several branches, the largest wins
		byDay    map[int64]float64
		commits  int
		last     int64
		weekRaw  map[int64]float64
		weekN    map[int64]int
		goalRaw  map[int64]float64
	}
	people := map[string]*person{}
	flt := filter.New(cfg.Ignore)
	limit := s.now().Add(futureSlack).Unix()
	for _, c := range commits {
		t := c.AuthorTime.Unix()
		if c.Parents > 1 || t > limit || isBot(c) {
			continue
		}
		key := c.MailmapMail
		if mine[c.AuthorEmail] || mine[c.MailmapMail] {
			key = "\x00me" // all of the user's addresses are one hiker
		}
		p := people[key]
		if p == nil {
			p = &person{
				id: s.st.ID("member", key)[:12], name: c.AuthorName, me: key == "\x00me", seen: map[string]entry{},
				byDay: map[int64]float64{}, weekRaw: map[int64]float64{}, weekN: map[int64]int{}, goalRaw: map[int64]float64{},
			}
			people[key] = p
		}
		m := s.score.CommitMeters(countLines(flt, c)) * s.score.Scale(levelAt(cfg, t)) // teammates: always today's rules
		if m <= 0 {
			continue
		}
		ck := c.AuthorEmail + "|" + strconv.FormatInt(t, 10)
		if prev, ok := p.seen[ck]; !ok || m > prev.m {
			p.seen[ck] = entry{t, m}
		}
	}
	for _, p := range people {
		for _, e := range p.seen {
			d := s.dayOf(e.t)
			if e.t >= j.assign.Since {
				p.byDay[d] += e.m
				p.commits++
				if e.t > p.last {
					p.last = e.t
				}
			}
			if d >= prevStart {
				p.weekRaw[d] += e.m
				p.weekN[d]++
			}
			if goalRoute != nil && e.t >= goal.Since {
				p.goalRaw[d] += e.m
			}
		}
	}
	level := levelAt(cfg, s.now().Unix())
	eff := func(raw float64) float64 { return s.score.DailyEffective(raw, level) }

	r := j.route
	out := &protocol.Team{Scope: j.scope, RouteID: r.ID, LengthM: r.LengthM}
	for _, p := range people {
		if p.commits == 0 {
			continue
		}
		dist, todayM := 0.0, 0.0
		for d, raw := range p.byDay {
			e := eff(raw)
			dist += e
			if d == today {
				todayM = e
			}
		}
		d := math.Min(dist, r.LengthM)
		mb := protocol.Member{
			ID: p.id, Name: p.name, Me: p.me, DistanceM: score.Round1(d),
			Percent: math.Round(d/r.LengthM*1000) / 10, Finished: dist >= r.LengthM,
			Commits: p.commits, TodayM: score.Round1(todayM), LastCommitAt: p.last,
		}
		if e, ok := r.ElevationAt(d); ok {
			e = math.Round(e)
			mb.ElevationM = &e
		}
		out.Members = append(out.Members, mb)
	}
	if err := s.useOwnProgress(cfg, j, out); err != nil {
		return nil, err
	}
	sort.Slice(out.Members, func(a, b int) bool {
		ma, mb := out.Members[a], out.Members[b]
		if ma.DistanceM != mb.DistanceM {
			return ma.DistanceM > mb.DistanceM
		}
		return ma.Name < mb.Name
	})
	if len(out.Members) > maxTeam {
		// Keep the user on the list even when they are far behind.
		keep := out.Members[:maxTeam]
		for _, m := range out.Members[maxTeam:] {
			if m.Me {
				keep[maxTeam-1] = m
			}
		}
		out.Hidden = len(out.Members) - maxTeam
		out.Members = keep
	}
	if out.Members == nil {
		out.Members = []protocol.Member{}
	}

	// ---- the week ----
	date := func(d int64) string { return time.Unix(d*day, 0).UTC().Format(time.DateOnly) }
	w := protocol.TeamWeek{From: date(weekStart), To: date(weekStart + 6), DaysElapsed: int(today-weekStart) + 1, Members: []protocol.WeekMember{}}
	teamDay := map[int64]float64{}
	for _, p := range people {
		wm := protocol.WeekMember{ID: p.id, Name: p.name, Me: p.me}
		for d, raw := range p.weekRaw {
			e := eff(raw)
			if d >= weekStart {
				wm.DistanceM += e
				wm.Commits += p.weekN[d]
				wm.ActiveDays++
				teamDay[d] += e
			} else {
				w.PrevTotalM += e
			}
		}
		if wm.Commits == 0 {
			continue
		}
		wm.DistanceM = score.Round1(wm.DistanceM)
		w.TotalM += wm.DistanceM
		w.Commits += wm.Commits
		w.Members = append(w.Members, wm)
	}
	for d, m := range teamDay {
		if m > w.BestDayM || (m == w.BestDayM && date(d) < w.BestDay) {
			w.BestDay, w.BestDayM = date(d), m
		}
	}
	w.ActiveDays = len(teamDay)
	w.TotalM, w.PrevTotalM, w.BestDayM = score.Round1(w.TotalM), score.Round1(w.PrevTotalM), score.Round1(w.BestDayM)
	sort.Slice(w.Members, func(a, b int) bool {
		if w.Members[a].DistanceM != w.Members[b].DistanceM {
			return w.Members[a].DistanceM > w.Members[b].DistanceM
		}
		return w.Members[a].Name < w.Members[b].Name
	})
	if len(w.Members) > maxTeam {
		w.Hidden = len(w.Members) - maxTeam
		w.Members = w.Members[:maxTeam]
	}
	out.Week = w

	// ---- the goal ----
	if goalRoute != nil {
		out.Goal = s.teamGoal(goalRoute, goal, chain, today, func(yield func(id, name string, me bool, days map[int64]float64)) {
			for _, p := range people {
				yield(p.id, p.name, p.me, p.goalRaw)
			}
		}, eff)
	}

	for _, rt := range s.Routes(lang) {
		out.Routes = append(out.Routes, protocol.RouteChoice{ID: rt.ID, Name: rt.Name, LengthM: rt.LengthM})
	}
	return out, nil
}

// teamGoal adds up everyone's distance since the goal was set.
func (s *Service) teamGoal(r *routes.Route, g *store.TeamGoal, chain []string, today int64,
	each func(func(id, name string, me bool, days map[int64]float64)), eff func(float64) float64,
) *protocol.TeamGoal {
	out := &protocol.TeamGoal{RouteID: r.ID, RouteName: r.T(chain, "name"), LengthM: r.LengthM, Since: g.Since, Members: []protocol.GoalMember{}}
	total, recent := 0.0, 0.0
	each(func(id, name string, me bool, days map[int64]float64) {
		m := 0.0
		for d, raw := range days {
			e := eff(raw)
			m += e
			if d > today-14 {
				recent += e
			}
		}
		if m > 0 {
			out.Members = append(out.Members, protocol.GoalMember{ID: id, Name: name, Me: me, DistanceM: score.Round1(m)})
			total += m
		}
	})
	sort.Slice(out.Members, func(a, b int) bool {
		if out.Members[a].DistanceM != out.Members[b].DistanceM {
			return out.Members[a].DistanceM > out.Members[b].DistanceM
		}
		return out.Members[a].Name < out.Members[b].Name
	})
	d := math.Min(total, r.LengthM)
	out.DistanceM, out.Percent, out.Finished = score.Round1(d), math.Round(d/r.LengthM*1000)/10, total >= r.LengthM
	// The pace: the last 14 days, or fewer when the goal is younger.
	span := math.Min(14, float64(today-s.dayOf(g.Since)+1))
	if span > 0 {
		out.PaceM = score.Round1(recent / span)
	}
	if !out.Finished && out.PaceM > 0 {
		out.DaysLeft = int(math.Ceil((r.LengthM - d) / out.PaceM))
	}
	dto := routeDTO(r, chain)
	for i := range dto.Waypoints {
		w := dto.Waypoints[i]
		if w.AtM <= d {
			out.LastWaypoint = &dto.Waypoints[i]
		} else if out.NextWaypoint == nil {
			out.NextWaypoint = &dto.Waypoints[i]
			out.ToNextM = score.Round1(w.AtM - d)
		}
	}
	return out
}

// SetTeamGoal sets the route the project's team walks together, counted from
// now; an empty routeID removes the goal.
func (s *Service) SetTeamGoal(repo, routeID string) error {
	unlock, err := s.lock()
	if err != nil {
		return err
	}
	defer unlock()
	cfg, err := s.st.LoadConfig()
	if err != nil {
		return err
	}
	pid, _, err := s.project(repo)
	if err != nil {
		return err
	}
	if routeID == "" {
		delete(cfg.TeamGoals, pid)
		return s.st.SaveConfig(cfg)
	}
	if _, ok := s.routes[routeID]; !ok {
		return fail(protocol.CodeUnknownRoute, "unknown route %q", routeID)
	}
	if cfg.TeamGoals == nil {
		cfg.TeamGoals = map[string]*store.TeamGoal{}
	}
	cfg.TeamGoals[pid] = &store.TeamGoal{RouteID: routeID, Since: s.now().Unix()}
	return s.st.SaveConfig(cfg)
}

// useOwnProgress puts the user's own numbers into the team: the same
// distance, commits and today's meters as on the trail. Recounting them from
// git would disagree with it: commits from before the current pace keep their
// old distance, only the user's own addresses count (not .mailmap aliases),
// and the daily limit is shared with the user's other projects.
func (s *Service) useOwnProgress(cfg *store.Config, j journey, out *protocol.Team) error {
	st, err := s.st.LoadState()
	if err != nil {
		return err
	}
	eff := s.effective(cfg, st)
	js := s.stats(st, eff, j)
	var last int64
	for id := range eff {
		r := st.Commits[id]
		if r.Time >= j.assign.Since && (j.pid == "" || r.Project == j.pid) && r.Time > last {
			last = r.Time
		}
	}
	idx := -1
	for i, m := range out.Members {
		if m.Me {
			idx = i
		}
	}
	if idx < 0 && js.commits == 0 {
		return nil // nothing of the user's on this trail and nothing in this repository
	}
	if idx < 0 {
		name := ""
		if len(cfg.Emails) > 0 {
			name, _, _ = strings.Cut(cfg.Emails[0], "@")
		}
		out.Members = append(out.Members, protocol.Member{ID: s.st.ID("member", "\x00me")[:12], Name: name, Me: true})
		idx = len(out.Members) - 1
	}
	r := j.route
	d := math.Min(js.distance, r.LengthM)
	m := &out.Members[idx]
	m.DistanceM, m.Percent, m.Finished = score.Round1(d), math.Round(d/r.LengthM*1000)/10, js.distance >= r.LengthM
	m.Commits, m.TodayM, m.LastCommitAt = js.commits, score.Round1(js.byDay[s.today()]), last
	m.ElevationM = nil
	if e, ok := r.ElevationAt(d); ok {
		e = math.Round(e)
		m.ElevationM = &e
	}
	return nil
}

// isBot skips automated committers (dependency updaters, CI).
func isBot(c gitlog.Commit) bool {
	n := strings.ToLower(c.AuthorName)
	return strings.HasSuffix(n, "[bot]") || strings.Contains(c.MailmapMail, "[bot]") ||
		strings.HasPrefix(c.MailmapMail, "noreply@") || n == "github" || n == "gitlab"
}
