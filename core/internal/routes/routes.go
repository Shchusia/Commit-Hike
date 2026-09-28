// Package routes loads route packs: a route.json with geometry, waypoints,
// an elevation profile, story beats, facts, objects and achievement rules,
// plus one translation file per language and optional assets.
//
//	routes/<id>/route.json
//	routes/<id>/locales/en.json
//	routes/<id>/locales/uk.json
//	routes/<id>/assets/*.svg|png|jpg|webp|gif|html   (objects, map image)
//
// Packs come from the binary (built-in) and, optionally, from the user's
// data directory, so people can add routes without rebuilding anything.
package routes

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"math"
	"path"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/Shchusia/commit-hike/core/internal/achievements"
	"github.com/Shchusia/commit-hike/core/internal/i18n"
)

// Waypoint is a named stop at a distance along the route.
type Waypoint struct {
	ID         string  `json:"id"`
	AtM        float64 `json:"at_m"`
	Kind       string  `json:"kind,omitempty"`        // map symbol, one of WaypointKinds
	ElevationM float64 `json:"elevation_m,omitempty"` // shown next to peaks and passes
	// Real position, for routes with a track: the map pins the stop exactly here.
	Lat *float64 `json:"lat,omitempty"`
	Lon *float64 `json:"lon,omitempty"`
}

// Biome sets the terrain drawn from AtM until the next biome starts.
type Biome struct {
	AtM  float64 `json:"at_m"`
	Type string  `json:"type"` // one of BiomeTypes
}

// WaypointKinds are the map symbols front ends know how to draw.
var WaypointKinds = []string{
	"start", "finish", "peak", "pass", "lake", "river", "bridge", "hut", "village", "landmark",
	"castle", "tower", "ruin", "cave", "spring", "viewpoint", "camp", "volcano", "lighthouse", "harbor",
}

// BiomeTypes are the kinds of terrain front ends know how to draw. Front ends
// blend neighbouring biomes smoothly, so a route can go from forest to
// desert without a hard edge.
var BiomeTypes = []string{
	"forest",   // conifers
	"grove",    // deciduous woods
	"meadow",   // grass, flowers
	"fields",   // farmland
	"steppe",   // dry grassland
	"desert",   // dunes, sand
	"rock",     // scree, crags
	"snow",     // glaciers, snowfields
	"tundra",   // alpine tussock, lichen
	"water",    // lakes
	"swamp",    // reeds, bogs
	"coast",    // beach and sea
	"volcanic", // ash, lava rock, steam
	"village",  // houses and fences
}

// IDPattern is what route ids may look like: they become folder names on
// every OS, so only lowercase letters, digits and single dashes.
var IDPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// itemIDPattern is for waypoint, fact, object and story ids: they become
// translation keys.
var itemIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// AssetExts are the file types an asset may have. HTML assets are static:
// front ends strip scripts and event handlers before showing them.
var AssetExts = map[string]string{
	".svg": "image/svg+xml", ".png": "image/png", ".jpg": "image/jpeg", ".jpeg": "image/jpeg",
	".webp": "image/webp", ".gif": "image/gif", ".html": "text/html",
}

// Span is a stretch of the route, e.g. underground: from FromM to ToM.
type Span struct {
	FromM float64 `json:"from_m"`
	ToM   float64 `json:"to_m"`
}

// Danger is an encounter or threat at AtM (a pursuer on the road, a storm);
// its text is locales/<lang>.json dangers.<id>. Front ends darken the scene
// around it and mark it once passed.
type Danger struct {
	ID  string  `json:"id"`
	AtM float64 `json:"at_m"`
}

// ObjectLayers say where an object stands in the side view.
var ObjectLayers = []string{"trail", "far"}

// ProfilePoint is one known elevation along the route.
type ProfilePoint struct {
	AtM        float64 `json:"at_m"`
	ElevationM float64 `json:"elevation_m"`
}

// Fact is a short piece of real-world or in-world knowledge tied to a
// position; its text is locales/<lang>.json facts.<id>.
type Fact struct {
	ID  string  `json:"id"`
	AtM float64 `json:"at_m"`
}

// Object is a picture (SVG, PNG, JPG, WebP, GIF) or a static HTML snippet that
// stands at AtM. Front ends fade it in as the walker approaches and out once
// they have passed it. An optional caption is objects.<id> in the locales.
type Object struct {
	ID      string  `json:"id"`
	AtM     float64 `json:"at_m"`
	Asset   string  `json:"asset"`               // path inside the pack, under assets/
	HeightP float64 `json:"height_px,omitempty"` // drawn height in the side view (default 90)
	LiftP   float64 `json:"lift_px,omitempty"`   // raise above the ground line (e.g. birds, signs)
	OffsetM float64 `json:"offset_m,omitempty"`  // shift along the trail without moving the fade centre
	FadeM   float64 `json:"fade_m,omitempty"`    // distance at which it is fully faded out (default 350)
	Layer   string  `json:"layer,omitempty"`     // trail (default) | far
}

