package dev.commithike

import com.intellij.ide.util.PropertiesComponent
import com.intellij.notification.NotificationAction
import com.intellij.notification.NotificationGroupManager
import com.intellij.notification.NotificationType
import com.intellij.openapi.application.EDT
import com.intellij.openapi.components.Service
import com.intellij.openapi.diagnostic.logger
import com.intellij.openapi.fileEditor.FileEditorManager
import com.intellij.openapi.fileEditor.FileEditorManagerEvent
import com.intellij.openapi.fileEditor.FileEditorManagerListener
import com.intellij.openapi.project.Project
import com.intellij.openapi.startup.ProjectActivity
import com.intellij.openapi.ui.Messages
import com.intellij.openapi.util.IconLoader
import com.intellij.openapi.wm.ToolWindowManager
import com.intellij.util.messages.Topic
import dev.commithike.core.CoreException
import dev.commithike.core.ScanResult
import dev.commithike.core.Scope
import dev.commithike.core.Status
import dev.commithike.core.formatDistance
import dev.commithike.core.gitGlobalEmail
import dev.commithike.ui.RouteDialog
import dev.commithike.ui.SetupDialog
import git4idea.repo.GitRepository
import git4idea.repo.GitRepositoryChangeListener
import git4idea.repo.GitRepositoryManager
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import java.util.concurrent.ConcurrentHashMap
import java.util.concurrent.atomic.AtomicBoolean

private val LOG = logger<ProjectTrek>()

object CommitHikeIcons {
    @JvmField val Trail = IconLoader.getIcon("/icons/commit-hike.svg", CommitHikeIcons::class.java)
}

/** Published on the project bus (on the EDT) whenever [ProjectTrek.view] changes. */
fun interface TrailListener {
    fun trailChanged()

    companion object {
        @Topic.ProjectLevel
        val TOPIC = Topic(TrailListener::class.java, Topic.BroadcastDirection.NONE)
    }
}

/** What the status bar widget and the tool window render. */
data class TrailView(
    val state: String, // loading | not_initialized | ok | error
    val error: String? = null,
    val repo: String? = null,
    val status: Status? = null,
    val flash: String? = null, // "+86 m", shown briefly after a commit
)

class CommitHikeStartup : ProjectActivity {
    override suspend fun execute(project: Project) {
        project.getService(ProjectTrek::class.java).start()
    }
}

/**
 * Per project: detects commits through Git4Idea (HEAD changes), asks the core
 * to count them, and keeps the view state. Never writes to the repository.
 */
@Service(Service.Level.PROJECT)
class ProjectTrek(private val project: Project, private val cs: CoroutineScope) {
    @Volatile var view = TrailView("loading")
        private set

    private val app get() = CommitHikeApp.getInstance()
    private val started = AtomicBoolean(false)
    private val heads = ConcurrentHashMap<String, String>()
    private val pending = ConcurrentHashMap<String, Job>()

    @Volatile private var currentRepo: String? = null

    @Volatile private var flash: String? = null
    private var flashJob: Job? = null

    fun start() {
        if (!started.compareAndSet(false, true)) return
        val bus = project.messageBus.connect(cs)
        bus.subscribe(GitRepository.GIT_REPO_CHANGE, GitRepositoryChangeListener { onRepoChanged(it) })
        bus.subscribe(
            FileEditorManagerListener.FILE_EDITOR_MANAGER,
            object : FileEditorManagerListener {
                override fun selectionChanged(event: FileEditorManagerEvent) {
                    cs.launch { followActiveRepo() }
                }
            },
        )
        cs.launch {
            try {
                app.ensureLoaded()
            } catch (e: Exception) {
                LOG.warn("Commit Hike failed to start", e)
                publish(TrailView("error", error = e.message))
                return@launch
            }
            repositories().forEach { onRepoChanged(it) }
            followActiveRepo(force = true)
            if (!app.initialized) offerSetupOnce()
        }
    }

    fun requestRefresh() {
        cs.launch { refresh() }
    }

