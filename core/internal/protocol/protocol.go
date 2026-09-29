// Package protocol defines the JSON contract between the core and IDE
// plugins. It is the only package plugins need to know about.
//
// Every response is one envelope:
//
//	{"api": 1, "ok": true,  "data": {...}}
//	{"api": 1, "ok": false, "error": {"code": "not_initialized", "message": "..."}}
//
// Rules for evolving it: adding fields is fine at any time; renaming or
// removing a field, or changing its meaning, requires bumping Version.
package protocol

import "github.com/Shchusia/commit-hike/core/internal/render"

// Version of the envelope and all types below.
const Version = 1

// Error codes plugins can rely on. Messages are English and for logs only:
// plugins show their own translated text per code.
const (
	CodeNotInitialized  = "not_initialized"
	CodeInvalidArgument = "invalid_argument"
	CodeUnknownRoute    = "unknown_route"
	CodeNotARepo        = "not_a_repo"
	CodeBusy            = "busy"
	CodeInvalidRoute    = "invalid_route" // an imported route pack is broken; message says why
	CodeRouteExists     = "route_exists"
	CodeRouteInUse      = "route_in_use"
	CodeInvalidImage    = "invalid_image" // a hiker icon that isn't a usable PNG
	CodeInternal        = "internal"
)

// Envelope wraps every response the core prints.
type Envelope struct {
	API   int    `json:"api"`
	OK    bool   `json:"ok"`
	Data  any    `json:"data,omitempty"`
	Error *Error `json:"error,omitempty"`
}

// Error describes a failed command.
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Config is the part of the user settings plugins may show.
type Config struct {
	Mode   string   `json:"mode"`
	Emails []string `json:"emails"`
	Locale string   `json:"locale,omitempty"`
}

// Waypoint is a named stop on a route, with translated texts.
type Waypoint struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Text       string   `json:"text,omitempty"`
	AtM        float64  `json:"at_m"`
	Kind       string   `json:"kind,omitempty"` // map symbol: peak, lake, bridge, hut…
	ElevationM float64  `json:"elevation_m,omitempty"`
	Lat        *float64 `json:"lat,omitempty"` // real position, on routes with a track
	Lon        *float64 `json:"lon,omitempty"`
	X          *float64 `json:"x,omitempty"` // position on the drawn map (0..1), on routes with a path
	Y          *float64 `json:"y,omitempty"`
}

// Biome is the terrain from AtM until the next biome.
type Biome struct {
	AtM  float64 `json:"at_m"`
	Type string  `json:"type"` // forest, grove, meadow, fields, steppe, desert, rock, snow, tundra, water, swamp, coast, volcanic, village
}

// ProfilePoint is a known elevation along a route.
type ProfilePoint struct {
	AtM        float64 `json:"at_m"`
	ElevationM float64 `json:"elevation_m"`
}

// Fact is a short piece of knowledge tied to a place on the route, translated.
type Fact struct {
	ID   string  `json:"id"`
	AtM  float64 `json:"at_m"`
	Text string  `json:"text"`
}

// Object is a picture or static HTML standing at AtM. Front ends fade it in
// as the walker approaches and out once they have passed it. The file itself
// comes from `route assets`.
type Object struct {
	ID      string  `json:"id"`
	AtM     float64 `json:"at_m"`
	Asset   string  `json:"asset"` // key into RouteAssets.Images or RouteAssets.HTML
	HeightP float64 `json:"height_px"`
	LiftP   float64 `json:"lift_px,omitempty"`
	OffsetM float64 `json:"offset_m,omitempty"`
	FadeM   float64 `json:"fade_m"`
	Layer   string  `json:"layer"` // trail | far
	Caption string  `json:"caption,omitempty"`
}

// RouteAssets holds a route's pictures as data URLs and its HTML snippets as
// text. HTML is static: front ends must strip scripts and event handlers.
type RouteAssets struct {
	ID     string            `json:"id"`
	Images map[string]string `json:"images"`
	HTML   map[string]string `json:"html"`
}

// Day is the distance walked on one day (the user's local calendar day).
type Day struct {
	Date string  `json:"date"` // YYYY-MM-DD
	M    float64 `json:"m"`
}

