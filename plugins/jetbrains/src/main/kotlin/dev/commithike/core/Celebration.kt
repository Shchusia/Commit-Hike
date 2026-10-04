package dev.commithike.core

/**
 * What to tell the user after a scan: the distance flash in the status bar,
 * whether to ask for a rating, and the notifications to show. Decided here,
 * from the scan and the user's choice of notifications; the plugin only shows it.
 */
data class Celebration(
    val flash: String?, // "+86 m" for the status bar, or null
    val askForRating: Boolean,
    val rewritten: Boolean, // history was rewritten: worth a line in the log
    val notes: List<Note>,
) {
    enum class Action { SHOW_TRAIL, CHOOSE_GLOBAL_ROUTE, CHOOSE_PROJECT_ROUTE, WALK_ROUTE }

    /** [routeId]: the route to walk on to, for [Action.WALK_ROUTE]. */
    data class Note(val title: String, val text: String, val button: String, val action: Action, val routeId: String? = null)

    companion object {
        /** [notifications]: all | milestones (stops, achievements, the finish) | off. */
        fun plan(res: ScanResult, notifications: String): Celebration {
            val moved = res.addedM > 0
            val flash = if (moved && notifications == "all") "+" + I18n.distance(res.addedM) else null
            val base = Celebration(flash, moved, res.rewritten && res.removedCommits > 0, emptyList())
            if (notifications == "off") return base
            val notes = mutableListOf<Note>()
            val events = res.events.orEmpty()
            val show = I18n.t("showTrail")
            // An encounter on the road is announced when it's the only news of this commit.
            events.lastOrNull { it.type == "danger" }?.danger?.let { d ->
                if (events.size <= 3 && notifications == "all") notes += Note("⚠", d.text, show, Action.SHOW_TRAIL)
            }
            // Achievements always get a notification of their own.
            for (e in events.filter { it.type == "achievement" }) {
                val a = e.achievement ?: continue
                notes += Note("🏅 ${a.name ?: ""}", a.description ?: "", show, Action.SHOW_TRAIL)
            }
            // Importing history can pass many stops at once: announce only the latest per journey.
            for (scope in Scope.entries) {
                val mine = events.filter { it.journey == scope.cli }
                val journey = if (scope == Scope.GLOBAL) res.global else res.project
                if (journey != null && mine.any { it.type == "finished" }) {
                    val next = journey.nextRoute
                    notes += if (next != null) {
                        // a series goes on: offer the next route
                        Note(
                            I18n.t("finished", journey.route.name),
                            I18n.t("seriesNextText", next.seriesName, next.name),
                            I18n.t("walkOn", next.name),
                            Action.WALK_ROUTE,
                            next.id,
                        )
                    } else {
                        val choose = if (scope == Scope.GLOBAL) Action.CHOOSE_GLOBAL_ROUTE else Action.CHOOSE_PROJECT_ROUTE
                        Note(I18n.t("finished", journey.route.name), I18n.t("finishedText"), I18n.t("chooseTrail"), choose)
                    }
                } else {
                    val w = mine.lastOrNull { it.type == "waypoint" }?.waypoint ?: continue
                    notes += Note(w.name, w.text ?: I18n.t("reached", w.name), show, Action.SHOW_TRAIL)
                }
            }
            return base.copy(notes = notes)
        }
    }
}
