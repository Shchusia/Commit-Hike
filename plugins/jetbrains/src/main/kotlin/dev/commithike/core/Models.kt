// Data returned by the commit-hike core, protocol api 1
// (mirrors core/internal/protocol). Field names map to the core's snake_case
// JSON through Gson's LOWER_CASE_WITH_UNDERSCORES policy; unknown fields are
// ignored, so newer cores with extra fields keep working.
//
// Every property has a default so Kotlin generates a no-arg constructor and
// Gson keeps the defaults. Lists the core may omit are nullable.
package dev.commithike.core

const val API_VERSION = 1

data class Waypoint(val id: String = "", val name: String = "", val text: String? = null, val atM: Double = 0.0)

data class Story(val id: String = "", val text: String = "", val atM: Double = 0.0)

data class Achievement(
    val id: String = "",
    val name: String? = null, // null while a hidden achievement is locked
    val description: String? = null,
    val hidden: Boolean = false,
    val unlockedAt: Long = 0,
)

data class Route(
    val id: String = "",
    val name: String = "",
    val description: String = "",
    val lengthM: Double = 0.0,
    val locales: List<String>? = null,
    val builtin: Boolean = false,
    val waypoints: List<Waypoint>? = null,
    val achievements: Int = 0,
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
)

data class Status(
    val tracked: Boolean = false,
    val reason: String? = null,
    val locale: String = "en",
    val global: Journey? = null,
    val project: Journey? = null,
    val todayM: Double = 0.0,
    val totalM: Double = 0.0,
)

data class JourneyEvent(
    val type: String = "", // waypoint | story | achievement | finished
    val journey: String = "", // global | project
    val waypoint: Waypoint? = null,
    val story: Story? = null,
    val achievement: Achievement? = null,
)

/** `scan` returns the Status fields inline plus scan details. */
data class ScanResult(
    val tracked: Boolean = false,
    val reason: String? = null,
    val locale: String = "en",
    val global: Journey? = null,
    val project: Journey? = null,
    val todayM: Double = 0.0,
    val totalM: Double = 0.0,
    val newCommits: Int = 0,
    val updatedCommits: Int = 0,
    val addedM: Double = 0.0,
    val events: List<JourneyEvent>? = null,
) {
    fun status() = Status(tracked, reason, locale, global, project, todayM, totalM)
}

data class VerifyResult(val added: Int = 0, val updated: Int = 0, val removed: Int = 0)

data class CoreConfig(val mode: String = "all", val emails: List<String>? = null, val locale: String? = null)

enum class Scope(val cli: String) { GLOBAL("global"), PROJECT("project") }