// Story is a piece of narration between waypoints.
type Story struct {
	ID   string  `json:"id"`
	Text string  `json:"text"`
	AtM  float64 `json:"at_m"`
}

// Achievement is a goal on a route, locked or unlocked.
type Achievement struct {
	ID          string `json:"id"`
	Name        string `json:"name,omitempty"`        // empty while a hidden one is locked
	Description string `json:"description,omitempty"` // same
	Hidden      bool   `json:"hidden,omitempty"`
	UnlockedAt  int64  `json:"unlocked_at,omitempty"` // unix seconds; 0 = locked
}

// Route describes a trail and its waypoints, translated.
type Route struct {
	ID           string     `json:"id"`
	Name         string     `json:"name"`
	Description  string     `json:"description"`
	LengthM      float64    `json:"length_m"`
	Locales      []string   `json:"locales"`
	Builtin      bool       `json:"builtin"`
	Waypoints    []Waypoint `json:"waypoints"`
	Biomes       []Biome    `json:"biomes,omitempty"`
	Achievements int        `json:"achievements"` // how many there are

	Profile       []ProfilePoint `json:"profile,omitempty"` // waypoint heights + extra points, sorted
	AscentM       float64        `json:"ascent_m,omitempty"`
	MinElevationM float64        `json:"min_elevation_m,omitempty"`
	MaxElevationM float64        `json:"max_elevation_m,omitempty"`
	Facts         []Fact         `json:"facts,omitempty"`
	Objects       []Object       `json:"objects,omitempty"`
	Path          [][2]float64   `json:"path,omitempty"`      // hand-drawn trail shape, points in 0..1
	Track         [][2]float64   `json:"track,omitempty"`     // real trail, [latitude, longitude] points; maps draw it to scale
	Loop          bool           `json:"loop,omitempty"`      // round trip: the finish is back at the start
	MapImage      string         `json:"map_image,omitempty"` // asset drawn under the path on the map
	Underground   []Span         `json:"underground,omitempty"`
	Dangers       []Danger       `json:"dangers,omitempty"`
}

// Span is a stretch of a route.
type Span struct {
	FromM float64 `json:"from_m"`
	ToM   float64 `json:"to_m"`
}

// Danger is an encounter or threat on the route, translated.
type Danger struct {
	ID   string  `json:"id"`
	AtM  float64 `json:"at_m"`
	Text string  `json:"text"`
}

// Journey is progress along one route: the global one or a project's own.
type Journey struct {
	Scope        string        `json:"scope"` // "global" | "project"
	Route        Route         `json:"route"`
	DistanceM    float64       `json:"distance_m"`
	Percent      float64       `json:"percent"`
	Finished     bool          `json:"finished"`
	LastWaypoint *Waypoint     `json:"last_waypoint,omitempty"`
	NextWaypoint *Waypoint     `json:"next_waypoint,omitempty"`
	ToNextM      float64       `json:"to_next_m,omitempty"`
	Story        *Story        `json:"story,omitempty"` // latest story beat passed
	Achievements []Achievement `json:"achievements"`
	Commits      int           `json:"commits"`
	StreakDays   int           `json:"streak_days"`
	Daily        []Day         `json:"daily"` // last 14 days, oldest first, today last
	Day          int           `json:"day"`   // day of the journey: 1 on the day it started

	ElevationM    *float64 `json:"elevation_m,omitempty"`     // where the walker stands; absent without a profile
	AscentM       float64  `json:"ascent_m,omitempty"`        // climbed so far on this journey
	MaxElevationM float64  `json:"max_elevation_m,omitempty"` // highest point reached so far
	ToNextClimbM  float64  `json:"to_next_climb_m,omitempty"` // climb left until the next stop
	Underground   bool     `json:"underground,omitempty"`     // the walker is in a tunnel right now
}

// Avatar is the hiker icon: a custom PNG as a data URL, or empty for the default.
type Avatar struct {
	Custom  bool   `json:"custom"`
	DataURL string `json:"data_url,omitempty"`
}

