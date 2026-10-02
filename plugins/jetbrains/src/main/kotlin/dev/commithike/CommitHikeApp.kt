package dev.commithike

import com.intellij.DynamicBundle
import com.intellij.openapi.application.PathManager
import com.intellij.openapi.components.Service
import com.intellij.openapi.components.service
import com.intellij.openapi.project.ProjectManager
import dev.commithike.core.Avatar
import dev.commithike.core.BuildInfo
import dev.commithike.core.CoreCli
import dev.commithike.core.CoreException
import dev.commithike.core.CorePlatform
import dev.commithike.core.I18n
import dev.commithike.core.LocaleInfo
import dev.commithike.core.Route
import dev.commithike.core.RouteAssets
import dev.commithike.core.Status
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.sync.Mutex
import kotlinx.coroutines.sync.withLock
import kotlinx.coroutines.withContext
import java.nio.file.Files
import java.nio.file.Path
import java.nio.file.Paths
import java.nio.file.StandardCopyOption
import java.util.concurrent.ConcurrentHashMap

/**
 * One per IDE: owns the core binary and state shared by all open projects.
 * CLI calls are serialized so this IDE never contends with itself for the
 * core's file lock (other IDEs are handled by the lock itself).
 */
@Service(Service.Level.APP)
class CommitHikeApp {
    private val mutex = Mutex()

    // The last errors from any project and from the panel, for the developer report.
    private val errors = ArrayDeque<String>()

    fun noteError(message: String) = synchronized(errors) {
        errors.addLast("${java.time.Instant.now()} $message".take(600))
        while (errors.size > 10) errors.removeFirst()
    }

    fun recentErrors(): List<String> = synchronized(errors) { errors.toList() }
    private var cli: CoreCli? = null

    @Volatile var loaded = false
        private set

    @Volatile var initialized = false
        private set

    @Volatile var avatar: Avatar = Avatar()
        private set

    @Volatile var routes: Map<String, Route> = emptyMap()
        private set

    /** The language setting; null until the core is set up. */
    @Volatile var localeInfo: LocaleInfo? = null
        private set

    /** Pictures of route objects and custom maps, per route id. They change only on import. */
    private val assets = ConcurrentHashMap<String, RouteAssets>()

    suspend fun <T> call(block: (CoreCli) -> T): T = mutex.withLock {
        withContext(Dispatchers.IO) {
            val c = cli ?: CoreCli(BinaryLocator.locate()).also { cli = it }
            // Route texts follow the IDE language (Ukrainian with the language pack installed).
            c.lang = DynamicBundle.getLocale().toLanguageTag()
            block(c)
        }
    }

    /** Loads routes and checks whether first-run setup is needed. Idempotent. */
    suspend fun ensureLoaded() {
        if (loaded) return
        I18n.setLanguage(DynamicBundle.getLocale().toLanguageTag()) // until the core says otherwise
        routes = call { it.routes() }.associateBy { it.id }
        avatar = call { it.avatar() }
        initialized = try {
            call { it.config() }
            true
        } catch (e: CoreException) {
            if (e.notInitialized) false else throw e
        }
        if (initialized) reloadLocale()
        loaded = true
    }

    /** Reads (or, with [set], changes) the language and applies it to plugin texts. */
    suspend fun reloadLocale(set: String? = null) {
        localeInfo = call { it.locale(set) }
        I18n.setLanguage(localeInfo?.effective)
    }

    /** Fetches pictures for the routes a status shows, once per route. */
    suspend fun ensureAssets(status: Status?) {
        for (id in listOfNotNull(status?.global?.route?.id, status?.project?.route?.id)) {
            if (assets.containsKey(id)) continue
            try {
                assets[id] = call { it.routeAssets(id) }
            } catch (e: CoreException) {
                com.intellij.openapi.diagnostic.logger<CommitHikeApp>().warn("Commit Hike: no assets for $id", e)
            }
        }
    }

    fun assetsFor(status: Status?): Map<String, RouteAssets> =
        listOfNotNull(status?.global?.route?.id, status?.project?.route?.id).mapNotNull { id -> assets[id]?.let { id to it } }.toMap()

    /** Re-reads the hiker icon after it was changed (in this or another IDE). */
    suspend fun reloadAvatar() {
        avatar = call { it.avatar() }
    }

    /** Re-reads the route list, e.g. after importing or removing a route. */
    suspend fun reloadRoutes() {
        routes = call { it.routes() }.associateBy { it.id }
        assets.clear() // an imported route may bring new pictures
    }

    fun markInitialized(value: Boolean) {
        initialized = value
    }

    /** Progress is global: a commit in one project changes what every window shows. */
    fun refreshAllProjects() {
        for (p in ProjectManager.getInstance().openProjects) {
            if (!p.isDisposed) p.service<ProjectTrek>().requestRefresh()
        }
    }

    companion object {
        fun getInstance(): CommitHikeApp = service()
    }
}

/**
 * The core binaries ship inside the plugin jar (resources /bin/...). A file
 * inside a jar can't be executed, so the right one is copied once per plugin
 * version into the IDE's system directory.
 */
internal object BinaryLocator {
    fun locate(): Path {
        System.getenv("COMMIT_HIKE_BINARY")?.let { return Paths.get(it) } // for development
        val name = CorePlatform.binaryName()
        val bytes = BinaryLocator::class.java.getResourceAsStream("/bin/$name")?.use { it.readBytes() }
            ?: throw CoreException("This build of Commit Hike doesn't include $name.")
        val version = BuildInfo.current.version // one copy per plugin version
        val target = Paths.get(PathManager.getSystemPath(), "commit-hike", version, name)

        val upToDate = Files.isRegularFile(target) &&
            Files.size(target) == bytes.size.toLong() &&
            Files.readAllBytes(target).contentEquals(bytes)
        if (!upToDate) {
            Files.createDirectories(target.parent)
            val tmp = Files.createTempFile(target.parent, "commit-hike", ".tmp")
            try {
                Files.write(tmp, bytes)
                tmp.toFile().setExecutable(true, true)
                Files.move(tmp, target, StandardCopyOption.REPLACE_EXISTING, StandardCopyOption.ATOMIC_MOVE)
            } catch (e: Exception) {
                Files.deleteIfExists(tmp)
                // On Windows another IDE may be running the old copy; keep using it if present.
                if (!Files.isRegularFile(target)) throw e
            }
        }
        target.toFile().setExecutable(true, true)
        return target
    }
}
