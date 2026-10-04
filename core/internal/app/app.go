// Package app holds the use cases: init, scan, status, journeys, verify and
// render. It knows nothing about command lines or JSON envelopes, so the same
// logic can later sit behind a long-running server or a library binding.
package app

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/Shchusia/commit-hike/core/content"
	"github.com/Shchusia/commit-hike/core/internal/gitlog"
	"github.com/Shchusia/commit-hike/core/internal/i18n"
	"github.com/Shchusia/commit-hike/core/internal/protocol"
	"github.com/Shchusia/commit-hike/core/internal/routes"
	"github.com/Shchusia/commit-hike/core/internal/score"
	"github.com/Shchusia/commit-hike/core/internal/store"
)

// Journey scopes.
const (
	ScopeGlobal  = "global"
	ScopeProject = "project"
)

// Error carries a protocol error code alongside the message.
type Error struct {
	Code string
	Err  error
}

func (e *Error) Error() string { return e.Err.Error() }
func (e *Error) Unwrap() error { return e.Err }

func fail(code, format string, args ...any) error {
	return &Error{Code: code, Err: fmt.Errorf(format, args...)}
}

// Service is the core engine. Create it with New.
type Service struct {
	st            *store.Store
	routes        map[string]*routes.Route
	series        []routes.Series // built-in routes walked one after another
	routeWarnings []error
	score         score.Config
	now           func() time.Time
	loc           *time.Location // the user's time zone: days follow their calendar
}

// New opens the data directory and loads built-in and user routes.
func New(dir string) (*Service, error) {
	st, err := store.Open(dir)
	if err != nil {
		return nil, err
	}
	builtin, err := routes.Load(content.FS, "routes", true)
	if err != nil {
		return nil, fmt.Errorf("built-in routes are broken: %w", err)
	}
	series, err := routes.LoadSeries(content.FS, "series.json", builtin)
	if err != nil {
		return nil, fmt.Errorf("built-in series are broken: %w", err)
	}
	s := &Service{st: st, routes: builtin, series: series, score: score.Default(), now: time.Now, loc: time.Local}

	// User routes are optional; a broken one must not break the app.
	if _, err := os.Stat(st.UserRoutesDir()); err == nil {
		user, errs := routes.LoadEach(os.DirFS(st.UserRoutesDir()), ".", false)
		s.routeWarnings = append(s.routeWarnings, errs...)
		s.routes, errs = routes.Merge(builtin, user)
		s.routeWarnings = append(s.routeWarnings, errs...)
	} else if !errors.Is(err, fs.ErrNotExist) {
		s.routeWarnings = append(s.routeWarnings, err)
	}
	return s, nil
}

// RouteWarnings reports user routes that could not be loaded.
func (s *Service) RouteWarnings() []error { return s.routeWarnings }

// ---------- setup & config ----------

// InitOptions are the first-run setup choices.
type InitOptions struct {
	Emails      []string
	Mode        string
	RouteID     string
	FromHistory bool
	Locale      string
	Difficulty  string // easy | medium | hard; chosen at setup it applies to the whole history
}

// Init creates or updates the user settings. Empty options keep current values.
func (s *Service) Init(o InitOptions) (*protocol.Config, error) {
	unlock, err := s.lock()
	if err != nil {
		return nil, err
	}
	defer unlock()

	cfg, err := s.st.LoadConfig()
	if errors.Is(err, store.ErrNotInitialized) {
		zero := int64(0) // a new install walks all its history at the real pace
		cfg = &store.Config{
			Mode:            store.ModeAll,
			ProjectJourneys: map[string]*store.Assignment{},
			EnabledProjects: map[string]bool{},
			PaceFrom:        &zero,
		}
	} else if err != nil {
		return nil, err
	}

	if emails := normEmails(o.Emails); len(emails) > 0 {
		cfg.Emails = emails
	}
	if len(cfg.Emails) == 0 {
		if e := gitlog.GlobalEmail(); e != "" {
			cfg.Emails = []string{e}
		} else {
			return nil, fail(protocol.CodeInvalidArgument, "no email given and `git config --global user.email` is empty")
		}
	}
	switch o.Mode {
	case "":
	case store.ModeAll, store.ModeSelected:
		cfg.Mode = o.Mode
	default:
		return nil, fail(protocol.CodeInvalidArgument, "mode must be %q or %q", store.ModeAll, store.ModeSelected)
	}
	if o.Locale != "" {
		cfg.Locale = o.Locale
	}
	s.migratePace(cfg)
	if o.Difficulty != "" {
		if !score.ValidLevel(o.Difficulty) {
			return nil, fail(protocol.CodeInvalidArgument, "difficulty must be easy, medium or hard")
		}
		if len(cfg.Difficulty) == 0 {
			cfg.Difficulty = []store.DifficultyChange{{At: 0, Level: o.Difficulty}}
		} else if levelAt(cfg, s.now().Unix()) != o.Difficulty {
			cfg.Difficulty = append(cfg.Difficulty, store.DifficultyChange{At: s.now().Unix(), Level: o.Difficulty})
		}
	}
	if cfg.GlobalJourney == nil || o.RouteID != "" {
		id := o.RouteID
		if id == "" {
			id = s.defaultRoute()
		}
		if _, ok := s.routes[id]; !ok {
			return nil, fail(protocol.CodeUnknownRoute, "unknown route %q", id)
		}
		cfg.GlobalJourney = &store.Assignment{RouteID: id, Since: s.since(o.FromHistory)}
	}
	if err := s.st.SaveConfig(cfg); err != nil {
		return nil, err
	}
	return publicConfig(cfg), nil
}

