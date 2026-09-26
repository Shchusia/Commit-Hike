// Package routes loads route packs: a route.json with geometry, waypoints,
// story beats and achievement rules, plus one translation file per language.
//
//	routes/<id>/route.json
//	routes/<id>/locales/en.json
//	routes/<id>/locales/uk.json
//
// Packs come from the binary (built-in) and, optionally, from the user's
// data directory, so people can add routes without rebuilding anything.
package routes

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"

	"github.com/commit-hike/commit-hike/core/internal/achievements"
	"github.com/commit-hike/commit-hike/core/internal/i18n"
)

type Waypoint struct {
	ID  string  `json:"id"`
	AtM float64 `json:"at_m"`
}

// StoryBeat is a piece of narration shown when the walker passes AtM.
type StoryBeat struct {
	ID  string  `json:"id"`
	AtM float64 `json:"at_m"`
}

type Route struct {
	ID            string             `json:"id"`
	Version       int                `json:"version"`
	DefaultLocale string             `json:"default_locale"`
	LengthM       float64            `json:"length_m"`
	Path          [][2]float64       `json:"path,omitempty"` // optional hand-drawn shape, points in 0..1
	Waypoints     []Waypoint         `json:"waypoints"`
	Story         []StoryBeat        `json:"story,omitempty"`
	Achievements  []achievements.Def `json:"achievements,omitempty"`

	Texts   i18n.Bundle `json:"-"`
	Builtin bool        `json:"-"`
}

// T returns route text for key, following the locale chain and ending with
// the route's own default locale.
func (r *Route) T(chain []string, key string) string {
	// Full slice expression: never write into the caller's backing array.
	return r.Texts.Text(append(chain[:len(chain):len(chain)], r.DefaultLocale), key)
}

// Locales lists available translations, sorted.
func (r *Route) Locales() []string {
	out := make([]string, 0, len(r.Texts))
	for l := range r.Texts {
		out = append(out, l)
	}
	sort.Strings(out)
	return out
}

// WaypointPositions maps waypoint ids to their distance along the route.
func (r *Route) WaypointPositions() map[string]float64 {
	m := make(map[string]float64, len(r.Waypoints))
	for _, w := range r.Waypoints {
		m[w.ID] = w.AtM
	}
	return m
}

// Load reads every route pack under dir and fails on the first broken one.
// Use it for built-in routes, which must always be valid.
func Load(fsys fs.FS, dir string, builtin bool) (map[string]*Route, error) {
	out, errs := LoadEach(fsys, dir, builtin)
	if len(errs) > 0 {
		return nil, errs[0]
	}
	return out, nil
}

// LoadEach loads every valid pack and reports the broken ones separately,
// so one bad user route doesn't hide the good ones.
func LoadEach(fsys fs.FS, dir string, builtin bool) (map[string]*Route, []error) {
	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return nil, []error{err}
	}
	out := map[string]*Route{}
	var errs []error
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		r, err := loadOne(fsys, path.Join(dir, e.Name()))
		switch {
		case err != nil:
			errs = append(errs, fmt.Errorf("route %s: %w", e.Name(), err))
		case r.ID != e.Name():
			errs = append(errs, fmt.Errorf("route %s: id %q must match its folder name", e.Name(), r.ID))
		default:
			r.Builtin = builtin
			out[r.ID] = r
		}
	}
	return out, errs
}

// Merge adds user routes to built-in ones. A user route may not replace a
// built-in route: that would silently change people's progress.
func Merge(builtin, user map[string]*Route) (map[string]*Route, []error) {
	out := make(map[string]*Route, len(builtin)+len(user))
	for id, r := range builtin {
		out[id] = r
	}
	var errs []error
	for id, r := range user {
		if _, taken := out[id]; taken {
			errs = append(errs, fmt.Errorf("user route %q ignored: a built-in route has the same id", id))
			continue
		}
		out[id] = r
	}
	return out, errs
}

func loadOne(fsys fs.FS, dir string) (*Route, error) {
	raw, err := fs.ReadFile(fsys, path.Join(dir, "route.json"))
	if err != nil {
		return nil, err
	}
	var r Route
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields() // typos in route files fail loudly
	if err := dec.Decode(&r); err != nil {
		return nil, fmt.Errorf("route.json: %w", err)
	}
	if r.DefaultLocale == "" {
		r.DefaultLocale = i18n.DefaultLocale
	}

	r.Texts = i18n.Bundle{}
	files, err := fs.Glob(fsys, path.Join(dir, "locales", "*.json"))
	if err != nil {
		return nil, err
	}
	for _, f := range files {
		data, err := fs.ReadFile(fsys, f)
		if err != nil {
			return nil, err
		}
		cat, err := i18n.ParseNested(data)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path.Base(f), err)
		}
		r.Texts[strings.ToLower(strings.TrimSuffix(path.Base(f), ".json"))] = cat
	}
	return &r, r.Validate()
}

// RequiredKeys lists the translation keys every route must define in its
// default locale.
func (r *Route) RequiredKeys() []string {
	keys := []string{"name", "description"}
	for _, w := range r.Waypoints {
		keys = append(keys, "waypoints."+w.ID+".name")
	}
	for _, s := range r.Story {
		keys = append(keys, "story."+s.ID)
	}
	for _, a := range r.Achievements {
		keys = append(keys, "achievements."+a.ID+".name", "achievements."+a.ID+".description")
	}
	return keys
}

// Validate checks structure and the default-locale texts.
func (r *Route) Validate() error {
	if r.ID == "" || strings.ContainsAny(r.ID, " /\\.") {
		return errors.New("id is required and may contain only letters, digits and dashes")
	}
	if r.LengthM <= 0 {
		return errors.New("length_m must be positive")
	}
	seen := map[string]bool{}
	for _, w := range r.Waypoints {
		if w.ID == "" || seen[w.ID] {
			return fmt.Errorf("waypoint id %q is empty or duplicated", w.ID)
		}
		seen[w.ID] = true
		if w.AtM < 0 || w.AtM > r.LengthM {
			return fmt.Errorf("waypoint %q is outside the route", w.ID)
		}
	}
	for _, s := range r.Story {
		if s.ID == "" || s.AtM < 0 || s.AtM > r.LengthM {
			return fmt.Errorf("story beat %q is invalid", s.ID)
		}
	}
	for _, p := range r.Path {
		if p[0] < 0 || p[0] > 1 || p[1] < 0 || p[1] > 1 {
			return errors.New("path points must be within 0..1")
		}
	}
	if len(r.Path) == 1 {
		return errors.New("path needs at least two points")
	}
	if err := achievements.Validate(r.Achievements, r.WaypointPositions()); err != nil {
		return err
	}
	def, ok := r.Texts[r.DefaultLocale]
	if !ok {
		return fmt.Errorf("missing locales/%s.json for the default locale", r.DefaultLocale)
	}
	var missing []string
	for _, k := range r.RequiredKeys() {
		if def[k] == "" {
			missing = append(missing, k)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return fmt.Errorf("locales/%s.json is missing: %s", r.DefaultLocale, strings.Join(missing, ", "))
	}
	sort.Slice(r.Waypoints, func(i, j int) bool { return r.Waypoints[i].AtM < r.Waypoints[j].AtM })
	sort.Slice(r.Story, func(i, j int) bool { return r.Story[i].AtM < r.Story[j].AtM })
	return nil
}
