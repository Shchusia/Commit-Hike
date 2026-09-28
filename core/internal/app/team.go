package app

import (
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Shchusia/commit-hike/core/internal/filter"
	"github.com/Shchusia/commit-hike/core/internal/gitlog"
	"github.com/Shchusia/commit-hike/core/internal/protocol"
	"github.com/Shchusia/commit-hike/core/internal/score"
)

const maxTeam = 40

// Team puts everyone who commits to the project on the same route: the
// project's own journey if it has one, otherwise the global one. It uses the
// same rules as the user's own progress (filters, per-commit and daily caps,
// the journey's start date), counted only within this repository.
//
// Privacy: names come from the local git history (after .mailmap), are
// computed on request and are never written anywhere.
func (s *Service) Team(repo string) (*protocol.Team, error) {
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
	var since time.Time
	if j.assign.Since > 0 {
		since = time.Unix(j.assign.Since, 0).Add(-querySlack)
	}
	commits, err := gitlog.Log(top, since)
	if err != nil {
		return nil, err
	}

	mine := map[string]bool{}
	for _, e := range cfg.Emails {
		mine[e] = true
	}
	type person struct {
		id, name string
		me       bool
		byDay    map[int64]float64
		keys     map[string]float64 // dedup: the same commit on several branches
		commits  int
		last     int64
	}
	people := map[string]*person{}
	flt := filter.New(cfg.Ignore)
	limit := s.now().Add(futureSlack).Unix()
	for _, c := range commits {
		t := c.AuthorTime.Unix()
		if c.Parents > 1 || t > limit || t < j.assign.Since || isBot(c) {
			continue
		}
		key := c.MailmapMail
		if mine[c.AuthorEmail] || mine[c.MailmapMail] {
			key = "\x00me" // all of the user's addresses are one hiker
		}
		p := people[key]
		if p == nil {
			p = &person{
				id: s.st.ID("member", key)[:12], name: c.AuthorName, me: key == "\x00me",
				byDay: map[int64]float64{}, keys: map[string]float64{},
			}
			people[key] = p
		}
		m := s.score.CommitMeters(countLines(flt, c)) * s.score.Scale(levelAt(cfg, t)) // teammates: always today's rules
		if m <= 0 {
			continue
		}
		ck := c.AuthorEmail + "|" + strconv.FormatInt(t, 10)
		if prev, seen := p.keys[ck]; seen {
			if m <= prev {
				continue
			}
			p.byDay[s.dayOf(t)] -= prev
			p.commits--
		}
		p.keys[ck] = m
		p.byDay[s.dayOf(t)] += m
		p.commits++
		if t > p.last {
			p.last = t
		}
	}

	r := j.route
	today := s.today()
	out := &protocol.Team{Scope: j.scope, RouteID: r.ID, LengthM: r.LengthM}
	for _, p := range people {
		if p.commits == 0 {
			continue
		}
		dist, todayM := 0.0, 0.0
		for d, raw := range p.byDay {
			eff := s.score.DailyEffective(raw, levelAt(cfg, s.now().Unix()))
			dist += eff
			if d == today {
				todayM = eff
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
	return out, nil
}

// isBot skips automated committers (dependency updaters, CI).
func isBot(c gitlog.Commit) bool {
	n := strings.ToLower(c.AuthorName)
	return strings.HasSuffix(n, "[bot]") || strings.Contains(c.MailmapMail, "[bot]") ||
		strings.HasPrefix(c.MailmapMail, "noreply@") || n == "github" || n == "gitlab"
}
