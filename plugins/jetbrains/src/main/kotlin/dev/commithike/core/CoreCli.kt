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

    fun status(repo: String? = null): Status = call(listOf("status") + repoArgs(repo) + langArgs(), Status::class.java)

    fun scan(repo: String): ScanResult = call(listOf("scan", "--repo", repo) + langArgs(), ScanResult::class.java)

    fun verify(repo: String): VerifyResult = call(listOf("verify", "--repo", repo), VerifyResult::class.java)

    fun init(emails: List<String>, mode: String, fromHistory: Boolean): CoreConfig = call(
        listOf("init", "--email", emails.joinToString(","), "--mode", mode, "--from-history=$fromHistory"),
        CoreConfig::class.java,
    )

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

    private fun <T> call(args: List<String>, type: Type): T {
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
        val json: JsonElement = env.get("data") ?: com.google.gson.JsonNull.INSTANCE
        return gson.fromJson(json, type)
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

fun formatDistance(m: Double): String {
    if (m < 1000) return "${m.roundToInt()} m"
    val km = m / 1000
    return if (km < 100) "%.1f km".format(java.util.Locale.ROOT, km) else "${km.roundToInt()} km"
}

/** `git config --global user.email`, or "" when git or the setting is missing. */
fun gitGlobalEmail(): String = try {
    val p = ProcessBuilder("git", "config", "--global", "user.email").redirectErrorStream(true).start()
    val out = p.inputStream.readBytes().toString(Charsets.UTF_8).trim()
    if (p.waitFor(10, TimeUnit.SECONDS) && p.exitValue() == 0) out else ""
} catch (e: Exception) {
    ""
}
