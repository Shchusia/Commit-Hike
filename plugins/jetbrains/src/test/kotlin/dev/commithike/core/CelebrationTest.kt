package dev.commithike.core

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class CelebrationTest {
    private val route = Route(id = "r", name = "Ridge", lengthM = 24000.0)
    private val journey = Journey(route = route)

    private fun scan(vararg events: JourneyEvent, added: Double = 120.0) =
        ScanResult(global = journey, project = journey, addedM = added, events = events.toList())

    private fun waypoint(name: String, scope: String = "global") =
        JourneyEvent(type = "waypoint", journey = scope, waypoint = Waypoint(id = name, name = name))

    @Test
    fun everythingWithAll() {
        val p = Celebration.plan(
            scan(
                JourneyEvent(type = "danger", journey = "global", danger = Danger(text = "A bear")),
                JourneyEvent(type = "achievement", achievement = Achievement(id = "a", name = "First", description = "d")),
                waypoint("Lake"),
            ),
            "all",
        )
        assertTrue(p.flash!!.startsWith("+"))
        assertTrue(p.askForRating)
        assertEquals(listOf("⚠", "🏅 First", "Lake"), p.notes.map { it.title })
    }

    @Test
    fun milestonesDropTheDistanceAndEncounters() {
        val p = Celebration.plan(
            scan(JourneyEvent(type = "danger", journey = "global", danger = Danger(text = "A bear")), waypoint("Lake")),
            "milestones",
        )
        assertNull(p.flash)
        assertEquals(listOf("Lake"), p.notes.map { it.title })
    }

    @Test
    fun offShowsNothingButStillCounts() {
        val p = Celebration.plan(scan(waypoint("Lake")), "off")
        assertNull(p.flash)
        assertTrue(p.notes.isEmpty())
        assertTrue(p.askForRating)
    }

    @Test
    fun onlyTheLatestStopPerJourneyAndTheFinishOffersANewTrail() {
        val p = Celebration.plan(
            scan(waypoint("A"), waypoint("B"), waypoint("C", "project"), JourneyEvent(type = "finished", journey = "project")),
            "all",
        )
        assertEquals(listOf("B", Celebration.Action.CHOOSE_PROJECT_ROUTE), listOf(p.notes[0].title, p.notes[1].action))
        assertEquals(2, p.notes.size)
    }

    @Test
    fun anEncounterIsShownOnlyWhenItIsTheMainNews() {
        val many = Array(4) { waypoint("W$it") } + JourneyEvent(type = "danger", journey = "global", danger = Danger(text = "A bear"))
        assertFalse(Celebration.plan(scan(*many), "all").notes.any { it.title == "⚠" })
        assertNull(Celebration.plan(scan(added = 0.0), "all").flash)
        assertFalse(Celebration.plan(scan(added = 0.0), "all").askForRating)
    }

    @Test
    fun aFinishedRouteOfASeriesOffersTheNextOne() {
        val next = NextRoute(id = "svydovets-ridge", name = "Svydovets Ridge", seriesName = "The Carpathians")
        val res = ScanResult(global = journey.copy(nextRoute = next), events = listOf(JourneyEvent(type = "finished", journey = "global")))
        val note = Celebration.plan(res, "milestones").notes.single()
        assertEquals(Celebration.Action.WALK_ROUTE, note.action)
        assertEquals("svydovets-ridge", note.routeId)
        assertTrue(note.text.contains("The Carpathians") && note.button.contains("Svydovets Ridge"))
    }
}
