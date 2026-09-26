package app

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/commit-hike/commit-hike/core/internal/i18n"
	"github.com/commit-hike/commit-hike/core/internal/protocol"
	"github.com/commit-hike/commit-hike/core/internal/routes"
)

// Limits for imported packs: a route is a few small JSON files, so anything
// bigger is a mistake (or an attack) rather than a route.
const (
	maxPackFiles = 64
	maxPackBytes = 2 << 20 // 2 MiB in total
)

var routeIDPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// ImportRoute validates a route pack (a folder or a .zip) and copies it into
// the user's routes directory. With replace, an existing user route with the
// same id is overwritten; built-in routes can never be replaced.
func (s *Service) ImportRoute(src string, replace bool, lang string) (*protocol.Route, error) {
	unlock, err := s.lock()
	if err != nil {
		return nil, err
	}
	defer unlock()

	fsys, dir, closeFn, err := openPack(src)
	if err != nil {
		return nil, fail(protocol.CodeInvalidRoute, "%s", err)
	}
	defer closeFn()

	r, err := routes.LoadPack(fsys, dir)
	if err != nil {
		return nil, fail(protocol.CodeInvalidRoute, "%s", err)
	}
	if existing, ok := s.routes[r.ID]; ok {
		if existing.Builtin {
			return nil, fail(protocol.CodeRouteExists, "%q is a built-in route; give your route another id", r.ID)
		}
		if !replace {
			return nil, fail(protocol.CodeRouteExists, "a route with id %q is already installed", r.ID)
		}
	}
	files, err := packFiles(fsys, dir)
	if err != nil {
		return nil, fail(protocol.CodeInvalidRoute, "%s", err)
	}

	// Copy into a temporary folder next to the target, then swap it in, so a
	// failure never leaves a half-copied route behind.
	root := s.st.UserRoutesDir()
	if err := os.MkdirAll(root, 0o700); err != nil {
		return nil, err
	}
	tmp, err := os.MkdirTemp(root, ".import-")
	if err != nil {
		return nil, err
	}
	defer func() { _ = os.RemoveAll(tmp) }() // no-op after the rename below
	for _, f := range files {
		data, err := fs.ReadFile(fsys, path.Join(dir, f))
		if err != nil {
			return nil, err
		}
		dst := filepath.Join(tmp, filepath.FromSlash(f))
		if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
			return nil, err
		}
		if err := os.WriteFile(dst, data, 0o600); err != nil {
			return nil, err
		}
	}
	target := filepath.Join(root, r.ID)
	if err := os.RemoveAll(target); err != nil {
		return nil, err
	}
	if err := os.Rename(tmp, target); err != nil {
		return nil, err
	}

	s.routes[r.ID] = r
	dto := routeDTO(r, i18n.Chain(lang))
	return &dto, nil
}

// RemoveRoute deletes a user route. Built-in routes and routes that a
// journey is currently using can't be removed.
func (s *Service) RemoveRoute(id string) error {
	unlock, err := s.lock()
	if err != nil {
		return err
	}
	defer unlock()
	r, ok := s.routes[id]
	switch {
	case !ok:
		return fail(protocol.CodeUnknownRoute, "unknown route %q", id)
	case r.Builtin:
		return fail(protocol.CodeInvalidArgument, "%q is a built-in route and can't be removed", id)
	}
	if cfg, err := s.st.LoadConfig(); err == nil {
		inUse := cfg.GlobalJourney != nil && cfg.GlobalJourney.RouteID == id
		for _, a := range cfg.ProjectJourneys {
			inUse = inUse || a.RouteID == id
		}
		if inUse {
			return fail(protocol.CodeRouteInUse, "route %q is used by a journey; choose another trail first", id)
		}
	}
	if err := os.RemoveAll(filepath.Join(s.st.UserRoutesDir(), id)); err != nil {
		return err
	}
	delete(s.routes, id)
	return nil
}

// RouteTemplate writes a small, valid route pack to dir/<id> as a starting
// point for people making their own routes. It returns the folder path.
func (s *Service) RouteTemplate(id, dir string) (string, error) {
	if !routeIDPattern.MatchString(id) {
		return "", fail(protocol.CodeInvalidArgument,
			"route id %q may contain only lowercase letters, digits and single dashes, e.g. my-trail", id)
	}
	if _, taken := s.routes[id]; taken {
		return "", fail(protocol.CodeRouteExists, "a route with id %q already exists", id)
	}
	target := filepath.Join(dir, id)
	if _, err := os.Stat(target); err == nil {
		return "", fail(protocol.CodeInvalidArgument, "%s already exists", target)
	}
	files := map[string]any{
		"route.json":      templateRoute(id),
		"locales/en.json": templateTexts("en"),
		"locales/uk.json": templateTexts("uk"),
	}
	for name, content := range files {
		b, err := json.MarshalIndent(content, "", "  ")
		if err != nil {
			return "", err
		}
		p := filepath.Join(target, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
			return "", err
		}
		if err := os.WriteFile(p, append(b, '\n'), 0o600); err != nil {
			return "", err
		}
	}
	return target, nil
}

