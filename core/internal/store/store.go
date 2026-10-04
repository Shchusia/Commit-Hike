// Package store keeps all Commit Hike data locally, outside any repository.
//
// Layout (inside os.UserConfigDir()/commit-hike, or $COMMIT_HIKE_HOME):
//
//	config.json  user settings (mode, emails, journeys)
//	state.json   counted commits and per-project bookkeeping
//	key          32 random bytes, used to HMAC every commit/project id
//	state.lock   short-lived lock so several IDEs can share the data
//
// Privacy: no source code, commit messages, file paths, repo names or raw
// commit hashes are ever stored. Ids are HMAC-SHA256 with a local key, so the
// files alone do not reveal which repositories the user worked on.
package store

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// File format version and permissions.
const (
	SchemaVersion = 1
	dirPerm       = 0o700
	filePerm      = 0o600
	lockTimeout   = 5 * time.Second
	staleLock     = 30 * time.Second
)

// Which projects count.
const (
	ModeAll      = "all"      // every repo opened in the IDE counts
	ModeSelected = "selected" // only projects the user explicitly enabled
)

// Assignment binds a route to the global journey or to one project.
type Assignment struct {
	RouteID string `json:"route_id"`
	Since   int64  `json:"since"` // unix seconds; 0 = include full history
}

// PastJourney is a journey the user left for another route; it still counts
// for the passport.
type PastJourney struct {
	Scope   string `json:"scope"`             // global | project
	Project string `json:"project,omitempty"` // project id, for project journeys
	RouteID string `json:"route_id"`
	Since   int64  `json:"since"`
	Until   int64  `json:"until"`
}

// Config is the user settings file (config.json).
type Config struct {
	Version         int                    `json:"version"`
	Locale          string                 `json:"locale,omitempty"` // "" = follow the IDE (--lang)
	Mode            string                 `json:"mode"`
	Emails          []string               `json:"emails"`
	Ignore          []string               `json:"ignore,omitempty"` // extra filter patterns
	GlobalJourney   *Assignment            `json:"global_journey,omitempty"`
	ProjectJourneys map[string]*Assignment `json:"project_journeys,omitempty"` // key: project id
	PastJourneys    []PastJourney          `json:"past_journeys,omitempty"`    // newest last, at most 100
	EnabledProjects map[string]bool        `json:"enabled_projects,omitempty"` // for ModeSelected
	// Projects where the user wants to see teammates on the trail. Teammates
	// are computed from git history on demand and never stored.
	TeamProjects map[string]bool `json:"team_projects,omitempty"`
	// TeamGoals: a route a project's team walks together, per project id.
	// Only the route and the start are stored; the team's distance is counted
	// from git history every time, like the rest of the team view.
	TeamGoals map[string]*TeamGoal `json:"team_goals,omitempty"`
	// Difficulty over time: each change applies to commits from At on, so
	// switching never rewrites distance already walked. Empty = medium.
	Difficulty []DifficultyChange `json:"difficulty,omitempty"`
	// PaceFrom is when real-scale pace started to apply. Commits before it
	// keep the distance they had (1 base point = 1 m), so upgrading never
	// changes what was already walked. Nil = an older config, not migrated yet.
	PaceFrom *int64 `json:"pace_from,omitempty"`
	// RestDays are weekdays off (0 = Sunday … 6 = Saturday): a day off without
	// commits doesn't break a streak, a day off with commits still counts.
	RestDays []int `json:"rest_days,omitempty"`
	// Prefs are panel and notification choices, shared by every IDE on this computer.
	Prefs Prefs `json:"prefs,omitempty"`
}

// TeamGoal is a route the whole team walks together from Since on.
type TeamGoal struct {
	RouteID string `json:"route"`
	Since   int64  `json:"since"` // unix seconds
}

// Prefs are the panel and notification choices; "" means the default (auto, auto, all).
type Prefs struct {
	ReduceMotion  string `json:"reduce_motion,omitempty"` // auto | on | off
	HighContrast  string `json:"high_contrast,omitempty"` // auto | on | off
	Notifications string `json:"notifications,omitempty"` // all | milestones | off
}

// DifficultyChange is one switch of the difficulty level.
type DifficultyChange struct {
	At    int64  `json:"at"` // unix seconds; 0 = from the very beginning
	Level string `json:"level"`
}

