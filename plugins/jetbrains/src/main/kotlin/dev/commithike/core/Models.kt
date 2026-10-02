// Data returned by the commit-hike core, protocol api 1
// (mirrors core/internal/protocol). Field names map to the core's snake_case
// JSON through Gson's LOWER_CASE_WITH_UNDERSCORES policy; unknown fields are
// ignored, so newer cores with extra fields keep working.
//
// Every property has a default so Kotlin generates a no-arg constructor and
// Gson keeps the defaults. Lists the core may omit are nullable.
package dev.commithike.core

import com.google.gson.JsonElement

const val API_VERSION = 1

data class Waypoint(
    val id: String = "",
    val name: String = "",
    val text: String? = null,
    val atM: Double = 0.0,
    val kind: String? = null, // map symbol: peak, lake, bridge, hut…
    val elevationM: Double = 0.0,
    val lat: Double? = null, // real position, on routes with a GPS track
    val lon: Double? = null,
    val x: Double? = null, // position on a drawn map (0..1), on routes with a path
    val y: Double? = null,
)

data class Biome(val atM: Double = 0.0, val type: String = "")

data class Day(val date: String = "", val m: Double = 0.0)

data class Story(val id: String = "", val text: String = "", val atM: Double = 0.0)

data class Achievement(
    val id: String = "",
    val name: String? = null, // null while a hidden achievement is locked
    val description: String? = null,
    val hidden: Boolean = false,
    val unlockedAt: Long = 0,
)

data class Span(val fromM: Double = 0.0, val toM: Double = 0.0)

data class Danger(val id: String = "", val atM: Double = 0.0, val text: String = "")

data class RestDaysInfo(val days: List<Int> = emptyList())

data class BadgeResult(val svg: String = "", val fileName: String = "", val markdown: String = "")

/** Panel and notification choices, shared by every IDE on this computer. */
data class Settings(
    val reduceMotion: String = "auto", // auto | on | off
    val highContrast: String = "auto", // auto | on | off
    val notifications: String = "all", // all | milestones | off
)

data class BackupResult(
    val path: String = "",
    val createdAt: Long = 0,
    val commits: Int = 0,
    val routes: Int = 0,
    val avatar: Boolean = false,
)

data class DifficultyInfo(val level: String = "medium", val levels: List<String>? = null, val typicalDayM: Map<String, Double>? = null)

data class ProfilePoint(val atM: Double = 0.0, val elevationM: Double = 0.0)

data class Fact(val id: String = "", val atM: Double = 0.0, val text: String = "")

/** A picture or static HTML standing on the route; the file itself comes from [RouteAssets]. */
data class RouteObject(
    val id: String = "",
    val atM: Double = 0.0,
    val asset: String = "",
    val heightPx: Double = 90.0,
    val liftPx: Double = 0.0,
    val offsetM: Double = 0.0,
    val fadeM: Double = 350.0,
    val layer: String = "trail",
    val caption: String? = null,
)

data class Route(
    val id: String = "",
    val name: String = "",
    val description: String = "",
    val lengthM: Double = 0.0,
    val locales: List<String>? = null,
    val builtin: Boolean = false,
    val waypoints: List<Waypoint>? = null,
    val biomes: List<Biome>? = null,
    val achievements: Int = 0,
    val profile: List<ProfilePoint>? = null,
    val ascentM: Double = 0.0,
    val minElevationM: Double = 0.0,
    val maxElevationM: Double = 0.0,
    val facts: List<Fact>? = null,
    val objects: List<RouteObject>? = null,
    val path: List<List<Double>>? = null,
    val track: List<List<Double>>? = null, // real trail, [latitude, longitude] points
    val loop: Boolean = false, // round trip: the finish is back at the start
    val mapImage: String? = null,
    val underground: List<Span>? = null,
    val dangers: List<Danger>? = null,
)

data class Journey(
    val scope: String = "",
    val route: Route = Route(),
    val distanceM: Double = 0.0,
    val percent: Double = 0.0,
    val finished: Boolean = false,
    val lastWaypoint: Waypoint? = null,
    val nextWaypoint: Waypoint? = null,
    val toNextM: Double? = null,
    val story: Story? = null,
    val achievements: List<Achievement>? = null,
    val commits: Int = 0,
    val streakDays: Int = 0,
    val daily: List<Day>? = null,
    val day: Int = 1,
    val elevationM: Double? = null, // absent when the route has no heights
    val ascentM: Double = 0.0,
    val maxElevationM: Double = 0.0,
    val toNextClimbM: Double = 0.0,
    val underground: Boolean = false,
)

data class Status(
    val tracked: Boolean = false,
    val reason: String? = null,
    val reasonCode: String? = null, // not_a_repo | not_enabled
    val team: Boolean = false, // teammates shown for this project
    val difficulty: String? = null, // easy | medium | hard
    val typicalDayM: Double = 0.0, // a typical day of commits at that level
    val locale: String = "en",
    val global: Journey? = null,
    val project: Journey? = null,
    val todayM: Double = 0.0,
    val totalM: Double = 0.0,
    // The core's answer exactly as it came, for the trail panel. Transient: Gson
    // neither fills nor writes it, so it never goes out of sync with the fields above.
    val settings: Settings? = null, // panel and notification choices
    @Transient val raw: JsonElement? = null,
)

data class JourneyEvent(
    val type: String = "", // waypoint | story | achievement | finished | fact | danger
    val journey: String = "", // global | project
    val waypoint: Waypoint? = null,
    val story: Story? = null,
    val achievement: Achievement? = null,
    val fact: Fact? = null,
    val danger: Danger? = null,
)

/** `scan` returns the Status fields inline plus scan details. */
data class ScanResult(
    val tracked: Boolean = false,
    val reason: String? = null,
    val reasonCode: String? = null,
    val team: Boolean = false,
    val difficulty: String? = null,
    val typicalDayM: Double = 0.0,
    val locale: String = "en",
    val global: Journey? = null,
    val project: Journey? = null,
    val todayM: Double = 0.0,
    val totalM: Double = 0.0,
    val newCommits: Int = 0,
    val updatedCommits: Int = 0,
    val removedCommits: Int = 0,
    val rewritten: Boolean = false, // history was rewritten and recounted in full
    val addedM: Double = 0.0,
    val events: List<JourneyEvent>? = null,
) {
    fun status() = Status(tracked, reason, reasonCode, team, difficulty, typicalDayM, locale, global, project, todayM, totalM)
}

/** The hiker icon: a custom PNG as a data URL, or the panel's default when not custom. */
data class Avatar(val custom: Boolean = false, val dataUrl: String? = null)

data class VerifyResult(val added: Int = 0, val updated: Int = 0, val removed: Int = 0)

data class CoreConfig(val mode: String = "all", val emails: List<String>? = null, val locale: String? = null)

data class Member(
    val id: String = "", // stable per machine, for colors; not an email
    val name: String = "",
    val me: Boolean = false,
    val distanceM: Double = 0.0,
    val percent: Double = 0.0,
    val finished: Boolean = false,
    val commits: Int = 0,
    val todayM: Double = 0.0,
    val lastCommitAt: Long = 0,
    val elevationM: Double? = null,
)

data class Team(
    val scope: String = "",
    val routeId: String = "",
    val lengthM: Double = 0.0,
    val members: List<Member>? = null,
    val hidden: Int = 0,
)

data class LocaleInfo(val locale: String = "", val effective: String = "en", val available: List<String>? = null)

data class RouteAssets(val id: String = "", val images: Map<String, String>? = null, val html: Map<String, String>? = null)

enum class Scope(val cli: String) { GLOBAL("global"), PROJECT("project") }
