package app

import (
	"strconv"
	"time"

	"github.com/commit-hike/commit-hike/core/internal/filter"
	"github.com/commit-hike/commit-hike/core/internal/gitlog"
	"github.com/commit-hike/commit-hike/core/internal/i18n"
	"github.com/commit-hike/commit-hike/core/internal/protocol"
	"github.com/commit-hike/commit-hike/core/internal/render"
	"github.com/commit-hike/commit-hike/core/internal/score"
	"github.com/commit-hike/commit-hike/core/internal/store"
)

// Incremental scans look back this far (by committer date) to catch commits
// that arrived late, e.g. pushed from another machine.
const rescanOverlap = 7 * 24 * time.Hour

// Scan counts new commits in the repository containing repo. Plugins call it
// when HEAD moves and on IDE start-up.
func (s *Service) Scan(repo, lang string) (*protocol.ScanResult, error) {
	unlock, err := s.lock()
	if err != nil {
		return nil, err
	}
	defer unlock()
	cfg, err := s.st.LoadConfig()
	if err != nil {
		return nil, err
	}
	st, err := s.st.LoadState()
	if err != nil {
		return nil, err
	}
	lang = s.lang(cfg, lang)
	chain := i18n.Chain(lang)

	pid, top, err := s.project(repo)
	if err != nil {
		return &protocol.ScanResult{Status: protocol.Status{Tracked: false, Reason: err.Error(), Locale: s.resolvedLocale(chain)}}, nil
	}
	if !s.tracked(cfg, pid) {
		return &protocol.ScanResult{Status: protocol.Status{Tracked: false, Reason: "project is not enabled", Locale: s.resolvedLocale(chain)}}, nil
	}

	before := s.status(cfg, st, pid, lang)
	started := s.now()
	var since time.Time
	if pr := st.Projects[pid]; pr != nil && pr.LastScan > 0 {
		since = time.Unix(pr.LastScan, 0).Add(-rescanOverlap)
	}
	commits, err := gitlog.Log(top, since)
	if err != nil {
		return nil, err
	}
	res := &protocol.ScanResult{}
	for key, m := range s.measure(cfg, commits) {
		rec := st.Commits[key.id]
		switch {
		case rec == nil:
			if m > 0 {
				st.Commits[key.id] = &store.CommitRec{Project: pid, Time: key.time, Meters: m}
				res.NewCommits++
			}
		case rec.Project == pid && rec.Meters != m: // amended or rewritten
			rec.Meters = m
			res.UpdatedCommits++
		}
	}
	st.Projects[pid] = &store.ProjectRec{LastScan: started.Unix()}

	achievementEvents := s.unlockAchievements(cfg, st, pid, chain)
	after := s.status(cfg, st, pid, lang)
	res.Status = after
	res.AddedM = score.Round1(after.TotalM - before.TotalM)
	for _, j := range s.journeys(cfg, pid) {
		b, a := before.Global, after.Global
		if j.scope == ScopeProject {
			b, a = before.Project, after.Project
		}
		if b != nil && a != nil {
			res.Events = append(res.Events, events(j.scope, j.route, b.DistanceM, a.DistanceM, chain)...)
		}
	}
	res.Events = append(res.Events, achievementEvents...)

	// Always saved: LastScan changes even when nothing new was found.
	if err := s.st.SaveState(st); err != nil {
		return nil, err
	}
	return res, nil
}