// CommitRec is one counted commit. It holds no hash, path or message.
type CommitRec struct {
	Project string  `json:"p"`
	Time    int64   `json:"t"` // author time, unix seconds
	Meters  float64 `json:"m"` // raw meters before the daily cap
	// Clone that first counted it: HMAC(work tree path). Only that clone may
	// drop the record when the commit disappears from its history, so two
	// clones of one project with different unpushed commits don't fight.
	Src string `json:"s,omitempty"`
}

// ProjectRec is per-project bookkeeping.
type ProjectRec struct {
	LastScan int64 `json:"last_scan"`
}

// State is everything counted so far (state.json).
type State struct {
	Version  int                    `json:"version"`
	Commits  map[string]*CommitRec  `json:"commits"`  // key: HMAC(author email|author time)
	Projects map[string]*ProjectRec `json:"projects"` // key: HMAC(root commit)
	// Unlocked achievements: journey key -> achievement id -> unix time.
	// Journey key is "global:<route>" or "<project id>:<route>", so switching
	// routes never mixes achievements.
	Achievements map[string]map[string]int64 `json:"achievements,omitempty"`
	// Work trees seen so far: HMAC(path) -> project id. When a project's root
	// commit is rewritten its id changes; this map lets progress move along.
	Paths map[string]string `json:"paths,omitempty"`
}

// Store gives access to the data directory. Create it with Open.
type Store struct {
	Dir string
	key []byte
}

// ErrNotInitialized means the user has not run first-time setup yet.
var ErrNotInitialized = errors.New("commit-hike is not initialized, run `commit-hike init`")

// UserRoutesDir is where people can drop their own route packs.
func (s *Store) UserRoutesDir() string { return filepath.Join(s.Dir, "routes") }

// DefaultDir resolves the per-user data directory for this OS.
func DefaultDir() (string, error) {
	if d := os.Getenv("COMMIT_HIKE_HOME"); d != "" {
		return d, nil
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "commit-hike"), nil
}

// Open creates the data directory if needed and loads or creates the HMAC key.
func Open(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, dirPerm); err != nil {
		return nil, err
	}
	_ = os.Chmod(dir, dirPerm)
	s := &Store{Dir: dir}
	key, err := os.ReadFile(s.path("key"))
	switch {
	case err == nil && len(key) == 32:
		s.key = key
	case errors.Is(err, os.ErrNotExist):
		key = make([]byte, 32)
		if _, err := rand.Read(key); err != nil {
			return nil, err
		}
		if err := writeAtomic(s.path("key"), key); err != nil {
			return nil, err
		}
		s.key = key
	case err == nil:
		return nil, fmt.Errorf("key file is corrupted (%d bytes)", len(key))
	default:
		return nil, err
	}
	return s, nil
}

func (s *Store) path(name string) string { return filepath.Join(s.Dir, name) }

// ID returns a short keyed hash; the same input always maps to the same id
// on this machine, but the id cannot be reversed or correlated elsewhere.
func (s *Store) ID(parts ...string) string {
	m := hmac.New(sha256.New, s.key)
	for _, p := range parts {
		m.Write([]byte(p))
		m.Write([]byte{0})
	}
	return hex.EncodeToString(m.Sum(nil))[:32]
}

