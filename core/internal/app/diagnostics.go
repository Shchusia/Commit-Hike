package app

import (
	"errors"
	"os"
	"runtime"
	"sort"

	"github.com/Shchusia/commit-hike/core/internal/protocol"
	"github.com/Shchusia/commit-hike/core/internal/score"
	"github.com/Shchusia/commit-hike/core/internal/store"
)

// Diagnostics describes this installation for a bug report. It never includes
// e-mail addresses, paths, or project and repository names.
func (s *Service) Diagnostics(coreVersion string) (*protocol.Diagnostics, error) {
	d := &protocol.Diagnostics{CoreVersion: coreVersion, OS: runtime.GOOS, Arch: runtime.GOARCH, GoVersion: runtime.Version()}
	cfg, err := s.st.LoadConfig()
	if errors.Is(err, store.ErrNotInitialized) {
		return d, nil
	}
	if err != nil {
		return nil, err
	}
	st, err := s.st.LoadState()
	if err != nil {
		return nil, err
	}
	d.Initialized = true
	d.Emails, d.Mode, d.Locale, d.Difficulty = len(cfg.Emails), cfg.Mode, cfg.Locale, levelAt(cfg, s.now().Unix())
	d.RestDays = cfg.RestDays
	set := settingsOf(cfg)
	d.Settings = &set
	projects := map[string]bool{}
	for _, c := range st.Commits {
		projects[c.Project] = true
	}
	d.Commits, d.Projects = len(st.Commits), len(projects)
	for id, r := range s.routes {
		if !r.Builtin {
			d.UserRoutes = append(d.UserRoutes, id) // route ids are content names, not personal
		}
	}
	sort.Strings(d.UserRoutes)
	eff := s.effective(cfg, st)
	for _, j := range s.journeys(cfg, "") {
		d.Journeys = append(d.Journeys, protocol.DiagnosticTrip{Scope: j.scope, RouteID: j.route.ID, DistanceM: score.Round1(s.stats(st, eff, j).distance)})
	}
	if _, err := os.Stat(s.avatarPath()); err == nil {
		d.AvatarSet = true
	}
	return d, nil
}
