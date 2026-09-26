package dev.commithike

import com.intellij.DynamicBundle
import com.intellij.ide.plugins.PluginManagerCore
import com.intellij.openapi.application.PathManager
import com.intellij.openapi.components.Service
import com.intellij.openapi.components.service
import com.intellij.openapi.extensions.PluginId
import com.intellij.openapi.project.ProjectManager
import dev.commithike.core.CoreCli
import dev.commithike.core.CoreException
import dev.commithike.core.CorePlatform
import dev.commithike.core.Route
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.sync.Mutex
import kotlinx.coroutines.sync.withLock
import kotlinx.coroutines.withContext
import java.nio.file.Files
import java.nio.file.Path
import java.nio.file.Paths
import java.nio.file.StandardCopyOption

/**
 * One per IDE: owns the core binary and state shared by all open projects.
 * CLI calls are serialized so this IDE never contends with itself for the
 * core's file lock (other IDEs are handled by the lock itself).
 */
@Service(Service.Level.APP)
class CommitHikeApp {
    private val mutex = Mutex()
    private var cli: CoreCli? = null

    @Volatile var loaded = false
        private set

    @Volatile var initialized = false
        private set

    @Volatile var routes: Map<String, Route> = emptyMap()
        private set

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
        routes = call { it.routes() }.associateBy { it.id }
        initialized = try {
            call { it.config() }
            true
        } catch (e: CoreException) {
            if (e.notInitialized) false else throw e
        }
        loaded = true
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
    private const val PLUGIN_ID = "dev.commithike"

    fun locate(): Path {
        System.getenv("COMMIT_HIKE_BINARY")?.let { return Paths.get(it) } // for development
        val name = CorePlatform.binaryName()
        val bytes = BinaryLocator::class.java.getResourceAsStream("/bin/$name")?.use { it.readBytes() }
            ?: throw CoreException("This build of Commit Hike doesn't include $name.")
        val version = PluginManagerCore.getPlugin(PluginId.getId(PLUGIN_ID))?.version ?: "dev"
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
