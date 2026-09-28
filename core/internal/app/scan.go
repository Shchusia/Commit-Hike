package app

import (
	"strconv"
	"strings"
	"time"

	"github.com/Shchusia/commit-hike/core/internal/filter"
	"github.com/Shchusia/commit-hike/core/internal/gitlog"
	"github.com/Shchusia/commit-hike/core/internal/i18n"
	"github.com/Shchusia/commit-hike/core/internal/protocol"
	"github.com/Shchusia/commit-hike/core/internal/render"
	"github.com/Shchusia/commit-hike/core/internal/score"
	"github.com/Shchusia/commit-hike/core/internal/store"
)

const (
	// Incremental scans look back this far (by committer date) to catch commits
	// that arrived late, e.g. pushed from another machine. Commits inside the
	// window that git no longer has (squash, rebase, reset) are dropped.
	rescanOverlap = 7 * 24 * time.Hour
	// The query reaches a bit further back than the reconcile cut, so a commit
	// whose committer date is slightly older than its author date is still seen.
	querySlack = 24 * time.Hour
	// Commits dated further in the future than this are ignored: a wrong clock
	// or a forged date must not bypass the daily cap.
	futureSlack = 24 * time.Hour
)

// ScanOptions tune a scan.
type ScanOptions struct {
	// PrevHead is the HEAD the plugin saw before this change. If it is not an
	// ancestor of the current HEAD, history was rewritten and the whole
	// project is recounted.
	PrevHead string
	// Full re-reads the whole history and drops what git no longer has.
	Full bool
}

// Scan counts new commits in the repository containing repo. Plugins call it
// when HEAD moves and on IDE start-up.
func (s *Service) Scan(repo, lang string) (*protocol.ScanResult, error) {
	return s.ScanWith(repo, lang, ScanOptions{})
}

// ScanWith is Scan with options.
func (s *Service) ScanWith(repo, lang string, o ScanOptions) (*protocol.ScanResult, error) {
	res, _, err := s.sync(repo, lang, o, true)
	return res, err
}

// Verify rebuilds a project's records from git history: fixes meters, adds
// missing commits and drops records git no longer has (a hand-edited or
// corrupted state.json heals itself this way).
func (s *Service) Verify(repo string) (*protocol.VerifyResult, error) {
	_, v, err := s.sync(repo, "", ScanOptions{Full: true}, false)
	return v, err
}

// sync is the shared body of Scan and Verify. The slow part (git) runs
// without the lock, so another IDE is never blocked by a big repository; the
// merge then happens under the lock on freshly re-read state.
func (s *Service) sync(repo, lang string, o ScanOptions, scanning bool) (*protocol.ScanResult, *protocol.VerifyResult, error) {
	// ---- phase 1: no lock ----
	cfg, err := s.st.LoadConfig()
	if err != nil {
		return nil, nil, err
	}
	untracked := func(code, reason string) *protocol.ScanResult {
		chain := i18n.Chain(s.lang(cfg, lang))
		return &protocol.ScanResult{Status: protocol.Status{Tracked: false, Reason: reason, ReasonCode: code, Locale: s.resolvedLocale(chain)}}
	}
	pid, top, err := s.project(repo)
	if err != nil {
		if scanning {
			return untracked(protocol.ReasonNotARepo, err.Error()), nil, nil
		}
		return nil, nil, err
	}
	st0, err := s.st.LoadState()
	if err != nil {
		return nil, nil, err
	}
	pathID := s.st.ID("path", top)
	oldPID := st0.Paths[pathID]
	// In "selected" mode a project whose root was rewritten is still the
	// project the user enabled: judge by the id it had.
	if scanning && !s.tracked(cfg, pid) && (oldPID == "" || !s.tracked(cfg, oldPID)) {
		return untracked(protocol.ReasonNotEnabled, "project is not enabled"), nil, nil
	}
	full := o.Full
	rewritten := false
	if oldPID != "" && oldPID != pid { // the root commit changed (or another repository lives here now)
		full, rewritten = true, true
	}
	if !gitlog.IsAncestor(top, o.PrevHead) { // amend, rebase, reset, squash…
		full, rewritten = true, true
	}
	var since, cut time.Time
	if pr := st0.Projects[pid]; !full && pr != nil && pr.LastScan > 0 {
		cut = time.Unix(pr.LastScan, 0).Add(-rescanOverlap)
		since = cut.Add(-querySlack)
	}
	started := s.now()
	commits, err := gitlog.Log(top, since)
	if err != nil {
		return nil, nil, err
	}
	measured := s.measure(cfg, commits)

	// ---- phase 2: locked merge ----
	unlock, err := s.lock()
	if err != nil {
		return nil, nil, err
	}
	defer unlock()
	if cfg, err = s.st.LoadConfig(); err != nil {
		return nil, nil, err
	}
	st, err := s.st.LoadState()
	if err != nil {
		return nil, nil, err
	}
	lang = s.lang(cfg, lang)
	chain := i18n.Chain(lang)

	cfgChanged := s.migrate(cfg, st, pathID, pid, measured)
	cfgChanged = s.migratePace(cfg) || cfgChanged
	before := s.status(cfg, st, pid, lang)
	v := s.merge(st, pid, pathID, measured, full, cut)
	st.Projects[pid] = &store.ProjectRec{LastScan: started.Unix()}
	st.Paths[pathID] = pid

	achievementEvents := s.unlockAchievements(cfg, st, pid, chain)
	after := s.status(cfg, st, pid, lang)
	res := &protocol.ScanResult{
		Status: after, NewCommits: v.Added, UpdatedCommits: v.Updated, RemovedCommits: v.Removed,
		Rewritten: rewritten, AddedM: score.Round1(after.TotalM - before.TotalM),
	}
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

	if cfgChanged {
		if err := s.st.SaveConfig(cfg); err != nil {
			return nil, nil, err
		}
	}
	// Always saved: LastScan changes even when nothing new was found.
	if err := s.st.SaveState(st); err != nil {
		return nil, nil, err
	}
	return res, &v, nil
}

