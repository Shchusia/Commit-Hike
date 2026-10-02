package dev.commithike.core

/** Pure helpers for the developer report, so they are unit-tested. */
object Report {
    private val email = Regex("""[\w.+-]+@[\w-]+(\.[\w-]+)+""")

    /** Takes personal details out of a report: the home folder and e-mail addresses. */
    fun scrub(text: String, home: String?): String {
        val noHome = if (home != null && home.length > 1) text.replace(home, "~") else text
        return email.replace(noHome, "<email>")
    }
}
