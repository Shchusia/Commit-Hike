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
	"time"
)

const (
	SchemaVersion = 1
	dirPerm       = 0o700
	filePerm      = 0o600
	lockTimeout   = 5 * time.Second
	staleLock     = 30 * time.Second
)

const (
	ModeAll      = "all"      // every repo opened in the IDE counts
	ModeSelected = "selected" // only projects the user explicitly enabled
)

// Assignment binds a route to the global journey or to one project.
type Assignment struct {
	RouteID string `json:"route_id"`
	Since   int64  `json:"since"` // unix seconds; 0 = include full history
}

type Config struct {
	Version         int                    `json:"version"`
	Locale          string                 `json:"locale,omitempty"` // "" = follow the IDE (--lang)
	Mode            string                 `json:"mode"`
	Emails          []string               `json:"emails"`
	Ignore          []string               `json:"ignore,omitempty"` // extra filter patterns
	GlobalJourney   *Assignment            `json:"global_journey,omitempty"`
	ProjectJourneys map[string]*Assignment `json:"project_journeys,omitempty"` // key: project id
	EnabledProjects map[string]bool        `json:"enabled_projects,omitempty"` // for ModeSelected
}

type CommitRec struct {
	Project string  `json:"p"`
	Time    int64   `json:"t"` // author time, unix seconds
	Meters  float64 `json:"m"` // raw meters before the daily cap
}

type ProjectRec struct {
	LastScan int64 `json:"last_scan"`
}

type State struct {
	Version  int                    `json:"version"`
	Commits  map[string]*CommitRec  `json:"commits"`  // key: HMAC(author email|author time)
	Projects map[string]*ProjectRec `json:"projects"` // key: HMAC(root commit)
	// Unlocked achievements: journey key -> achievement id -> unix time.
	// Journey key is "global:<route>" or "<project id>:<route>", so switching
	// routes never mixes achievements.
	Achievements map[string]map[string]int64 `json:"achievements,omitempty"`
}

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
			fmt.Fprintf(f, "%d\n", os.Getpid())
			f.Close()
			return func() { os.Remove(p) }, nil
		}
		if !errors.Is(err, os.ErrExist) {
			return nil, err
		}
		if fi, err := os.Stat(p); err == nil && time.Since(fi.ModTime()) > staleLock {
			os.Remove(p) // owner crashed
			continue
		}
		if time.Now().After(deadline) {
			return nil, errors.New("data is locked by another process")
		}
		time.Sleep(50 * time.Millisecond)
	}
}

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
	return &c, nil
}

func (s *Store) SaveConfig(c *Config) error {
	c.Version = SchemaVersion
	return writeJSON(s.path("config.json"), c)
}

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
	return &st, nil
}

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

// writeAtomic writes to a temp file and renames it into place, so a crash or
// power loss never leaves a half-written file behind.
func writeAtomic(p string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(p), ".tmp-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name()) // no-op after successful rename
	if err := tmp.Chmod(filePerm); err != nil && !isWindows() {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), p)
}

func isWindows() bool { return os.PathSeparator == '\\' }