// Config returns the settings plugins may show.
func (s *Service) Config() (*protocol.Config, error) {
	cfg, err := s.st.LoadConfig()
	if err != nil {
		return nil, err
	}
	return publicConfig(cfg), nil
}

// Routes lists all routes, translated.
func (s *Service) Routes(lang string) []protocol.Route {
	chain := i18n.Chain(lang)
	out := make([]protocol.Route, 0, len(s.routes))
	for _, r := range s.routes {
		out = append(out, routeDTO(r, chain))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// SetJourney assigns a route to the global journey or a project; route
// "none" removes a project's journey.
func (s *Service) SetJourney(scope, repo, routeID string, fromHistory bool, lang string) (*protocol.Status, error) {
	unlock, err := s.lock()
	if err != nil {
		return nil, err
	}
	defer unlock()
	cfg, err := s.st.LoadConfig()
	if err != nil {
		return nil, err
	}
	remove := routeID == "" || routeID == "none"
	if !remove {
		if _, ok := s.routes[routeID]; !ok {
			return nil, fail(protocol.CodeUnknownRoute, "unknown route %q", routeID)
		}
	}
	a := &store.Assignment{RouteID: routeID, Since: s.since(fromHistory)}
	pid := ""
	// the journey being left still counts for the passport
	keepPast := func(scope, pid string, old *store.Assignment) {
		if old == nil || (!remove && old.RouteID == routeID && old.Since == a.Since) {
			return
		}
		cfg.PastJourneys = append(cfg.PastJourneys, store.PastJourney{Scope: scope, Project: pid, RouteID: old.RouteID, Since: old.Since, Until: s.now().Unix()})
		if n := len(cfg.PastJourneys); n > 100 {
			cfg.PastJourneys = cfg.PastJourneys[n-100:]
		}
	}
	switch scope {
	case ScopeGlobal:
		if remove {
			return nil, fail(protocol.CodeInvalidArgument, "the global journey cannot be removed, choose another route")
		}
		keepPast(ScopeGlobal, "", cfg.GlobalJourney)
		cfg.GlobalJourney = a
	case ScopeProject:
		if pid, _, err = s.project(repo); err != nil {
			return nil, err
		}
		keepPast(ScopeProject, pid, cfg.ProjectJourneys[pid])
		if remove {
			delete(cfg.ProjectJourneys, pid)
		} else {
			cfg.ProjectJourneys[pid] = a
			cfg.EnabledProjects[pid] = true // choosing a journey implies tracking
		}
	default:
		return nil, fail(protocol.CodeInvalidArgument, "scope must be %q or %q", ScopeGlobal, ScopeProject)
	}
	if err := s.st.SaveConfig(cfg); err != nil {
		return nil, err
	}
	st, err := s.st.LoadState()
	if err != nil {
		return nil, err
	}
	// History may already satisfy some achievements: unlock them quietly.
	s.unlockAchievements(cfg, st, pid, i18n.Chain(s.lang(cfg, lang)))
	if err := s.st.SaveState(st); err != nil {
		return nil, err
	}
	status := s.status(cfg, st, pid, s.lang(cfg, lang))
	return &status, nil
}

// SetProjectEnabled matters in "selected" mode.
func (s *Service) SetProjectEnabled(repo string, on bool) error {
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
	if on {
		cfg.EnabledProjects[pid] = true
	} else {
		delete(cfg.EnabledProjects, pid)
	}
	return s.st.SaveConfig(cfg)
}

// SetTeam turns the teammates view on or off for a project.
func (s *Service) SetTeam(repo string, on bool) error {
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
	if on {
		cfg.TeamProjects[pid] = true
	} else {
		delete(cfg.TeamProjects, pid)
	}
	return s.st.SaveConfig(cfg)
}

// Locale reads the language setting and, with set, changes it: "auto" (or
// "") follows the IDE, anything else fixes the language for every IDE.
func (s *Service) Locale(set *string, requested string) (*protocol.LocaleInfo, error) {
	cfg, err := s.st.LoadConfig()
	if err != nil {
		return nil, err
	}
	if set != nil {
		v := strings.ToLower(strings.TrimSpace(*set))
		if v == "auto" {
			v = ""
		}
		if v != "" && !localePattern.MatchString(v) {
			return nil, fail(protocol.CodeInvalidArgument, "locale must be auto or a language tag like uk or en")
		}
		unlock, err := s.lock()
		if err != nil {
			return nil, err
		}
		defer unlock()
		if cfg, err = s.st.LoadConfig(); err != nil {
			return nil, err
		}
		cfg.Locale = v
		if err := s.st.SaveConfig(cfg); err != nil {
			return nil, err
		}
	}
	avail := map[string]bool{i18n.DefaultLocale: true}
	for _, l := range i18n.UILocales {
		avail[l] = true
	}
	for _, r := range s.routes {
		for _, l := range r.Locales() {
			avail[l] = true
		}
	}
	out := &protocol.LocaleInfo{Locale: cfg.Locale, Effective: s.resolvedLocale(i18n.Chain(s.lang(cfg, requested)))}
	for l := range avail {
		out.Available = append(out.Available, l)
	}
	sort.Strings(out.Available)
	return out, nil
}

// migratePace starts the real pace now for a config from before it existed.
// It reports whether the config changed.
func (s *Service) migratePace(cfg *store.Config) bool {
	if cfg.PaceFrom != nil {
		return false
	}
	now := s.now().Unix()
	cfg.PaceFrom = &now
	return true
}

// Difficulty reads the difficulty and, with set, changes it. A change applies
// to commits from now on: distance already walked stays as it was.
func (s *Service) Difficulty(set string) (*protocol.DifficultyInfo, error) {
	if set != "" {
		if !score.ValidLevel(set) {
			return nil, fail(protocol.CodeInvalidArgument, "difficulty must be easy, medium or hard")
		}
		unlock, err := s.lock()
		if err != nil {
			return nil, err
		}
		defer unlock()
		cfg, err := s.st.LoadConfig()
		if err != nil {
			return nil, err
		}
		now := s.now().Unix()
		changed := s.migratePace(cfg)
		if levelAt(cfg, now) != set {
			cfg.Difficulty = append(cfg.Difficulty, store.DifficultyChange{At: now, Level: set})
			changed = true
		}
		if changed {
			if err := s.st.SaveConfig(cfg); err != nil {
				return nil, err
			}
		}
	}
	cfg, err := s.st.LoadConfig()
	if err != nil {
		return nil, err
	}
	out := &protocol.DifficultyInfo{Level: levelAt(cfg, s.now().Unix()), Levels: score.Levels, TypicalDayM: map[string]float64{}}
	for _, l := range score.Levels {
		out.TypicalDayM[l] = s.score.TypicalDay(l)
	}
	return out, nil
}

var localePattern = regexp.MustCompile(`^[a-z]{2,3}([-_][a-z0-9]{2,8})*$`)

// ---------- helpers ----------

func (s *Service) lock() (func(), error) {
	unlock, err := s.st.Lock()
	if err != nil {
		return nil, &Error{Code: protocol.CodeBusy, Err: err}
	}
	return unlock, nil
}

func (s *Service) project(repo string) (id, top string, err error) {
	if repo == "" {
		return "", "", fail(protocol.CodeInvalidArgument, "repository path is required")
	}
	top, err = gitlog.TopLevel(repo)
	if err != nil {
		return "", "", &Error{Code: protocol.CodeNotARepo, Err: err}
	}
	root, err := gitlog.RootCommit(top)
	if err != nil {
		return "", "", &Error{Code: protocol.CodeNotARepo, Err: err}
	}
	return s.st.ID("project", root), top, nil
}

func (s *Service) tracked(cfg *store.Config, pid string) bool {
	return cfg.Mode != store.ModeSelected || cfg.EnabledProjects[pid]
}

func (s *Service) since(fromHistory bool) int64 {
	if fromHistory {
		return 0
	}
	return s.now().Unix()
}

// lang: an explicit user choice in config wins over the IDE's language.
func (s *Service) lang(cfg *store.Config, requested string) string {
	if cfg.Locale != "" {
		return cfg.Locale
	}
	if requested != "" {
		return requested
	}
	return i18n.DefaultLocale
}

func (s *Service) defaultRoute() string {
	if _, ok := s.routes["demo-trail"]; ok {
		return "demo-trail"
	}
	ids := make([]string, 0, len(s.routes))
	for id := range s.routes {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids[0]
}

func publicConfig(c *store.Config) *protocol.Config {
	return &protocol.Config{Mode: c.Mode, Emails: c.Emails, Locale: c.Locale}
}

func normEmails(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, e := range in {
		e = strings.ToLower(strings.TrimSpace(e))
		if e != "" && !seen[e] {
			seen[e] = true
			out = append(out, e)
		}
	}
	return out
}
