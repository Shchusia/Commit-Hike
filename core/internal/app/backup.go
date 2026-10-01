package app

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/Shchusia/commit-hike/core/internal/protocol"
	"github.com/Shchusia/commit-hike/core/internal/store"
)

// A backup is one JSON file with everything needed to continue on another
// computer. It includes the local key: commits and projects are stored under
// ids keyed with it, so without the key the same commits would count again.
// That also makes the file private, like the data directory itself.
const (
	backupFormat   = "commit-hike-backup"
	backupVersion  = 1
	maxBackupBytes = 64 << 20
)

type backupFile struct {
	Format     string            `json:"format"`
	Version    int               `json:"version"`
	CreatedAt  int64             `json:"created_at"`
	AppVersion string            `json:"app_version,omitempty"`
	Key        []byte            `json:"key"`
	Config     json.RawMessage   `json:"config"`
	State      json.RawMessage   `json:"state"`
	Avatar     []byte            `json:"avatar,omitempty"`
	Routes     map[string][]byte `json:"routes,omitempty"` // path inside the user routes folder -> content
}

// ExportBackup writes all progress, settings, the hiker icon and user routes
// to one file at path.
func (s *Service) ExportBackup(path, appVersion string) (*protocol.BackupResult, error) {
	if path == "" {
		return nil, fail(protocol.CodeInvalidArgument, "--path is required")
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
	st, err := s.st.LoadState()
	if err != nil {
		return nil, err
	}
	b := backupFile{Format: backupFormat, Version: backupVersion, CreatedAt: s.now().Unix(), AppVersion: appVersion, Key: s.st.Key()}
	if b.Config, err = json.Marshal(cfg); err != nil {
		return nil, err
	}
	if b.State, err = json.Marshal(st); err != nil {
		return nil, err
	}
	if data, err := os.ReadFile(s.avatarPath()); err == nil {
		b.Avatar = data
	} else if !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	b.Routes = map[string][]byte{}
	root := s.st.UserRoutesDir()
	err = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if errors.Is(err, fs.ErrNotExist) && p == root {
			return fs.SkipAll // no user routes
		}
		if err != nil || d.IsDir() {
			return err
		}
		if !d.Type().IsRegular() {
			return nil // symlinks and the like: never follow them out of the routes folder
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		// Only regular files of the private (0700) data directory get here; swapping one
		// for a link mid-walk would need write access to that directory already.
		data, err := os.ReadFile(p) //nolint:gosec // see above; os.Root needs Go 1.24

		b.Routes[filepath.ToSlash(rel)] = data
		return err
	})
	if err != nil {
		return nil, err
	}
	out, err := json.Marshal(b)
	if err != nil {
		return nil, err
	}
	if err := store.WriteFileAt(path, out); err != nil {
		return nil, fail(protocol.CodeInvalidArgument, "can't write %s: %v", path, err)
	}
	return &protocol.BackupResult{Path: path, CreatedAt: b.CreatedAt, Commits: len(st.Commits), Routes: len(b.Routes), Avatar: b.Avatar != nil}, nil
}

// ImportBackup restores a backup. Existing progress is replaced only when
// replace is true, so nobody overwrites their trail by accident.
func (s *Service) ImportBackup(path string, replace bool) (*protocol.BackupResult, error) {
	if path == "" {
		return nil, fail(protocol.CodeInvalidArgument, "--path is required")
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, fail(protocol.CodeInvalidArgument, "can't read %s: %v", path, err)
	}
	if info.Size() > maxBackupBytes {
		return nil, fail(protocol.CodeInvalidBackup, "%s is too large for a Commit Hike backup", path)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fail(protocol.CodeInvalidArgument, "can't read %s: %v", path, err)
	}
	var b backupFile
	if err := json.Unmarshal(raw, &b); err != nil || b.Format != backupFormat {
		return nil, fail(protocol.CodeInvalidBackup, "%s is not a Commit Hike backup", path)
	}
	if b.Version > backupVersion {
		return nil, fail(protocol.CodeInvalidBackup, "this backup was made by a newer Commit Hike (format %d): update the plugin first", b.Version)
	}
	if len(b.Key) != 32 {
		return nil, fail(protocol.CodeInvalidBackup, "the backup is damaged: no valid key")
	}
	var cfg store.Config
	var st store.State
	if json.Unmarshal(b.Config, &cfg) != nil || json.Unmarshal(b.State, &st) != nil || cfg.Version == 0 {
		return nil, fail(protocol.CodeInvalidBackup, "the backup is damaged: settings or progress can't be read")
	}
	for rel := range b.Routes {
		if !filepath.IsLocal(filepath.FromSlash(rel)) {
			return nil, fail(protocol.CodeInvalidBackup, "the backup is damaged: route file %q points outside the routes folder", rel)
		}
	}

	unlock, err := s.lock()
	if err != nil {
		return nil, err
	}
	defer unlock()
	if _, err := s.st.LoadConfig(); err == nil && !replace {
		return nil, fail(protocol.CodeDataExists, "this computer already has Commit Hike progress: replace it to restore the backup")
	} else if err != nil && !errors.Is(err, store.ErrNotInitialized) {
		return nil, err
	}

	if err := s.st.ReplaceKey(b.Key); err != nil {
		return nil, err
	}
	if err := s.st.SaveConfig(&cfg); err != nil {
		return nil, err
	}
	if err := s.st.SaveState(&st); err != nil {
		return nil, err
	}
	if b.Avatar != nil {
		if err := s.st.WriteFile(avatarFile, b.Avatar); err != nil {
			return nil, err
		}
	} else if err := os.Remove(s.avatarPath()); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	root := s.st.UserRoutesDir()
	if err := os.RemoveAll(root); err != nil {
		return nil, err
	}
	for rel, data := range b.Routes {
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
			return nil, err
		}
		if err := store.WriteFileAt(p, data); err != nil {
			return nil, err
		}
	}
	return &protocol.BackupResult{Path: path, CreatedAt: b.CreatedAt, Commits: len(st.Commits), Routes: len(b.Routes), Avatar: b.Avatar != nil}, nil
}
