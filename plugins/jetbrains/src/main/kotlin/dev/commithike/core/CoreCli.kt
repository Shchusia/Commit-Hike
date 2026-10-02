// Talks to the commit-hike core binary. Uses no IntelliJ APIs, so it can be
// unit-tested with plain JUnit. All calls block: run them off the EDT.
package dev.commithike.core

import com.google.gson.FieldNamingPolicy
import com.google.gson.GsonBuilder
import com.google.gson.JsonElement
import com.google.gson.JsonParser
import com.google.gson.reflect.TypeToken
import java.lang.reflect.Type
import java.nio.file.Path
import java.util.concurrent.TimeUnit
import kotlin.concurrent.thread
import kotlin.math.roundToInt

class CoreException(message: String, val code: String? = null) : Exception(message) {
    val notInitialized: Boolean get() = code == "not_initialized"
}

class CoreCli(
    private val binary: Path,
    private val timeoutSeconds: Long = 300, // first scan of a huge repo can be slow
    private val extraEnv: Map<String, String> = emptyMap(),
) {
    /** Language for route texts, e.g. "uk" or "en". */
    @Volatile var lang: String = "en"

    private val gson = GsonBuilder()
        .setFieldNamingPolicy(FieldNamingPolicy.LOWER_CASE_WITH_UNDERSCORES)
        .create()

    fun config(): CoreConfig = call(listOf("config"), CoreConfig::class.java)

    fun routes(): List<Route> = call(listOf("routes") + langArgs(), object : TypeToken<List<Route>>() {}.type)

    /**
     * Status for the plugin's own use, plus the untouched JSON in [Status.raw] for
     * the trail panel: fields the Kotlin models don't know about still reach it.
     */
    fun status(repo: String? = null): Status {
        val raw = callRaw(listOf("status") + repoArgs(repo) + langArgs())
        return gson.fromJson(raw, Status::class.java).copy(raw = raw)
    }

    /** [prevHead]: the HEAD seen before this change; if it's no longer an ancestor the core recounts in full. */
    fun scan(repo: String, prevHead: String? = null): ScanResult = call(
        listOf("scan", "--repo", repo) + (if (prevHead.isNullOrEmpty()) emptyList() else listOf("--prev-head", prevHead)) + langArgs(),
        ScanResult::class.java,
    )

    /** Everyone who commits to the project, on the same route. Computed from git history, never stored. */
    fun team(repo: String): Team = call(listOf("team", "--repo", repo), Team::class.java)

    fun setTeam(repo: String, on: Boolean) {
        call<JsonElement>(listOf("project", if (on) "team-on" else "team-off", "--repo", repo), JsonElement::class.java)
    }

    /** Reads the language setting; with [set] ("auto", "en", "uk"…) changes it for every IDE. */
    fun locale(set: String? = null): LocaleInfo =
        call(listOf("locale") + (if (set == null) emptyList() else listOf("--set", set)) + langArgs(), LocaleInfo::class.java)

    /** Pictures (data URLs) and static HTML of a route's objects and map. */
    fun routeAssets(id: String): RouteAssets = call(listOf("route", "assets", "--id", id), RouteAssets::class.java)

    fun verify(repo: String): VerifyResult = call(listOf("verify", "--repo", repo), VerifyResult::class.java)

    fun init(emails: List<String>, mode: String, fromHistory: Boolean, difficulty: String? = null): CoreConfig = call(
        listOf("init", "--email", emails.joinToString(","), "--mode", mode, "--from-history=$fromHistory") +
            (if (difficulty == null) emptyList() else listOf("--difficulty", difficulty)),
        CoreConfig::class.java,
    )

    /** Reads the difficulty; with [set] (easy, medium, hard) changes it for commits from now on. */
    /** Weekdays off (0 = Sunday … 6 = Saturday); [set] like "sat,sun" or "none" changes them. */
    fun restDays(set: String? = null): RestDaysInfo =
        call(listOf("rest-days") + (if (set == null) emptyList() else listOf("--set", set)), RestDaysInfo::class.java)

    /** Changes the given settings (null = keep) and returns them all. */
    fun settings(reduceMotion: String? = null, highContrast: String? = null, notifications: String? = null): Settings = call(
        listOf("settings") +
            listOfNotNull(
                reduceMotion?.let {
                    "--reduce-motion=$it"
                },
                highContrast?.let { "--high-contrast=$it" },
                notifications?.let { "--notifications=$it" },
            ),
        Settings::class.java,
    )

    /** An SVG badge for a README, for the journey the panel shows. */
    fun badge(repo: String? = null): BadgeResult = call(listOf("badge") + repoArgs(repo) + langArgs(), BadgeResult::class.java)

    /** Versions and counts for a bug report, pretty-printed: nothing personal. */
    fun diagnosticsJson(): String = GsonBuilder().setPrettyPrinting().create().toJson(callRaw(listOf("diagnostics")))

    fun exportBackup(path: String): BackupResult = call(listOf("backup", "export", "--path", path), BackupResult::class.java)

    /** Throws [CoreException] with code "data_exists" when there is progress and [replace] is false. */
    fun importBackup(path: String, replace: Boolean): BackupResult =
        call(listOf("backup", "import", "--path", path) + (if (replace) listOf("--replace") else emptyList()), BackupResult::class.java)

    fun difficulty(set: String? = null): DifficultyInfo =
        call(listOf("difficulty") + (if (set == null) emptyList() else listOf("--set", set)), DifficultyInfo::class.java)

    fun setJourney(scope: Scope, route: String, fromHistory: Boolean, repo: String? = null): Status = call(
        listOf("journey", "--scope", scope.cli, "--route", route, "--from-history=$fromHistory") + repoArgs(repo) + langArgs(),
        Status::class.java,
    )

    fun importRoute(path: String, replace: Boolean): Route = call(
        listOf("route", "import", "--path", path) + (if (replace) listOf("--replace") else emptyList()) + langArgs(),
        Route::class.java,
    )

    fun removeRoute(id: String) {
        call<JsonElement>(listOf("route", "remove", "--id", id), JsonElement::class.java)
    }

    /** Writes a template route pack to dir/id and returns its folder. */
    fun routeTemplate(id: String, dir: String): String =
        call<JsonElement>(listOf("route", "template", "--id", id, "--path", dir), JsonElement::class.java)
            .asJsonObject.get("path").asString

    fun avatar(): Avatar = call(listOf("avatar", "get"), Avatar::class.java)

    fun setAvatar(pngPath: String): Avatar = call(listOf("avatar", "set", "--path", pngPath), Avatar::class.java)

    fun resetAvatar() {
        call<JsonElement>(listOf("avatar", "reset"), JsonElement::class.java)
    }

    fun setProjectEnabled(repo: String, on: Boolean) {
        call<JsonElement>(listOf("project", if (on) "enable" else "disable", "--repo", repo), JsonElement::class.java)
    }

    private fun repoArgs(repo: String?) = if (repo == null) emptyList() else listOf("--repo", repo)

    private fun langArgs() = listOf("--lang", lang)

    private fun <T> call(args: List<String>, type: Type): T = gson.fromJson(callRaw(args), type)

    /** Runs the core and returns the "data" of its envelope, or throws [CoreException]. */
    private fun callRaw(args: List<String>): JsonElement {
        val pb = ProcessBuilder(listOf(binary.toString()) + args)
        pb.environment().putAll(extraEnv)
        val process = pb.start()
        process.outputStream.close()

        // Drain both streams concurrently so a full pipe can never block the child.
        var stdout = ""
        var stderr = ""
        val outReader = thread(name = "commit-hike-stdout") { stdout = process.inputStream.readBytes().toString(Charsets.UTF_8) }
        val errReader = thread(name = "commit-hike-stderr") { stderr = process.errorStream.readBytes().toString(Charsets.UTF_8) }
        if (!process.waitFor(timeoutSeconds, TimeUnit.SECONDS)) {
            process.destroyForcibly()
            throw CoreException("The core did not answer within $timeoutSeconds seconds.")
        }
        outReader.join()
        errReader.join()

        // Every response is an envelope: {"api":1,"ok":true,"data":...} or {"api":1,"ok":false,"error":{...}}
        val env = try {
            JsonParser.parseString(stdout).asJsonObject
        } catch (e: Exception) {
            throw CoreException(stderr.trim().ifEmpty { "The core returned invalid output." })
        }
        val api = env.get("api")?.asInt
        if (api != API_VERSION) {
            throw CoreException("Core speaks protocol $api, this plugin expects $API_VERSION.", "protocol_mismatch")
        }
        if (env.get("ok")?.asBoolean != true) {
            val err = env.getAsJsonObject("error")
            throw CoreException(err?.get("message")?.asString ?: "Unknown error", err?.get("code")?.asString)
        }
        return env.get("data") ?: com.google.gson.JsonNull.INSTANCE
    }
}

