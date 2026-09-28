package dev.commithike.core

import dev.commithike.core.RatingPrompt.DAY_MS
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class RatingPromptTest {
    private val t0 = 1_788_264_000_000L // 2026-09-01 12:00 UTC

    private fun day(n: Int) = t0 + n * DAY_MS

    private fun activeUser(): RatingState {
        var s = RatingPrompt.initial(t0)
        for (n in listOf(1, 3, 5)) s = RatingPrompt.recordProgress(s, day(n))
        return s
    }

    @Test
    fun asksOnlyRealUsersAfterAWeek() {
        val once = RatingPrompt.recordProgress(RatingPrompt.recordProgress(RatingPrompt.initial(t0), day(1)), day(1))
        assertEquals(1, once.activeDays.size)
        assertFalse(RatingPrompt.shouldAsk(activeUser(), day(6)))
        assertTrue(RatingPrompt.shouldAsk(activeUser(), day(7)))
        val twoDays = RatingPrompt.recordProgress(RatingPrompt.recordProgress(RatingPrompt.initial(t0), day(1)), day(2))
        assertFalse(RatingPrompt.shouldAsk(twoDays, day(30)))
    }

    @Test
    fun ratedOrNeverMeansNeverAgain() {
        for (a in listOf("rate", "never")) {
            val s = RatingPrompt.answered(RatingPrompt.asked(activeUser(), day(7)), a)
            assertFalse(a, RatingPrompt.shouldAsk(s, day(60)))
        }
    }

    @Test
    fun laterOrDismissedAsksAgainAtMostThreeTimes() {
        var s = RatingPrompt.asked(activeUser(), day(7))
        assertFalse(RatingPrompt.shouldAsk(s, day(10)))
        assertTrue(RatingPrompt.shouldAsk(s, day(14)))
        s = RatingPrompt.answered(RatingPrompt.asked(s, day(14)), "later")
        s = RatingPrompt.asked(s, day(21))
        assertFalse(RatingPrompt.shouldAsk(s, day(60)))
    }

    @Test
    fun storedStateRoundTripsAndDamagedStateStartsOver() {
        val s = RatingPrompt.asked(activeUser(), day(7))
        assertEquals(s, RatingPrompt.load(RatingPrompt.save(s), day(8)))
        assertEquals(RatingPrompt.initial(t0), RatingPrompt.load(null, t0))
        assertEquals(RatingPrompt.initial(t0), RatingPrompt.load("{not json", t0))
        assertEquals("pending", RatingPrompt.load("""{"firstSeen":1,"status":"weird"}""", t0).status)
    }
}