// Status is the full picture for the current project.
type Status struct {
	Tracked bool   `json:"tracked"`
	Reason  string `json:"reason,omitempty"` // English, for logs
	// ReasonCode says why a project isn't tracked: not_a_repo | not_enabled.
	ReasonCode string `json:"reason_code,omitempty"`
	// Team is true when the user turned on teammates for this project.
	Team bool `json:"team,omitempty"`
	// Difficulty is easy | medium | hard; TypicalDayM is how far a typical
	// day of commits (≈6 of ~60 lines) goes at that level.
	Difficulty  string   `json:"difficulty,omitempty"`
	TypicalDayM float64  `json:"typical_day_m,omitempty"`
	Locale      string   `json:"locale"` // language actually used for texts
	Global      *Journey `json:"global,omitempty"`
	Project     *Journey `json:"project,omitempty"`
	TodayM      float64  `json:"today_m"`
	TotalM      float64  `json:"total_m"`
}

// Event types reported by scan.
const (
	EventWaypoint    = "waypoint"
	EventStory       = "story"
	EventAchievement = "achievement"
	EventFinished    = "finished"
	EventFact        = "fact"
	EventDanger      = "danger"
)

// Reason codes for an untracked project.
const (
	ReasonNotARepo   = "not_a_repo"
	ReasonNotEnabled = "not_enabled"
)

// Event is something that happened during a scan, for notifications.
type Event struct {
	Type        string       `json:"type"`
	Journey     string       `json:"journey"` // "global" | "project"
	Waypoint    *Waypoint    `json:"waypoint,omitempty"`
	Story       *Story       `json:"story,omitempty"`
	Achievement *Achievement `json:"achievement,omitempty"`
	Fact        *Fact        `json:"fact,omitempty"`
	Danger      *Danger      `json:"danger,omitempty"`
}

// ScanResult is Status plus what the scan changed.
type ScanResult struct {
	Status
	NewCommits     int     `json:"new_commits"`
	UpdatedCommits int     `json:"updated_commits"`
	RemovedCommits int     `json:"removed_commits,omitempty"` // gone from history (squash, rebase, reset)
	Rewritten      bool    `json:"rewritten,omitempty"`       // history was rewritten and recounted in full
	AddedM         float64 `json:"added_m"`
	Events         []Event `json:"events,omitempty"`
}

// VerifyResult reports how a recount changed the stored data.
type VerifyResult struct {
	Added   int `json:"added"`
	Updated int `json:"updated"`
	Removed int `json:"removed"`
}

// Render holds a drawn journey: SVG markup or a scene to draw yourself.
type Render struct {
	Format string        `json:"format"` // "svg" | "scene"
	SVG    string        `json:"svg,omitempty"`
	Scene  *render.Scene `json:"scene,omitempty"`
}

// Member is one person on the team view of a project.
type Member struct {
	ID           string   `json:"id"` // stable per machine, for colors; not an email
	Name         string   `json:"name"`
	Me           bool     `json:"me,omitempty"`
	DistanceM    float64  `json:"distance_m"`
	Percent      float64  `json:"percent"`
	Finished     bool     `json:"finished,omitempty"`
	Commits      int      `json:"commits"`
	TodayM       float64  `json:"today_m"`
	LastCommitAt int64    `json:"last_commit_at"`
	ElevationM   *float64 `json:"elevation_m,omitempty"`
}

// Team places everyone who commits to a project on the same route. It is
// computed from git history on demand and never stored.
type Team struct {
	Scope   string   `json:"scope"` // the journey whose route is used: project, else global
	RouteID string   `json:"route_id"`
	LengthM float64  `json:"length_m"`
	Members []Member `json:"members"`          // furthest first
	Hidden  int      `json:"hidden,omitempty"` // people beyond the list limit
}

// LocaleInfo is the language setting.
type LocaleInfo struct {
	Locale    string   `json:"locale"`    // fixed language, "" = follow the IDE
	Effective string   `json:"effective"` // what texts are shown in right now
	Available []string `json:"available"` // languages route texts exist in
}

// DifficultyInfo is the difficulty setting.
type DifficultyInfo struct {
	Level       string             `json:"level"`
	Levels      []string           `json:"levels"`
	TypicalDayM map[string]float64 `json:"typical_day_m"` // per level
}
