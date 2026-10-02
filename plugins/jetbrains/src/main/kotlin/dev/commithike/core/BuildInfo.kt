package dev.commithike.core

import java.io.InputStream
import java.util.Properties

/**
 * Version and flavor baked into the plugin by the Gradle build (writeBuildInfo).
 * Read from a resource instead of asking the platform, whose plugin lookup is internal API.
 */
data class BuildInfo(val version: String, val flavor: String, val repo: String = "") {
    companion object {
        const val RESOURCE = "/commit-hike-build.properties"

        val current: BuildInfo by lazy { load(BuildInfo::class.java.getResourceAsStream(RESOURCE)) }

        /** Missing or broken data gives "dev" and the prod flavor. */
        fun load(input: InputStream?): BuildInfo {
            val p = Properties()
            input?.use { runCatching { p.load(it) } }
            val flavor = p.getProperty("flavor", "prod").let { if (it == "dev") "dev" else "prod" }
            return BuildInfo(p.getProperty("version")?.takeIf { it.isNotBlank() } ?: "dev", flavor, p.getProperty("repo", "").trim())
        }
    }
}
