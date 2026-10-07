package dev.commithike.core

import com.google.gson.JsonObject
import java.net.URI

/**
 * The routes website (commit-hike.dev): its address, the pages the panel
 * may open there, and the panel's search turned into core arguments.
 */
object Site {
    const val DEFAULT = "https://commit-hike.dev"
    private val origin = Regex("^https?://[^/\\s]+$")
    private val page = Regex("^/(routes/[a-z0-9-]+|u/\\d+)?$")
    val routeId = Regex("^[a-z0-9]+(-[a-z0-9]+)*$")
    private val oneOf = mapOf(
        "kind" to setOf("real", "story"),
        "length" to setOf("s", "m", "l", "xl"),
        "sort" to setOf("top", "rating", "rating_asc", "popular", "downloads_asc", "new", "short", "long"),
    )

    /** The site; COMMIT_HIKE_SITE overrides it (the core reads the same variable). */
    fun url(env: Map<String, String> = System.getenv()): String {
        val v = env["COMMIT_HIKE_SITE"].orEmpty().trim().trimEnd('/')
        return if (origin.matches(v)) v else DEFAULT
    }

    /** Whether [link] is a route or author page of the site (opened from a route card). */
    fun isPage(link: String, base: String = url()): Boolean {
        if (link.length > 2000) return false
        return try {
            val u = URI(link)
            val b = URI(base)
            u.scheme == b.scheme && u.host == b.host && u.port == b.port && u.userInfo == null && page.matches(u.rawPath.orEmpty())
        } catch (e: Exception) {
            false
        }
    }

    /** The panel's search as arguments for `commit-hike site routes`; anything odd is dropped. */
    fun searchArgs(query: JsonObject?): List<String> {
        val args = mutableListOf("site", "routes")
        val q = query ?: return args
        fun str(key: String) = q.get(key)?.takeIf { it.isJsonPrimitive && it.asJsonPrimitive.isString }?.asString
        str("q")?.trim()?.takeIf { it.isNotEmpty() }?.let { args += listOf("--q", it.take(100)) }
        for ((key, allowed) in oneOf) str(key)?.takeIf { it in allowed }?.let { args += listOf("--$key", it) }
        val page = q.get("page")?.takeIf { it.isJsonPrimitive && it.asJsonPrimitive.isNumber }?.asDouble
        if (page != null && page == Math.floor(page) && page > 1 && page < 10_000) args += listOf("--page", page.toInt().toString())
        return args
    }
}
