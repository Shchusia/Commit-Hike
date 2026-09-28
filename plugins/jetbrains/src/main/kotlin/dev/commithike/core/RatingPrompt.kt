// When to ask for a rating. Pure logic, no IntelliJ APIs, so it's unit-tested.
//
// JetBrains Marketplace doesn't tell a plugin whether someone left a review, so
// "rated" means the user pressed "Rate it": after that we never ask again.
// We ask only people who really use Commit Hike (a week since install, commits
// on at least 3 days), only right after a commit moved them along the trail,
// at most 3 times, and never again after "Don't ask again".
package dev.commithike.core

import com.google.gson.Gson
import java.time.Instant
import java.time.ZoneOffset

data class RatingState(
    val firstSeen: Long = 0, // ms since epoch
    val activeDays: List<String> = emptyList(), // YYYY-MM-DD with progress, most recent last
    val status: String = "pending", // pending | rated | never
    val snoozedUntil: Long = 0,
    val asks: Int = 0,
)

object RatingPrompt {
    const val DAY_MS = 86_400_000L
    const val MIN_AGE_DAYS = 7
    const val MIN_ACTIVE_DAYS = 3
    const val MAX_ASKS = 3
    const val SNOOZE_DAYS = 7
    const val REVIEWS_URL = "https://plugins.jetbrains.com/plugin/34595-commit-hike/reviews"

    private val gson = Gson()

    fun initial(now: Long) = RatingState(firstSeen = now)

    /** Reads stored state; missing or damaged values start over instead of failing. */
    fun load(json: String?, now: Long): RatingState {
        val s = runCatching { gson.fromJson(json, RatingState::class.java) }.getOrNull()
        if (s == null || s.firstSeen <= 0) return initial(now)
        return s.copy(
            activeDays = s.activeDays.orEmpty().takeLast(30),
            status = if (s.status == "rated" || s.status == "never") s.status else "pending",
        )
    }

    fun save(s: RatingState): String = gson.toJson(s)

    /** Records a commit that moved the user along the trail today. */
    fun recordProgress(s: RatingState, now: Long): RatingState {
        val day = Instant.ofEpochMilli(now).atZone(ZoneOffset.UTC).toLocalDate().toString()
        if (s.activeDays.lastOrNull() == day) return s
        return s.copy(activeDays = (s.activeDays + day).takeLast(30))
    }

    fun shouldAsk(s: RatingState, now: Long): Boolean = s.status == "pending" &&
        now - s.firstSeen >= MIN_AGE_DAYS * DAY_MS &&
        s.activeDays.size >= MIN_ACTIVE_DAYS &&
        now >= s.snoozedUntil &&
        s.asks < MAX_ASKS

    /** Called when the prompt is shown: closing it without an answer counts as "later". */
    fun asked(s: RatingState, now: Long) = s.copy(asks = s.asks + 1, snoozedUntil = now + SNOOZE_DAYS * DAY_MS)

    fun answered(s: RatingState, answer: String): RatingState = when (answer) {
        "rate" -> s.copy(status = "rated")
        "never" -> s.copy(status = "never")
        else -> s // "later": asked() already snoozed it
    }
}