// Verify rebuilds a project's records from git history: fixes meters, adds
// missing commits and drops records git no longer has (a hand-edited or
// corrupted state.json heals itself this way).
func (s *Service) Verify(repo string) (*protocol.VerifyResult, error) {
	unlock, err := s.lock()
	if err != nil {
		return nil, err
	}
	defer unlock()
	cfg, err := s.st.LoadConfig()
	if err != nil {
		return nil, err
	}
	st, err := s.st.LoadState()
	if err != nil {
		return nil, err
	}
	pid, top, err := s.project(repo)
	if err != nil {
		return nil, err
	}
	commits, err := gitlog.Log(top, time.Time{})
	if err != nil {
		return nil, err
	}
	res := &protocol.VerifyResult{}
	seen := map[string]bool{}
	for key, m := range s.measure(cfg, commits) {
		seen[key.id] = true
		rec := st.Commits[key.id]
		switch {
		case rec == nil && m > 0:
			st.Commits[key.id] = &store.CommitRec{Project: pid, Time: key.time, Meters: m}
			res.Added++
		case rec != nil && rec.Project == pid && m == 0:
			delete(st.Commits, key.id)
			res.Removed++
		case rec != nil && rec.Project == pid && (rec.Meters != m || rec.Time != key.time):
			rec.Meters, rec.Time = m, key.time
			res.Updated++
		}
	}
	for id, rec := range st.Commits {
		if rec.Project == pid && !seen[id] {
			delete(st.Commits, id)
			res.Removed++
		}
	}
	st.Projects[pid] = &store.ProjectRec{LastScan: s.now().Unix()}
	return res, s.st.SaveState(st)
}

// Status is read-only and lock-free: files are replaced atomically.
func (s *Service) Status(repo, lang string) (*protocol.Status, error) {
	cfg, err := s.st.LoadConfig()
	if err != nil {
		return nil, err
	}
	st, err := s.st.LoadState()
	if err != nil {
		return nil, err
	}
	pid := ""
	if repo != "" {
		if id, _, err := s.project(repo); err == nil && s.tracked(cfg, id) {
			pid = id
		}
	}
	out := s.status(cfg, st, pid, s.lang(cfg, lang))
	return &out, nil
}

// Render draws a journey as SVG, or returns the scene for front ends that
// draw it themselves.
func (s *Service) Render(scope, repo, format, lang string, width float64) (*protocol.Render, error) {
	status, err := s.Status(repo, lang)
	if err != nil {
		return nil, err
	}
	j := status.Global
	if scope == ScopeProject {
		j = status.Project
	}
	if j == nil {
		return nil, fail(protocol.CodeInvalidArgument, "no %s journey to render", scope)
	}
	r := s.routes[j.Route.ID]
	markers := make([]render.MarkerInput, 0, len(j.Route.Waypoints))
	for _, w := range j.Route.Waypoints {
		markers = append(markers, render.MarkerInput{ID: w.ID, Label: w.Name, AtM: w.AtM})
	}
	scene := render.Build(r.ID, r.Path, r.LengthM, j.DistanceM, markers)
	switch format {
	case "scene":
		return &protocol.Render{Format: format, Scene: &scene}, nil
	case "svg", "":
		if width <= 0 {
			width = 300
		}
		return &protocol.Render{Format: "svg", SVG: render.SVG(scene, j.Route.Name, width)}, nil
	}
	return nil, fail(protocol.CodeInvalidArgument, "format must be svg or scene")
}

type commitKey struct {
	id   string
	time int64
}

// measure keeps the user's own non-merge commits and computes raw meters.
// The dedup key is HMAC(author email | author time): it survives amend,
// rebase and cherry-pick, so rewritten history is never counted twice. If
// several versions of one commit are reachable, the largest wins.
func (s *Service) measure(cfg *store.Config, commits []gitlog.Commit) map[commitKey]float64 {
	mine := make(map[string]bool, len(cfg.Emails))
	for _, e := range cfg.Emails {
		mine[e] = true
	}
	flt := filter.New(cfg.Ignore)
	out := map[commitKey]float64{}
	for _, c := range commits {
		if c.Parents > 1 || !mine[c.AuthorEmail] {
			continue
		}
		lines := 0
		for _, f := range c.Files {
			if !f.Binary && !flt.Ignored(f.Path) {
				lines += f.Added + f.Deleted
			}
		}
		t := c.AuthorTime.Unix()
		k := commitKey{id: s.st.ID("commit", c.AuthorEmail, strconv.FormatInt(t, 10)), time: t}
		if m := s.score.CommitMeters(lines); m >= out[k] {
			out[k] = m
		}
	}
	return out
}