// StoryBeat is a piece of narration shown when the walker passes AtM.
type StoryBeat struct {
	ID  string  `json:"id"`
	AtM float64 `json:"at_m"`
}

// Route is a loaded, validated route pack.
type Route struct {
	ID            string             `json:"id"`
	Version       int                `json:"version"`
	DefaultLocale string             `json:"default_locale"`
	LengthM       float64            `json:"length_m"`
	Path          [][2]float64       `json:"path,omitempty"`      // optional hand-drawn shape, points in 0..1
	Track         [][2]float64       `json:"track,omitempty"`     // optional real trail: [latitude, longitude] points in walking order
	MapImage      string             `json:"map_image,omitempty"` // optional asset drawn under the path on the map
	Waypoints     []Waypoint         `json:"waypoints"`
	Profile       []ProfilePoint     `json:"profile,omitempty"` // extra elevations between waypoints
	Biomes        []Biome            `json:"biomes,omitempty"`
	Story         []StoryBeat        `json:"story,omitempty"`
	Facts         []Fact             `json:"facts,omitempty"`
	Objects       []Object           `json:"objects,omitempty"`
	Underground   []Span             `json:"underground,omitempty"` // tunnels, mines, caves walked through
	Dangers       []Danger           `json:"dangers,omitempty"`
	Achievements  []achievements.Def `json:"achievements,omitempty"`

	Texts   i18n.Bundle `json:"-"`
	Builtin bool        `json:"-"`

	fsys fs.FS  // where the pack was loaded from, for assets
	dir  string // the pack folder inside fsys
}

// T returns route text for key, following the locale chain and ending with
// the route's own default locale.
func (r *Route) T(chain []string, key string) string {
	// Full slice expression: never write into the caller's backing array.
	return r.Texts.Text(append(chain[:len(chain):len(chain)], r.DefaultLocale), key)
}

// Tmaybe is T for optional texts (T already returns "" when missing; the
// name documents intent at call sites).
func (r *Route) Tmaybe(chain []string, key string) string { return r.T(chain, key) }

// UndergroundAt reports whether m lies in an underground stretch.
func (r *Route) UndergroundAt(m float64) bool {
	for _, u := range r.Underground {
		if m >= u.FromM && m <= u.ToM {
			return true
		}
	}
	return false
}

// MaxElevation is the highest point between 0 and m (0 without heights).
func (r *Route) MaxElevation(m float64) float64 {
	best, any := 0.0, false
	for _, p := range r.Elevations() {
		if p.AtM <= m && (!any || p.ElevationM > best) {
			best, any = p.ElevationM, true
		}
	}
	if e, ok := r.ElevationAt(m); ok && (!any || e > best) {
		best = e
	}
	return best
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

// Elevations merges waypoint heights and profile points into one sorted
// profile. Empty when the route knows no heights at all.
func (r *Route) Elevations() []ProfilePoint {
	var out []ProfilePoint
	for _, w := range r.Waypoints {
		if w.ElevationM != 0 {
			out = append(out, ProfilePoint{AtM: w.AtM, ElevationM: w.ElevationM})
		}
	}
	out = append(out, r.Profile...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].AtM < out[j].AtM })
	// Two heights at the same spot: the later one (profile) wins.
	dedup := out[:0]
	for _, p := range out {
		if n := len(dedup); n > 0 && dedup[n-1].AtM == p.AtM {
			dedup[n-1] = p
			continue
		}
		dedup = append(dedup, p)
	}
	return dedup
}

// ElevationAt interpolates the height at m linearly between known points.
// ok is false when the route has no heights.
func (r *Route) ElevationAt(m float64) (e float64, ok bool) {
	pts := r.Elevations()
	if len(pts) == 0 {
		return 0, false
	}
	if m <= pts[0].AtM {
		return pts[0].ElevationM, true
	}
	for i := 1; i < len(pts); i++ {
		a, b := pts[i-1], pts[i]
		if m <= b.AtM {
			k := (m - a.AtM) / math.Max(b.AtM-a.AtM, 1e-9)
			return a.ElevationM + (b.ElevationM-a.ElevationM)*k, true
		}
	}
	return pts[len(pts)-1].ElevationM, true
}