    // ---------- commit detection ----------

    private fun repositories(): List<GitRepository> = GitRepositoryManager.getInstance(project).repositories

    private fun onRepoChanged(repo: GitRepository) {
        val root = repo.root.path
        val head = repo.currentRevision ?: ""
        val previous = heads.put(root, head)
        if (previous != head) schedule(root) // first sighting also scans: catches commits made while the IDE was closed
    }

    // git touches several files per commit; coalesce into one scan.
    private fun schedule(root: String) {
        pending.remove(root)?.cancel()
        pending[root] = cs.launch {
            delay(800)
            pending.remove(root)
            scan(root)
        }
    }

    private suspend fun scan(root: String) {
        if (!app.loaded || !app.initialized) return
        try {
            celebrate(app.call { it.scan(root) })
        } catch (e: Exception) {
            LOG.warn("Commit Hike scan failed for $root", e)
        }
        app.refreshAllProjects()
    }

    private suspend fun followActiveRepo(force: Boolean = false) {
        val file = withContext(Dispatchers.EDT) { FileEditorManager.getInstance(project).selectedFiles.firstOrNull() }
        val manager = GitRepositoryManager.getInstance(project)
        val roots = manager.repositories.map { it.root.path }
        val repo = file?.let { manager.getRepositoryForFileQuick(it)?.root?.path }
            ?: currentRepo?.takeIf { it in roots }
            ?: roots.firstOrNull()
        if (force || repo != currentRepo) {
            currentRepo = repo
            refresh()
        }
    }

    // ---------- state ----------

    private suspend fun refresh() {
        if (!app.loaded) return
        val next = if (!app.initialized) {
            TrailView("not_initialized")
        } else {
            try {
                TrailView("ok", repo = currentRepo, status = app.call { it.status(currentRepo) })
            } catch (e: CoreException) {
                if (e.notInitialized) {
                    app.markInitialized(false)
                    TrailView("not_initialized")
                } else {
                    TrailView("error", error = e.message)
                }
            }
        }
        publish(next)
    }

    private suspend fun publish(next: TrailView) {
        view = next.copy(flash = flash)
        withContext(Dispatchers.EDT) {
            if (!project.isDisposed) project.messageBus.syncPublisher(TrailListener.TOPIC).trailChanged()
        }
    }

    private suspend fun celebrate(res: ScanResult) {
        if (res.addedM > 0) {
            flash = "+" + formatDistance(res.addedM)
            flashJob?.cancel()
            flashJob = cs.launch {
                delay(4000)
                flash = null
                publish(view)
            }
        }
        val events = res.events.orEmpty()
        // Achievements always get a notification of their own.
        for (e in events.filter { it.type == "achievement" }) {
            val a = e.achievement ?: continue
            notify("🏅 ${a.name ?: ""}", a.description ?: "", "Show trail") { showTrail() }
        }
        // Importing history can pass many stops at once: announce only the latest per journey.
        for (scope in Scope.entries) {
            val mine = events.filter { it.journey == scope.cli }
            val journey = if (scope == Scope.GLOBAL) res.global else res.project
            if (journey != null && mine.any { it.type == "finished" }) {
                notify("You finished ${journey.route.name}!", "Pick your next trail.", "Choose trail") { chooseRoute(scope) }
            } else {
                val w = mine.lastOrNull { it.type == "waypoint" }?.waypoint ?: continue
                notify(w.name, w.text ?: "You reached ${w.name}.", "Show trail") { showTrail() }
            }
        }
    }

    // ---------- user flows ----------

    fun showTrail() {
        cs.launch(Dispatchers.EDT) { ToolWindowManager.getInstance(project).getToolWindow(TOOL_WINDOW_ID)?.show() }
    }