// merge folds freshly measured commits into the state. Records of this
// project that git no longer has are dropped: inside the rescan window
// (from cut on) for a normal scan, everywhere for a full one. A record is
// only dropped by the clone that counted it, so clones with different
// unpushed commits don't undo each other.
func (s *Service) merge(st *store.State, pid, src string, measured map[commitKey]float64, full bool, cut time.Time) protocol.VerifyResult {
	var v protocol.VerifyResult
	seen := make(map[string]bool, len(measured))
	owned := func(r *store.CommitRec) bool { return r.Src == "" || r.Src == src }
	for key, m := range measured {
		seen[key.id] = true
		rec := st.Commits[key.id]
		switch {
		case rec == nil:
			if m > 0 {
				st.Commits[key.id] = &store.CommitRec{Project: pid, Time: key.time, Meters: m, Src: src}
				v.Added++
			}
		case rec.Project != pid:
			// The same commit counted in another project first (cherry-pick
			// into an unrelated repository): it stays there.
		case m == 0:
			if owned(rec) { // e.g. the ignore list changed
				delete(st.Commits, key.id)
				v.Removed++
			}
		default:
			if rec.Src == "" {
				rec.Src = src // legacy record: this clone adopts it
			}
			if rec.Meters != m || rec.Time != key.time { // amended or rewritten
				rec.Meters, rec.Time = m, key.time
				v.Updated++
			}
		}
	}
	for id, rec := range st.Commits {
		if rec.Project != pid || seen[id] || !owned(rec) {
			continue
		}
		if !full && !cut.IsZero() && rec.Time < cut.Unix() {
			continue // outside the window this scan looked at
		}
		delete(st.Commits, id)
		v.Removed++
	}
	return v
}

// migrate moves a project's progress to its new id after its root commit was
// rewritten (amending the first commit, filter-repo, squashing everything).
// It reports whether the config changed. A different repository that now
// lives at the same path is told apart by sharing no commits with the old
// one and by the old one having been seen elsewhere.
func (s *Service) migrate(cfg *store.Config, st *store.State, pathID, pid string, measured map[commitKey]float64) bool {
	old := st.Paths[pathID]
	if old == "" || old == pid {
		return false
	}
	overlap := false
	for key := range measured {
		if r := st.Commits[key.id]; r != nil && r.Project == old {
			overlap = true
			break
		}
	}
	if !overlap {
		// No shared commits: still the same project if it was only ever seen
		// here (its whole history was squashed into a new root).
		for p, id := range st.Paths {
			if id == old && p != pathID {
				return false
			}
		}
		for _, r := range st.Commits {
			if r.Project == old && r.Src != "" && r.Src != pathID {
				return false
			}
		}
	}
	for _, r := range st.Commits {
		if r.Project == old {
			r.Project = pid
		}
	}
	for p, id := range st.Paths {
		if id == old {
			st.Paths[p] = pid
		}
	}
	delete(st.Projects, old)
	for key, got := range st.Achievements {
		if strings.HasPrefix(key, old+":") {
			nk := pid + key[len(old):]
			if _, taken := st.Achievements[nk]; !taken {
				st.Achievements[nk] = got
			}
			delete(st.Achievements, key)
		}
	}
	changed := false
	if a := cfg.ProjectJourneys[old]; a != nil {
		if cfg.ProjectJourneys[pid] == nil {
			cfg.ProjectJourneys[pid] = a
		}
		delete(cfg.ProjectJourneys, old)
		changed = true
	}
	for _, m := range []map[string]bool{cfg.EnabledProjects, cfg.TeamProjects} {
		if m[old] {
			m[pid] = true
			delete(m, old)
			changed = true
		}
	}
	return changed
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
	var out protocol.Status
	if repo != "" {
		id, _, err := s.project(repo)
		switch {
		case err != nil:
			out = s.status(cfg, st, "", s.lang(cfg, lang))
			out.Tracked, out.ReasonCode, out.Reason = false, protocol.ReasonNotARepo, err.Error()
			return &out, nil
		case !s.tracked(cfg, id):
			out = s.status(cfg, st, "", s.lang(cfg, lang))
			out.Tracked, out.ReasonCode, out.Reason = false, protocol.ReasonNotEnabled, "project is not enabled"
			return &out, nil
		}
		pid = id
	}
	out = s.status(cfg, st, pid, s.lang(cfg, lang))
	out.Team = pid != "" && cfg.TeamProjects[pid]
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
	limit := s.now().Add(futureSlack).Unix()
	out := map[commitKey]float64{}
	for _, c := range commits {
		if c.Parents > 1 || !mine[c.AuthorEmail] || c.AuthorTime.Unix() > limit {
			continue
		}
		t := c.AuthorTime.Unix()
		k := commitKey{id: s.st.ID("commit", c.AuthorEmail, strconv.FormatInt(t, 10)), time: t}
		if m := s.score.CommitMeters(countLines(flt, c)); m >= out[k] {
			out[k] = m
		}
	}
	return out
}

// countLines sums significant changed lines of a commit.
func countLines(flt *filter.Filter, c gitlog.Commit) int {
	lines := 0
	for _, f := range c.Files {
		if !f.Binary && !flt.Ignored(f.Path) {
			lines += f.Added + f.Deleted
		}
	}
	return lines
}