// Ascent is the total climb (sum of positive height changes) between 0 and m.
func (r *Route) Ascent(m float64) float64 {
	pts := r.Elevations()
	up := 0.0
	for i := 1; i < len(pts); i++ {
		a, b := pts[i-1], pts[i]
		if a.AtM >= m {
			break
		}
		eb := b.ElevationM
		if b.AtM > m { // partial segment
			eb, _ = r.ElevationAt(m)
		}
		if eb > a.ElevationM {
			up += eb - a.ElevationM
		}
	}
	return up
}

// Asset reads a file of the pack. name must be one of the pack's assets.
func (r *Route) Asset(name string) ([]byte, error) {
	if r.fsys == nil || !validAssetPath(name) {
		return nil, fs.ErrNotExist
	}
	return fs.ReadFile(r.fsys, path.Join(r.dir, name))
}

// AssetNames lists the assets the route refers to, without duplicates.
func (r *Route) AssetNames() []string {
	seen := map[string]bool{}
	var out []string
	add := func(a string) {
		if a != "" && !seen[a] {
			seen[a] = true
			out = append(out, a)
		}
	}
	add(r.MapImage)
	for _, o := range r.Objects {
		add(o.Asset)
	}
	return out
}

func validAssetPath(p string) bool {
	if !strings.HasPrefix(p, "assets/") || !fs.ValidPath(p) {
		return false
	}
	_, ok := AssetExts[strings.ToLower(path.Ext(p))]
	return ok
}

// LoadPack loads and validates a single route pack in dir.
func LoadPack(fsys fs.FS, dir string) (*Route, error) { return loadOne(fsys, dir) }

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
	r.fsys, r.dir = fsys, dir

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
	for _, f := range r.Facts {
		keys = append(keys, "facts."+f.ID)
	}
	for _, d := range r.Dangers {
		keys = append(keys, "dangers."+d.ID)
	}
	for _, a := range r.Achievements {
		keys = append(keys, "achievements."+a.ID+".name", "achievements."+a.ID+".description")
	}
	return keys
}

