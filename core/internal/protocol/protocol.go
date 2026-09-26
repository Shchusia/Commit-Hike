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

import "github.com/commit-hike/commit-hike/core/internal/render"

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
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	Text       string  `json:"text,omitempty"`
	AtM        float64 `json:"at_m"`
	Kind       string  `json:"kind,omitempty"` // map symbol: peak, lake, bridge, hut…
	ElevationM float64 `json:"elevation_m,omitempty"`
}

// Biome is the terrain from AtM until the next biome.
type Biome struct {
	AtM  float64 `json:"at_m"`
	Type string  `json:"type"` // forest, meadow, rock, snow, water, village
}

// Day is the distance walked on one day (UTC).
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
}

// Avatar is the hiker icon: a custom PNG as a data URL, or empty for the default.
type Avatar struct {
	Custom  bool   `json:"custom"`
	DataURL string `json:"data_url,omitempty"`
}

// Status is the full picture for the current project.
type Status struct {
	Tracked bool     `json:"tracked"`
	Reason  string   `json:"reason,omitempty"`
	Locale  string   `json:"locale"` // language actually used for texts
	Global  *Journey `json:"global,omitempty"`
	Project *Journey `json:"project,omitempty"`
	TodayM  float64  `json:"today_m"`
	TotalM  float64  `json:"total_m"`
}

// Event types reported by scan.
const (
	EventWaypoint    = "waypoint"
	EventStory       = "story"
	EventAchievement = "achievement"
	EventFinished    = "finished"
)

// Event is something that happened during a scan, for notifications.
type Event struct {
	Type        string       `json:"type"`
	Journey     string       `json:"journey"` // "global" | "project"
	Waypoint    *Waypoint    `json:"waypoint,omitempty"`
	Story       *Story       `json:"story,omitempty"`
	Achievement *Achievement `json:"achievement,omitempty"`
}

// ScanResult is Status plus what the scan changed.
type ScanResult struct {
	Status
	NewCommits     int     `json:"new_commits"`
	UpdatedCommits int     `json:"updated_commits"`
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
