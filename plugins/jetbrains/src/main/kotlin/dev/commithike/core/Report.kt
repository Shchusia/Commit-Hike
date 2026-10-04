package dev.commithike.core

/** Pure helpers for the developer report, so they are unit-tested. */
object Report {
    private val email = Regex("""[\w.+-]+@[\w-]+(\.[\w-]+)+""")

    /** Takes personal details out of a report: the home folder and e-mail addresses. */
    fun scrub(text: String, home: String?): String {
        val noHome = if (home != null && home.length > 1) text.replace(home, "~") else text
        return email.replace(noHome, "<email>")
    }

    /** The whole report for a GitHub issue, with personal details taken out. */
    fun build(plugin: String, ide: String, os: String, coreJson: String, errors: List<String>, home: String?): String = scrub(
        (
            listOf("Commit Hike: report for the developer", "Plugin: $plugin", "IDE: $ide", "OS: $os") +
                listOf("", "Core:", "```json", coreJson, "```", "", "Recent errors:") +
                errors.ifEmpty { listOf("none") }.map { "- $it" }
            ).joinToString("\n"),
        home,
    )
}
