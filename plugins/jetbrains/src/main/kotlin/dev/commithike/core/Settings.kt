package dev.commithike.core

/** Weekdays off as the core's `rest-days --set` wants them. */
object RestDays {
    private val names = listOf("sun", "mon", "tue", "wed", "thu", "fri", "sat")

    /** 0 = Sunday … 6 = Saturday -> "sat,sun"-style, or "none". */
    fun arg(days: List<Int>): String = days.filter { it in 0..6 }.distinct().sorted().joinToString(",") { names[it] }.ifEmpty { "none" }
}

/** The last errors from any project and from the panel, for the developer report. */
class ErrorLog(private val keep: Int = 10, private val clock: () -> String = { java.time.Instant.now().toString() }) {
    private val errors = ArrayDeque<String>()

    fun note(message: String) = synchronized(errors) {
        errors.addLast("${clock()} $message".take(600))
        while (errors.size > keep) errors.removeFirst()
    }

    fun recent(): List<String> = synchronized(errors) { errors.toList() }
}