// openPack returns a file system for a pack given as a folder or a .zip,
// and the directory inside it that holds route.json.
func openPack(src string) (fsys fs.FS, dir string, closeFn func(), err error) {
	fi, err := os.Stat(src)
	if err != nil {
		return nil, "", nil, err
	}
	if fi.IsDir() {
		return os.DirFS(src), ".", func() {}, nil
	}
	if !strings.EqualFold(filepath.Ext(src), ".zip") {
		return nil, "", nil, errors.New("choose a route folder or a .zip file")
	}
	zr, err := zip.OpenReader(src)
	if err != nil {
		return nil, "", nil, fmt.Errorf("can't read the zip: %w", err)
	}
	closeFn = func() { _ = zr.Close() }
	// The pack may sit at the zip root or inside one top-level folder.
	if _, err := fs.Stat(zr, "route.json"); err == nil {
		return zr, ".", closeFn, nil
	}
	entries, err := fs.ReadDir(zr, ".")
	if err == nil && len(entries) == 1 && entries[0].IsDir() {
		if _, err := fs.Stat(zr, path.Join(entries[0].Name(), "route.json")); err == nil {
			return zr, entries[0].Name(), closeFn, nil
		}
	}
	closeFn()
	return nil, "", nil, errors.New("route.json not found in the zip")
}

// packFiles lists the files to copy: route.json, locales/*.json and an
// optional README.md. Anything else is ignored; size limits apply.
func packFiles(fsys fs.FS, dir string) ([]string, error) {
	files := []string{"route.json"}
	locales, err := fs.Glob(fsys, path.Join(dir, "locales", "*.json"))
	if err != nil {
		return nil, err
	}
	for _, l := range locales {
		files = append(files, "locales/"+path.Base(l))
	}
	if _, err := fs.Stat(fsys, path.Join(dir, "README.md")); err == nil {
		files = append(files, "README.md")
	}
	if len(files) > maxPackFiles {
		return nil, fmt.Errorf("too many files in the route pack (max %d)", maxPackFiles)
	}
	var total int64
	for _, f := range files {
		fi, err := fs.Stat(fsys, path.Join(dir, f))
		if err != nil {
			return nil, err
		}
		total += fi.Size()
	}
	if total > maxPackBytes {
		return nil, fmt.Errorf("the route pack is too large (max %d KiB)", maxPackBytes>>10)
	}
	return files, nil
}

func templateRoute(id string) map[string]any {
	return map[string]any{
		"id":             id,
		"version":        1,
		"default_locale": "en",
		"length_m":       5000,
		"waypoints": []map[string]any{
			{"id": "start", "at_m": 0, "kind": "start"},
			{"id": "viewpoint", "at_m": 2500, "kind": "peak", "elevation_m": 900},
			{"id": "finish", "at_m": 5000, "kind": "finish"},
		},
		"biomes": []map[string]any{
			{"at_m": 0, "type": "meadow"},
			{"at_m": 1200, "type": "forest"},
			{"at_m": 3500, "type": "rock"},
		},
		"story": []map[string]any{{"id": "halfway", "at_m": 2000}},
		"achievements": []map[string]any{
			{"id": "first-steps", "rule": map[string]any{"type": "distance", "min_m": 100}},
			{"id": "top", "rule": map[string]any{"type": "waypoint", "waypoint": "viewpoint"}},
			{"id": "done", "rule": map[string]any{"type": "finish"}},
		},
	}
}

func templateTexts(lang string) map[string]any {
	type wp = map[string]string
	if lang == "uk" {
		return map[string]any{
			"name":        "Моя стежка",
			"description": "Маршрут, який я створив сам.",
			"waypoints": map[string]any{
				"start":     wp{"name": "Старт", "text": "Відредагуй locales/*.json, щоб написати власну історію."},
				"viewpoint": wp{"name": "Оглядовий майданчик", "text": "Тексти точок з'являються, коли ти до них доходиш."},
				"finish":    wp{"name": "Фініш", "text": "Кінець стежки."},
			},
			"story": wp{"halfway": "Історія показується між точками маршруту."},
			"achievements": map[string]any{
				"first-steps": wp{"name": "Перші кроки", "description": "Пройди 100 м."},
				"top":         wp{"name": "Вид згори", "description": "Дійди до оглядового майданчика."},
				"done":        wp{"name": "Готово", "description": "Пройди стежку до кінця."},
			},
		}
	}
	return map[string]any{
		"name":        "My Trail",
		"description": "A route I made myself.",
		"waypoints": map[string]any{
			"start":     wp{"name": "Start", "text": "Edit locales/*.json to write your own story."},
			"viewpoint": wp{"name": "Viewpoint", "text": "Waypoint texts appear when you arrive."},
			"finish":    wp{"name": "Finish", "text": "The end of the trail."},
		},
		"story": wp{"halfway": "Story beats show up between waypoints."},
		"achievements": map[string]any{
			"first-steps": wp{"name": "First Steps", "description": "Walk 100 m."},
			"top":         wp{"name": "View from the Top", "description": "Reach the viewpoint."},
			"done":        wp{"name": "Done", "description": "Finish the trail."},
		},
	}
}