    fun setup() = guarded {
        val email = withContext(Dispatchers.IO) { gitGlobalEmail() }
        val result = withContext(Dispatchers.EDT) {
            val dialog = SetupDialog(project, email)
            if (dialog.showAndGet()) dialog.result() else null
        } ?: return@guarded

        app.call { it.init(result.emails, result.mode, result.fromHistory) }
        app.markInitialized(true)
        val repo = currentRepo
        if (result.mode == "selected" && repo != null) {
            val count = withContext(Dispatchers.EDT) {
                Messages.showYesNoDialog(project, "Count commits in this project?", "Commit Hike", "Count It", "Not Now", null) ==
                    Messages.YES
            }
            if (count) app.call { it.setProjectEnabled(repo, true) }
        }
        for (r in repositories()) app.call { it.scan(r.root.path) }
        app.refreshAllProjects()
        showTrail()
    }

    fun chooseRoute(scope: Scope) = guarded {
        if (!app.initialized) {
            setup()
            return@guarded
        }
        val repo = currentRepo
        if (scope == Scope.PROJECT && repo == null) {
            notify("", "Open a file from a git repository to choose a trail for that project.")
            return@guarded
        }
        val canRemove = scope == Scope.PROJECT && view.status?.project != null
        val choice = withContext(Dispatchers.EDT) {
            val dialog = RouteDialog(project, scope, app.routes.values.sortedBy { it.name }, canRemove)
            if (dialog.showAndGet()) dialog.result() else null
        } ?: return@guarded
        app.call { it.setJourney(scope, choice.routeId, choice.fromHistory, if (scope == Scope.PROJECT) repo else null) }
        refresh()
    }

    fun setProjectEnabled(on: Boolean) = guarded {
        if (!app.initialized) {
            setup()
            return@guarded
        }
        val repo = currentRepo ?: run {
            notify("", "Open a file from a git repository first.")
            return@guarded
        }
        if (app.call { it.config() }.mode == "all") {
            notify("", "All your projects are already counted. To choose projects one by one, run Tools | Commit Hike | Set Up.")
            return@guarded
        }
        app.call { it.setProjectEnabled(repo, on) }
        if (on) scan(repo) else refresh()
    }

    fun verify() = guarded {
        val repo = currentRepo ?: return@guarded
        if (!app.initialized) return@guarded
        val r = app.call { it.verify(repo) }
        app.refreshAllProjects()
        val changes = r.added + r.updated + r.removed
        notify(
            "Recount finished",
            if (changes == 0) {
                "Everything already matched your git history."
            } else {
                "${r.added} added, ${r.updated} corrected, ${r.removed} removed."
            },
        )
    }

    private suspend fun offerSetupOnce() {
        val props = PropertiesComponent.getInstance()
        if (props.getBoolean(SETUP_OFFERED_KEY)) return
        props.setValue(SETUP_OFFERED_KEY, true)
        notify(
            "Commit Hike",
            "Turn your commits into a hiking journey. Commit Hike never writes to your projects and keeps everything on this computer.",
            "Set up",
        ) { setup() }
    }

    // ---------- helpers ----------

    private fun guarded(block: suspend CoroutineScope.() -> Unit): Job = cs.launch {
        try {
            block()
        } catch (e: Exception) {
            if (e is kotlinx.coroutines.CancellationException) throw e
            LOG.warn("Commit Hike action failed", e)
            notify("Commit Hike", e.message ?: e.toString(), type = NotificationType.ERROR)
        }
    }

    private fun notify(
        title: String,
        content: String,
        actionText: String? = null,
        type: NotificationType = NotificationType.INFORMATION,
        action: (() -> Unit)? = null,
    ) {
        val n = NotificationGroupManager.getInstance().getNotificationGroup(NOTIFICATION_GROUP)
            .createNotification(title, content, type)
        if (actionText != null && action != null) n.addAction(NotificationAction.createSimpleExpiring(actionText) { action() })
        n.notify(project)
    }

    companion object {
        const val TOOL_WINDOW_ID = "Commit Hike"
        const val NOTIFICATION_GROUP = "Commit Hike"
        private const val SETUP_OFFERED_KEY = "commit-hike.setupOffered"
    }
}