// Validate checks structure and the default-locale texts.
func (r *Route) Validate() error {
	if !IDPattern.MatchString(r.ID) {
		return errors.New("id is required and may contain only lowercase letters, digits and single dashes, e.g. my-trail")
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
	for _, w := range r.Waypoints {
		if w.Kind != "" && !slices.Contains(WaypointKinds, w.Kind) {
			return fmt.Errorf("waypoint %q: unknown kind %q (use one of %s)", w.ID, w.Kind, strings.Join(WaypointKinds, ", "))
		}
	}
	for _, b := range r.Biomes {
		if !slices.Contains(BiomeTypes, b.Type) {
			return fmt.Errorf("biome at %v m: unknown type %q (use one of %s)", b.AtM, b.Type, strings.Join(BiomeTypes, ", "))
		}
		if b.AtM < 0 || b.AtM > r.LengthM {
			return fmt.Errorf("biome at %v m is outside the route", b.AtM)
		}
	}
	for _, s := range r.Story {
		if s.ID == "" || s.AtM < 0 || s.AtM > r.LengthM {
			return fmt.Errorf("story beat %q is invalid", s.ID)
		}
	}
	if err := r.validateExtras(); err != nil {
		return err
	}
	for _, p := range r.Path {
		if p[0] < 0 || p[0] > 1 || p[1] < 0 || p[1] > 1 {
			return errors.New("path points must be within 0..1")
		}
	}
	if len(r.Path) == 1 {
		return errors.New("path needs at least two points")
	}
	if len(r.Track) == 1 {
		return errors.New("track needs at least two points")
	}
	for i, p := range r.Track {
		if p[0] < -90 || p[0] > 90 || p[1] < -180 || p[1] > 180 {
			return fmt.Errorf("track point %d is not [latitude, longitude]: %v", i, p)
		}
	}
	for _, w := range r.Waypoints {
		if (w.Lat == nil) != (w.Lon == nil) {
			return fmt.Errorf("waypoint %q needs both lat and lon", w.ID)
		}
		if w.Lat != nil && len(r.Track) == 0 {
			return fmt.Errorf("waypoint %q has coordinates but the route has no track", w.ID)
		}
		if w.Lat != nil && (*w.Lat < -90 || *w.Lat > 90 || *w.Lon < -180 || *w.Lon > 180) {
			return fmt.Errorf("waypoint %q: lat/lon out of range", w.ID)
		}
	}
	if len(r.Track) > 0 && len(r.Path) > 0 {
		return errors.New("use either track (real coordinates) or path (a drawn shape), not both")
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
	sort.Slice(r.Biomes, func(i, j int) bool { return r.Biomes[i].AtM < r.Biomes[j].AtM })
	sort.Slice(r.Facts, func(i, j int) bool { return r.Facts[i].AtM < r.Facts[j].AtM })
	sort.Slice(r.Dangers, func(i, j int) bool { return r.Dangers[i].AtM < r.Dangers[j].AtM })
	sort.Slice(r.Objects, func(i, j int) bool { return r.Objects[i].AtM < r.Objects[j].AtM })
	sort.Slice(r.Profile, func(i, j int) bool { return r.Profile[i].AtM < r.Profile[j].AtM })
	return nil
}

// validateExtras checks the profile, facts, objects and assets.
func (r *Route) validateExtras() error {
	for _, w := range r.Waypoints {
		if !itemIDPattern.MatchString(w.ID) {
			return fmt.Errorf("waypoint id %q may contain only letters, digits, - and _", w.ID)
		}
	}
	for _, p := range r.Profile {
		if p.AtM < 0 || p.AtM > r.LengthM {
			return fmt.Errorf("profile point at %v m is outside the route", p.AtM)
		}
		if p.ElevationM < -500 || p.ElevationM > 9000 {
			return fmt.Errorf("profile point at %v m: elevation %v m is not on this planet", p.AtM, p.ElevationM)
		}
	}
	seen := map[string]bool{}
	for _, f := range r.Facts {
		if !itemIDPattern.MatchString(f.ID) || seen[f.ID] {
			return fmt.Errorf("fact id %q is empty, duplicated or has characters other than letters, digits, - and _", f.ID)
		}
		seen[f.ID] = true
		if f.AtM < 0 || f.AtM > r.LengthM {
			return fmt.Errorf("fact %q is outside the route", f.ID)
		}
	}
	seen = map[string]bool{}
	for _, o := range r.Objects {
		if !itemIDPattern.MatchString(o.ID) || seen[o.ID] {
			return fmt.Errorf("object id %q is empty, duplicated or has characters other than letters, digits, - and _", o.ID)
		}
		seen[o.ID] = true
		if o.AtM < 0 || o.AtM > r.LengthM {
			return fmt.Errorf("object %q is outside the route", o.ID)
		}
		if o.HeightP < 0 || o.HeightP > 400 || o.FadeM < 0 || o.LiftP < -100 || o.LiftP > 400 {
			return fmt.Errorf("object %q: height_px must be 0..400, lift_px -100..400 and fade_m positive", o.ID)
		}
		if o.Layer != "" && !slices.Contains(ObjectLayers, o.Layer) {
			return fmt.Errorf("object %q: unknown layer %q (use one of %s)", o.ID, o.Layer, strings.Join(ObjectLayers, ", "))
		}
		if o.Asset == "" {
			return fmt.Errorf("object %q needs an asset", o.ID)
		}
	}
	seen = map[string]bool{}
	for _, d := range r.Dangers {
		if !itemIDPattern.MatchString(d.ID) || seen[d.ID] {
			return fmt.Errorf("danger id %q is empty, duplicated or has characters other than letters, digits, - and _", d.ID)
		}
		seen[d.ID] = true
		if d.AtM < 0 || d.AtM > r.LengthM {
			return fmt.Errorf("danger %q is outside the route", d.ID)
		}
	}
	sort.Slice(r.Underground, func(i, j int) bool { return r.Underground[i].FromM < r.Underground[j].FromM })
	for i, u := range r.Underground {
		if u.FromM < 0 || u.ToM > r.LengthM || u.ToM <= u.FromM {
			return fmt.Errorf("underground stretch %v–%v m must lie inside the route and end after it starts", u.FromM, u.ToM)
		}
		if i > 0 && u.FromM < r.Underground[i-1].ToM {
			return fmt.Errorf("underground stretches overlap at %v m", u.FromM)
		}
	}
	if r.MapImage != "" && len(r.Path) < 2 {
		return errors.New("map_image needs a path: the trail is drawn over the image along path")
	}
	for _, a := range r.AssetNames() {
		if !validAssetPath(a) {
			return fmt.Errorf("asset %q must be a file under assets/ with one of: svg, png, jpg, webp, gif, html", a)
		}
		if r.fsys != nil {
			if _, err := fs.Stat(r.fsys, path.Join(r.dir, a)); err != nil {
				return fmt.Errorf("asset %q is missing from the pack", a)
			}
		}
	}
	return nil
}