// Lock serializes access between processes (e.g. VS Code and IntelliJ open
// at the same time). The returned func releases the lock.
func (s *Store) Lock() (func(), error) {
	p := s.path("state.lock")
	deadline := time.Now().Add(lockTimeout)
	for {
		f, err := os.OpenFile(p, os.O_CREATE|os.O_EXCL|os.O_WRONLY, filePerm)
		if err == nil {
			_, werr := fmt.Fprintf(f, "%d\n", os.Getpid()) // the pid is only for humans debugging a stuck lock
			if err := errors.Join(werr, f.Close()); err != nil {
				_ = os.Remove(p)
				return nil, fmt.Errorf("writing lock file: %w", err)
			}
			// Heartbeat: a long-held lock never looks stale to others. If the
			// process dies, the heartbeat stops and the lock expires.
			stop := make(chan struct{})
			go func() {
				t := time.NewTicker(staleLock / 4)
				defer t.Stop()
				for {
					select {
					case <-stop:
						return
					case now := <-t.C:
						_ = os.Chtimes(p, now, now)
					}
				}
			}()
			var once sync.Once
			// If removal fails, the lock simply expires as stale after staleLock.
			return func() { once.Do(func() { close(stop); _ = os.Remove(p) }) }, nil
		}
		if !errors.Is(err, os.ErrExist) {
			return nil, err
		}
		if fi, err := os.Stat(p); err == nil && time.Since(fi.ModTime()) > staleLock {
			_ = os.Remove(p) // owner crashed; if this fails, the next OpenFile reports it
			continue
		}
		if time.Now().After(deadline) {
			return nil, errors.New("data is locked by another process")
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// LoadConfig reads config.json; ErrNotInitialized if setup hasn't run.
func (s *Store) LoadConfig() (*Config, error) {
	var c Config
	if err := readJSON(s.path("config.json"), &c); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrNotInitialized
		}
		return nil, err
	}
	if c.ProjectJourneys == nil {
		c.ProjectJourneys = map[string]*Assignment{}
	}
	if c.EnabledProjects == nil {
		c.EnabledProjects = map[string]bool{}
	}
	if c.TeamProjects == nil {
		c.TeamProjects = map[string]bool{}
	}
	return &c, nil
}

// SaveConfig writes config.json atomically.
func (s *Store) SaveConfig(c *Config) error {
	c.Version = SchemaVersion
	return writeJSON(s.path("config.json"), c)
}

// LoadState reads state.json; a missing file is an empty state.
func (s *Store) LoadState() (*State, error) {
	st := State{Version: SchemaVersion}
	if err := readJSON(s.path("state.json"), &st); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("state.json is unreadable (run `commit-hike verify` in your repos to rebuild): %w", err)
	}
	if st.Commits == nil {
		st.Commits = map[string]*CommitRec{}
	}
	if st.Projects == nil {
		st.Projects = map[string]*ProjectRec{}
	}
	if st.Achievements == nil {
		st.Achievements = map[string]map[string]int64{}
	}
	if st.Paths == nil {
		st.Paths = map[string]string{}
	}
	return &st, nil
}

// SaveState writes state.json atomically.
func (s *Store) SaveState(st *State) error {
	st.Version = SchemaVersion
	return writeJSON(s.path("state.json"), st)
}

func readJSON(p string, v any) error {
	b, err := os.ReadFile(p)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}

func writeJSON(p string, v any) error {
	b, err := json.MarshalIndent(v, "", " ")
	if err != nil {
		return err
	}
	return writeAtomic(p, b)
}

// WriteFile writes a file inside the data directory atomically.
func (s *Store) WriteFile(name string, data []byte) error { return writeAtomic(s.path(name), data) }

// ReadFile reads a file from the data directory.
func (s *Store) ReadFile(name string) ([]byte, error) { return os.ReadFile(s.path(name)) }

// Key returns a copy of the local secret key (for backups only).
func (s *Store) Key() []byte { return append([]byte(nil), s.key...) }

// ReplaceKey installs a key from a backup, so the ids stored with it match.
func (s *Store) ReplaceKey(key []byte) error {
	if len(key) != 32 {
		return fmt.Errorf("key must be 32 bytes, got %d", len(key))
	}
	if err := writeAtomic(s.path("key"), key); err != nil {
		return err
	}
	s.key = append([]byte(nil), key...)
	return nil
}

// WriteFileAt writes a private file anywhere (a backup the user exported).
func WriteFileAt(path string, data []byte) error { return writeAtomic(path, data) }

// writeAtomic writes to a temp file and renames it into place, so a crash or
// power loss never leaves a half-written file behind.
func writeAtomic(p string, data []byte) (err error) {
	tmp, err := os.CreateTemp(filepath.Dir(p), ".tmp-*")
	if err != nil {
		return err
	}
	defer func() {
		if err != nil { // clean up; the first error is the one worth reporting
			_ = tmp.Close()
			_ = os.Remove(tmp.Name())
		}
	}()
	if err = tmp.Chmod(filePerm); err != nil && !isWindows() {
		return err
	}
	if _, err = tmp.Write(data); err != nil {
		return err
	}
	if err = tmp.Sync(); err != nil {
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), p)
}

func isWindows() bool { return os.PathSeparator == '\\' }