/** Maps the JVM's OS/CPU names to the bundled binary, e.g. commit-hike-darwin-arm64. */
object CorePlatform {
    fun binaryName(osName: String = System.getProperty("os.name"), osArch: String = System.getProperty("os.arch")): String {
        val os = osName.lowercase().let {
            when {
                it.startsWith("windows") -> "windows"
                it.startsWith("mac") || it.startsWith("darwin") -> "darwin"
                it.startsWith("linux") -> "linux"
                else -> throw CoreException("Commit Hike doesn't support $osName yet.")
            }
        }
        val arch = when (osArch.lowercase()) {
            "amd64", "x86_64", "x64" -> "amd64"
            "aarch64", "arm64" -> "arm64"
            else -> throw CoreException("Commit Hike doesn't support $osArch processors yet.")
        }
        return "commit-hike-$os-$arch" + if (os == "windows") ".exe" else ""
    }
}

fun formatDistance(m: Double, lang: String = "en"): String {
    val uk = lang == "uk"
    if (m < 1000) return "${m.roundToInt()} " + if (uk) "м" else "m"
    val km = m / 1000
    val unit = if (uk) "км" else "km"
    if (km >= 100) return "${km.roundToInt()} $unit"
    val s = "%.1f".format(java.util.Locale.ROOT, km)
    return (if (uk) s.replace('.', ',') else s) + " " + unit
}

/** `git config --global user.email`, or "" when git or the setting is missing. */
fun gitGlobalEmail(): String = try {
    val p = ProcessBuilder("git", "config", "--global", "user.email").redirectErrorStream(true).start()
    val out = p.inputStream.readBytes().toString(Charsets.UTF_8).trim()
    if (p.waitFor(10, TimeUnit.SECONDS) && p.exitValue() == 0) out else ""
} catch (e: Exception) {
    ""
}
